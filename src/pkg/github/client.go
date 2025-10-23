package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/models"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/template"
	"github.com/google/go-github/v66/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var logger = log.WithField("package", "github")

const GH_COMMENT_MARKER = template.ToolCommentSignature

// GitHubClient defines the interface for GitHub API operations
type GitHubClient interface {
	// GetPR retrieves pull request information
	GetPR(ctx context.Context, repo string, number int) (*models.PullRequest, error)
	// CreateComment creates a new comment on a pull request
	CreateComment(ctx context.Context, repo string, number int, body string) (*models.Comment, error)
	// UpdateComment updates an existing comment
	UpdateComment(ctx context.Context, repo string, commentID int64, body string) error
	// GetComments retrieves all comments for a pull request
	GetComments(ctx context.Context, repo string, number int) ([]*models.Comment, error)
	// FindToolComment finds an existing tool-generated comment
	FindToolComment(ctx context.Context, repo string, prNumber int) (*models.Comment, error)
	// SparseCheckoutAtPath clones with treeless and sparse checks out specific ref at path
	SparseCheckoutAtPath(ctx context.Context, cloneURL, ref, path string) (string, error)
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
func (c *Client) GetPR(ctx context.Context, repo string, number int) (*models.PullRequest, error) {
	owner, repo, err := ParseOwnerRepo(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}
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
func (c *Client) CreateComment(ctx context.Context, repo string, number int, body string) (*models.Comment, error) {
	owner, repo, err := ParseOwnerRepo(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}
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
func (c *Client) UpdateComment(ctx context.Context, repo string, commentID int64, body string) error {
	owner, repo, err := ParseOwnerRepo(repo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	commentRes, res, err := c.client.Issues.EditComment(ctx, owner, repo, commentID, comment)
	log.WithField("comment", commentRes).WithField("response", res).Debug("Updated comment")
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// GetComments retrieves all comments for a pull request
// Current limitation it will only fetch first 200 comments, hopefully it contains override messages..
func (c *Client) GetComments(ctx context.Context, repo string, prNumber int) ([]*models.Comment, error) {
	owner, repo, err := ParseOwnerRepo(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 200},
	}

	var allComments []*models.Comment
	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, owner, repo, prNumber, opts)
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
func (c *Client) FindToolComment(ctx context.Context, repo string, prNumber int) (*models.Comment, error) {
	comments, err := c.GetComments(ctx, repo, prNumber)
	if err != nil {
		return nil, err
	}

	var latestComment *models.Comment
	for _, comment := range comments {
		if strings.Contains(comment.Body, GH_COMMENT_MARKER) {
			// If multiple comments exist, for optmization reason, get the first one
			latestComment = comment
			break
		}
	}

	return latestComment, nil // Returns nil if not found
}

// SparseCheckoutAtPath clones with treeless and sparse checks out specific ref at path
// returns the directory containing the checked out files
// It does the following commands:
// 1. git clone --filter=blob:none --depth 1 --no-checkout --single-branch -b branch cloneURL directory
// 2. git sparse-checkout set --no-cone path
// 3. git checkout branch
// 4. return directory
func (c *Client) SparseCheckoutAtPath(ctx context.Context, repo, branch, path string) (string, error) {
	logger.WithField("repo", repo).WithField("branch", branch).WithField("path", path).Info("Sparse checking out at path")

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	chkoutName := strings.ReplaceAll(branch, "/", "_")
	checkoutDir := filepath.Join(cwd, fmt.Sprintf("tmp-checkout-%s-%d", chkoutName, time.Now().Unix()))
	cloneURL, err := GetCloneURLForRepo(repo)
	if err != nil {
		return "", fmt.Errorf("failed to get clone URL: %w", err)
	}

	// 1. git clone --filter=blob:none --depth 1 --no-checkout --single-branch -b branch cloneURL directory
	logger.WithField("checkoutDir", checkoutDir).Info("Cloning...")
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--filter=blob:none", "--depth", "1", "--no-checkout", "--single-branch", "-b", branch, cloneURL, checkoutDir)
	if err := cloneCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone: %w", err)
	}

	// 2. git sparse-checkout set --no-cone path
	logger.WithField("checkoutDir", checkoutDir).Info("Checking out path...")
	sparseCmd := exec.CommandContext(ctx, "git", "sparse-checkout", "set", "--no-cone", path)
	sparseCmd.Dir = checkoutDir
	if err := sparseCmd.Run(); err != nil {
		_ = os.RemoveAll(checkoutDir)
		return "", fmt.Errorf("failed to set sparse checkout: %w", err)
	}

	// 3. git checkout branch
	logger.WithField("checkoutDir", checkoutDir).Info("Checking out branch...")
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", branch)
	checkoutCmd.Dir = checkoutDir
	if err := checkoutCmd.Run(); err != nil {
		_ = os.RemoveAll(checkoutDir)
		return "", fmt.Errorf("failed to checkout: %w", err)
	}

	// 4. return directory
	absPath, err := filepath.Abs(checkoutDir)
	logger.WithField("checkoutDir", checkoutDir).WithField("absPath", absPath).Info("Absolute path...")
	if err != nil {
		_ = os.RemoveAll(checkoutDir)
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}
