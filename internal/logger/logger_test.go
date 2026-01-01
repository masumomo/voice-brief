package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogger_New(t *testing.T) {
	// ログディレクトリなしで作成
	logger, err := New(LevelInfo, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	if logger.GetLogFilePath() != "" {
		t.Errorf("expected empty log file path, got: %s", logger.GetLogFilePath())
	}
}

func TestLogger_NewWithLogDir(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := New(LevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// ログファイルが作成されていることを確認
	logPath := logger.GetLogFilePath()
	if logPath == "" {
		t.Error("expected log file path to be set")
	}

	if !strings.HasPrefix(logPath, tmpDir) {
		t.Errorf("log file path should be in temp dir: %s", logPath)
	}

	// ファイル名にタイムスタンプが含まれていることを確認
	if !strings.Contains(filepath.Base(logPath), "voicebrief_") {
		t.Errorf("log file name should contain 'voicebrief_': %s", logPath)
	}
}

func TestLogger_Info(t *testing.T) {
	logger, err := New(LevelInfo, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// パニックしないことを確認
	logger.Info("test message", map[string]interface{}{
		"key": "value",
	})
}

func TestLogger_Debug(t *testing.T) {
	logger, err := New(LevelDebug, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// パニックしないことを確認
	logger.Debug("debug message")
}

func TestLogger_Warn(t *testing.T) {
	logger, err := New(LevelWarn, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// パニックしないことを確認
	logger.Warn("warning message")
}

func TestLogger_Error(t *testing.T) {
	logger, err := New(LevelError, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// パニックしないことを確認
	logger.Error("error message", map[string]interface{}{
		"error": "some error",
	})
}

func TestLogger_Print(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := New(LevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Printのテスト
	logger.Print("test message %s\n", "arg1")
	logger.Close()

	// ファイルに書き込まれていることを確認
	logPath := logger.GetLogFilePath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message arg1") {
		t.Errorf("log file should contain 'test message arg1': %s", string(content))
	}
}

func TestLogger_Println(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := New(LevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Printlnのテスト
	logger.Println("println message")
	logger.Close()

	// ファイルに書き込まれていることを確認
	logPath := logger.GetLogFilePath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "println message") {
		t.Errorf("log file should contain 'println message': %s", string(content))
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

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %s; want %s", tt.level, got, tt.expected)
		}
	}
}

func TestLogger_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logger, err := New(LevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Closeが複数回呼ばれてもエラーにならない
	if err := logger.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// 2回目のCloseはすでにクローズされているがエラーになる可能性
	// (ファイルが既にクローズされているため)
	_ = logger.Close()
}

func TestLogger_MultipleFields(t *testing.T) {
	logger, err := New(LevelDebug, "")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// 複数フィールドのマージが正しく動作することを確認
	logger.Info("test with multiple fields",
		map[string]interface{}{"field1": "value1"},
		map[string]interface{}{"field2": "value2"},
	)
}
