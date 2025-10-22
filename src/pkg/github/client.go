package github

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

const GH_COMMENT_MARKER = template.DefaultCommentTemplate

// GitHubClient defines the interface for GitHub API operations
type GitHubClient interface {
	// GetPR retrieves pull request information
	GetPR(ctx context.Context, owner, repo string, number int) (*models.PullRequest, error)
	// CreateComment creates a new comment on a pull request
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (*models.Comment, error)
	// UpdateComment updates an existing comment
	UpdateComment(ctx context.Context, owner, repo string, commentID int64, body string) error
	// GetComments retrieves all comments for a pull request
	GetComments(ctx context.Context, owner, repo string, number int) ([]*models.Comment, error)
	// FindToolComment finds an existing tool-generated comment
	FindToolComment(ctx context.Context, owner, repo string, number int) (*models.Comment, error)
}

// Client handles GitHub API interactions using go-github
type Client struct {
	client *github.Client
}

// Ensure Client implements GitHubClient
var _ GitHubClient = (*Client)(nil)

// NewClient creates a new GitHub client
func NewClient() (*Client, error) {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub token not found. Set GH_TOKEN or GITHUB_TOKEN environment variable")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &Client{
		client: client,
	}, nil
}

// GetPR retrieves pull request information
func (c *Client) GetPR(ctx context.Context, owner, repo string, number int) (*models.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	return &models.PullRequest{
		Number:  pr.GetNumber(),
		BaseRef: pr.GetBase().GetRef(),
		BaseSHA: pr.GetBase().GetSHA(),
		HeadRef: pr.GetHead().GetRef(),
		HeadSHA: pr.GetHead().GetSHA(),
	}, nil
}

// CreateComment creates a new comment on a pull request
func (c *Client) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*models.Comment, error) {
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	created, _, err := c.client.Issues.CreateComment(ctx, owner, repo, number, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return &models.Comment{
		ID:   created.GetID(),
		Body: created.GetBody(),
	}, nil
}

// UpdateComment updates an existing comment
func (c *Client) UpdateComment(ctx context.Context, owner, repo string, commentID int64, body string) error {
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	_, _, err := c.client.Issues.EditComment(ctx, owner, repo, commentID, comment)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// GetComments retrieves all comments for a pull request
func (c *Client) GetComments(ctx context.Context, owner, repo string, number int) ([]*models.Comment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allComments []*models.Comment
	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get comments: %w", err)
		}

		for _, c := range comments {
			allComments = append(allComments, &models.Comment{
				ID:   c.GetID(),
				Body: c.GetBody(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// FindToolComment finds an existing tool-generated comment
// If multiple comments with the same marker exist, returns the latest one (highest ID)
func (c *Client) FindToolComment(ctx context.Context, owner, repo string, number int) (*models.Comment, error) {
	comments, err := c.GetComments(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}

	var latestComment *models.Comment
	for _, comment := range comments {
		if strings.Contains(comment.Body, GH_COMMENT_MARKER) {
			// If multiple comments exist, keep the one with the highest ID (latest)
			if latestComment == nil || comment.ID > latestComment.ID {
				latestComment = comment
			}
		}
	}

	return latestComment, nil // Returns nil if not found
}
