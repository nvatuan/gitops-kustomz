package models

type DiffResult struct {
	Content          string
	LineCount        int
	AddedLineCount   int
	DeletedLineCount int
}
