package diff

import (
	"testing"
)

// ExampleDiffer_Diff demonstrates how to use the Differ
func ExampleDiffer_Diff() {
	d := NewDiffer()

	before := []byte("line1\nline2\nline3")
	after := []byte("line1\nline2_modified\nline3")

	result, err := d.Diff(before, after)
	if err != nil {
		// handle error
		return
	}

	// result will contain the unified diff
	_ = result
}

// ExampleDiffer_DiffText demonstrates how to use DiffText
func ExampleDiffer_DiffText() {
	d := NewDiffer()

	before := "line1\nline2\nline3"
	after := "line1\nline2_modified\nline3"

	result, err := d.DiffText(before, after)
	if err != nil {
		// handle error
		return
	}

	// result will contain the unified diff
	_ = result
}

// TestDiffer_WithMocks demonstrates how to use mocks for testing
func TestDiffer_WithMocks(t *testing.T) {
	helper := NewTestHelper()

	// Set up test content
	before, after := helper.CreateTestContent()

	// Test with the actual differ
	d := NewDiffer()
	result, err := d.Diff(before, after)

	if err != nil {
		t.Errorf("Diff() error = %v", err)
	}

	if result == "" {
		t.Error("Diff() should return diff for different content")
	}

	// Verify the result contains expected diff markers
	if !contains(result, "---") || !contains(result, "+++") {
		t.Errorf("Diff() result should contain diff markers, got: %s", result)
	}
}

// TestDiffer_Performance demonstrates performance testing
func TestDiffer_Performance(t *testing.T) {
	helper := NewTestHelper()

	// Create large test content
	before, after := helper.CreateLargeTestContent()

	d := NewDiffer()

	// Test performance with large content
	result, err := d.Diff(before, after)
	if err != nil {
		t.Errorf("Diff(large content) error = %v", err)
	}

	if result == "" {
		t.Error("Diff(large content) should return diff")
	}
}

// TestDiffer_ErrorHandling demonstrates error handling
func TestDiffer_ErrorHandling(t *testing.T) {
	d := NewDiffer()

	// Test with nil inputs
	result, err := d.Diff(nil, nil)
	if err != nil {
		t.Errorf("Diff(nil, nil) error = %v, want nil", err)
	}
	if result != "" {
		t.Errorf("Diff(nil, nil) = %v, want empty string", result)
	}

	// Test with one nil input
	result, err = d.Diff([]byte("content"), nil)
	if err != nil {
		t.Errorf("Diff(content, nil) error = %v, want nil", err)
	}
	if result == "" {
		t.Error("Diff(content, nil) should return diff")
	}
}

// TestDiffer_Concurrency demonstrates concurrent usage
func TestDiffer_Concurrency(t *testing.T) {
	d := NewDiffer()

	// Test concurrent access
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			before := []byte("test content " + string(rune(id)))
			after := []byte("modified test content " + string(rune(id)))

			result, err := d.Diff(before, after)
			if err != nil {
				t.Errorf("Concurrent Diff() error = %v", err)
			}
			if result == "" {
				t.Errorf("Concurrent Diff() should return diff")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
