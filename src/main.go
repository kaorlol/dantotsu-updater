package main

import (
	"artifact-downloader/src/utils/actions"
	"artifact-downloader/src/utils/info"
	"fmt"
	"time"
)

func main() {
	println("Getting workflow latest run...")
	prevTime := time.Now()
	latestRun, err := actions.GetWorkflowLatestRun()
	if err != nil {
		fmt.Println(err)
		return
	}

	actions.DownloadArtifacts(latestRun)
	info.UpdateInfo(info.Info{ ElapsedTime: int64(time.Since(prevTime).Seconds()) })
}