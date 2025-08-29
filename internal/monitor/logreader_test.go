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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogReader(t *testing.T) {
	lr := NewLogReader()

	assert.NotNil(t, lr)
	assert.NotNil(t, lr.logPaths)
	assert.NotNil(t, lr.logCache)
	assert.Empty(t, lr.logPaths)
	assert.Empty(t, lr.logCache)
}

func TestLogReader_SetAndHasLogPath(t *testing.T) {
	lr := NewLogReader()

	// Initially no log path
	assert.False(t, lr.HasLogPath("client1"))

	// Set log path
	lr.SetLogPath("client1", "/var/log/client1.log")
	assert.True(t, lr.HasLogPath("client1"))

	// Set empty log path
	lr.SetLogPath("client2", "")
	assert.False(t, lr.HasLogPath("client2"))

	// Check non-existent client
	assert.False(t, lr.HasLogPath("client3"))
}

func TestLogReader_ReadLogs_NoPath(t *testing.T) {
	lr := NewLogReader()

	// Read logs without setting path
	logs, err := lr.ReadLogs("client1")
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "[No log path configured]", logs[0])

	// Cached version should be the same
	cached := lr.GetCachedLogs("client1")
	assert.Equal(t, logs, cached)
}

func TestLogReader_ReadLogs_NonExistentFile(t *testing.T) {
	lr := NewLogReader()

	// Set path to non-existent file
	lr.SetLogPath("client1", "/non/existent/path/client1.log")

	logs, err := lr.ReadLogs("client1")
	assert.NoError(t, err) // ReadLogs doesn't return error for missing files
	assert.Len(t, logs, 1)
	assert.Contains(t, logs[0], "[Unable to read log file:")
}

func TestLogReader_ReadLogs_Success(t *testing.T) {
	// Create temporary log file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Write test content
	content := []string{
		"2024-01-01 10:00:00 INFO Starting service",
		"2024-01-01 10:00:01 DEBUG Connecting to peer",
		"2024-01-01 10:00:02 WARN Connection timeout",
		"2024-01-01 10:00:03 ERROR Failed to connect",
		"2024-01-01 10:00:04 INFO Retrying connection",
	}
	err := ioutil.WriteFile(logFile, []byte(strings.Join(content, "\n")), 0644)
	assert.NoError(t, err)

	lr := NewLogReader()
	lr.SetLogPath("client1", logFile)

	logs, err := lr.ReadLogs("client1")
	assert.NoError(t, err)
	assert.Equal(t, content, logs)
}

func TestLogReader_ReadLogs_EmptyFile(t *testing.T) {
	// Create empty log file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "empty.log")

	err := ioutil.WriteFile(logFile, []byte(""), 0644)
	assert.NoError(t, err)

	lr := NewLogReader()
	lr.SetLogPath("client1", logFile)

	logs, err := lr.ReadLogs("client1")
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "[Log file is empty]", logs[0])
}

func TestLogReader_GetCachedLogs(t *testing.T) {
	lr := NewLogReader()

	// No path configured
	cached := lr.GetCachedLogs("client1")
	assert.Equal(t, []string{"[No log path configured]"}, cached)

	// Path configured but not read yet
	lr.SetLogPath("client1", "/var/log/client1.log")
	cached = lr.GetCachedLogs("client1")
	assert.Equal(t, []string{"[Logs not loaded yet]"}, cached)

	// After reading (non-existent file)
	lr.ReadLogs("client1")
	cached = lr.GetCachedLogs("client1")
	assert.Contains(t, cached[0], "[Unable to read log file:")
}

func TestTailFile_LargeFile(t *testing.T) {
	// Create a large log file with many lines
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "large.log")

	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("2024-01-01 10:00:%02d INFO Log line %d", i, i))
	}

	err := ioutil.WriteFile(logFile, []byte(strings.Join(lines, "\n")), 0644)
	assert.NoError(t, err)

	file, err := os.Open(logFile)
	assert.NoError(t, err)
	defer file.Close()

	// Read last 15 lines
	result, err := tailFile(file, maxLogLines)
	assert.NoError(t, err)
	assert.Len(t, result, maxLogLines)

	// Verify we got the last lines
	expectedStart := len(lines) - maxLogLines
	for i := 0; i < maxLogLines; i++ {
		assert.Equal(t, lines[expectedStart+i], result[i])
	}
}

func TestTailFile_SmallFile(t *testing.T) {
	// Create a small log file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "small.log")

	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	err := ioutil.WriteFile(logFile, []byte(strings.Join(content, "\n")), 0644)
	assert.NoError(t, err)

	file, err := os.Open(logFile)
	assert.NoError(t, err)
	defer file.Close()

	// Request more lines than available
	result, err := tailFile(file, 10)
	assert.NoError(t, err)
	assert.Equal(t, content, result)
}

func TestTailFile_WithEmptyLines(t *testing.T) {
	// Create log file with empty lines
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "empty_lines.log")

	content := "Line 1\n\n\nLine 2\n\nLine 3\n\n"
	err := ioutil.WriteFile(logFile, []byte(content), 0644)
	assert.NoError(t, err)

	file, err := os.Open(logFile)
	assert.NoError(t, err)
	defer file.Close()

	result, err := tailFile(file, 5)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Line 1", "Line 2", "Line 3"}, result)
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"2024-01-01 ERROR Failed to connect", "ERROR"},
		{"[FATAL] System crashed", "ERROR"},
		{"WARN: Connection timeout", "WARN"},
		{"WARNING - Low memory", "WARN"},
		{"INFO Starting service", "INFO"},
		{"[info] Service started", "INFO"},
		{"DEBUG: Verbose logging enabled", "DEBUG"},
		{"[debug] Connection details", "DEBUG"},
		{"Regular log line without level", "INFO"},
		{"", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			level := ParseLogLevel(tt.line)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestLogReader_ConcurrentAccess(t *testing.T) {
	lr := NewLogReader()

	// Create temp file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "concurrent.log")
	err := ioutil.WriteFile(logFile, []byte("Test log line"), 0644)
	assert.NoError(t, err)

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientName := fmt.Sprintf("client%d", idx)
			lr.SetLogPath(clientName, logFile)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientName := fmt.Sprintf("client%d", idx)
			lr.ReadLogs(clientName)
		}(i)
	}

	// Concurrent cached reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientName := fmt.Sprintf("client%d", idx)
			lr.GetCachedLogs(clientName)
		}(i)
	}

	// Concurrent has checks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientName := fmt.Sprintf("client%d", idx)
			lr.HasLogPath(clientName)
		}(i)
	}

	wg.Wait()

	// Verify all clients have paths set
	for i := 0; i < 10; i++ {
		clientName := fmt.Sprintf("client%d", i)
		assert.True(t, lr.HasLogPath(clientName))
	}
}

func TestLogReader_MultipleClients(t *testing.T) {
	lr := NewLogReader()

	// Create different log files for different clients
	tempDir := t.TempDir()

	clients := map[string][]string{
		"client1": {"Client 1 log line 1", "Client 1 log line 2"},
		"client2": {"Client 2 log line 1", "Client 2 log line 2"},
		"client3": {"Client 3 log line 1", "Client 3 log line 2"},
	}

	// Create log files
	for client, lines := range clients {
		logFile := filepath.Join(tempDir, client+".log")
		err := ioutil.WriteFile(logFile, []byte(strings.Join(lines, "\n")), 0644)
		assert.NoError(t, err)
		lr.SetLogPath(client, logFile)
	}

	// Read logs for each client
	for client, expectedLines := range clients {
		logs, err := lr.ReadLogs(client)
		assert.NoError(t, err)
		assert.Equal(t, expectedLines, logs)
	}

	// Verify caches are separate
	for client, expectedLines := range clients {
		cached := lr.GetCachedLogs(client)
		assert.Equal(t, expectedLines, cached)
	}
}

func TestTailFile_VeryLargeFile(t *testing.T) {
	// Create a very large file to test chunked reading
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "very_large.log")

	file, err := os.Create(logFile)
	assert.NoError(t, err)

	// Write 10MB of data
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(file, "This is a long log line number %d with some padding to make it longer %s\n",
			i, strings.Repeat("x", 100))
	}
	file.Close()

	// Read the file
	file, err = os.Open(logFile)
	assert.NoError(t, err)
	defer file.Close()

	// Should efficiently read last 15 lines
	result, err := tailFile(file, maxLogLines)
	assert.NoError(t, err)
	assert.Len(t, result, maxLogLines)

	// Verify we got the last lines
	for i := 0; i < maxLogLines; i++ {
		expectedLineNum := 10000 - maxLogLines + i
		assert.Contains(t, result[i], fmt.Sprintf("number %d", expectedLineNum))
	}
}
