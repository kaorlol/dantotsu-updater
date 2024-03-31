package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"strconv"

	"github.com/google/go-github/v60/github"
)

const owner = "rebelonion"
const repo = "Dantotsu"
const branch = "dev"
var workspacePath = os.Getenv("GITHUB_WORKSPACE")
var tempDir = filepath.Join(workspacePath, "temp")

func main() {
	logFile, err := os.OpenFile("updater.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetPrefix("[Dantotsu Updater] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	println("Starting Dantotsu Updater...")

	pat := os.Getenv("TOKEN_PAT")
	client := github.NewClient(nil).WithAuthToken(pat)

	println("Getting latest workflow run...")

	workflowId, name := getLatestWorkflow(client)
	os.Setenv("workflow_name", name)

	println("Downloading Dantotsu artifact...")

	artifacts := getArtifacts(client, workflowId)
	artifactId := getZipArtifactId(artifacts)
	downloadDantotsu(client, workflowId, artifactId)

	println("Dantotsu artifact downloaded successfully")
}

func getLatestWorkflow(client *github.Client) (int64, string) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{ Branch: branch })
	if err != nil {
		log.Fatalf("Error getting workflow runs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	workflowId := latestRun.GetID()
	workflowName := latestRun.GetDisplayTitle()

	println("Workflow ID: ", workflowId)
	println("Workflow Name: ", workflowName)

	if compareWorkflowIds(workflowId) {
		println("Workflow ID is the same as the last run, waiting for new run...")
		time.Sleep(10 * time.Second)
		return getLatestWorkflow(client)
	}
	os.Setenv("ids_same", strconv.Itoa(1))

	if latestRun.GetStatus() != "completed" {
		println("Latest workflow run is not completed, waiting for completion...")
		time.Sleep(10 * time.Second)
		return getLatestWorkflow(client)
	}
	os.Setenv("completed", strconv.Itoa(1))

	log.Printf("Latest workflow run ID: %d, name: %s",workflowId, workflowName)
	return workflowId, workflowName
}

func compareWorkflowIds(workflowId int64) bool {
	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	if _, err := os.Stat(workflowIdFile); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(workflowIdFile)
	if err != nil {
		log.Fatalf("Error reading workflow ID file: %v", err)
	}

	oldWorkflowId, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		log.Fatalf("Error parsing old workflow ID: %v", err)
	}
	return oldWorkflowId == workflowId
}

func getArtifacts(client *github.Client, workflowId int64) []*github.Artifact {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		log.Fatalf("Error getting workflow run artifacts: %v", err)
	}
	return artifacts.Artifacts
}

func getZipArtifactId(Artifacts []*github.Artifact) int64 {
	for _, artifact := range Artifacts {
		if artifact.GetName() == "Dantotsu" {
			log.Printf("Found Dantotsu artifact with ID: %d", artifact.GetID())
			return artifact.GetID()
		}
	}

	log.Fatalf("Dantotsu artifact not found")
	return 0
}

func downloadDantotsu(client *github.Client, workflowId int64, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		log.Fatalf("Error downloading artifact: %v", err)
	}

	err = downloadAndExtractAPK(artifactDownloadUrl.String(), tempDir)
	if err != nil {
		log.Fatalf("Error downloading and extracting APK: %v", err)
	}

	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	err = os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		log.Fatalf("Error writing workflow ID to file: %v", err)
	}

	log.Printf("Artifact downloaded and extracted successfully")
	log.Printf("New Workflow ID: %d", workflowId)
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

			log.Printf("APK extracted successfully: %s", extractedAPK)
			break
		}
	}

	return nil
}
