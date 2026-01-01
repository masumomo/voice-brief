package util

import (
	"strings"
	"unicode/utf8"
)

// SanitizeUTF8 は無効なUTF-8バイトを除去し、有効なUTF-8文字列を返します
func SanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var builder strings.Builder
	builder.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// 無効なバイトはスキップ
			i++
			continue
		}
		builder.WriteRune(r)
		i += size
	}
	return builder.String()
}

// TruncateText はテキストを指定長で切り詰めます
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
