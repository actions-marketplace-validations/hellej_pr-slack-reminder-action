package githubclient

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/google/go-github/v78/github"
)

// FetchLatestArtifactByName downloads the most recent GitHub Actions artifact by name,
// extracts a JSON file from the zip archive, and unmarshals it into the provided struct.
// The target parameter should be a pointer to the target struct for JSON deserialization.
func (client *client) FetchLatestArtifactByName(
	ctx context.Context,
	owner, repo, artifactName, jsonFilename string,
	target any,
) error {
	opts := &github.ListArtifactsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Name:        &artifactName,
	}
	res, resp, err := client.actionsService.ListArtifacts(ctx, owner, repo, opts)
	if err != nil {
		statusText := ""
		if resp != nil && resp.Status != "" {
			statusText = " status=" + resp.Status
		}
		return fmt.Errorf("failed to list artifacts: %w%s", err, statusText)
	}

	artifacts := res.Artifacts
	if len(artifacts) == 0 {
		return fmt.Errorf("no artifacts found with name %q", artifactName)
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].GetCreatedAt().Time.After(artifacts[j].GetCreatedAt().Time)
	})

	latest := artifacts[0]
	artifactID := latest.GetID()

	archiveURL := latest.GetArchiveDownloadURL()
	if archiveURL == "" {
		archiveURL = fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/actions/artifacts/%d/zip",
			owner, repo, artifactID,
		)
	}

	req, err := client.requests.NewRequest("GET", archiveURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	ghResp, err := client.requests.Do(ctx, req, nil) // v == nil -> do not attempt JSON decode
	if err != nil {
		return fmt.Errorf("download artifact zip: %w", err)
	}
	httpResp := ghResp.Response
	defer httpResp.Body.Close()

	tmpFile, err := os.CreateTemp("", "artifact-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, httpResp.Body); err != nil {
		return fmt.Errorf("write zip to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	zr, err := zip.OpenReader(tmpPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	found := false
	for _, f := range zr.File {
		if f.Name == jsonFilename {
			found = true
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open file inside zip: %w", err)
			}
			dec := json.NewDecoder(rc)
			if err := dec.Decode(target); err != nil {
				_ = rc.Close()
				return fmt.Errorf("decode json %q: %w", jsonFilename, err)
			}
			_ = rc.Close()
			break
		}
	}

	if !found {
		return fmt.Errorf("json file %q not found inside artifact zip", jsonFilename)
	}

	return nil
}
