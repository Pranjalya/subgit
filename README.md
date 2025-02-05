# subgit

Downloads a subdirectory from a GitHub repository. This tool allows you to fetch only the specific files you need from a larger repository, saving time and bandwidth.

## Features

*   **Selective Download:** Downloads only the specified subdirectory from a GitHub repository.
*   **Cross-Platform:** Built with Go, providing binaries for Linux, macOS, and Windows.
*   **Concurrency:** Uses concurrency to speed up the download process.
*   **SSL Verification:** Supports SSL certificate verification (enabled by default).
*   **GitHub Personal Access Token (PAT):**  Option to use a PAT for accessing private repositories or to increase rate limits.

## Installation

### From Binaries

Download the pre-built binaries for your operating system from the [Releases](https://github.com/pranjalya/subgit/releases) page.  Extract the archive and place the `subgit` executable in a directory included in your system's `PATH`.


### Using Go

```bash
go install github.com/pranjalya/subgit@latest
```

## Usage

```bash
subgit -url <github_url> -root_dir <local_directory> [options]
```

**Arguments:**

*   `-url`:  GitHub URL to the subdirectory (e.g., `https://github.com/user/repo/tree/branch/subfolder`).
*   `-root_dir`: Local directory to save the files.

**Options:**

*   `-no-verify-ssl`: Disable SSL certificate verification (not recommended).
*   `-pat-token`: GitHub Personal Access Token (PAT).

**Example:**

```bash
subgit -url https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/api -root_dir ./k8s_api
```

This command will download the contents of the `staging/src/k8s.io/api` subdirectory from the `kubernetes/kubernetes` repository and save them to the `./k8s_api` directory.

**Using a PAT (Personal Access Token):**

To access private repositories or increase rate limits, provide a PAT:

```bash
subgit -url https://github.com/private_org/private_repo/tree/main/my_subfolder -root_dir ./my_subfolder -pat-token <your_pat>
```

**Disabling SSL Verification (Not Recommended):**

```bash
subgit -url https://github.com/user/repo/tree/branch/subfolder -root_dir ./my_folder -no-verify-ssl
```

## Building from Source

1.  **Install Go:** Make sure you have Go installed (version 1.21 or later).
2.  **Clone the Repository:**

    ```bash
    git clone https://github.com/pranjalya/subgit.git
    cd subgit
    ```

3.  **Build the Application:**

    ```bash
    cd golang
    go build -o subgit .
    ```

    This will create an executable named `subgit` in the `golang` directory.

## Contributing

Contributions are welcome! Please feel free to submit pull requests.

## License

[Apache 2.0 License](LICENSE)