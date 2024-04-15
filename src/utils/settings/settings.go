package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const path = "data/settings.json"

type Workflow struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

type Settings struct {
	Workflow Workflow `json:"workflow"`
	Delay    int64     `json:"delay"`
}

func GetSettings() Settings {
	file := filepath.Join(path)
	data, _ := os.ReadFile(file)

	var settings Settings
	json.Unmarshal(data, &settings)

	return settings
}