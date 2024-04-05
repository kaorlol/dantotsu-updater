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

var tempDir = GetTempFolder()
var tokenPat = GetGitHubToken()

func main() {
	println("Starting Dantotsu Updater...")
	client := github.NewClient(nil).WithAuthToken(tokenPat)
	
	println("Getting latest workflow job...")
	workflowId, workflowName := GetLatestWorkflowInfo(client)
	artifactId := GetZipArtifactId(client, workflowId)
	if artifactId == 0 {
		println("No Dantotsu artifact found.\nUpdating saved workflow id...")
		UpdateWorkflowId(workflowId)
		println("Updating saved status...")
		UpdateStatus("failed")
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

func GetGitHubToken() string {
	tokenPat := os.Getenv("TOKEN_PAT")
	if tokenPat != "" {
		return tokenPat
	}

	token_pat_file := filepath.Join(".", "github_pat.txt")
	data, _ := os.ReadFile(token_pat_file)
	return string(data)
}

func GetLatestWorkflowInfo(client *github.Client) (int64, string) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{ Branch: branch })
	if err != nil {
		fmt.Printf("Error getting workflow jobs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	workflowId := latestRun.GetID()
	workflowStatus := latestRun.GetStatus()
	workflowName := latestRun.GetDisplayTitle()

	savedIdFile := filepath.Join(tempDir, "workflow-id.txt")
	savedIdBytes, _ := os.ReadFile(savedIdFile)
	savedWorkflowId, _ := strconv.ParseInt(string(savedIdBytes), 10, 64)
	if savedWorkflowId == workflowId || workflowStatus != "completed" {
		time.Sleep(5 * time.Second)
		return GetLatestWorkflowInfo(client)
	}

	fmt.Printf("Found new workflow job '%s'\n", workflowName)
	return workflowId, workflowName
}

func GetZipArtifactId(client *github.Client, workflowId int64) int64 {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting workflow job artifacts: %v\n", err)
	}

	for _, artifact := range artifacts.Artifacts {
		if artifact.GetName() == "Dantotsu" {
			fmt.Printf("Found Dantotsu artifact with ID: %d\n", artifact.GetID())
			return artifact.GetID()
		}
	}

	return 0
}

func UpdateWorkflowId(workflowId int64) {
	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	err := os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow ID to file: %v\n", err)
	}
}

func UpdateWorkflowName(workflowName string) {
	workflowNameFile := filepath.Join(tempDir, "workflow-name.txt")
	err := os.WriteFile(workflowNameFile, []byte(workflowName), os.ModePerm)
	if err != nil {
		fmt.Printf("Error writing workflow name to file: %v\n", err)
	}
}

func UpdateStatus(status string) {
	statusFile := filepath.Join(tempDir, "status.txt")
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

	err = downloadAndExtractAPK(artifactDownloadUrl.String(), tempDir)
	if err != nil {
		fmt.Printf("Error downloading and extracting APK: %v\n", err)
	}

	UpdateWorkflowId(workflowId)
	UpdateWorkflowName(workflowName)
	UpdateStatus("success")

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

			fmt.Printf("APK extracted successfully: %s\n", extractedAPK)
			break
		}
	}

	return nil
}
