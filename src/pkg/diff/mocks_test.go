package diff

import (
	"os"
	"path/filepath"
	"strings"
)

// MockFileSystem provides a mock implementation for file operations
// This can be used for more advanced testing scenarios
type MockFileSystem struct {
	tempFiles map[string]*MockFile
	nextID    int
	errors    map[string]error // operation -> error
}

type MockFile struct {
	name     string
	content  []byte
	closed   bool
	removed  bool
	writeErr error
	closeErr error
}

func (f *MockFile) Write(data []byte) (int, error) {
	if f.closed {
		return 0, os.ErrClosed
	}
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	f.content = append(f.content, data...)
	return len(data), nil
}

func (f *MockFile) Close() error {
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closed = true
	return nil
}

func (f *MockFile) Name() string {
	return f.name
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		tempFiles: make(map[string]*MockFile),
		errors:    make(map[string]error),
	}
}

func (fs *MockFileSystem) CreateTemp(dir, pattern string) (*MockFile, error) {
	if err, exists := fs.errors["CreateTemp"]; exists {
		return nil, err
	}

	fs.nextID++
	name := filepath.Join(dir, strings.Replace(pattern, "*", "temp", 1))
	file := &MockFile{
		name:    name,
		content: []byte{},
	}
	fs.tempFiles[name] = file
	return file, nil
}

func (fs *MockFileSystem) Remove(name string) error {
	if err, exists := fs.errors["Remove"]; exists {
		return err
	}

	if file, exists := fs.tempFiles[name]; exists {
		file.removed = true
		return nil
	}
	return os.ErrNotExist
}

func (fs *MockFileSystem) SetError(operation string, err error) {
	fs.errors[operation] = err
}

func (fs *MockFileSystem) GetFile(name string) *MockFile {
	return fs.tempFiles[name]
}

// MockCommandExecutor provides a mock implementation for command execution
type MockCommandExecutor struct {
	outputs map[string][]byte // command -> output
	errors  map[string]error  // command -> error
}

func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		outputs: make(map[string][]byte),
		errors:  make(map[string]error),
	}
}

func (m *MockCommandExecutor) SetOutput(command string, output []byte) {
	m.outputs[command] = output
}

func (m *MockCommandExecutor) SetError(command string, err error) {
	m.errors[command] = err
}

func (m *MockCommandExecutor) CombinedOutput() ([]byte, error) {
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

// TestHelper provides helper functions for testing
type TestHelper struct {
	fileSystem *MockFileSystem
	executor   *MockCommandExecutor
}

func NewTestHelper() *TestHelper {
	return &TestHelper{
		fileSystem: NewMockFileSystem(),
		executor:   NewMockCommandExecutor(),
	}
}

func (th *TestHelper) GetFileSystem() *MockFileSystem {
	return th.fileSystem
}

func (th *TestHelper) GetExecutor() *MockCommandExecutor {
	return th.executor
}

// CreateTestContent creates test content for diffing
func (th *TestHelper) CreateTestContent() ([]byte, []byte) {
	before := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: test
    image: nginx:1.20
`)

	after := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: test
    image: nginx:1.21
`)

	return before, after
}

// CreateLargeTestContent creates large test content for performance testing
func (th *TestHelper) CreateLargeTestContent() ([]byte, []byte) {
	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		lines[i] = "line " + string(rune(i))
	}

	before := []byte(strings.Join(lines, "\n"))
	after := []byte(strings.Join(lines, "\n") + "\nextra line")

	return before, after
}
