package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level はログレベルを表す
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// toSlogLevel はLevelをslog.Levelに変換します
func (l Level) toSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger は構造化ログを出力するロガー（slogベース）
type Logger struct {
	slogger  *slog.Logger
	level    Level
	logFile  *os.File
	mu       sync.Mutex
}

// New は新しいLoggerを作成します
// logDirが空の場合は標準出力のみ、指定された場合はファイルにも出力
func New(level Level, logDir string) (*Logger, error) {
	var writers []io.Writer
	var logFile *os.File

	// 常にstdoutに出力
	writers = append(writers, os.Stdout)

	// logDirが指定されていればファイルにも出力
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// タイムスタンプ付きファイル名
		filename := fmt.Sprintf("voicebrief_%s.log", time.Now().Format("2006-01-02_15-04-05"))
		logPath := filepath.Join(logDir, filename)

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logFile = f
		writers = append(writers, f)
	}

	multiWriter := io.MultiWriter(writers...)

	// slogハンドラの作成（テキスト形式）
	opts := &slog.HandlerOptions{
		Level: level.toSlogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// タイムスタンプのフォーマットをカスタマイズ
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			return a
		},
	}

	handler := slog.NewTextHandler(multiWriter, opts)
	slogger := slog.New(handler)

	return &Logger{
		slogger: slogger,
		level:   level,
		logFile: logFile,
	}, nil
}

// Close はログファイルをクローズします
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Debug はDEBUGレベルのログを出力します
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.logWithFields(slog.LevelDebug, msg, mergeFields(fields...))
}

// Info はINFOレベルのログを出力します
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.logWithFields(slog.LevelInfo, msg, mergeFields(fields...))
}

// Warn はWARNレベルのログを出力します
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.logWithFields(slog.LevelWarn, msg, mergeFields(fields...))
}

// Error はERRORレベルのログを出力します
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.logWithFields(slog.LevelError, msg, mergeFields(fields...))
}

// logWithFields はfieldsをslog.Attrに変換してログ出力します
func (l *Logger) logWithFields(level slog.Level, msg string, fields map[string]interface{}) {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	l.slogger.Log(context.Background(), level, msg, attrs...)
}

// Print はログファイルと標準出力の両方に出力します（ログレベルなし）
// fmt.Printfの代替として使用
func (l *Logger) Print(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// slogを通さず直接出力（タイムスタンプなしでユーザーフレンドリー）
	fmt.Print(msg)
	// ファイルにはタイムスタンプ付きで出力
	if l.logFile != nil {
		l.mu.Lock()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(l.logFile, "[%s] %s", timestamp, msg)
		l.mu.Unlock()
	}
}

// Println はログファイルと標準出力の両方に出力します（改行付き）
func (l *Logger) Println(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	fmt.Print(msg)
	if l.logFile != nil {
		l.mu.Lock()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(l.logFile, "[%s] %s", timestamp, msg)
		l.mu.Unlock()
	}
}

// mergeFields は複数のfieldsマップをマージします
func mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// ParseLevel は文字列からログレベルを解析します
func ParseLevel(s string) (Level, error) {
	switch s {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s", s)
	}
}

// GetLogFilePath はログファイルのパスを返します
func (l *Logger) GetLogFilePath() string {
	if l.logFile != nil {
		return l.logFile.Name()
	}
	return ""
}
