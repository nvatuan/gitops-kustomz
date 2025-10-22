package diff

import "strings"

// CalcLineChangesFromDiffContent calculates the number of added and deleted lines from a diff content
// returns: addedLines, deletedLines, totalLines
// operate on `diff -u` output, so other options may not work
func CalcLineChangesFromDiffContent(diffContent string) (int, int, int) {
	addedLines := 0
	deletedLines := 0
	for _, line := range strings.Split(diffContent, "\n") {
		if strings.HasPrefix(line, "+ ") {
			addedLines++
		}
		if strings.HasPrefix(line, "- ") {
			deletedLines++
		}
	}
	return addedLines, deletedLines, addedLines + deletedLines
}
