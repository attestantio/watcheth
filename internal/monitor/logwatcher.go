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
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/watcheth/watcheth/internal/logger"
)

const (
	defaultLogBufferSize = 100 // Keep more lines for smoother scrolling
	defaultPollInterval  = 100 * time.Millisecond
)

type LogUpdate struct {
	ClientName string
	Lines      []string
	Timestamp  time.Time
}

type fileWatcher struct {
	path       string
	lastSize   int64 // Track last known file size instead of keeping file open
	buffer     []string
	bufferSize int
	mu         sync.RWMutex
}

func (fw *fileWatcher) readNewLines() ([]string, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Open file fresh each time to avoid stale handles
	file, err := os.Open(fw.path)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Best effort close for read-only file
		_ = file.Close()
	}()

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	currentSize := stat.Size()

	// If file was truncated or this is first read, read tail
	if currentSize < fw.lastSize || fw.lastSize == 0 {
		// File was truncated or first read - read last N lines
		lines, err := tailFile(file, fw.bufferSize)
		if err != nil {
			return nil, err
		}
		fw.buffer = lines
		fw.lastSize = currentSize

		// On first read, return empty to avoid duplicate initial display
		if fw.lastSize == 0 {
			fw.lastSize = currentSize
			return []string{}, nil
		}
		return lines, nil
	}

	// If no new data, return empty
	if currentSize == fw.lastSize {
		return []string{}, nil
	}

	// Read only the new data
	bytesToRead := currentSize - fw.lastSize
	if bytesToRead > 0 {
		// Seek to where we left off
		_, err = file.Seek(fw.lastSize, io.SeekStart)
		if err != nil {
			return nil, err
		}

		// Read new content
		newContent := make([]byte, bytesToRead)
		_, err = io.ReadFull(file, newContent)
		if err != nil {
			return nil, err
		}

		// Split into lines
		scanner := bufio.NewScanner(strings.NewReader(string(newContent)))
		var newLines []string
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				newLines = append(newLines, line)
				fw.buffer = append(fw.buffer, line)
			}
		}

		// Trim buffer if needed
		if len(fw.buffer) > fw.bufferSize {
			fw.buffer = fw.buffer[len(fw.buffer)-fw.bufferSize:]
		}

		fw.lastSize = currentSize
		return newLines, nil
	}

	return []string{}, nil
}

func (fw *fileWatcher) getBuffer() []string {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	result := make([]string, len(fw.buffer))
	copy(result, fw.buffer)
	return result
}

func (fw *fileWatcher) close() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// No file handles to close since we open/close on each read
	// Just reset the state
	fw.lastSize = 0
	fw.buffer = []string{}
}

type LogWatcher struct {
	mu         sync.RWMutex
	watchers   map[string]*fileWatcher
	updateChan chan LogUpdate
	watcher    *fsnotify.Watcher
	ctx        context.Context
	cancel     context.CancelFunc
	pollTicker *time.Ticker
	bufferSize int
}

func NewLogWatcher(bufferSize int, pollInterval time.Duration) (*LogWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if bufferSize <= 0 {
		bufferSize = defaultLogBufferSize
	}

	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	ctx, cancel := context.WithCancel(context.Background())

	lw := &LogWatcher{
		watchers:   make(map[string]*fileWatcher),
		updateChan: make(chan LogUpdate, 100),
		watcher:    watcher,
		ctx:        ctx,
		cancel:     cancel,
		pollTicker: time.NewTicker(pollInterval),
		bufferSize: bufferSize,
	}

	go lw.watchLoop()
	return lw, nil
}

func (lw *LogWatcher) AddLogFile(clientName, logPath string) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Remove old watcher if exists
	if old, exists := lw.watchers[clientName]; exists {
		old.close()
		if err := lw.watcher.Remove(old.path); err != nil {
			// Log but don't fail - the file might not exist anymore
			logger.Debug("Failed to remove watcher for %s: %v", old.path, err)
		}
	}

	// Create new file watcher
	fw := &fileWatcher{
		path:       logPath,
		buffer:     make([]string, 0, lw.bufferSize),
		bufferSize: lw.bufferSize,
	}

	// Try to add to fsnotify watcher
	// File might not exist yet, but we'll still poll it
	_ = lw.watcher.Add(logPath)

	lw.watchers[clientName] = fw

	// Do initial read to populate buffer with tail
	go lw.initialRead(clientName, fw)

	return nil
}

func (lw *LogWatcher) initialRead(clientName string, fw *fileWatcher) {
	// Read last N lines like the original implementation
	file, err := os.Open(fw.path)
	if err != nil {
		return
	}
	defer func() {
		// Best effort close for read-only file
		_ = file.Close()
	}()

	// Get file size for tracking
	stat, err := file.Stat()
	if err != nil {
		return
	}

	lines, err := tailFile(file, fw.bufferSize)
	if err == nil && len(lines) > 0 {
		fw.mu.Lock()
		fw.buffer = lines
		fw.lastSize = stat.Size() // Set the initial file size
		fw.mu.Unlock()

		// Send initial update
		select {
		case lw.updateChan <- LogUpdate{
			ClientName: clientName,
			Lines:      lines,
			Timestamp:  time.Now(),
		}:
		case <-lw.ctx.Done():
		}
	}
}

func (lw *LogWatcher) watchLoop() {
	defer lw.pollTicker.Stop()

	for {
		select {
		case <-lw.ctx.Done():
			return

		case event, ok := <-lw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				lw.handleFileChange(event.Name)
			}

		case <-lw.pollTicker.C:
			// Poll all files for changes (backup for when fsnotify doesn't work)
			lw.pollAllFiles()

		case err, ok := <-lw.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue
			_ = err
		}
	}
}

func (lw *LogWatcher) handleFileChange(path string) {
	lw.mu.RLock()
	defer lw.mu.RUnlock()

	for clientName, fw := range lw.watchers {
		if fw.path == path {
			if newLines, err := fw.readNewLines(); err == nil && len(newLines) > 0 {
				select {
				case lw.updateChan <- LogUpdate{
					ClientName: clientName,
					Lines:      fw.getBuffer(),
					Timestamp:  time.Now(),
				}:
				case <-lw.ctx.Done():
				}
			}
			break
		}
	}
}

func (lw *LogWatcher) pollAllFiles() {
	lw.mu.RLock()
	defer lw.mu.RUnlock()

	for clientName, fw := range lw.watchers {
		if newLines, err := fw.readNewLines(); err == nil && len(newLines) > 0 {
			select {
			case lw.updateChan <- LogUpdate{
				ClientName: clientName,
				Lines:      fw.getBuffer(),
				Timestamp:  time.Now(),
			}:
			default:
				// Don't block if channel is full
			}
		}
	}
}

func (lw *LogWatcher) GetLogBuffer(clientName string) []string {
	lw.mu.RLock()
	defer lw.mu.RUnlock()

	if fw, exists := lw.watchers[clientName]; exists {
		return fw.getBuffer()
	}
	return []string{"[No log file configured]"}
}

func (lw *LogWatcher) Updates() <-chan LogUpdate {
	return lw.updateChan
}

func (lw *LogWatcher) Close() error {
	lw.cancel()

	lw.mu.Lock()
	defer lw.mu.Unlock()

	for _, fw := range lw.watchers {
		fw.close()
	}

	return lw.watcher.Close()
}
