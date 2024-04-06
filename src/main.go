package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"regexp"

	"github.com/google/go-github/v60/github"
)

// TODO: Use multiple go files and cleanup junk code, make it more optimized, efficient, and robust.

const (
	owner = "rebelonion"
	repo = "Dantotsu"
	branch = "dev"
)

var (
	discordLinkRegex = regexp.MustCompile(`https://cdn\.discordapp\.com/attachments/\d+/\d+/(app-google-[^?]+)\?ex=[^&]+&is=[^&]+&hm=[^&]+&`)
	tempDir = GetTempFolder()
	infoDir = GetInfoFolder()
	tokenPat = GetGitHubToken()
)

func main() {
	println("Starting Dantotsu Updater...")
	client := github.NewClient(nil).WithAuthToken(tokenPat)
	
	println("Getting latest workflow run...")
	workflowId, workflowName := GetLatestWorkflowInfo(client)
	artifactId := GetZipArtifactId(client, workflowId)
	if artifactId == 0 {
		println("No Dantotsu artifact found.\nUpdating saved workflow id...")
		UpdateWorkflowId(workflowId)
		println("Trying the backup download method...")
		DownloadApkBackup(client, workflowId, workflowName)
		return
	}

	println("Downloading Dantotsu artifact...")
	DownloadDantotsu(client, workflowId, workflowName, artifactId)
	println("Dantotsu artifact downloaded successfully")
}

func GetTempFolder() string {
	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	if workspacePath != "" {
		return filepath.Join(workspacePath, "temp");
	}
	return filepath.Join(".", "temp");
}

func GetInfoFolder() string {
	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	if workspacePath != "" {
		return filepath.Join(workspacePath, "info");
	}
	return filepath.Join(".", "info");
}

func GetGitHubToken() string {
	tokenPat := os.Getenv("TOKEN_PAT")
	if tokenPat == "" {
		token_pat_file := filepath.Join(infoDir, "github_pat.txt")
		data, _ := os.ReadFile(token_pat_file)
		return string(data)
	}
	return tokenPat
}

func GetLatestWorkflowInfo(client *github.Client) (int64, string) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{ Branch: branch })
	if err != nil {
		fmt.Printf("Error getting workflow runs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	workflowId := latestRun.GetID()
	workflowStatus := latestRun.GetStatus()
	workflowName := latestRun.GetDisplayTitle()

	savedIdFile := filepath.Join(infoDir, "workflow-id.txt")
	savedIdBytes, _ := os.ReadFile(savedIdFile)
	savedWorkflowId, _ := strconv.ParseInt(string(savedIdBytes), 10, 64)
	if savedWorkflowId == workflowId || workflowStatus != "completed" {
		time.Sleep(5 * time.Second)
		return GetLatestWorkflowInfo(client)
	}

	fmt.Printf("Found new workflow run '%s'\n", workflowName)
	return workflowId, workflowName
}

func GetZipArtifactId(client *github.Client, workflowId int64) int64 {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting workflow run artifacts: %v\n", err)
	}

	for _, artifact := range artifacts.Artifacts {
		if artifact.GetName() == "Dantotsu-Split" {
			fmt.Printf("Found Dantotsu artifact with ID: %d\n", artifact.GetID())
			return artifact.GetID()
		}
	}

	return 0
}

func GetDiscordLinks(logText io.ReadCloser) []map[string]string {
	logBytes, err := io.ReadAll(logText)
	if err != nil {
		fmt.Printf("Error reading log text: %v\n", err)
	}

	matches := discordLinkRegex.FindAllStringSubmatch(string(logBytes), -1)
	tables := make([]map[string]string, 0)

	for _, match := range matches {
		table := map[string]string{
			"name": match[1],
			"link": match[0],
		}
		tables = append(tables, table)
    }

    return tables
}

func UpdateWorkflowId(workflowId int64) {
	workflowIdFile := filepath.Join(infoDir, "workflow-id.txt")
	err := os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow ID to file: %v\n", err)
	}
}

func UpdateWorkflowName(workflowName string) {
	workflowNameFile := filepath.Join(infoDir, "workflow-name.txt")
	err := os.WriteFile(workflowNameFile, []byte(workflowName), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow name to file: %v\n", err)
	}
}

func UpdateStatus(status string) {
	statusFile := filepath.Join(infoDir, "status.txt")
	err := os.WriteFile(statusFile, []byte(status), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing status to file: %v\n", err)
	}
}

func DownloadDantotsu(client *github.Client, workflowId int64, workflowName string, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		fmt.Printf("Error downloading artifact: %v\n", err)
	}

	err = DownloadAndExtractAPKs(artifactDownloadUrl.String(), tempDir)
	if err != nil {
		fmt.Printf("Error downloading and extracting APK: %v\n", err)
	}

	UpdateWorkflowId(workflowId)
	UpdateWorkflowName(workflowName)
	UpdateStatus("success")

	fmt.Println("APKs downloaded successfully")

	tempZipFile := filepath.Join(tempDir, "temp.zip")
	os.Remove(tempZipFile)
}

func DownloadFile(url string, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func DownloadApkBackup(client *github.Client, workflowId int64, workflowName string) {
    jobs, _, err := client.Actions.ListWorkflowJobs(context.Background(), owner, repo, workflowId, &github.ListWorkflowJobsOptions{})
    if err != nil {
        fmt.Printf("Error getting workflow jobs: %v\n", err)
    }

    for _, job := range jobs.Jobs {
        if job.GetName() == "build" {
            fmt.Printf("Found build job with ID: %d\n", job.GetID())

            logs, _, err := client.Actions.GetWorkflowJobLogs(context.Background(), owner, repo, job.GetID(), 0)
            if err != nil {
                fmt.Printf("Error getting job logs: %v\n", err)
            }

            logUrl := logs.String()
			logText, err := http.Get(logUrl)
			if err != nil {
				fmt.Printf("Error requesting job logs: %v\n", err)
			}

			successfullyDownloaded := 0;
			downloadTable := GetDiscordLinks(logText.Body)
			for _, download := range downloadTable {
				resp, err := http.Get(download["link"])
				if err != nil {
					fmt.Printf("Error requesting download link: %v\n", err)
				}

				if resp.StatusCode == 404 {
					println("Download link expired")
					continue
				}

				err = DownloadFile(download["link"], filepath.Join(tempDir, download["name"]))
				if err != nil {
					fmt.Printf("Error downloading APK: %v\n", err)
				}

				successfullyDownloaded++
				fmt.Printf("Downloaded APK: %s\n", download["name"])
			}

			if len(downloadTable) == 0 || successfullyDownloaded != len(downloadTable) {
				println("Failed to download APKs")
				UpdateStatus("failed")
				return
			}

			UpdateWorkflowName(workflowName)
			UpdateStatus("success")
			fmt.Println("APKs downloaded successfully")

			tempZipFile := filepath.Join(tempDir, "temp.zip")
			os.Remove(tempZipFile)
        }
    }
}

func DownloadAndExtractAPKs(downloadUrl, outputDir string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return fmt.Errorf("error downloading APK: %v", err)
	}
	defer resp.Body.Close()

	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	tempZipFile := filepath.Join(outputDir, "temp.zip")
	out, err := os.Create(tempZipFile)
	if err != nil {
		return fmt.Errorf("error creating temporary zip file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error writing APK to temporary zip file: %v", err)
	}

	r, err := zip.OpenReader(tempZipFile)
	if err != nil {
		return fmt.Errorf("error opening temporary zip file: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".apk") {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("error opening APK file in zip: %v", err)
			}
			defer rc.Close()

			extractedAPK := filepath.Join(outputDir, f.Name)
			extractedFile, err := os.Create(extractedAPK)
			if err != nil {
				return fmt.Errorf("error creating extracted APK file: %v", err)
			}
			defer extractedFile.Close()

			_, err = io.Copy(extractedFile, rc)
			if err != nil {
				return fmt.Errorf("error writing APK to extracted file: %v", err)
			}

			fmt.Printf("Extracted APK: %s\n", f.Name)
		}
	}

	return nil
}
