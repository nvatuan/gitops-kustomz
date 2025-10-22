package github

import (
	"fmt"
	"strings"
)

// ParseRepo parses a repository string into owner and repository
// Example: "owner/repository" -> "owner", "repository"
// Example: "owner/repository/subpath" -> "owner", "repository"
func ParseOwnerRepo(repo string) (owner, repository string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository format: %s", repo)
	}
	owner = parts[0]
	repository = parts[1]
	return owner, repository, nil
}

func ShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
