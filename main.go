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

	"github.com/google/go-github/v60/github"
)

const owner = "rebelonion"
const repo = "Dantotsu"
const branch = "dev"
var workspacePath = os.Getenv("GITHUB_WORKSPACE")
var tempDir = filepath.Join(workspacePath, "temp")

type WorkflowStatus struct {
	success []string
	failure []string
}

var workflowStatus = WorkflowStatus{
	success: []string{
		"completed",
		"success",
	},
	failure: []string{
		"failure",
		"timed_out",
		"action_required",
		"cancelled",
		"skipped",
		"neutral",
	},
}

func main() {
	duration := 5*time.Hour + 30*time.Minute
    time.AfterFunc(duration, func() {
        fmt.Println("Program has been running for 5 hours and 30 minutes.")
    })

	println("Starting Dantotsu Updater...")
	pat := os.Getenv("TOKEN_PAT")
	client := github.NewClient(nil).WithAuthToken(pat)

	println("Getting latest workflow run...")
	workflowId, err := getLatestWorkflow(client)
	if err != nil {
		println("Error getting latest workflow run. Exiting...")
		return
	}

	artifacts := getArtifacts(client, workflowId)
	artifactId := getZipArtifactId(artifacts)
	if artifactId == 0 {
		println("No Dantotsu artifact found. Updating workflow ID...")
		updateWorkflowId(workflowId)
		return
	}

	println("Downloading Dantotsu artifact...")
	downloadDantotsu(client, workflowId, artifactId)
	println("Dantotsu artifact downloaded successfully")
}

func getWorkflowStatus(status string) WorkflowStatus {
	if contains(workflowStatus.success, status) {
		return workflowStatus
	}
	if contains(workflowStatus.failure, status) {
		return workflowStatus
	}
	return WorkflowStatus{}
}

func contains(status []string, s string) bool {
	for _, a := range status {
		if a == s {
			return true
		}
	}
	return false
}

func getLatestWorkflow(client *github.Client) (int64, error) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{ Branch: branch })
	if err != nil {
		fmt.Printf("Error getting workflow runs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	workflowId := latestRun.GetID()
	workflowName := latestRun.GetDisplayTitle()

	if compareWorkflowIds(workflowId) {
		time.Sleep(5 * time.Second)
		return getLatestWorkflow(client)
	}

	// if latestRun.GetStatus() == "failure" {
	// 	return workflowId, fmt.Errorf("latest workflow run failed")
	// }

	// if latestRun.GetStatus() != "completed" {
		// time.Sleep(5 * time.Second)
		// return getLatestWorkflow(client)
	// }

	if getWorkflowStatus(latestRun.GetStatus()).failure != nil {
		return workflowId, fmt.Errorf("latest workflow run failed")
	}

	if getWorkflowStatus(latestRun.GetStatus()).success != nil {
		time.Sleep(5 * time.Second)
		return getLatestWorkflow(client)
	} 

	fmt.Printf("Latest workflow run ID: %d, name: %s", workflowId, workflowName)
	return workflowId, nil
}

func compareWorkflowIds(workflowId int64) bool {
	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	if _, err := os.Stat(workflowIdFile); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(workflowIdFile)
	if err != nil {
		fmt.Printf("Error reading workflow ID file: %v", err)
	}

	cleanedData := strings.ReplaceAll(strings.ReplaceAll(string(data), " ", ""), "\n", "")
	oldWorkflowId, err := strconv.ParseInt(cleanedData, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing old workflow ID: %v", err)
	}
	return oldWorkflowId == workflowId
}

func getArtifacts(client *github.Client, workflowId int64) []*github.Artifact {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting workflow run artifacts: %v", err)
	}
	return artifacts.Artifacts
}

func getZipArtifactId(Artifacts []*github.Artifact) int64 {
	for _, artifact := range Artifacts {
		if artifact.GetName() == "Dantotsu" {
			fmt.Printf("Found Dantotsu artifact with ID: %d", artifact.GetID())
			return artifact.GetID()
		}
	}

	fmt.Println("Dantotsu artifact not found")
	return 0
}

func updateWorkflowId(workflowId int64) {
	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	err := os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow ID to file: %v", err)
	}
}

func downloadDantotsu(client *github.Client, workflowId int64, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		fmt.Printf("Error downloading artifact: %v", err)
	}

	err = downloadAndExtractAPK(artifactDownloadUrl.String(), tempDir)
	if err != nil {
		fmt.Printf("Error downloading and extracting APK: %v", err)
	}

	updateWorkflowId(workflowId)

	fmt.Println("Artifact downloaded and extracted successfully")
	fmt.Printf("New Workflow ID: %d", workflowId)
}


func downloadAndExtractAPK(downloadUrl, outputDir string) error {
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

			extractedAPK := filepath.Join(outputDir, filepath.Base(f.Name))
			extractedFile, err := os.Create(extractedAPK)
			if err != nil {
				return fmt.Errorf("error creating extracted APK file: %v", err)
			}
			defer extractedFile.Close()

			_, err = io.Copy(extractedFile, rc)
			if err != nil {
				return fmt.Errorf("error writing APK to extracted file: %v", err)
			}

			fmt.Printf("APK extracted successfully: %s", extractedAPK)
			break
		}
	}

	return nil
}
