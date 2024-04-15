package actions

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"artifact-downloader/src/utils/info"
	"artifact-downloader/src/utils/modules"
	"artifact-downloader/src/utils/settings"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

var (
	token            = info.GetGitHubToken()
	workflowInfo     = info.GetInfo()
	workflowSettings = settings.GetSettings()
	client           = createClient()
)

func createClient() *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}

func GetWorkflowLatestRun() (int64, error) {
	owner := workflowSettings.Workflow.Owner
	repo := workflowSettings.Workflow.Repo
	name := workflowSettings.Workflow.Name

	workflowRuns, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), owner, repo, name, &github.ListWorkflowRunsOptions{
		Branch: workflowSettings.Workflow.Branch,
		Status: "success",
	})

	if _, ok := err.(*github.RateLimitError); ok {
		info.UpdateInfo(info.Info{Status: "hit rate limit"})
		return 0, fmt.Errorf("hit rate limit")
	}

	if len(workflowRuns.WorkflowRuns) == 0 {
		info.UpdateInfo(info.Info{Status: "no workflow runs found"})
		return 0, fmt.Errorf("no workflow runs found")
	}

	latestRun := workflowRuns.WorkflowRuns[0]
	oldRun := workflowRuns.WorkflowRuns[1]
	if latestRun.GetID() == workflowInfo.Workflow.ID {
		time.Sleep(time.Duration(workflowSettings.Delay) * time.Second)
		return GetWorkflowLatestRun()
	}

	fmt.Printf("Workflow run named: '%s' found with id %d\n", latestRun.GetName(), latestRun.GetID())
	workflowInfo = info.UpdateInfo(info.Info{
		Workflow: info.Workflow{
			ID:    latestRun.GetID(),
			Title: latestRun.GetDisplayTitle(),
		},
	})

	getCommitHistory(oldRun.GetCreatedAt().Time, latestRun.GetCreatedAt().Time)
	return latestRun.GetID(), nil
}

func DownloadArtifacts(runID int64) error {
	owner := workflowSettings.Workflow.Owner
	repo := workflowSettings.Workflow.Repo

	artifacts, _, err := client.Actions.ListWorkflowRunArtifacts(context.Background(), owner, repo, runID, &github.ListOptions{})
	if _, ok := err.(*github.RateLimitError); ok {
		info.UpdateInfo(info.Info{Status: "hit rate limit"})
		return fmt.Errorf("hit rate limit")
	}

	modules.MakeDir("archive")
	err = modules.Parallel(artifacts.Artifacts, func(artifact *github.Artifact) error {
		if artifact.GetExpired() {
			return fmt.Errorf("artifact expired")
		}

		artifactDownloadUrl, _, err := client.Actions.DownloadArtifact(context.Background(), owner, repo, artifact.GetID(), 0)
		if _, ok := err.(*github.RateLimitError); ok {
			info.UpdateInfo(info.Info{Status: "hit rate limit"})
			return fmt.Errorf("hit rate limit")
		}

		err = modules.DownloadFile(artifactDownloadUrl.String(), "archive")
		if err != nil {
			return err
		}

		err = modules.ExtractFromZip("archive/"+artifact.GetName()+".zip", ".apk", "archive")
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	files, _ := os.ReadDir("archive")
	modules.Parallel(files, func(file os.DirEntry) {
		if file.IsDir() || file.Name()[len(file.Name())-4:] != ".apk" {
			os.Remove("archive/" + file.Name())
		}
	})

	println("Artifacts downloaded successfully")
	return nil
}

func getCommitHistory(since, until time.Time) error {
	owner := workflowSettings.Workflow.Owner
	repo := workflowSettings.Workflow.Repo

	println("Getting commit history...")
	commits, _, err := client.Repositories.ListCommits(context.Background(), owner, repo, &github.CommitsListOptions{
		SHA:   workflowSettings.Workflow.Branch,
		Since: since,
		Until: until,
	})

	if _, ok := err.(*github.RateLimitError); ok {
		info.UpdateInfo(info.Info{Status: "hit rate limit"})
		return fmt.Errorf("hit rate limit")
	}

	commitLog := ""
	for _, commit := range commits {
		message := commit.GetCommit().GetMessage()
		author := commit.GetCommit().GetAuthor().GetName()

		if !strings.Contains(author, "(bot)") {
			commitLog += fmt.Sprintf("- %s ~%s\n", strings.Trim(message, " \t\n\r"), author)
		}
	}

	workflowInfo = info.UpdateInfo(info.Info{CommitLog: commitLog})
	println("Commit history updated successfully")
	return nil
}

// func GetRateLimit() int {
// 	rateLimit, _, err := client.RateLimit.Get(context.Background());
// 	if _, ok := err.(*github.RateLimitError); ok {
// 		println("hit rate limit")
// 		return 0
// 	}

// 	return rateLimit.Core.Remaining
// }
