package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/go-github/v60/github"
	"dantotsu-update/src/downloader"
	"dantotsu-update/src/info"
)

// TODO: Use a JSON file instead of storing info in multiple files.

func main() {
	client := github.NewClient(nil).WithAuthToken(info.Token)
	
	println("Getting latest workflow run...")
	workflowId, workflowName := GetLatestWorkflowInfo(client)
	fmt.Printf("Got latest workflow run '%s' with ID: %d\n", workflowName, workflowId)

	artifactId := GetZipArtifactId(client, workflowId)
	if artifactId == 0 {
		println("No Dantotsu artifact found.\nUpdating saved workflow id...")
		info.UpdateWorkflowId(workflowId)
		println("Trying the backup download method...")
		DownloadApkBackup(client, workflowId, workflowName)
		return
	}

	println("\nDownloading Dantotsu APKs...")
	DownloadDantotsu(client, workflowId, workflowName, artifactId)
	println("Finished downloading Dantotsu APKs")
}

func GetLatestWorkflowInfo(client *github.Client) (int64, string) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), info.Owner, info.Repo, "beta.yml", &github.ListWorkflowRunsOptions{ Branch: info.Branch })
	if err != nil {
		fmt.Printf("Error getting workflow runs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	workflowId := latestRun.GetID()
	workflowStatus := latestRun.GetStatus()
	workflowName := latestRun.GetDisplayTitle()

	savedIdFile := filepath.Join(info.InfoDir, "workflow-id.txt")
	savedIdBytes, _ := os.ReadFile(savedIdFile)
	savedWorkflowId, _ := strconv.ParseInt(string(savedIdBytes), 10, 64)
	if savedWorkflowId == workflowId || workflowStatus != "completed" {
		time.Sleep(5 * time.Second)
		return GetLatestWorkflowInfo(client)
	}
	return workflowId, workflowName
}

func GetZipArtifactId(client *github.Client, workflowId int64) int64 {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), info.Owner, info.Repo, workflowId, &github.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting workflow run artifacts: %v\n", err)
	}

	artifactId, commitLogId := int64(0), int64(0)
	for _, artifact := range artifacts.Artifacts {
		if artifact.GetName() == "Dantotsu-Split" {
			artifactId = artifact.GetID()
		} else if artifact.GetName() == "commit-log" {
			commitLogId = artifact.GetID()
		}
	}

	if artifactId != 0 && commitLogId != 0 {
		info.UpdateCommitLog(client, commitLogId)
		return artifactId
	}

	return 0
}

func GetDiscordLinks(logText io.ReadCloser) []map[string]string {
	logBytes, err := io.ReadAll(logText)
	if err != nil {
		fmt.Printf("Error reading log text: %v\n", err)
	}

	matches := info.DiscordLinkRegex.FindAllStringSubmatch(string(logBytes), -1)
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

func DownloadDantotsu(client *github.Client, workflowId int64, workflowName string, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), info.Owner, info.Repo, artifactId, 0)
	if err != nil {
		fmt.Printf("Error downloading artifact: %v\n", err)
	}

	err = downloader.DownloadAndExtract(artifactDownloadUrl.String(), info.TempDir, ".apk")
	if err != nil {
		fmt.Printf("Error downloading and extracting APK: %v\n", err)
	}

	info.UpdateWorkflowId(workflowId)
	info.UpdateWorkflowName(workflowName)
	info.UpdateStatus("success")
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
    jobs, _, err := client.Actions.ListWorkflowJobs(context.Background(), info.Owner, info.Repo, workflowId, &github.ListWorkflowJobsOptions{})
    if err != nil {
        fmt.Printf("Error getting workflow jobs: %v\n", err)
    }

    for _, job := range jobs.Jobs {
        if job.GetName() == "build" {
            fmt.Printf("Found build job with ID: %d\n", job.GetID())

            logs, _, err := client.Actions.GetWorkflowJobLogs(context.Background(), info.Owner, info.Repo, job.GetID(), 0)
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

				err = DownloadFile(download["link"], filepath.Join(info.TempDir, download["name"]))
				if err != nil {
					fmt.Printf("Error downloading APK: %v\n", err)
				}

				successfullyDownloaded++
				fmt.Printf("Downloaded APK: %s\n", download["name"])
			}

			if len(downloadTable) == 0 || successfullyDownloaded != len(downloadTable) {
				println("Failed to download APKs")
				info.UpdateStatus("failed")
				return
			}

			info.UpdateWorkflowName(workflowName)
			info.UpdateStatus("success")
			fmt.Println("APKs downloaded successfully")

			tempZipFile := filepath.Join(info.TempDir, "temp.zip")
			os.Remove(tempZipFile)
        }
    }
}