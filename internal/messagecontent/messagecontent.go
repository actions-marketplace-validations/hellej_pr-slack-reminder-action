package messagecontent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hellej/pr-slack-reminder-action/internal/config"
	"github.com/hellej/pr-slack-reminder-action/internal/prparser"
)

type Content struct {
	SummaryText   string
	PRListHeading string
	PRList        []prparser.PR
}

func (c Content) HasPRs() bool {
	return len(c.PRList) > 0
}

type PRCategory struct {
	Heading string
	PRs     []prparser.PR
}

func formatListHeading(heading string, prCount int) string {
	return strings.ReplaceAll(heading, "<pr_count>", strconv.Itoa(prCount))
}

func getSummaryText(prCount int) string {
	if prCount == 1 {
		return "1 open PR is waiting for attention 👀"
	}
	return fmt.Sprintf("%d open PRs are waiting for attention 👀", prCount)
}

func setPRIsOldProperty(prs []prparser.PR, oldPRThresholdHours int) []prparser.PR {
	if oldPRThresholdHours == 0 {
		return prs
	}

	processedPRs := make([]prparser.PR, len(prs))
	for i, pr := range prs {
		processedPRs[i] = pr
		processedPRs[i].IsOldPR = pr.IsOlderThan(oldPRThresholdHours)
	}
	return processedPRs
}

func GetContent(openPRs []prparser.PR, contentInputs config.ContentInputs) Content {
	processedPRs := setPRIsOldProperty(openPRs, contentInputs.OldPRThresholdHours)

	switch {
	case len(processedPRs) == 0:
		return Content{
			SummaryText: contentInputs.NoPRsMessage,
		}
	default:
		return Content{
			SummaryText:   getSummaryText(len(processedPRs)),
			PRListHeading: formatListHeading(contentInputs.PRListHeading, len(processedPRs)),
			PRList:        processedPRs,
		}
	}
}
