package models

type PullRequestRef struct {
	Repository Repository `json:"repository"`
	Number     int        `json:"number"`
}
