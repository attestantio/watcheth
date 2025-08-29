// Copyright © 2025 Attestant Limited.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"io"
	"os"
	"strings"
	"sync"

	"github.com/watcheth/watcheth/internal/logger"
)

const (
	maxLogLines = 15 // Maximum number of log lines to keep in buffer
)

type LogReader struct {
	mu       sync.RWMutex
	logPaths map[string]string   // clientName -> logPath
	logCache map[string][]string // clientName -> last N log lines
}

func NewLogReader() *LogReader {
	return &LogReader{
		logPaths: make(map[string]string),
		logCache: make(map[string][]string),
	}
}

// SetLogPath sets the log file path for a client
func (lr *LogReader) SetLogPath(clientName, logPath string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	lr.logPaths[clientName] = logPath
}

// HasLogPath checks if a log path is configured for a client
func (lr *LogReader) HasLogPath(clientName string) bool {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	logPath, exists := lr.logPaths[clientName]
	return exists && logPath != ""
}

// ReadLogs reads the last N lines from a client's log file
func (lr *LogReader) ReadLogs(clientName string) ([]string, error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	logPath, exists := lr.logPaths[clientName]
	if !exists || logPath == "" {
		// No log path configured - this is fine, just return empty
		lr.logCache[clientName] = []string{"[No log path configured]"}
		return lr.logCache[clientName], nil
	}

	// Try to open the log file
	file, err := os.Open(logPath)
	if err != nil {
		// File doesn't exist or no permission - return empty
		lr.logCache[clientName] = []string{"[Unable to read log file: " + err.Error() + "]"}
		return lr.logCache[clientName], nil
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log file close error is not critical for reading
			logger.Debug("Failed to close log file %s: %v", logPath, err)
		}
	}()

	// Read the last N lines efficiently
	lines, err := tailFile(file, maxLogLines)
	if err != nil {
		lr.logCache[clientName] = []string{"[Error reading log file: " + err.Error() + "]"}
		return lr.logCache[clientName], nil
	}

	lr.logCache[clientName] = lines
	return lines, nil
}

// GetCachedLogs returns cached logs for a client without re-reading the file
func (lr *LogReader) GetCachedLogs(clientName string) []string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	if logs, exists := lr.logCache[clientName]; exists {
		return logs
	}

	// Check if log path exists but no cache yet
	if logPath, exists := lr.logPaths[clientName]; exists && logPath != "" {
		return []string{"[Logs not loaded yet]"}
	}

	return []string{"[No log path configured]"}
}

// tailFile reads the last n lines from a file efficiently
func tailFile(file *os.File, n int) ([]string, error) {
	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Start from the end and work backwards
	fileSize := stat.Size()
	if fileSize == 0 {
		return []string{"[Log file is empty]"}, nil
	}

	// Read from the end in chunks
	bufferSize := int64(8192) // 8KB chunks
	if bufferSize > fileSize {
		bufferSize = fileSize
	}

	var lines []string
	var leftover []byte

	for offset := fileSize; offset > 0 && len(lines) < n; {
		// Calculate how much to read
		readSize := bufferSize
		if offset < bufferSize {
			readSize = offset
		}
		offset -= readSize

		// Seek to position and read
		buffer := make([]byte, readSize)
		_, err := file.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Combine with leftover from previous iteration
		if leftover != nil {
			buffer = append(buffer, leftover...)
		}

		// Split into lines
		content := string(buffer)
		contentLines := strings.Split(content, "\n")

		// First line might be partial, save it as leftover
		if offset > 0 {
			leftover = []byte(contentLines[0])
			contentLines = contentLines[1:]
		}

		// Prepend lines (we're reading backwards)
		for i := len(contentLines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(contentLines[i])
			if line != "" {
				lines = append([]string{line}, lines...)
				if len(lines) >= n {
					break
				}
			}
		}
	}

	// Ensure we don't exceed the requested number of lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	if len(lines) == 0 {
		return []string{"[No recent log entries]"}, nil
	}

	return lines, nil
}

// ParseLogLevel attempts to determine the log level from a log line
func ParseLogLevel(line string) string {
	upperLine := strings.ToUpper(line)
	if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FATAL") {
		return "ERROR"
	} else if strings.Contains(upperLine, "WARN") || strings.Contains(upperLine, "WARNING") {
		return "WARN"
	} else if strings.Contains(upperLine, "INFO") {
		return "INFO"
	} else if strings.Contains(upperLine, "DEBUG") {
		return "DEBUG"
	}
	return "INFO"
}
