package models

const (
	DiffContentTypeText       = "text"
	DiffContentTypeGHArtifact = "ext_ghartifact"
)

type DiffResult struct {
	ContentType      string // "text" or "ext_ghartifact"
	Content          string // diff text OR artifact URL
	LineCount        int
	AddedLineCount   int
	DeletedLineCount int
}
