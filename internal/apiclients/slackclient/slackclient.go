// Package slackclient provides Slack API integration for sending messages.
// It handles channel resolution by name and message posting with Block Kit formatting.
package slackclient

import (
	"errors"
	"fmt"
	"log"

	"github.com/hellej/pr-slack-reminder-action/internal/utilities"
	"github.com/slack-go/slack"
)

type Client interface {
	GetChannelIDByName(channelName string) (string, error)
	SendMessage(channelID string, blocks slack.Message, summaryText string) error
}

func GetAuthenticatedClient(token string) Client {
	return NewClient(slack.New(token))
}

func NewClient(slackAPI SlackAPI) Client {
	return &client{slackAPI: slackAPI}
}

// represents the Slack API methods relevant to us from github.com/slack-go/slack
type SlackAPI interface {
	GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type client struct {
	slackAPI SlackAPI
}

func (c *client) GetChannelIDByName(channelName string) (string, error) {
	for _, channelType := range []string{"public_channel", "private_channel"} {
		channels, err := c.fetchChannels([]string{channelType})
		if err != nil {
			return "", fmt.Errorf(
				"%v (check channel name, token and permissions or use channel ID input instead)",
				err,
			)
		}
		channel, found := utilities.Find(channels, func(ch slack.Channel) bool {
			return ch.Name == channelName
		})
		if found {
			return channel.ID, nil
		}
	}
	return "", errors.New("channel not found")
}

func (c *client) SendMessage(channelID string, blocks slack.Message, summaryText string) error {
	_, _, err := c.slackAPI.PostMessage(
		channelID,
		slack.MsgOptionBlocks(blocks.Blocks.BlockSet...),
		slack.MsgOptionText(summaryText, false),
	)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %v", err)
	}
	log.Printf("Sent message to Slack channel: %s", channelID)
	return nil
}

func (c *client) fetchChannels(types []string) ([]slack.Channel, error) {
	channels, cursor := []slack.Channel{}, ""

	for {
		result, nextCursor, err := c.slackAPI.GetConversations(&slack.GetConversationsParameters{
			Limit:           999,
			Cursor:          cursor,
			Types:           types,
			ExcludeArchived: true,
		})
		if err != nil {
			return nil, err
		}
		channels = append(channels, result...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return channels, nil
}
