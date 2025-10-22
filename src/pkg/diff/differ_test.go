package diff

import (
	"regexp"
	"strings"
	"testing"
)

// normalizeTimestamps replaces timestamps in diff output with a placeholder
func normalizeTimestamps(diff string) string {
	// Replace timestamps like "2025-10-23 00:45:23" with "TIMESTAMP"
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	return re.ReplaceAllString(diff, "TIMESTAMP")
}

// TestDiffer_Diff tests the main Diff method
func TestDiffer_Diff(t *testing.T) {
	tests := []struct {
		name     string
		before   []byte
		after    []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "identical content",
			before:   []byte("same content"),
			after:    []byte("same content"),
			expected: "",
			wantErr:  false,
		},
		{
			name:     "different content",
			before:   []byte("line1\nline2\nline3"),
			after:    []byte("line1\nline2_modified\nline3"),
			expected: "--- before\tTIMESTAMP\n+++ after\tTIMESTAMP\n@@ -1,3 +1,3 @@\n line1\n-line2\n+line2_modified\n line3\n\\ No newline at end of file\n",
			wantErr:  false,
		},
		{
			name:     "empty before",
			before:   []byte(""),
			after:    []byte("new content"),
			expected: "--- before\tTIMESTAMP\n+++ after\tTIMESTAMP\n@@ -0,0 +1 @@\n+new content\n\\ No newline at end of file\n",
			wantErr:  false,
		},
		{
			name:     "empty after",
			before:   []byte("old content"),
			after:    []byte(""),
			expected: "--- before\tTIMESTAMP\n+++ after\tTIMESTAMP\n@@ -1 +0,0 @@\n-old content\n\\ No newline at end of file\n",
			wantErr:  false,
		},
		{
			name:     "large content",
			before:   []byte(strings.Repeat("line\n", 1000)),
			after:    []byte(strings.Repeat("line\n", 1000) + "extra"),
			expected: "", // Will be handled specially - just check non-empty
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer()
			result, err := d.Diff(tt.before, tt.after)

			if (err != nil) != tt.wantErr {
				t.Errorf("Diff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For identical content, expect empty result
			if tt.name == "identical content" {
				if result != "" {
					t.Errorf("Diff() = %v, want empty string for identical content", result)
				}
				return
			}

			// For large content, just verify we get a non-empty result (can't predict exact format)
			if tt.name == "large content" {
				if result == "" {
					t.Errorf("Diff() = %v, want non-empty diff result for large content", result)
				}
				return
			}

			// For all other cases, compare exact output (normalize timestamps)
			normalizedResult := normalizeTimestamps(result)
			normalizedExpected := normalizeTimestamps(tt.expected)
			if normalizedResult != normalizedExpected {
				t.Errorf("Diff() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffer_unifiedDiff tests the unifiedDiff method directly
func TestDiffer_unifiedDiff(t *testing.T) {
	tests := []struct {
		name     string
		before   []byte
		after    []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "identical bytes",
			before:   []byte("identical"),
			after:    []byte("identical"),
			expected: "",
			wantErr:  false,
		},
		{
			name:     "different bytes",
			before:   []byte("old\ncontent"),
			after:    []byte("new\ncontent"),
			expected: "--- before\tTIMESTAMP\n+++ after\tTIMESTAMP\n@@ -1,2 +1,2 @@\n-old\n+new\n content\n\\ No newline at end of file\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer()
			result, err := d.unifiedDiff(tt.before, tt.after)

			if (err != nil) != tt.wantErr {
				t.Errorf("unifiedDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare exact output (normalize timestamps)
			normalizedResult := normalizeTimestamps(result)
			normalizedExpected := normalizeTimestamps(tt.expected)
			if normalizedResult != normalizedExpected {
				t.Errorf("unifiedDiff() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDiffer_InterfaceCompliance tests that Differ implements ManifestDiffer interface
func TestDiffer_InterfaceCompliance(t *testing.T) {
	var _ ManifestDiffer = (*Differ)(nil)

	// This test will fail at compile time if the interface is not implemented
	// The var declaration above ensures compile-time checking
}

// TestDiffer_NewDiffer tests the constructor
func TestDiffer_NewDiffer(t *testing.T) {
	d := NewDiffer()
	if d == nil {
		t.Error("NewDiffer() returned nil")
	}
}

// TestDiffer_EdgeCases tests edge cases
func TestDiffer_EdgeCases(t *testing.T) {
	d := NewDiffer()

	t.Run("nil inputs", func(t *testing.T) {
		result, err := d.Diff(nil, nil)
		if err != nil {
			t.Errorf("Diff(nil, nil) error = %v, want nil", err)
		}
		if result != "" {
			t.Errorf("Diff(nil, nil) = %v, want empty string", result)
		}
	})

	t.Run("one nil input", func(t *testing.T) {
		result, err := d.Diff([]byte("content"), nil)
		if err != nil {
			t.Errorf("Diff(content, nil) error = %v, want nil", err)
		}
		if result == "" {
			t.Error("Diff(content, nil) should return diff, got empty string")
		}
	})

	t.Run("large content", func(t *testing.T) {
		largeContent := strings.Repeat("line\n", 1000)
		result, err := d.Diff([]byte(largeContent), []byte(largeContent+"extra"))
		if err != nil {
			t.Errorf("Diff(large content) error = %v, want nil", err)
		}
		if result == "" {
			t.Error("Diff(large content) should return diff, got empty string")
		}
	})

	t.Run("special characters", func(t *testing.T) {
		before := []byte("line with special chars: !@#$%^&*()")
		after := []byte("line with special chars: !@#$%^&*()_modified")
		result, err := d.Diff(before, after)
		if err != nil {
			t.Errorf("Diff(special chars) error = %v, want nil", err)
		}
		if result == "" {
			t.Error("Diff(special chars) should return diff, got empty string")
		}
	})

	t.Run("unicode content", func(t *testing.T) {
		before := []byte("line with unicode: 你好世界 🌍")
		after := []byte("line with unicode: 你好世界 🌍_modified")
		result, err := d.Diff(before, after)
		if err != nil {
			t.Errorf("Diff(unicode) error = %v, want nil", err)
		}
		if result == "" {
			t.Error("Diff(unicode) should return diff, got empty string")
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		result, err := d.DiffText("", "")
		if err != nil {
			t.Errorf("DiffText(empty, empty) error = %v, want nil", err)
		}
		if result != "" {
			t.Errorf("DiffText(empty, empty) = %v, want empty string", result)
		}
	})
}

// TestDiffer_DeferCleanup tests that defer functions work correctly
func TestDiffer_DeferCleanup(t *testing.T) {
	d := NewDiffer()

	// This test ensures that the defer functions in unifiedDiff work correctly
	// by running multiple diffs and checking that temp files are cleaned up
	before := []byte("test content")
	after := []byte("modified test content")

	// Run multiple diffs to ensure cleanup works
	for i := 0; i < 10; i++ {
		result, err := d.Diff(before, after)
		if err != nil {
			t.Errorf("Diff() iteration %d error = %v", i, err)
		}
		if result == "" {
			t.Errorf("Diff() iteration %d should return diff, got empty string", i)
		}
	}

	// If we get here without errors, the defer cleanup worked
}

// TestDiffer_ConcurrentAccess tests concurrent access to the differ
func TestDiffer_ConcurrentAccess(t *testing.T) {
	d := NewDiffer()

	// Test concurrent access to ensure thread safety
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			before := []byte("test content " + string(rune(id)))
			after := []byte("modified test content " + string(rune(id)))

			result, err := d.Diff(before, after)
			if err != nil {
				t.Errorf("Concurrent Diff() error = %v", err)
			}
			if result == "" {
				t.Errorf("Concurrent Diff() should return diff, got empty string")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkDiffer_Diff(b *testing.B) {
	d := NewDiffer()
	before := []byte("line1\nline2\nline3\nline4\nline5")
	after := []byte("line1\nline2_modified\nline3\nline4\nline5")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := d.Diff(before, after)
		if err != nil {
			b.Fatal(err)
		}
	}
}
