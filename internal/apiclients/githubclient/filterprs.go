package githubclient

import (
	"github.com/hellej/pr-slack-reminder-action/internal/config"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
)

func filterPRs(
	prs []PR,
	getFiltersForRepository func(repo models.Repository) config.Filters,
) []PR {
	return utilities.Filter(prs, func(pr PR) bool {
		return pr.isMatch(getFiltersForRepository(pr.Repository))
	})
}
