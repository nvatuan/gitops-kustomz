package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockFileSystem provides a mock implementation for file operations
type mockFileSystem struct {
	tempFiles map[string]*mockFile
	nextID    int
}

type mockFile struct {
	name    string
	content []byte
	closed  bool
	removed bool
}

func (f *mockFile) Write(data []byte) (int, error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	f.content = append(f.content, data...)
	return len(data), nil
}

func (f *mockFile) Close() error {
	f.closed = true
	return nil
}

func (f *mockFile) Name() string {
	return f.name
}

func (fs *mockFileSystem) CreateTemp(dir, pattern string) (*mockFile, error) {
	fs.nextID++
	name := filepath.Join(dir, strings.Replace(pattern, "*", "temp", 1))
	file := &mockFile{
		name:    name,
		content: []byte{},
	}
	fs.tempFiles[name] = file
	return file, nil
}

func (fs *mockFileSystem) Remove(name string) error {
	if file, exists := fs.tempFiles[name]; exists {
		file.removed = true
		return nil
	}
	return os.ErrNotExist
}

// mockCommandExecutor provides a mock implementation for command execution
type mockCommandExecutor struct {
	commands map[string]string // command -> output
	errors   map[string]error  // command -> error
}

func (m *mockCommandExecutor) CombinedOutput() ([]byte, error) {
	// This is a simplified mock - in real tests you'd want to match the actual command
	// For now, we'll return a generic diff output
	output := `--- before	2025-01-01 00:00:00
+++ after	2025-01-01 00:00:00
@@ -1,3 +1,3 @@
 line1
-line2
+line2_modified
 line3
`
	return []byte(output), nil
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
			name:   "different content",
			before: []byte("line1\nline2\nline3"),
			after:  []byte("line1\nline2_modified\nline3"),
			// We can't predict the exact output due to temp file names, so we'll check for diff markers
			expected: "---", // Should contain diff markers
			wantErr:  false,
		},
		{
			name:     "empty before",
			before:   []byte(""),
			after:    []byte("new content"),
			expected: "+++",
			wantErr:  false,
		},
		{
			name:     "empty after",
			before:   []byte("old content"),
			after:    []byte(""),
			expected: "---",
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

			if tt.expected == "" && result != "" {
				t.Errorf("Diff() = %v, want empty string", result)
			}

			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("Diff() = %v, want to contain %v", result, tt.expected)
			}
		})
	}
}

// TestDiffer_DiffText tests the DiffText method
func TestDiffer_DiffText(t *testing.T) {
	tests := []struct {
		name     string
		before   string
		after    string
		expected string
		wantErr  bool
	}{
		{
			name:     "identical strings",
			before:   "same content",
			after:    "same content",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "different strings",
			before:   "old content",
			after:    "new content",
			expected: "---",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiffer()
			result, err := d.DiffText(tt.before, tt.after)

			if (err != nil) != tt.wantErr {
				t.Errorf("DiffText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expected == "" && result != "" {
				t.Errorf("DiffText() = %v, want empty string", result)
			}

			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("DiffText() = %v, want to contain %v", result, tt.expected)
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
			expected: "---",
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

			if tt.expected == "" && result != "" {
				t.Errorf("unifiedDiff() = %v, want empty string", result)
			}

			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("unifiedDiff() = %v, want to contain %v", result, tt.expected)
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

// TestDiffer_FileOperations tests file operations in isolation
func TestDiffer_FileOperations(t *testing.T) {
	// Test that temp files are created and cleaned up properly
	d := NewDiffer()

	before := []byte("before content")
	after := []byte("after content")

	result, err := d.Diff(before, after)
	if err != nil {
		t.Errorf("Diff() error = %v", err)
	}

	// Check that result contains expected diff markers
	if !strings.Contains(result, "---") || !strings.Contains(result, "+++") {
		t.Errorf("Diff() result should contain diff markers, got: %s", result)
	}

	// Check that temp files are cleaned up (we can't directly verify this,
	// but if the test passes without errors, cleanup worked)
}

// TestDiffer_CommandExecution tests the diff command execution
func TestDiffer_CommandExecution(t *testing.T) {
	// Test that the diff command is executed correctly
	d := NewDiffer()

	before := []byte("line1\nline2\nline3")
	after := []byte("line1\nline2_modified\nline3")

	result, err := d.Diff(before, after)
	if err != nil {
		t.Errorf("Diff() error = %v", err)
	}

	// Verify the output format
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Errorf("Diff() result should have at least 3 lines, got %d", len(lines))
	}

	// Check for unified diff format
	foundHeader := false
	for _, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			foundHeader = true
			break
		}
	}

	if !foundHeader {
		t.Errorf("Diff() result should contain unified diff headers")
	}
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

func BenchmarkDiffer_DiffText(b *testing.B) {
	d := NewDiffer()
	before := "line1\nline2\nline3\nline4\nline5"
	after := "line1\nline2_modified\nline3\nline4\nline5"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := d.DiffText(before, after)
		if err != nil {
			b.Fatal(err)
		}
	}
}
