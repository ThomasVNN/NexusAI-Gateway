package tee

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// TeeConfig contains tee system configuration
type TeeConfig struct {
	OutputDir      string
	MaxFileSize    int64
	MaxFiles       int
	RetentionDays  int
	Enabled        bool
}

// TeeOutput represents captured command output
type TeeOutput struct {
	ID        string
	Command   string
	ExitCode  int
	Stdout    string
	Stderr    string
	Combined  string
	Timestamp time.Time
	FilePath  string
}

// TeeWriter writes command output to files on failure
type TeeWriter struct {
	config  TeeConfig
	mu      sync.Mutex
	writers map[string]*os.File
}

// NewTeeWriter creates a new tee writer
func NewTeeWriter(config TeeConfig) (*TeeWriter, error) {
	if config.Enabled && config.OutputDir == "" {
		return nil, fmt.Errorf("output directory is required when tee is enabled")
	}

	if config.Enabled {
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return &TeeWriter{
		config:  config,
		writers: make(map[string]*os.File),
	}, nil
}

// Write writes output to tee destination
func (t *TeeWriter) Write(ctx context.Context, output *TeeOutput) (string, error) {
	if !t.config.Enabled {
		return "", nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	filePath := t.GetFilePath(output)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Check file size limit
	if t.config.MaxFileSize > 0 {
		if info, err := os.Stat(filePath); err == nil && info.Size() >= t.config.MaxFileSize {
			return "", fmt.Errorf("file size limit exceeded")
		}
	}

	// Open file for writing
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Encode output as JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	if _, err := f.Write(append(data, []byte{'\n'}...)); err != nil {
		return "", fmt.Errorf("failed to write output: %w", err)
	}

	output.FilePath = filePath
	return filePath, nil
}

// WriteOnFailure writes output only if exit code != 0
func (t *TeeWriter) WriteOnFailure(ctx context.Context, output *TeeOutput) (string, error) {
	if output.ExitCode == 0 {
		return "", nil
	}
	return t.Write(ctx, output)
}

// Close closes all open file handles
func (t *TeeWriter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lastErr error
	for name, f := range t.writers {
		if err := f.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close %s: %w", name, err)
		}
	}
	t.writers = make(map[string]*os.File)
	return lastErr
}

// RotationManager manages file rotation
type RotationManager struct {
	config TeeConfig
	mu     sync.Mutex
}

// NewRotationManager creates a new rotation manager
func NewRotationManager(config TeeConfig) *RotationManager {
	return &RotationManager{config: config}
}

// ShouldRotate checks if rotation is needed
func (r *RotationManager) ShouldRotate(dir string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.config.MaxFiles <= 0 {
		return false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}

	return count >= r.config.MaxFiles
}

// Rotate rotates old files
func (r *RotationManager) Rotate(dir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (oldest first)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].modTime.Before(files[i].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove oldest files until we're under max
	targetCount := r.config.MaxFiles - 1
	if targetCount < 0 {
		targetCount = 0
	}

	removed := 0
	for i := len(files) - 1; i >= targetCount && i >= 0; i-- {
		if err := os.Remove(files[i].path); err != nil {
			return fmt.Errorf("failed to remove old file: %w", err)
		}
		removed++
	}

	return nil
}

// Cleanup removes old files beyond retention
func (r *RotationManager) Cleanup(dir string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.config.RetentionDays <= 0 {
		return 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -r.config.RetentionDays)
	removed := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return removed, fmt.Errorf("failed to remove old file: %w", err)
			}
			removed++
		}
	}

	return removed, nil
}

// GetFilePath returns the file path for an output
func (t *TeeWriter) GetFilePath(output *TeeOutput) string {
	if output.ID != "" {
		return filepath.Join(t.config.OutputDir, fmt.Sprintf("%s_%s.json", GenerateFilename(output.Command, output.Timestamp), output.ID))
	}
	return filepath.Join(t.config.OutputDir, fmt.Sprintf("%s.json", GenerateFilename(output.Command, output.Timestamp)))
}

// GenerateFilename generates a unique filename
func GenerateFilename(command string, timestamp time.Time) string {
	// Sanitize command for use in filename
	safeCmd := sanitizeFilename(command)
	if len(safeCmd) > 50 {
		safeCmd = safeCmd[:50]
	}

	// Remove special characters from command
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	safeCmd = re.ReplaceAllString(safeCmd, "_")

	// Format: command_timestamp
	return fmt.Sprintf("%s_%s", safeCmd, timestamp.Format("20060102_150405"))
}

// ReadOutput reads saved output from file
func ReadOutput(filePath string) (*TeeOutput, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var output TeeOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return &output, nil
}

// Helper function to sanitize filename
func sanitizeFilename(name string) string {
	// Remove path separators and other problematic characters
	re := regexp.MustCompile(`[/\\:*?"<>|]`)
	return re.ReplaceAllString(name, "_")
}

// EnsureDirectory creates directory if it doesn't exist
func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetDirectorySize returns the total size of files in a directory
func GetDirectorySize(dir string) (int64, error) {
	var size int64

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		size += info.Size()
	}

	return size, nil
}

// ListOutputFiles returns all output files in the directory
func ListOutputFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}

// ParseExitCode attempts to parse exit code from stderr
func ParseExitCode(stderr string) int {
	re := regexp.MustCompile(`exit status (\d+)`)
	matches := re.FindStringSubmatch(stderr)
	if len(matches) > 1 {
		if code, err := strconv.Atoi(matches[1]); err == nil {
			return code
		}
	}
	return -1
}
