package githubclient

import (
	"slices"
	"strings"

	"github.com/google/go-github/v78/github"
	"github.com/hellej/pr-slack-reminder-action/internal/config"
)

func includePR(pr *github.PullRequest, filters config.Filters) bool {
	title := pr.GetTitle()
	for _, ignoredTerm := range filters.IgnoredTerms {
		if strings.Contains(title, ignoredTerm) {
			return false
		}
	}

	if len(filters.IgnoredLabels) > 0 {
		if slices.ContainsFunc(pr.Labels, func(l *github.Label) bool {
			return slices.Contains(filters.IgnoredLabels, l.GetName())
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
