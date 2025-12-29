package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// Logger は構造化ログを出力するロガー
type Logger struct {
	level      Level
	output     io.Writer
	jsonFormat bool
}

// New は新しいLoggerを作成します
func New(level Level, jsonFormat bool) *Logger {
	return &Logger{
		level:      level,
		output:     os.Stderr,
		jsonFormat: jsonFormat,
	}
}

// SetOutput は出力先を設定します
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// logEntry はログエントリを表す
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log は内部ログ出力メソッド
func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := logEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Fields:    fields,
	}

	if l.jsonFormat {
		data, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(l.output, "Error marshaling log: %v\n", err)
			return
		}
		fmt.Fprintf(l.output, "%s\n", string(data))
	} else {
		// 人間が読みやすい形式
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		levelStr := level.String()

		// レベルに応じた絵文字
		emoji := ""
		switch level {
		case LevelDebug:
			emoji = "🔍"
		case LevelInfo:
			emoji = "ℹ️ "
		case LevelWarn:
			emoji = "⚠️ "
		case LevelError:
			emoji = "❌"
		}

		if len(fields) > 0 {
			fieldsStr, _ := json.Marshal(fields)
			fmt.Fprintf(l.output, "[%s] %s %s: %s %s\n", timestamp, emoji, levelStr, msg, string(fieldsStr))
		} else {
			fmt.Fprintf(l.output, "[%s] %s %s: %s\n", timestamp, emoji, levelStr, msg)
		}
	}
}

// Debug はDEBUGレベルのログを出力します
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.log(LevelDebug, msg, f)
}

// Info はINFOレベルのログを出力します
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.log(LevelInfo, msg, f)
}

// Warn はWARNレベルのログを出力します
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.log(LevelWarn, msg, f)
}

// Error はERRORレベルのログを出力します
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	l.log(LevelError, msg, f)
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
