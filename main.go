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

	"github.com/google/go-github/v60/github"
)

func main() {
	logFile, err := os.OpenFile("updater.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetPrefix("[Dantotsu Updater] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	pat := os.Getenv("TOKEN_PAT")
	if pat == "" {
		log.Fatalf("GitHub PAT not found in environment variables")
	}

	client := github.NewClient(nil).WithAuthToken(pat)

	owner := "rebelonion"
	repo := "Dantotsu"
	branch := "dev"

	workflowId, name := getLatestWorkflow(client, owner, repo, branch)
	os.Setenv("workflow_name", name)

	artifacts := getArtifacts(client, owner, repo, workflowId)
	artifactId := getZipArtifactId(artifacts)

	downloadDantotsu(client, owner, repo, workflowId, artifactId)
}

func getLatestWorkflow(client *github.Client, owner, repo, branch string) (int64, string) {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{Branch: branch})
	if err != nil {
		log.Fatalf("Error getting workflow runs: %v", err)
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	log.Printf("Latest workflow run ID: %d, status: %s", latestRun.GetID(), latestRun.GetName())
	return latestRun.GetID(), latestRun.GetName()
}

func getArtifacts(client *github.Client, owner, repo string, workflowId int64) []*github.Artifact {
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

func downloadDantotsu(client *github.Client, owner, repo string, workflowId int64, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		log.Fatalf("Error downloading artifact: %v", err)
	}

	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	tempDir := filepath.Join(workspacePath, "temp")

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
