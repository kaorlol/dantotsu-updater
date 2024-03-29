package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v60/github"
)

func main() {
	pat := os.Getenv("TOKEN_PAT")
	if pat == "" {
		panic("GitHub PAT not found in environment variables")
	}

	client := github.NewClient(nil).WithAuthToken(pat)

	owner := "rebelonion"
	repo := "Dantotsu"
	branch := "dev"

	workflowId := getLatestWorkflowId(client, owner, repo, branch)
	artifacts := getArtifacts(client, owner, repo, workflowId)
	artifactId, err := getArtifactId(artifacts)
	if err != nil {
		return
	}

	downloadDantotsu(client, owner, repo, workflowId, artifactId)
}

func getLatestWorkflowId(client *github.Client, owner, repo, branch string) int64 {
	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{Branch: branch})
	if err != nil {
		panic("Error getting workflow runs")
	}

	println(workflowRuns.WorkflowRuns[0].GetStatus())
	return workflowRuns.WorkflowRuns[0].GetID()
}

func getArtifacts(client *github.Client, owner, repo string, workflowId int64) []*github.Artifact {
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		panic("Error getting workflow run artifacts")
	}

	return artifacts.Artifacts
}

func getArtifactId(Artifacts []*github.Artifact) (int64, error) {
	for _, artifact := range Artifacts {
		if artifact.GetName() == "Dantotsu" {
			return artifact.GetID(), nil
		}
	}

	return 0, fmt.Errorf("artifact not found")
}

func downloadDantotsu(client *github.Client, owner, repo string, workflowId int64, artifactId int64) {
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		panic("Error downloading artifact")
	}

	workspacePath := os.Getenv("GITHUB_WORKSPACE")
	tempDir := filepath.Join(workspacePath, "temp")

	err = downloadAndExtractAPK(artifactDownloadUrl.String(), tempDir)
	if err != nil {
		panic("Error downloading and extracting APK")
	}

	workflowIdFile := filepath.Join(tempDir, "workflow-id.txt")
	err = os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		panic("Error writing workflow id to file")
	}

	fmt.Println("Artifact downloaded and extracted successfully")
	fmt.Println("Workflow ID:", workflowId)
}

func downloadAndExtractAPK(downloadUrl, outputDir string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	tempZipFile := filepath.Join(outputDir, "temp.zip")
	out, err := os.Create(tempZipFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(tempZipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".apk") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			extractedAPK := filepath.Join(outputDir, filepath.Base(f.Name))
			extractedFile, err := os.Create(extractedAPK)
			if err != nil {
				return err
			}
			defer extractedFile.Close()

			_, err = io.Copy(extractedFile, rc)
			if err != nil {
				return err
			}

			fmt.Println("APK extracted successfully:", extractedAPK)
			break
		}
	}

	return nil
}