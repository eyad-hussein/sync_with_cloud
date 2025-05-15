# sync_with_cloud

A simple tool I created to push local files to Google Drive so they're always updated and accessible from anywhere.

## Background

I built this tool because I needed a way to ensure my important files are always backed up to Google Drive and accessible from any device. Rather than manually uploading files or using the standard Drive client, this tool automates the process and provides more control over what gets synced.

The current version works well for basic sync operations, but I'm planning to extend it to run as a background service for continuous synchronization. I may also add a command-line interface (CLI) for more interactive control.

## Features

- One-way synchronization from local directories to Google Drive
- Selective syncing of specific directories
- Exclude specific files or directories from syncing
- Automatically creates folder structure in Google Drive
- Updates existing files if they've changed locally

## Prerequisites

- Go 1.18 or higher
- Google Drive API access
- Service account credentials JSON file

## Installation

1. Clone the repository:

```bash
git clone https://github.com/eyad-hussein/sync_with_cloud.git
cd sync_with_cloud
```

2. Build the application:

```bash
make build
```

## Configuration

The application is configured via a YAML file (`drive-sync.yaml`). Here's an example configuration:

```yaml
credentials_file: 'service-account-credentials.json'
root_folder_id: 'YOUR_GOOGLE_DRIVE_FOLDER_ID'

paths:
  '/path/to/local/directory': 'remote-directory-name'
  '/path/to/another/directory': 'another-remote-directory'

exclude:
  '/path/to/local/directory/file-to-exclude.txt': ''
  '/path/to/local/directory/directory-to-exclude': ''
```

### Configuration Options

- `credentials_file`: Path to the Google Drive service account credentials JSON file
- `root_folder_id`: The ID of the Google Drive folder where files will be synced to
- `paths`: A map of local paths to remote path names
- `exclude`: A map of paths to exclude from syncing

## Google Drive Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project
3. Enable the Google Drive API
4. Create a service account
5. Download the service account credentials JSON file
6. Save the credentials file to the same directory as the application
7. Share the target Google Drive folder with the service account email

## Usage

Run the sync process:

```bash
make run
```

Or manually run the application:

```bash
./bin/sync_with_cloud
```

## Future Plans

- Run as a background service for continuous synchronization
- Command-line interface for more control and manual operations
- Code refactoring to improve maintainability and performance
- Add comprehensive unit tests for all components

## Development

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```
