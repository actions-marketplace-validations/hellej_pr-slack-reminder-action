// Package config handles GitHub Action input parsing and validation.
// It converts environment variables to structured configuration with
// support for repository-specific filters, user mappings, and content settings.
package config

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hellej/pr-slack-reminder-action/internal/config/inputhelpers"

	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

const (
	EnvGithubRepository string = "GITHUB_REPOSITORY"

	InputGithubRepositories          string = "github-repositories"
	InputRunMode                     string = "mode"
	InputStateFilePath               string = "state-file-path"
	InputGithubToken                 string = "github-token"
	InputSlackBotToken               string = "slack-bot-token"
	InputSlackChannelName            string = "slack-channel-name"
	InputSlackChannelID              string = "slack-channel-id"
	InputSlackUserIdByGitHubUsername string = "github-user-slack-user-id-mapping"
	InputNoPRsMessage                string = "no-prs-message"
	InputPRListHeading               string = "main-list-heading"
	InputOldPRThresholdHours         string = "old-pr-threshold-hours"
	InputGlobalFilters               string = "filters"
	InputRepositoryFilters           string = "repository-filters"
	InputRepositoryPrefixes          string = "repository-prefixes"
	InputGroupByRepository           string = "group-by-repository"

	MaxRepositories int = 50

	DefaultRunMode       = RunModePost
	DefaultStateFilePath = ".pr-slack-reminder/state.json"
)

type RunMode string

const (
	RunModePost   RunMode = "post"
	RunModeUpdate RunMode = "update"
)

// ParseRunMode validates a raw string as a RunMode.
// It returns an error for unsupported values; defaulting is handled by callers.
func ParseRunMode(raw string) (RunMode, error) {
	switch raw {
	case string(RunModePost):
		return RunModePost, nil
	case string(RunModeUpdate):
		return RunModeUpdate, nil
	default:
		return "", fmt.Errorf("invalid run mode: %s (expected '%s' or '%s')", raw, RunModePost, RunModeUpdate)
	}
}

type Config struct {
	GithubToken   string
	SlackBotToken string

	RunMode       RunMode
	StateFilePath string

	repository   string
	Repositories []models.Repository

	SlackChannelName string
	SlackChannelID   string

	GlobalFilters     Filters
	RepositoryFilters map[string]Filters
	ContentInputs     ContentInputs
}

type ContentInputs struct {
	NoPRsMessage                string
	PRListHeading               string
	OldPRThresholdHours         int
	RepositoryPrefixes          map[string]string
	SlackUserIdByGitHubUsername map[string]string
	GroupByRepository           bool
}

func (c Config) Print() {
	copy := c
	if copy.GithubToken != "" {
		copy.GithubToken = "XXXXX"
	}
	if copy.SlackBotToken != "" {
		copy.SlackBotToken = "XXXXX"
	}
	asJson, _ := json.MarshalIndent(copy, "", "  ")
	log.Print("Configuration:")
	log.Println(string(asJson))
}

func GetConfig() (Config, error) {
	repository, err1 := inputhelpers.GetEnvRequired(EnvGithubRepository)
	githubToken, err2 := inputhelpers.GetInputRequired(InputGithubToken)
	slackToken, err3 := inputhelpers.GetInputRequired(InputSlackBotToken)
	mainListHeading := inputhelpers.GetInput(InputPRListHeading)
	oldPRsThresholdHours, err4 := inputhelpers.GetInputInt(InputOldPRThresholdHours)
	slackUserIdByGitHubUsername, err5 := inputhelpers.GetInputMapping(InputSlackUserIdByGitHubUsername)
	globalFilters, err6 := GetGlobalFiltersFromInput(InputGlobalFilters)
	repositoryFilters, err7 := GetRepositoryFiltersFromInput(InputRepositoryFilters)
	repositoryPrefixes, err8 := inputhelpers.GetInputMapping(InputRepositoryPrefixes)
	groupByRepository, err9 := inputhelpers.GetInputBool(InputGroupByRepository)
	runMode, err10 := ParseRunMode(inputhelpers.GetInputOr(InputRunMode, string(DefaultRunMode)))
	stateFilePath := inputhelpers.GetInputOr(InputStateFilePath, DefaultStateFilePath)

	if err := selectNonNilError(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10); err != nil {
		return Config{}, err
	}

	repositoryPaths := inputhelpers.GetInputList(InputGithubRepositories)
	if len(repositoryPaths) == 0 {
		repositoryPaths = []string{repository}
	}

	repositories, err := utilities.MapWithError(repositoryPaths, func(repoPath string) (models.Repository, error) {
		return models.ParseRepository(repoPath)
	})
	if err != nil {
		return Config{}, fmt.Errorf("invalid repositories input: %v", err)
	}

	config := Config{
		RunMode:          runMode,
		StateFilePath:    stateFilePath,
		repository:       repository,
		Repositories:     repositories,
		GithubToken:      githubToken,
		SlackBotToken:    slackToken,
		SlackChannelName: inputhelpers.GetInput(InputSlackChannelName),
		SlackChannelID:   inputhelpers.GetInput(InputSlackChannelID),
		ContentInputs: ContentInputs{
			SlackUserIdByGitHubUsername: slackUserIdByGitHubUsername,
			NoPRsMessage:                inputhelpers.GetInput(InputNoPRsMessage),
			PRListHeading:               mainListHeading,
			OldPRThresholdHours:         oldPRsThresholdHours,
			RepositoryPrefixes:          repositoryPrefixes,
			GroupByRepository:           groupByRepository,
		},
		GlobalFilters:     globalFilters,
		RepositoryFilters: repositoryFilters,
	}

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c Config) GetFiltersForRepository(repo models.Repository) Filters {
	filters, exists := c.RepositoryFilters[repo.Path]
	if exists {
		return filters
	}
	filters, exists = c.RepositoryFilters[repo.Name]
	if exists {
		return filters
	}
	return c.GlobalFilters
}

// validate performs post-construction validation of business rules for Config.
// It validates repository limits, Slack channel requirements and repository names.
func (c Config) validate() error {
	if c.RunMode != RunModePost && c.RunMode != RunModeUpdate {
		return fmt.Errorf("invalid run mode: %s (expected '%s' or '%s')", c.RunMode, RunModePost, RunModeUpdate)
	}
	if len(c.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be specified in %s or %s", EnvGithubRepository, InputGithubRepositories)
	}
	if c.SlackChannelID == "" && c.SlackChannelName == "" {
		return fmt.Errorf("either %s or %s must be set", InputSlackChannelID, InputSlackChannelName)
	}
	if len(c.Repositories) > MaxRepositories {
		return fmt.Errorf("too many repositories: maximum of %d repositories allowed, got %d", MaxRepositories, len(c.Repositories))
	}
	if err := c.validateRepositoryNames(); err != nil {
		return err
	}
	if err := c.validateHeadingOptions(); err != nil {
		return err
	}

	return nil
}

func (c Config) validateRepositoryNames() error {
	if err := validateDuplicateRepositories(c.Repositories); err != nil {
		return err
	}
	if err := validateRepositoryReferences(
		c.Repositories, c.RepositoryFilters, "repository-filters",
	); err != nil {
		return err
	}
	if err := validateRepositoryReferences(
		c.Repositories, c.ContentInputs.RepositoryPrefixes, "repository-prefixes",
	); err != nil {
		return err
	}
	return nil
}

func validateDuplicateRepositories(repositories []models.Repository) error {
	repositoryPaths := make(map[string]bool, len(repositories))
	for _, repo := range repositories {
		if repositoryPaths[repo.Path] {
			return fmt.Errorf("duplicate repository '%s' found in github-repositories", repo.Path)
		}
		repositoryPaths[repo.Path] = true
	}
	return nil
}

// validateRepositoryReferences checks if any repository name appears multiple times
// across different owners, which would make repository-specific configurations ambiguous.
func validateRepositoryReferences[V any](
	repositories []models.Repository,
	repoMapping map[string]V,
	inputName string,
) error {
	for repoNameOrPath := range repoMapping {
		matches := utilities.Filter(repositories, func(r models.Repository) bool {
			return r.Path == repoNameOrPath || r.Name == repoNameOrPath
		})

		switch len(matches) {
		case 1:
			continue
		case 0:
			return fmt.Errorf(
				"%s contains entry for '%s' which does not match any repository",
				inputName,
				repoNameOrPath,
			)
		default: // matches > 1
			return fmt.Errorf(
				"%s contains ambiguous entry for '%s' which matches "+
					"multiple repositories (needs owner/repo format)",
				inputName,
				repoNameOrPath,
			)
		}
	}
	return nil
}

func (c Config) validateHeadingOptions() error {
	if !c.ContentInputs.GroupByRepository && c.ContentInputs.PRListHeading == "" {
		return fmt.Errorf("%s is required when group-by-repository is false", InputPRListHeading)
	}
	return nil
}

func selectNonNilError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
