package tee

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTeeWriter(t *testing.T) {
	// Test with enabled but no output dir
	_, err := NewTeeWriter(TeeConfig{
		Enabled:    true,
		OutputDir:  "",
	})
	if err == nil {
		t.Error("expected error when enabled but no output dir")
	}

	// Test with disabled
	writer, err := NewTeeWriter(TeeConfig{
		Enabled: false,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if writer == nil {
		t.Error("expected writer to be created")
	}
}

func TestNewTeeWriterWithDir(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:    true,
		OutputDir:  tmpDir,
		MaxFiles:   100,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if writer == nil {
		t.Error("expected writer to be created")
	}

	// Verify directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:    true,
		OutputDir:  tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	output := &TeeOutput{
		ID:        "test-001",
		Command:   "echo 'hello world'",
		ExitCode:  0,
		Stdout:    "hello world\n",
		Stderr:    "",
		Combined:  "hello world\n",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	filePath, err := writer.Write(ctx, output)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if filePath == "" {
		t.Error("expected file path to be returned")
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}

	// Verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var saved TeeOutput
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if saved.ID != output.ID {
		t.Errorf("expected ID %s, got %s", output.ID, saved.ID)
	}
	if saved.Command != output.Command {
		t.Errorf("expected Command %s, got %s", output.Command, saved.Command)
	}
	if saved.ExitCode != output.ExitCode {
		t.Errorf("expected ExitCode %d, got %d", output.ExitCode, saved.ExitCode)
	}
}

func TestWriteDisabled(t *testing.T) {
	writer, err := NewTeeWriter(TeeConfig{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	output := &TeeOutput{
		ID:       "test-001",
		Command:  "echo 'hello'",
		ExitCode: 0,
		Stdout:   "hello\n",
	}

	ctx := context.Background()
	filePath, err := writer.Write(ctx, output)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if filePath != "" {
		t.Error("expected empty file path when disabled")
	}
}

func TestWriteOnFailure(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:   true,
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	ctx := context.Background()

	// Test with exit code 0 - should not write
	output1 := &TeeOutput{
		ID:       "test-001",
		Command:  "echo 'success'",
		ExitCode: 0,
		Stdout:   "success\n",
	}

	filePath, err := writer.WriteOnFailure(ctx, output1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if filePath != "" {
		t.Error("expected empty file path for exit code 0")
	}

	// Test with non-zero exit code - should write
	output2 := &TeeOutput{
		ID:       "test-002",
		Command:  "exit 1",
		ExitCode: 1,
		Stderr:   "error occurred",
	}

	filePath, err = writer.WriteOnFailure(ctx, output2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if filePath == "" {
		t.Error("expected file path for non-zero exit code")
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		command  string
		timestamp time.Time
		want     string
	}{
		{
			command:  "echo hello",
			timestamp: time.Date(2026, 6, 24, 10, 30, 45, 0, time.UTC),
			want:     "echo_hello_20260624_103045",
		},
		{
			command:  "npm test",
			timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     "npm_test_20260101_000000",
		},
	}

	for _, tt := range tests {
		got := GenerateFilename(tt.command, tt.timestamp)
		if got != tt.want {
			t.Errorf("GenerateFilename(%q, %v) = %q, want %q", tt.command, tt.timestamp, got, tt.want)
		}
	}
}

func TestGenerateFilenameLongCommand(t *testing.T) {
	longCommand := "this_is_a_very_long_command_that_exceeds_fifty_characters_and_should_be_truncated"
	timestamp := time.Date(2026, 6, 24, 10, 30, 45, 0, time.UTC)

	got := GenerateFilename(longCommand, timestamp)

	if len(got) > 60 { // filename + timestamp part
		t.Errorf("filename too long: %s", got)
	}
}

func TestGenerateFilenameSpecialChars(t *testing.T) {
	command := "echo 'hello world' && cat file.txt"
	timestamp := time.Date(2026, 6, 24, 10, 30, 45, 0, time.UTC)

	got := GenerateFilename(command, timestamp)

	// Should not contain special characters
	for _, c := range got {
		if c == ' ' || c == '\'' || c == '&' || c == '|' {
			t.Errorf("filename contains invalid character %c: %s", c, got)
		}
	}
}

func TestRotationManagerShouldRotate(t *testing.T) {
	tmpDir := t.TempDir()

	rm := NewRotationManager(TeeConfig{
		MaxFiles: 3,
	})

	// Should not rotate when under limit
	if rm.ShouldRotate(tmpDir) {
		t.Error("should not rotate when under limit")
	}

	// Create files to reach limit
	for i := 0; i < 3; i++ {
		f, err := os.Create(filepath.Join(tmpDir, "test_"+string(rune('a'+i))+".json"))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()
	}

	// Should rotate when at limit
	if !rm.ShouldRotate(tmpDir) {
		t.Error("should rotate when at limit")
	}
}

func TestRotationManagerShouldRotateDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	rm := NewRotationManager(TeeConfig{
		MaxFiles: 0, // Disabled
	})

	// Create many files
	for i := 0; i < 10; i++ {
		f, err := os.Create(filepath.Join(tmpDir, "test_"+string(rune('a'+i))+".json"))
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		f.Close()
	}

	// Should not rotate when disabled
	if rm.ShouldRotate(tmpDir) {
		t.Error("should not rotate when disabled")
	}
}

func TestRotationManagerRotate(t *testing.T) {
	tmpDir := t.TempDir()

	rm := NewRotationManager(TeeConfig{
		MaxFiles: 2,
	})

	// Create files with different ages
	oldFile := filepath.Join(tmpDir, "old.json")
	newFile := filepath.Join(tmpDir, "new.json")

	os.WriteFile(oldFile, []byte(`{"id": "old"}`), 0644)
	os.WriteFile(newFile, []byte(`{"id": "new"}`), 0644)

	// Set old file to be older
	time.Sleep(10 * time.Millisecond)
	os.Chtimes(oldFile, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))

	// Rotate
	if err := rm.Rotate(tmpDir); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// New file should exist, old file should be removed
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("new file should still exist")
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
}

func TestRotationManagerCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	rm := NewRotationManager(TeeConfig{
		RetentionDays: 1,
	})

	// Create old file
	oldFile := filepath.Join(tmpDir, "old.json")
	os.WriteFile(oldFile, []byte(`{"id": "old"}`), 0644)
	os.Chtimes(oldFile, time.Now().Add(-48*time.Hour), time.Now().Add(-48*time.Hour))

	// Create new file
	newFile := filepath.Join(tmpDir, "new.json")
	os.WriteFile(newFile, []byte(`{"id": "new"}`), 0644)

	// Cleanup
	removed, err := rm.Cleanup(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if removed != 1 {
		t.Errorf("expected 1 file removed, got %d", removed)
	}

	// New file should exist, old file should be removed
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("new file should still exist")
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
}

func TestRotationManagerCleanupDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	rm := NewRotationManager(TeeConfig{
		RetentionDays: 0, // Disabled
	})

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "file1.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.json"), []byte(`{}`), 0644)

	removed, err := rm.Cleanup(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if removed != 0 {
		t.Errorf("expected 0 files removed when disabled, got %d", removed)
	}
}

func TestReadOutput(t *testing.T) {
	tmpDir := t.TempDir()

	output := &TeeOutput{
		ID:        "test-001",
		Command:   "echo 'hello'",
		ExitCode:  0,
		Stdout:    "hello\n",
		Stderr:    "",
		Combined:  "hello\n",
		Timestamp: time.Now(),
	}

	filePath := filepath.Join(tmpDir, "test.json")
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	os.WriteFile(filePath, append(data, '\n'...), 0644)

	read, err := ReadOutput(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if read.ID != output.ID {
		t.Errorf("expected ID %s, got %s", output.ID, read.ID)
	}
	if read.Command != output.Command {
		t.Errorf("expected Command %s, got %s", output.Command, read.Command)
	}
	if read.ExitCode != output.ExitCode {
		t.Errorf("expected ExitCode %d, got %d", output.ExitCode, read.ExitCode)
	}
}

func TestReadOutputNotFound(t *testing.T) {
	_, err := ReadOutput("/nonexistent/file.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetFilePath(t *testing.T) {
	tmpDir := "/tmp/tee-test"

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:   true,
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	output := &TeeOutput{
		ID:        "test-001",
		Command:   "echo hello",
		Timestamp: time.Date(2026, 6, 24, 10, 30, 45, 0, time.UTC),
	}

	path := writer.GetFilePath(output)
	expected := "/tmp/tee-test/echo_hello_20260624_103045_test-001.json"

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestGetFilePathWithoutID(t *testing.T) {
	tmpDir := "/tmp/tee-test"

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:   true,
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	output := &TeeOutput{
		Command:   "echo hello",
		Timestamp: time.Date(2026, 6, 24, 10, 30, 45, 0, time.UTC),
	}

	path := writer.GetFilePath(output)
	expected := "/tmp/tee-test/echo_hello_20260624_103045.json"

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		stderr   string
		expected int
	}{
		{"exit status 1", 1},
		{"exit status 42", 42},
		{"some error\nexit status 255\n", 255},
		{"no exit code here", -1},
		{"", -1},
	}

	for _, tt := range tests {
		got := ParseExitCode(tt.stderr)
		if got != tt.expected {
			t.Errorf("ParseExitCode(%q) = %d, want %d", tt.stderr, got, tt.expected)
		}
	}
}

func TestEnsureDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "nested", "path", "dir")

	if err := EnsureDirectory(newDir); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestGetDirectorySize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(tmpDir, "file1.json"), []byte(`{"size": 10}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.json"), []byte(`{"size": 20}`), 0644)

	size, err := GetDirectorySize(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if size < 30 {
		t.Errorf("expected size >= 30, got %d", size)
	}
}

func TestListOutputFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create JSON files
	os.WriteFile(filepath.Join(tmpDir, "file1.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte(`not json`), 0644)

	files, err := ListOutputFiles(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 json files, got %d", len(files))
	}
}

func TestWriteMultipleOutputs(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewTeeWriter(TeeConfig{
		Enabled:   true,
		OutputDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	defer writer.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		output := &TeeOutput{
			ID:       "test-" + string(rune('0'+i)),
			Command:  "test command",
			ExitCode: i,
			Stdout:   "output",
		}

		filePath, err := writer.Write(ctx, output)
		if err != nil {
			t.Errorf("unexpected error on write %d: %v", i, err)
		}
		if filePath == "" {
			t.Error("expected file path")
		}
	}

	files, err := ListOutputFiles(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(files) != 5 {
		t.Errorf("expected 5 files, got %d", len(files))
	}
}
