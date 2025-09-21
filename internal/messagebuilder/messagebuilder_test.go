package messagebuilder_test

import (
	"testing"
	"time"

	"github.com/google/go-github/v72/github"
	"github.com/slack-go/slack"

	"github.com/hellej/pr-slack-reminder-action/internal/apiclients/githubclient"
	"github.com/hellej/pr-slack-reminder-action/internal/messagebuilder"
	"github.com/hellej/pr-slack-reminder-action/internal/messagecontent"
	"github.com/hellej/pr-slack-reminder-action/internal/prparser"
)

func TestComposeSlackBlocksMessage(t *testing.T) {
	t.Run("No PRs", func(t *testing.T) {
		content := messagecontent.Content{
			SummaryText: "No open PRs, happy coding! 🎉",
		}

		message, _ := messagebuilder.BuildMessage(content)

		blockLen := len(message.Blocks.BlockSet)
		if blockLen != 1 {
			t.Errorf("Expected there to be exactly one block, got %d", blockLen)
		}

		firstBlock := message.Blocks.BlockSet[0]
		if firstBlock.BlockType() != "rich_text" {
			t.Errorf("Expected first block to be of type 'rich_text', was '%s'", firstBlock.BlockType())
		}

		richTextElement := firstBlock.(*slack.RichTextBlock).Elements[0].(*slack.RichTextSection).Elements[0].(*slack.RichTextSectionTextElement)
		if richTextElement.Text != content.SummaryText {
			t.Errorf("Expected text to be '%s', got '%s'", content.SummaryText, richTextElement.Text)
		}
	})

	t.Run("Message summary", func(t *testing.T) {
		testPRs := getTestPRs()
		content := messagecontent.Content{
			SummaryText:   "1 open PRs are waiting for attention 👀",
			PRListHeading: "🚀 New PRs since 1 days ago",
			PRs:           testPRs.PRs,
		}
		_, got := messagebuilder.BuildMessage(content)
		if got != content.SummaryText {
			t.Errorf("Expected summary to be '%s', got '%s'", content.SummaryText, got)
		}
	})

	t.Run("One new PR", func(t *testing.T) {
		testPRs := getTestPRs()

		content := messagecontent.Content{
			SummaryText:   "1 open PRs are waiting for attention 👀",
			PRListHeading: "🚀 New PRs since 1 days ago",
			PRs:           testPRs.PRs,
		}
		got, _ := messagebuilder.BuildMessage(content)

		if len(got.Blocks.BlockSet) < 2 {
			t.Errorf("Expected non-empty blocks, got nil or empty")
		}
		firstBlock := got.Blocks.BlockSet[0]
		header := firstBlock.(*slack.RichTextBlock).Elements[0].(*slack.RichTextSection).Elements[0].(*slack.RichTextSectionTextElement)
		if header.Text != content.PRListHeading {
			t.Errorf("Expected '%s', got '%s'", content.PRListHeading, header.Text)
		}
		prBulletPointTextElements := got.Msg.Blocks.BlockSet[1].(*slack.RichTextBlock).Elements[0].(*slack.RichTextList).Elements[0].(*slack.RichTextSection).Elements
		prLinkElement := prBulletPointTextElements[0].(*slack.RichTextSectionLinkElement)
		prAgeElement := prBulletPointTextElements[1].(*slack.RichTextSectionTextElement)
		prBeforeUserElement := prBulletPointTextElements[2].(*slack.RichTextSectionTextElement)
		prUserElement := prBulletPointTextElements[3].(*slack.RichTextSectionUserElement)
		if prLinkElement.Text != *testPRs.PR1.Title {
			t.Errorf("Expected text to be '%s', got '%s'", *testPRs.PR1.Title, prLinkElement.Text)
		}
		expectedAgeText := " 3 hours ago"
		if prAgeElement.Text != expectedAgeText {
			t.Errorf("Expected text to be '%s', got '%s'", expectedAgeText, prAgeElement.Text)
		}
		expectedBeforeUserText := " by "
		if prBeforeUserElement.Text != expectedBeforeUserText {
			t.Errorf("Expected text to be '%s', got '%s'", expectedBeforeUserText, prAgeElement.Text)
		}
		if prUserElement.UserID != testPRs.PR1.Author.SlackUserID {
			t.Errorf("Expected text to be '%s', got '%s'", testPRs.PR1.Author.SlackUserID, prUserElement.UserID)
		}
	})

	t.Run("Grouped by repository", func(t *testing.T) {
		content := messagecontent.Content{
			SummaryText:         "2 open PRs are waiting for attention 👀",
			GroupedByRepository: true,
			PRsGroupedByRepository: []messagecontent.PRsOfRepository{
				{
					HeadingPrefix:       "Open PRs in ",
					RepositoryLinkLabel: "owner/repo-name",
					RepositoryLink:      "https://github.com/owner/repo-name",
					PRs:                 getTestPRs().PRs,
				},
				{
					HeadingPrefix:       "Open PRs in ",
					RepositoryLinkLabel: "another-org/special-chars_repo",
					RepositoryLink:      "https://github.com/another-org/special-chars_repo",
					PRs:                 getTestPRs().PRs,
				},
			},
		}

		message, summaryText := messagebuilder.BuildMessage(content)

		if summaryText != content.SummaryText {
			t.Errorf("Expected summary to be '%s', got '%s'", content.SummaryText, summaryText)
		}

		if len(message.Blocks.BlockSet) != 4 {
			t.Errorf("Expected 4 blocks, got %d", len(message.Blocks.BlockSet))
		}

		firstHeadingBlock := message.Blocks.BlockSet[0].(*slack.RichTextBlock)

		firstSection := firstHeadingBlock.Elements[0].(*slack.RichTextSection)
		if len(firstSection.Elements) != 3 { // prefix + link + colon
			t.Errorf("Expected 3 elements in first section, got %d", len(firstSection.Elements))
		}

		prefixElement := firstSection.Elements[0].(*slack.RichTextSectionTextElement)
		if prefixElement.Text != "Open PRs in " {
			t.Errorf("Expected prefix 'Open PRs in ', got '%s'", prefixElement.Text)
		}

		linkElement := firstSection.Elements[1].(*slack.RichTextSectionLinkElement)
		if linkElement.Text != "owner/repo-name" {
			t.Errorf("Expected link text 'owner/repo-name', got '%s'", linkElement.Text)
		}
		if linkElement.URL != "https://github.com/owner/repo-name" {
			t.Errorf("Expected link URL 'https://github.com/owner/repo-name', got '%s'", linkElement.URL)
		}
	})
}

type TestPRs struct {
	PRs []prparser.PR
	PR1 prparser.PR
}

func getTestPRs() TestPRs {
	pr1 := prparser.PR{
		PR: &githubclient.PR{
			PullRequest: &github.PullRequest{
				CreatedAt: &github.Timestamp{Time: time.Now().Add(-3 * time.Hour)}, // 1 day ago
				Title:     github.Ptr("This is a test PR"),
				User: &github.User{
					Login: github.Ptr("testuser"),
					Name:  github.Ptr("Test User"),
				},
			},
		},
	}
	pr1.Author = prparser.Collaborator{
		Collaborator: &githubclient.Collaborator{
			Login: "Test User",
		},
		SlackUserID: "U12345678",
	}
	return TestPRs{
		PRs: []prparser.PR{pr1},
		PR1: pr1,
	}
}
