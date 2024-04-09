package info

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"net/url"

	"dantotsu-update/src/downloader"
	"github.com/google/go-github/v60/github"
)

const (
	Owner = "rebelonion"
	Repo = "Dantotsu"
	Branch = "dev"
)

var (
	DiscordLinkRegex = regexp.MustCompile(`https://cdn\.discordapp\.com/attachments/\d+/\d+/(app-google-[^?]+)\?ex=[^&]+&is=[^&]+&hm=[^&]+&`)
	TempDir = GetTempDir()
	InfoDir = GetInfoDir()
	Token = GetGitHubToken()
)

func UpdateWorkflowId(workflowId int64) {
	workflowIdFile := filepath.Join(InfoDir, "workflow-id.txt")
	err := os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow ID to file: %v\n", err)
	}
}

func UpdateWorkflowName(workflowName string) {
	workflowNameFile := filepath.Join(InfoDir, "workflow-name.txt")
	err := os.WriteFile(workflowNameFile, []byte(workflowName), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow name to file: %v\n", err)
	}
}

func UpdateStatus(status string) {
	statusFile := filepath.Join(InfoDir, "status.txt")
	err := os.WriteFile(statusFile, []byte(status), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing status to file: %v\n", err)
	}
}

func UpdateCommitLog(client *github.Client, commitLogId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), Owner, Repo, commitLogId, 0)
	if err != nil {
		fmt.Printf("Error downloading artifact: %v\n", err)
	}

	println("\nDownloading commit log...")
	err = downloader.DownloadAndExtract(artifactDownloadUrl.String(), InfoDir, ".txt")
	if err != nil {
		fmt.Printf("Error downloading and extracting commit log: %v\n", err)
	}

	commitLogFile := filepath.Join(InfoDir, "commit_log.txt")
	commitLogText, _ := os.ReadFile(commitLogFile)
	decodedCommitLogText, err := url.QueryUnescape(string(commitLogText))
	if err != nil {
		fmt.Printf("Error decoding commit log text: %v\n", err)
	}

	err = os.WriteFile(commitLogFile, []byte(decodedCommitLogText), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing commit log to file: %v\n", err)
	}

	os.Rename(commitLogFile, filepath.Join(InfoDir, "commit-log.txt"))

	println("Finished downloading commit log")
}

func GetTempDir() string {
	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	if workspacePath == "" {
		workspacePath = "."
	}
	tempDir := filepath.Join(workspacePath, "temp")
	os.MkdirAll(tempDir, os.ModePerm)
	return tempDir
}

func GetInfoDir() string {
	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	if workspacePath != "" {
		return filepath.Join(workspacePath, "info");
	}
	return filepath.Join(".", "info");
}

func GetGitHubToken() string {
	tokenPat := os.Getenv("TOKEN_PAT")
	if tokenPat == "" {
		token_pat_file := filepath.Join(InfoDir, "github-pat.txt")
		data, _ := os.ReadFile(token_pat_file)
		return string(data)
	}
	return tokenPat
}