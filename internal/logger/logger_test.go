package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelDebug, true)
	logger.SetOutput(&buf)

	logger.Info("test message", map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	})

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected log to contain 'test message', got: %s", output)
	}

	// JSON形式として解析できることを確認
	var entry logEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Errorf("failed to parse JSON log: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("expected level INFO, got: %s", entry.Level)
	}

	if entry.Message != "test message" {
		t.Errorf("expected message 'test message', got: %s", entry.Message)
	}

	if entry.Fields["key1"] != "value1" {
		t.Errorf("expected fields['key1'] = 'value1', got: %v", entry.Fields["key1"])
	}
}

func TestLogger_HumanReadableFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, false)
	logger.SetOutput(&buf)

	logger.Info("human readable test")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected log to contain 'INFO', got: %s", output)
	}
	if !strings.Contains(output, "human readable test") {
		t.Errorf("expected log to contain 'human readable test', got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		loggerLevel   Level
		logLevel      Level
		expectOutput  bool
	}{
		{
			name:         "INFO logger should log INFO",
			loggerLevel:  LevelInfo,
			logLevel:     LevelInfo,
			expectOutput: true,
		},
		{
			name:         "INFO logger should not log DEBUG",
			loggerLevel:  LevelInfo,
			logLevel:     LevelDebug,
			expectOutput: false,
		},
		{
			name:         "DEBUG logger should log INFO",
			loggerLevel:  LevelDebug,
			logLevel:     LevelInfo,
			expectOutput: true,
		},
		{
			name:         "ERROR logger should not log INFO",
			loggerLevel:  LevelError,
			logLevel:     LevelInfo,
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(tt.loggerLevel, true)
			logger.SetOutput(&buf)

			switch tt.logLevel {
			case LevelDebug:
				logger.Debug("test")
			case LevelInfo:
				logger.Info("test")
			case LevelWarn:
				logger.Warn("test")
			case LevelError:
				logger.Error("test")
			}

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expectOutput {
				t.Errorf("expected output=%v, got output=%v", tt.expectOutput, hasOutput)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
		hasError bool
	}{
		{"debug", LevelDebug, false},
		{"info", LevelInfo, false},
		{"warn", LevelWarn, false},
		{"error", LevelError, false},
		{"invalid", LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLevel(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input '%s'", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if level != tt.expected {
					t.Errorf("expected level %v, got %v", tt.expected, level)
				}
			}
		})
	}
}

func TestLogger_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelDebug, true)
	logger.SetOutput(&buf)

	logger.Info("test with multiple fields",
		map[string]interface{}{"field1": "value1"},
		map[string]interface{}{"field2": "value2"},
	)

	var entry logEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if entry.Fields["field1"] != "value1" {
		t.Errorf("expected field1=value1, got: %v", entry.Fields["field1"])
	}
	if entry.Fields["field2"] != "value2" {
		t.Errorf("expected field2=value2, got: %v", entry.Fields["field2"])
	}
}
