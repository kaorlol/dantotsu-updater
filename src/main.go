package main

import (
	"fmt"
	"time"

	"artifact-downloader/src/utils/actions"
	"artifact-downloader/src/utils/info"
)

func main() {
	println("Getting workflow latest run...")
	prevTime := time.Now()
	latestRun, err := actions.GetWorkflowLatestRun()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = actions.DownloadArtifacts(latestRun)
	if err != nil {
		fmt.Println(err)
		return
	}

	info.UpdateInfo(info.Info{ElapsedTime: int64(time.Since(prevTime).Seconds())})
}
