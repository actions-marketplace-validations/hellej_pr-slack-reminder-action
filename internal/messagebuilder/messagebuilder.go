// Package messagebuilder constructs Slack Block Kit messages for PR reminders.
// It transforms structured PR content into rich text blocks with formatting,
// links, and user mentions suitable for Slack messaging.
package messagebuilder

import (
	"github.com/hellej/pr-slack-reminder-action/internal/messagecontent"
	"github.com/hellej/pr-slack-reminder-action/internal/prparser"
	"github.com/slack-go/slack"
)

func BuildMessage(content messagecontent.Content) (slack.Message, string) {
	var blocks []slack.Block

	if !content.HasPRs() {
		blocks = addNoPRsBlock(blocks, content.SummaryText)
		return slack.NewBlockMessage(blocks...), content.SummaryText
	}

	blocks = addPRListBLock(blocks, content.PRListHeading, content.PRs)
	return slack.NewBlockMessage(blocks...), content.SummaryText
}

func addNoPRsBlock(blocks []slack.Block, noPRsText string) []slack.Block {
	return append(blocks,
		slack.NewRichTextBlock("no_prs_block",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement(noPRsText, &slack.RichTextSectionTextStyle{}),
			),
		),
	)
}

func addPRListBLock(blocks []slack.Block, heading string, prs []prparser.PR) []slack.Block {
	return append(blocks,
		slack.NewRichTextBlock("pr_list_heading",
			slack.NewRichTextSection(
				slack.NewRichTextSectionTextElement(heading, &slack.RichTextSectionTextStyle{Bold: true}),
			),
		),
		makePRListBlock(prs),
	)
}

func makePRListBlock(openPRs []prparser.PR) *slack.RichTextBlock {
	var prBlocks []slack.RichTextElement
	for _, pr := range openPRs {
		prBlocks = append(prBlocks, buildPRBulletPointBlock(pr))
	}
	return slack.NewRichTextBlock(
		"open_prs",
		slack.NewRichTextList(slack.RichTextListElementType("bullet"), 0,
			prBlocks...,
		),
	)
}

func buildPRBulletPointBlock(pr prparser.PR) slack.RichTextElement {
	var ageElements []slack.RichTextSectionElement

	if pr.IsOldPR {
		ageElements = append(ageElements,
			slack.NewRichTextSectionTextElement(" 🚨 ", &slack.RichTextSectionTextStyle{}),
			slack.NewRichTextSectionTextElement(pr.GetPRAgeText(), &slack.RichTextSectionTextStyle{Bold: true}),
			slack.NewRichTextSectionTextElement(" 🚨", &slack.RichTextSectionTextStyle{}),
		)
	} else {
		ageElements = append(ageElements,
			slack.NewRichTextSectionTextElement(" "+pr.GetPRAgeText(), &slack.RichTextSectionTextStyle{}),
		)
	}

	titleAgeAndAuthorElements := []slack.RichTextSectionElement{}

	if pr.Prefix != "" {
		titleAgeAndAuthorElements = append(titleAgeAndAuthorElements,
			slack.NewRichTextSectionTextElement(pr.Prefix+" ", &slack.RichTextSectionTextStyle{}),
		)
	}

	titleAgeAndAuthorElements = append(titleAgeAndAuthorElements,
		slack.NewRichTextSectionLinkElement(pr.GetHTMLURL(), pr.GetTitle(), &slack.RichTextSectionTextStyle{Bold: true}),
	)
	titleAgeAndAuthorElements = append(titleAgeAndAuthorElements, ageElements...)
	titleAgeAndAuthorElements = append(titleAgeAndAuthorElements,
		slack.NewRichTextSectionTextElement(" by ", &slack.RichTextSectionTextStyle{}),
		getUserNameElement(pr),
	)

	return slack.NewRichTextSection(
		append(titleAgeAndAuthorElements, getReviewersElements(pr)...)...,
	)
}

func getUserNameElement(pr prparser.PR) slack.RichTextSectionElement {
	if pr.Author.SlackUserID != "" {
		return slack.NewRichTextSectionUserElement(
			pr.Author.SlackUserID, &slack.RichTextSectionTextStyle{},
		)
	}
	return slack.NewRichTextSectionTextElement(
		pr.Author.GetGitHubName(), &slack.RichTextSectionTextStyle{},
	)
}

func getReviewersElements(pr prparser.PR) []slack.RichTextSectionElement {
	var elements []slack.RichTextSectionElement
	approverCount := len(pr.Approvers)
	commenterCount := len(pr.Commenters)

	if approverCount == 0 && commenterCount == 0 {
		return append(
			elements, slack.NewRichTextSectionTextElement(
				" (no reviews)", &slack.RichTextSectionTextStyle{},
			),
		)
	}

	reviewerTextPrefix := " (reviewed by "
	if len(pr.Approvers) > 0 {
		reviewerTextPrefix = " (approved by "
	}
	elements = append(elements, slack.NewRichTextSectionTextElement(
		reviewerTextPrefix, &slack.RichTextSectionTextStyle{},
	))

	for idx, approver := range pr.Approvers {
		if idx > 0 {
			elements = append(elements, slack.NewRichTextSectionTextElement(
				", ", &slack.RichTextSectionTextStyle{},
			))
		}
		elements = append(elements, slack.NewRichTextSectionTextElement(
			approver.GetGitHubName(), &slack.RichTextSectionTextStyle{},
		))
	}

	if commenterCount == 0 {
		return append(elements, slack.NewRichTextSectionTextElement(
			")", &slack.RichTextSectionTextStyle{},
		))
	}

	if reviewerTextPrefix == " (approved by " {
		elements = append(elements, slack.NewRichTextSectionTextElement(
			" - reviewed by ", &slack.RichTextSectionTextStyle{},
		))
	}

	for idx, commenter := range pr.Commenters {
		if idx > 0 {
			elements = append(elements, slack.NewRichTextSectionTextElement(
				", ", &slack.RichTextSectionTextStyle{},
			))
		}
		elements = append(elements, slack.NewRichTextSectionTextElement(
			commenter.GetGitHubName(), &slack.RichTextSectionTextStyle{},
		))
	}

	return append(elements, slack.NewRichTextSectionTextElement(
		")", &slack.RichTextSectionTextStyle{},
	))
}
