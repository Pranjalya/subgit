package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"golang.org/x/sync/semaphore"
)

var (
	version = "dev" // Default value, can be overwritten by ldflags
	commit  = "none"
	date    = "unknown"
)

type GithubFetcher struct {
	RepoName    string
	Branch      string
	Subfolder   string
	RootDir     string
	VerifySSL   bool
	PATToken    string       // GitHub Personal Access Token
	Client      *http.Client // Use http.Client directly
	ProgressBar *pb.ProgressBar
}

func NewGithubFetcher(repoName, branch, subfolder, rootDir string, verifySSL bool, patToken string) *GithubFetcher {
	// Configure the HTTP client with TLS verification options.
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !verifySSL, // Disable verification if verifySSL is false
	}

	// Create a transport with the TLS configuration.
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Create a new HTTP client using the transport.
	client := &http.Client{
		Transport: transport,
	}
	return &GithubFetcher{
		RepoName:    repoName,
		Branch:      branch,
		Subfolder:   subfolder,
		RootDir:     rootDir,
		VerifySSL:   verifySSL,
		PATToken:    patToken,
		Client:      client,
		ProgressBar: nil,
	}
}

func (gf *GithubFetcher) GetFileContent(filepath string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/refs/heads/%s/%s", gf.RepoName, gf.Branch, filepath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	if gf.PATToken != "" {
		req.Header.Set("Authorization", "token "+gf.PATToken)
	}

	resp, err := gf.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error %d for %s", resp.StatusCode, url)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading body from %s: %w", url, err)
	}

	return string(bodyBytes), nil
}

func (gf *GithubFetcher) SaveFileContent(filepath_ string, content string) error {
	fullPath := filepath.Join(gf.RootDir, filepath_)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, os.ModeDir|0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dir, err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", fullPath, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", fullPath, err)
	}

	return nil
}

func (gf *GithubFetcher) ProcessFile(filepath string, wg *sync.WaitGroup, sem *semaphore.Weighted) {
	defer wg.Done()

	err := sem.Acquire(context.Background(), 1)
	if err != nil {
		log.Printf("Failed to acquire semaphore: %v\n", err)
		return
	}
	defer sem.Release(1)

	content, err := gf.GetFileContent(filepath)
	if err != nil {
		log.Println(err) // Log the error, but continue processing other files.
		return
	}

	if err := gf.SaveFileContent(filepath, content); err != nil {
		log.Println(err)
		return
	}

	gf.ProgressBar.Increment()
}

func (gf *GithubFetcher) FetchFiles() error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", gf.RepoName, gf.Branch)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	if gf.PATToken != "" {
		req.Header.Set("Authorization", "token "+gf.PATToken)
	}

	resp, err := gf.Client.Do(req)

	if err != nil {
		return fmt.Errorf("error fetching tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error %d for %s", resp.StatusCode, url)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %w", err)
	}

	var treeResponse struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}

	if err := json.Unmarshal(bodyBytes, &treeResponse); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	filesToFetch := []string{}
	for _, item := range treeResponse.Tree {
		if strings.HasPrefix(item.Path, gf.Subfolder) && item.Type == "blob" {
			filesToFetch = append(filesToFetch, item.Path)
		}
	}

	totalFiles := len(filesToFetch)
	if totalFiles == 0 {
		fmt.Println("No files found matching the criteria.")
		return nil
	}

	gf.ProgressBar = pb.StartNew(totalFiles)
	gf.ProgressBar.Set(pb.SIBytesPrefix, true)

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(8) // Limit concurrency to 8 (adjust as needed)
	for _, filepath := range filesToFetch {
		wg.Add(1)
		go gf.ProcessFile(filepath, &wg, sem)
	}

	wg.Wait()
	gf.ProgressBar.Finish()

	return nil
}

func ParseGithubURL(githubURL string) (string, string, string, error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", "", "", fmt.Errorf("error parsing URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 4 {
		return "", "", "", fmt.Errorf("invalid GitHub URL format")
	}

	repoName := path.Join(pathParts[0], pathParts[1])
	branch := pathParts[3]
	subfolder := strings.Join(pathParts[4:], "/")

	return repoName, branch, subfolder, nil
}

func main() {
	fmt.Printf("subgit - Version: %s, Commit: %s, Date: %s\n", version, commit, date)

	githubURL := flag.String("url", "", "GitHub URL to the subdirectory (e.g., https://github.com/user/repo/tree/branch/subfolder)")
	rootDir := flag.String("root_dir", "", "Local directory to save the files")
	noVerifySSL := flag.Bool("no-verify-ssl", false, "Disable SSL certificate verification (not recommended)")
	patToken := flag.String("pat-token", "", "GitHub Personal Access Token (PAT)")
	flag.Parse()

	if *githubURL == "" || *rootDir == "" {
		fmt.Println("Please provide both -url and -root_dir arguments.")
		flag.Usage()
		os.Exit(1)
	}

	repoName, branch, subfolder, err := ParseGithubURL(*githubURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fetcher := NewGithubFetcher(repoName, branch, subfolder, *rootDir, !*noVerifySSL, *patToken)

	if err := fetcher.FetchFiles(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Files downloaded successfully!")
}
