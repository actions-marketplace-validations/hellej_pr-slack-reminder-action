package githubclient

import (
	"cmp"
	"log"
	"slices"

	"github.com/google/go-github/v72/github"
	"github.com/hellej/pr-slack-reminder-action/internal/models"
)

type PRsOfRepoResult struct {
	prs        []*github.PullRequest
	repository models.Repository
	err        error
}

func (r PRsOfRepoResult) GetPRCount() int {
	if r.prs != nil {
		return len(r.prs)
	}
	return 0
}

type PR struct {
	*github.PullRequest
	Repository       models.Repository
	Author           Collaborator
	ApprovedByUsers  []Collaborator
	CommentedByUsers []Collaborator // reviewers who commented the PR but did not approve it
}

type FetchReviewsResult struct {
	pr         *github.PullRequest
	reviews    []*github.PullRequestReview
	repository models.Repository
	err        error
}

func (r FetchReviewsResult) printResult() {
	if r.err != nil {
		log.Printf("Unable to fetch reviews for PR #%d: %v", r.pr.GetNumber(), r.err)
	} else {
		log.Printf("Found %d reviews for PR %v/%d", len(r.reviews), r.repository, r.pr.GetNumber())
	}
	for _, review := range r.reviews {
		log.Printf(
			"Review by %s (name: %s): %s",
			review.GetUser().GetLogin(),
			review.GetUser().GetName(),
			*review.State,
		)
	}
}

type Collaborator struct {
	Login string // GitHub username
	Name  string // GitHub name if available
}

func NewCollaboratorFromUser(user *github.User) Collaborator {
	return Collaborator{
		Login: user.GetLogin(),
		Name:  user.GetName(),
	}
}

func isBot(user *github.User) bool {
	userType := user.GetType()
	return userType == "Bot"
}

// Returns the GitHub name if available, otherwise login.
func (c Collaborator) GetGitHubName() string {
	return cmp.Or(c.Name, c.Login)
}

func (r FetchReviewsResult) asPR() PR {
	approvedByUsers := []Collaborator{}
	commentedByUsers := []Collaborator{}

	for _, review := range r.reviews {
		user := review.GetUser()
		if user == nil {
			continue
		}
		login := user.GetLogin()
		if login == "" || isBot(user) {
			continue
		}
		if review.GetState() == "APPROVED" {
			if !slices.ContainsFunc(approvedByUsers, func(c Collaborator) bool {
				return c.Login == login
			}) {
				approvedByUsers = append(
					approvedByUsers, NewCollaboratorFromUser(user),
				)
			}
		} else {
			// add to commentedByUsers unless...
			if r.pr.GetUser().GetLogin() == login { // i.e. is the author
				continue
			}
			if slices.ContainsFunc(commentedByUsers, func(c Collaborator) bool {
				return c.Login == login
			}) || slices.ContainsFunc(approvedByUsers, func(c Collaborator) bool {
				return c.Login == login // has already approved
			}) {
				continue
			}
			commentedByUsers = append(
				commentedByUsers, NewCollaboratorFromUser(user),
			)
		}
	}

	return PR{
		PullRequest:      r.pr,
		Repository:       r.repository,
		Author:           NewCollaboratorFromUser(r.pr.GetUser()),
		ApprovedByUsers:  approvedByUsers,
		CommentedByUsers: commentedByUsers,
	}
}

type OwnerAndRepo struct {
	Owner string
	Repo  string
}
