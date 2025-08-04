package logger

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetDebugMode(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	defer func() {
		log.SetOutput(originalOutput)
		defaultLogger.debugEnabled = originalDebug
	}()

	// Test enabling debug mode
	SetDebugMode(true)
	assert.True(t, defaultLogger.debugEnabled)
	assert.Equal(t, os.Stderr, log.Writer())

	// Test disabling debug mode
	SetDebugMode(false)
	assert.False(t, defaultLogger.debugEnabled)
	assert.Equal(t, io.Discard, log.Writer())
}

func TestIsDebugEnabled(t *testing.T) {
	// Save original state
	originalDebug := defaultLogger.debugEnabled
	defer func() {
		defaultLogger.debugEnabled = originalDebug
	}()

	// Test when debug is disabled
	defaultLogger.debugEnabled = false
	assert.False(t, IsDebugEnabled())

	// Test when debug is enabled
	defaultLogger.debugEnabled = true
	assert.True(t, IsDebugEnabled())
}

func TestLogFunctions(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		defaultLogger.debugEnabled = originalDebug
	}()

	// Remove timestamp from logs for consistent testing
	log.SetFlags(0)

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		prefix   string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "Debug message",
			logFunc:  Debug,
			prefix:   "[DEBUG]",
			format:   "Test debug message: %s",
			args:     []interface{}{"test"},
			expected: "[DEBUG] Test debug message: test\n",
		},
		{
			name:     "Info message",
			logFunc:  Info,
			prefix:   "[INFO]",
			format:   "Test info message: %d",
			args:     []interface{}{42},
			expected: "[INFO] Test info message: 42\n",
		},
		{
			name:     "Error message",
			logFunc:  Error,
			prefix:   "[ERROR]",
			format:   "Test error: %v",
			args:     []interface{}{"something went wrong"},
			expected: "[ERROR] Test error: something went wrong\n",
		},
		{
			name:     "Warn message",
			logFunc:  Warn,
			prefix:   "[WARN]",
			format:   "Test warning: %s %d",
			args:     []interface{}{"count", 10},
			expected: "[WARN] Test warning: count 10\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)

			// Test with debug disabled - should not log
			defaultLogger.debugEnabled = false
			tt.logFunc(tt.format, tt.args...)
			assert.Empty(t, buf.String(), "Should not log when debug is disabled")

			// Test with debug enabled - should log
			buf.Reset()
			defaultLogger.debugEnabled = true
			tt.logFunc(tt.format, tt.args...)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestLogFormatting(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		defaultLogger.debugEnabled = originalDebug
	}()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defaultLogger.debugEnabled = true

	// Test various format strings
	Debug("Simple message")
	assert.Equal(t, "[DEBUG] Simple message\n", buf.String())

	buf.Reset()
	Info("Message with %s and %d", "string", 123)
	assert.Equal(t, "[INFO] Message with string and 123\n", buf.String())

	buf.Reset()
	Error("Error: %v", io.EOF)
	assert.Equal(t, "[ERROR] Error: EOF\n", buf.String())

	buf.Reset()
	Warn("Multiple: %s %d %v %t", "str", 42, 3.14, true)
	assert.Equal(t, "[WARN] Multiple: str 42 3.14 true\n", buf.String())
}

func TestConcurrentLogging(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		defaultLogger.debugEnabled = originalDebug
	}()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defaultLogger.debugEnabled = true

	var wg sync.WaitGroup
	numGoroutines := 100

	// Launch concurrent loggers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			switch id % 4 {
			case 0:
				Debug("Debug from goroutine %d", id)
			case 1:
				Info("Info from goroutine %d", id)
			case 2:
				Error("Error from goroutine %d", id)
			case 3:
				Warn("Warn from goroutine %d", id)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were logged
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, numGoroutines)

	// Count message types
	debugCount := 0
	infoCount := 0
	errorCount := 0
	warnCount := 0

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "[DEBUG]"):
			debugCount++
		case strings.HasPrefix(line, "[INFO]"):
			infoCount++
		case strings.HasPrefix(line, "[ERROR]"):
			errorCount++
		case strings.HasPrefix(line, "[WARN]"):
			warnCount++
		}
	}

	// Should have roughly equal distribution
	assert.Greater(t, debugCount, 0)
	assert.Greater(t, infoCount, 0)
	assert.Greater(t, errorCount, 0)
	assert.Greater(t, warnCount, 0)
	assert.Equal(t, numGoroutines, debugCount+infoCount+errorCount+warnCount)
}

func TestNoLoggingWhenDisabled(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	defer func() {
		log.SetOutput(originalOutput)
		defaultLogger.debugEnabled = originalDebug
	}()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defaultLogger.debugEnabled = false

	// None of these should produce output
	Debug("This should not appear")
	Info("This should not appear either")
	Error("Not this one")
	Warn("Nor this")

	assert.Empty(t, buf.String())
}

func TestEmptyFormat(t *testing.T) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		defaultLogger.debugEnabled = originalDebug
	}()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defaultLogger.debugEnabled = true

	// Test empty format strings
	Debug("")
	assert.Equal(t, "[DEBUG] \n", buf.String())

	buf.Reset()
	Info("")
	assert.Equal(t, "[INFO] \n", buf.String())
}

func TestLoggerState(t *testing.T) {
	// Save original state
	originalDebug := defaultLogger.debugEnabled
	defer func() {
		defaultLogger.debugEnabled = originalDebug
	}()

	// Test state changes
	SetDebugMode(true)
	assert.True(t, IsDebugEnabled())

	SetDebugMode(false)
	assert.False(t, IsDebugEnabled())

	SetDebugMode(true)
	assert.True(t, IsDebugEnabled())
}

func BenchmarkLogging(b *testing.B) {
	// Save original settings
	originalOutput := log.Writer()
	originalDebug := defaultLogger.debugEnabled
	defer func() {
		log.SetOutput(originalOutput)
		defaultLogger.debugEnabled = originalDebug
	}()

	// Direct to discard for benchmarking
	log.SetOutput(io.Discard)

	b.Run("DebugEnabled", func(b *testing.B) {
		defaultLogger.debugEnabled = true
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Debug("Benchmark message %d", i)
		}
	})

	b.Run("DebugDisabled", func(b *testing.B) {
		defaultLogger.debugEnabled = false
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Debug("Benchmark message %d", i)
		}
	})
}
