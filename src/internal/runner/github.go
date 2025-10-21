package runner

import (
	"context"
	"fmt"

	"github.com/gh-nvat/gitops-kustomz/src/pkg/config"
	"github.com/gh-nvat/gitops-kustomz/src/pkg/github"
)

type RunnerGitHub struct {
	ctx     context.Context
	options *Options

	ghclient *github.Client

	prInfo   *config.PullRequest
	comments []*config.Comment
}

func NewRunnerGitHub(
	ctx context.Context,
	options *Options,
	ghclient *github.Client,
) (*RunnerGitHub, error) {
	if ghclient == nil {
		return nil, fmt.Errorf("GitHub client is not initialized")
	}
	return &RunnerGitHub{
		ctx:      ctx,
		options:  options,
		ghclient: ghclient,
	}, nil
}

func (c *RunnerGitHub) Initialize() error {
	if err := c.fetchAndSetPullRequestInfo(); err != nil {
		return fmt.Errorf("failed to fetch pull request info: %w", err)
	}
	return nil
}

// Fetch and set pull request data into struct from GitHub
func (c *RunnerGitHub) fetchAndSetPullRequestInfo() error {
	owner, repo, err := github.ParseOwnerRepo(c.options.GhRepo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Create channels for parallel execution
	type prResult struct {
		pr  *config.PullRequest
		err error
	}
	type commentsResult struct {
		comments []*config.Comment
		err      error
	}

	prChan := make(chan prResult, 1)
	commentsChan := make(chan commentsResult, 1)

	// Fetch PR info in parallel
	go func() {
		pr, err := c.ghclient.GetPR(c.ctx, owner, repo, c.options.GhPrNumber)
		prChan <- prResult{pr: pr, err: err}
	}()

	// Fetch comments in parallel
	go func() {
		comments, err := c.ghclient.GetComments(c.ctx, owner, repo, c.options.GhPrNumber)
		commentsChan <- commentsResult{comments: comments, err: err}
	}()

	// Wait for both results
	select {
	case prRes := <-prChan:
		if prRes.err != nil {
			return fmt.Errorf("failed to get PR info: %w", prRes.err)
		}
		c.prInfo = prRes.pr
	case <-c.ctx.Done():
		return fmt.Errorf("PR fetch cancelled: %w", c.ctx.Err())
	}

	select {
	case commentsRes := <-commentsChan:
		if commentsRes.err != nil {
			return fmt.Errorf("failed to get PR comments: %w", commentsRes.err)
		}
		c.comments = commentsRes.comments
	case <-c.ctx.Done():
		return fmt.Errorf("comments fetch cancelled: %w", c.ctx.Err())
	}

	return nil
}
