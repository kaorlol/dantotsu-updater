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
		for _, e := range os.Environ() {
			fmt.Println(e)
		}
        panic("GitHub PAT not found in environment variables")
    }

	client := github.NewClient(nil).WithAuthToken(pat)

	owner := "rebelonion"
	repo := "Dantotsu"
	branch := "dev"

	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, "beta.yml", &github.ListWorkflowRunsOptions{Branch: branch})
	if err != nil {
		panic("Error getting workflow runs, error: " + err.Error())
	}

	workflowId := workflowRuns.WorkflowRuns[0].GetID()
	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, workflowId, &github.ListOptions{})
	if err != nil {
		panic("Error getting workflow run artifacts, error: " + err.Error())
	}

	artifactId := getArtifactId(artifacts.Artifacts)
	artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifactId, 0)
	if err != nil {
		panic("Error downloading artifact, error: " + err.Error())
	}

	err = downloadAndExtractAPK(artifactDownloadUrl.String())
	if err != nil {
		panic("Error downloading and extracting APK: " + err.Error())
	}

	workflowIdFile := filepath.Join("./temp", "workflow-id.txt")
	err = os.WriteFile(workflowIdFile, []byte(fmt.Sprintf("%d", workflowId)), os.ModePerm)
	if err != nil {
		panic("Error writing artifactId to file: " + err.Error())
	}
}

func getArtifactId(Artifacts []*github.Artifact) int64 {
	for _, artifact := range Artifacts {
		if artifact.GetName() == "Dantotsu" {
			return artifact.GetID()
		}
	}

	panic("Artifact not found")
}

func downloadAndExtractAPK(downloadUrl string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempDir := "./temp"
	err = os.MkdirAll(tempDir, os.ModePerm)
	if err != nil {
		return err
	}

	tempZipFile := filepath.Join(tempDir, "temp.zip")
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

			extractedAPK := filepath.Join(tempDir, filepath.Base(f.Name))
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
