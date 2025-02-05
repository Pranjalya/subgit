import argparse
import os
import asyncio
import aiohttp
from urllib.parse import urlparse
from tqdm import tqdm
import aiohttp.client_exceptions

class GithubFetcher:
    def __init__(self, reponame, branch, subfolder, root_dir, verify_ssl=True, pat_token=None):
        self.reponame = reponame
        self.branch = branch
        self.subfolder = subfolder
        self.root_dir = root_dir
        self.total_files = 0
        self.progress_bar = None
        self.verify_ssl = verify_ssl
        self.pat_token = pat_token
        self.headers = {}
        if self.pat_token:
            self.headers = {"Authorization": f"token {self.pat_token}"}

    async def get_file_content(self, session, filepath):
        url = f"https://raw.githubusercontent.com/{self.reponame}/refs/heads/{self.branch}/{filepath}"
        try:
            async with session.get(url, ssl=self.verify_ssl, headers=self.headers) as response:
                if response.status == 200:
                    return await response.text()
                else:
                    print(f"Error {response.status} for {url}")
                    return None
        except aiohttp.ClientError as e:
            print(f"Error fetching {url}: {e}")
            return None
        except aiohttp.client_exceptions.ClientConnectorError as e:
            print(f"Connection error for {url}: {e}")
            return None

    def save_file_content(self, filepath, content):
        full_path = os.path.join(self.root_dir, filepath)
        os.makedirs(os.path.dirname(full_path), exist_ok=True)
        with open(full_path, 'w') as file:
            file.write(content)

    async def process_file(self, session, item):
        filepath = item['path']
        content = await self.get_file_content(session, filepath)
        if content:
            self.save_file_content(filepath, content)
        self.progress_bar.update(1)  # Update progress bar
        return True

    async def fetch_files(self):
        url = f"https://api.github.com/repos/{self.reponame}/git/trees/{self.branch}?recursive=1"
        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(url, ssl=self.verify_ssl, headers=self.headers) as response: # And also here
                    if response.status == 200:
                        tree = (await response.json())['tree']
                        files_to_fetch = [item for item in tree if item['path'].startswith(self.subfolder) and item['type'] == 'blob']
                        self.total_files = len(files_to_fetch)

                        if self.total_files == 0:
                            print("No files found matching the criteria.")
                            return

                        self.progress_bar = tqdm(total=self.total_files, desc="Downloading Files")

                        tasks = [self.process_file(session, item) for item in files_to_fetch]
                        await asyncio.gather(*tasks)
                        self.progress_bar.close()

                    else:
                        print(f"Error {response.status} for {url}")
        except aiohttp.ClientError as e:
            print(f"Error fetching tree: {e}")
        except aiohttp.client_exceptions.ClientConnectorError as e:
            print(f"Connection error for tree: {e}")

def parse_github_url(url):
    parsed_url = urlparse(url)
    path_parts = parsed_url.path.strip('/').split('/')
    reponame = f"{path_parts[0]}/{path_parts[1]}"
    branch = path_parts[3]
    subfolder = '/'.join(path_parts[4:])
    return reponame, branch, subfolder

async def main(url, root_dir, verify_ssl, pat_token):
    reponame, branch, subfolder = parse_github_url(url)
    fetcher = GithubFetcher(reponame, branch, subfolder, root_dir, verify_ssl, pat_token)
    await fetcher.fetch_files()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Fetch and save subdirectory from a GitHub repository.")
    parser.add_argument("url", help="GitHub URL to the subdirectory (e.g., https://github.com/user/repo/tree/branch/subfolder)")
    parser.add_argument("root_dir", help="Local directory to save the files")
    parser.add_argument("--no-verify-ssl", action="store_false", dest="verify_ssl", help="Disable SSL certificate verification (not recommended)")
    parser.add_argument("--pat-token", help="GitHub Personal Access Token (PAT)")

    args = parser.parse_args()

    asyncio.run(main(args.url, args.root_dir, args.verify_ssl, args.pat_token))