package config_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hellej/pr-slack-reminder-action/internal/config"
)

// ConfigTestHelpers provides helper functions for setting up test environments
// similar to the ones in testhelpers/confighelpers.go but focused on config package testing
type ConfigTestHelpers struct {
	t *testing.T
}

func newConfigTestHelpers(t *testing.T) *ConfigTestHelpers {
	return &ConfigTestHelpers{t: t}
}

func (h *ConfigTestHelpers) setEnv(key, value string) {
	h.t.Setenv(key, value)
}

func (h *ConfigTestHelpers) setInput(inputName, value string) {
	envName := h.inputNameAsEnv(inputName)
	h.t.Setenv(envName, value)
}

func (h *ConfigTestHelpers) setInputInt(inputName string, value int) {
	h.setInput(inputName, strconv.Itoa(value))
}

func (h *ConfigTestHelpers) setInputMapping(inputName string, mapping map[string]string) {
	if mapping == nil {
		return
	}
	var parts []string
	for key, value := range mapping {
		parts = append(parts, key+":"+value)
	}
	h.setInput(inputName, strings.Join(parts, ";"))
}

func (h *ConfigTestHelpers) setInputList(inputName string, list []string) {
	if list == nil {
		return
	}
	h.setInput(inputName, strings.Join(list, ";"))
}

func (h *ConfigTestHelpers) inputNameAsEnv(name string) string {
	e := strings.ReplaceAll(name, " ", "_")
	e = strings.ToUpper(e)
	return "INPUT_" + e
}

// MinimalConfigOptions allows selectively disabling certain inputs in setupMinimalValidConfig
type MinimalConfigOptions struct {
	SkipGithubRepository bool // Skip setting GITHUB_REPOSITORY
	SkipGithubToken      bool // Skip setting github-token
	SkipSlackBotToken    bool // Skip setting slack-bot-token
	SkipSlackChannelName bool // Skip setting slack-channel-name
	SkipPRListHeading    bool // Skip setting main-list-heading
}

func (h *ConfigTestHelpers) setupMinimalValidConfig(opts ...MinimalConfigOptions) {
	var options MinimalConfigOptions
	if len(opts) > 1 {
		h.t.Fatalf("setupMinimalValidConfig accepts at most one MinimalConfigOptions argument, got %d", len(opts))
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	if !options.SkipGithubRepository {
		h.setEnv(config.EnvGithubRepository, "test-org/test-repo")
	} else {
		h.setEnv(config.EnvGithubRepository, "") // Explicitly clear if skipping
	}
	if !options.SkipGithubToken {
		h.setInput(config.InputGithubToken, "gh_token_123")
	}
	if !options.SkipSlackBotToken {
		h.setInput(config.InputSlackBotToken, "xoxb-slack-token")
	}
	if !options.SkipSlackChannelName {
		h.setInput(config.InputSlackChannelName, "test-channel")
	}
	if !options.SkipPRListHeading {
		h.setInput(config.InputPRListHeading, "PRs needing attention")
	}
}

func (h *ConfigTestHelpers) setupFullValidConfig() {
	h.setupMinimalValidConfig()
	h.setInput(config.InputSlackChannelID, "C1234567890")
	h.setInputInt(config.InputOldPRThresholdHours, 24)
	h.setInput(config.InputNoPRsMessage, "No PRs found!")
	h.setInputMapping(config.InputSlackUserIdByGitHubUsername, map[string]string{
		"alice": "U1234567890",
		"bob":   "U2234567890",
	})
	h.setInputList(config.InputGithubRepositories, []string{
		"test-org/repo1",
		"test-org/repo2",
	})
	h.setInput(config.InputGlobalFilters, `{"authors": ["alice"], "labels": ["feature"]}`)
	h.setInput(config.InputRepositoryFilters, `repo1: {"labels-ignore": ["wip"]}`)
	h.setInputMapping(config.InputRepositoryPrefixes, map[string]string{
		"repo1": "🚀",
		"repo2": "📦",
	})
}

func TestGetConfig_MinimalValid(t *testing.T) {
	h := newConfigTestHelpers(t)
	h.setupMinimalValidConfig()

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.GithubToken != "gh_token_123" {
		t.Errorf("Expected GithubToken 'gh_token_123', got '%s'", cfg.GithubToken)
	}
	if cfg.SlackBotToken != "xoxb-slack-token" {
		t.Errorf("Expected SlackBotToken 'xoxb-slack-token', got '%s'", cfg.SlackBotToken)
	}
	if cfg.SlackChannelName != "test-channel" {
		t.Errorf("Expected SlackChannelName 'test-channel', got '%s'", cfg.SlackChannelName)
	}
	if cfg.ContentInputs.PRListHeading != "PRs needing attention" {
		t.Errorf("Expected PRListHeading 'PRs needing attention', got '%s'", cfg.ContentInputs.PRListHeading)
	}

	if len(cfg.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(cfg.Repositories))
	}
	if cfg.Repositories[0].Path != "test-org/test-repo" {
		t.Errorf("Expected repository path 'test-org/test-repo', got '%s'", cfg.Repositories[0].Path)
	}
	if cfg.Repositories[0].Owner != "test-org" {
		t.Errorf("Expected repository owner 'test-org', got '%s'", cfg.Repositories[0].Owner)
	}
	if cfg.Repositories[0].Name != "test-repo" {
		t.Errorf("Expected repository name 'test-repo', got '%s'", cfg.Repositories[0].Name)
	}

	// Verify optional fields have default values
	if cfg.ContentInputs.OldPRThresholdHours != 0 {
		t.Errorf("Expected OldPRThresholdHours 0, got %d", cfg.ContentInputs.OldPRThresholdHours)
	}
	if cfg.ContentInputs.NoPRsMessage != "" {
		t.Errorf("Expected empty NoPRsMessage, got '%s'", cfg.ContentInputs.NoPRsMessage)
	}
}

func TestGetConfig_FullValid(t *testing.T) {
	h := newConfigTestHelpers(t)
	h.setupFullValidConfig()

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.SlackChannelID != "C1234567890" {
		t.Errorf("Expected SlackChannelID 'C1234567890', got '%s'", cfg.SlackChannelID)
	}
	if cfg.ContentInputs.OldPRThresholdHours != 24 {
		t.Errorf("Expected OldPRThresholdHours 24, got %d", cfg.ContentInputs.OldPRThresholdHours)
	}
	if cfg.ContentInputs.NoPRsMessage != "No PRs found!" {
		t.Errorf("Expected NoPRsMessage 'No PRs found!', got '%s'", cfg.ContentInputs.NoPRsMessage)
	}

	expectedUsers := map[string]string{
		"alice": "U1234567890",
		"bob":   "U2234567890",
	}
	for username, expectedSlackID := range expectedUsers {
		if slackID, exists := cfg.ContentInputs.SlackUserIdByGitHubUsername[username]; !exists {
			t.Errorf("Expected user mapping for '%s' to exist", username)
		} else if slackID != expectedSlackID {
			t.Errorf("Expected slack ID '%s' for user '%s', got '%s'", expectedSlackID, username, slackID)
		}
	}

	if len(cfg.Repositories) != 2 {
		t.Fatalf("Expected 2 repositories, got %d", len(cfg.Repositories))
	}
	expectedRepos := []string{"test-org/repo1", "test-org/repo2"}
	for i, expectedRepo := range expectedRepos {
		if cfg.Repositories[i].Path != expectedRepo {
			t.Errorf("Expected repository %d path '%s', got '%s'", i, expectedRepo, cfg.Repositories[i].Path)
		}
	}

	expectedPrefixes := map[string]string{
		"repo1": "🚀",
		"repo2": "📦",
	}
	for repo, expectedPrefix := range expectedPrefixes {
		if prefix, exists := cfg.ContentInputs.RepositoryPrefixes[repo]; !exists {
			t.Errorf("Expected repository prefix for '%s' to exist", repo)
		} else if prefix != expectedPrefix {
			t.Errorf("Expected prefix '%s' for repo '%s', got '%s'", expectedPrefix, repo, prefix)
		}
	}

	if len(cfg.GlobalFilters.Authors) != 1 || cfg.GlobalFilters.Authors[0] != "alice" {
		t.Errorf("Expected global authors filter ['alice'], got %v", cfg.GlobalFilters.Authors)
	}
	if len(cfg.GlobalFilters.Labels) != 1 || cfg.GlobalFilters.Labels[0] != "feature" {
		t.Errorf("Expected global labels filter ['feature'], got %v", cfg.GlobalFilters.Labels)
	}

	if len(cfg.RepositoryFilters) != 1 {
		t.Fatalf("Expected 1 repository filter, got %d", len(cfg.RepositoryFilters))
	}
	repo1Filters, exists := cfg.RepositoryFilters["repo1"]
	if !exists {
		t.Fatalf("Expected repository filters for 'repo1' to exist")
	}
	if len(repo1Filters.LabelsIgnore) != 1 || repo1Filters.LabelsIgnore[0] != "wip" {
		t.Errorf("Expected repo1 labels-ignore filter ['wip'], got %v", repo1Filters.LabelsIgnore)
	}
}

func TestGetConfig_MissingRequiredInputs(t *testing.T) {
	testCases := []struct {
		name           string
		setupOptions   MinimalConfigOptions
		expectedErrMsg string
	}{
		{
			name: "missing github repository",
			setupOptions: MinimalConfigOptions{
				SkipGithubRepository: true,
			},
			expectedErrMsg: "required input GITHUB_REPOSITORY is not set",
		},
		{
			name: "missing github token",
			setupOptions: MinimalConfigOptions{
				SkipGithubToken: true,
			},
			expectedErrMsg: "required input github-token is not set",
		},
		{
			name: "missing slack bot token",
			setupOptions: MinimalConfigOptions{
				SkipSlackBotToken: true,
			},
			expectedErrMsg: "required input slack-bot-token is not set",
		},
		{
			name: "missing PR list heading",
			setupOptions: MinimalConfigOptions{
				SkipPRListHeading: true,
			},
			expectedErrMsg: "required input main-list-heading is not set",
		},
		{
			name: "missing slack channel",
			setupOptions: MinimalConfigOptions{
				SkipSlackChannelName: true,
			},
			expectedErrMsg: "either slack-channel-id or slack-channel-name must be set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setupMinimalValidConfig(tc.setupOptions)

			_, err := config.GetConfig()
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestGetConfig_InvalidRepository(t *testing.T) {
	testCases := []struct {
		name           string
		repository     string
		expectedErrMsg string
	}{
		{
			name:           "too many slashes",
			repository:     "org/repo/extra",
			expectedErrMsg: "invalid owner/repository format: org/repo/extra",
		},
		{
			name:           "missing repository name",
			repository:     "org/",
			expectedErrMsg: "owner or repository name cannot be empty in: org/",
		},
		{
			name:           "missing owner",
			repository:     "/repo",
			expectedErrMsg: "owner or repository name cannot be empty in: /repo",
		},
		{
			name:           "no slash",
			repository:     "invalid",
			expectedErrMsg: "invalid owner/repository format: invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setupMinimalValidConfig()
			h.setEnv(config.EnvGithubRepository, tc.repository)

			_, err := config.GetConfig()
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestGetConfig_InvalidFilters(t *testing.T) {
	testCases := []struct {
		name           string
		setupFilters   func(*ConfigTestHelpers)
		expectedErrMsg string
	}{
		{
			name: "invalid global filters JSON",
			setupFilters: func(h *ConfigTestHelpers) {
				h.setInput(config.InputGlobalFilters, `{"invalid": "json"}`)
			},
			expectedErrMsg: "error reading input filters: unable to parse filters from",
		},
		{
			name: "conflicting global authors filters",
			setupFilters: func(h *ConfigTestHelpers) {
				h.setInput(config.InputGlobalFilters, `{"authors": ["alice"], "authors-ignore": ["bob"]}`)
			},
			expectedErrMsg: "cannot use both authors and authors-ignore filters at the same time",
		},
		{
			name: "conflicting global labels filters",
			setupFilters: func(h *ConfigTestHelpers) {
				h.setInput(config.InputGlobalFilters, `{"labels": ["feature"], "labels-ignore": ["feature"]}`)
			},
			expectedErrMsg: "labels filter cannot contain labels that are in labels-ignore filter",
		},
		{
			name: "invalid repository filters format",
			setupFilters: func(h *ConfigTestHelpers) {
				h.setInput(config.InputRepositoryFilters, "invalid-format")
			},
			expectedErrMsg: "error reading input repository-filters: invalid mapping format",
		},
		{
			name: "invalid repository filters JSON",
			setupFilters: func(h *ConfigTestHelpers) {
				h.setInput(config.InputRepositoryFilters, `repo1: {"invalid": "json"}`)
			},
			expectedErrMsg: "error parsing filters for repository repo1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newConfigTestHelpers(t)
			h.setupMinimalValidConfig()
			tc.setupFilters(h)

			_, err := config.GetConfig()
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Expected error to contain '%s', got '%s'", tc.expectedErrMsg, err.Error())
			}
		})
	}
}

func TestGetConfig_InvalidOldPRThreshold(t *testing.T) {
	h := newConfigTestHelpers(t)
	h.setupMinimalValidConfig()
	h.setInput(config.InputOldPRThresholdHours, "not-a-number")

	_, err := config.GetConfig()
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	expectedErrMsg := "error parsing input old-pr-threshold-hours as integer"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestConfigPrint(t *testing.T) {
	cfg := config.Config{
		GithubToken:   "secret-github-token",
		SlackBotToken: "secret-slack-token",
	}
	cfg.Print() // Should not panic
}
