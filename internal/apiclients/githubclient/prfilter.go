package githubclient

import (
	"slices"

	"github.com/google/go-github/v78/github"
	"github.com/hellej/pr-slack-reminder-action/internal/config"
)

func includePR(pr *github.PullRequest, filters config.Filters) bool {
	if len(filters.LabelsIgnore) > 0 {
		if slices.ContainsFunc(pr.Labels, func(l *github.Label) bool {
			return slices.Contains(filters.LabelsIgnore, l.GetName())
		}) {
			return false
		}
	}
	if len(filters.AuthorsIgnore) > 0 {
		if slices.Contains(filters.AuthorsIgnore, pr.GetUser().GetLogin()) {
			return false
		}
	}
	if len(filters.Labels) > 0 {
		if !slices.ContainsFunc(pr.Labels, func(l *github.Label) bool {
			return slices.Contains(filters.Labels, l.GetName())
		}) {
			return false
		}
	}
	if len(filters.Authors) > 0 {
		if !slices.Contains(filters.Authors, pr.GetUser().GetLogin()) {
			return false
		}
	}
	return true
}
