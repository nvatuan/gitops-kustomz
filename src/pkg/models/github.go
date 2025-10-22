package models

import "time"

// PullRequest represents GitHub pull request information
type PullRequest struct {
	Number  int
	Title   string
	Body    string
	BaseSHA string
	HeadSHA string
	BaseRef string
	HeadRef string
	State   string
	Merged  bool
	Created time.Time
	Updated time.Time
}

// Comment represents a GitHub comment
type Comment struct {
	ID        int64
	Body      string
	User      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
