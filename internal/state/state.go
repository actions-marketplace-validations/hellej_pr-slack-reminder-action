package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const CurrentSchemaVersion = 1

type SlackRef struct {
	ChannelID string `json:"channelId"`
	MessageTS string `json:"messageTs"`
}

type PullRequestRef struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

type State struct {
	SchemaVersion int              `json:"schemaVersion"`
	CreatedAt     time.Time        `json:"createdAt"`
	Slack         SlackRef         `json:"slack"`
	PullRequests  []PullRequestRef `json:"pullRequests"`
}

func Save(path string, state State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	jsonData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}

	return nil
}

func Load(path string) (*State, error) {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *State) Validate() error {
	if s.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %d, expected %d", s.SchemaVersion, CurrentSchemaVersion)
	}
	return nil
}
