package util

import (
	"testing"
)

func TestSanitizeUTF8_ValidString(t *testing.T) {
	input := "Hello, 世界! 🎉"
	result := SanitizeUTF8(input)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestSanitizeUTF8_InvalidBytes(t *testing.T) {
	// 無効なUTF-8バイト列を含む文字列
	input := "Hello\x80\x81World"
	result := SanitizeUTF8(input)
	expected := "HelloWorld"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeUTF8_EmptyString(t *testing.T) {
	result := SanitizeUTF8("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSanitizeUTF8_OnlyInvalidBytes(t *testing.T) {
	input := "\x80\x81\x82"
	result := SanitizeUTF8(input)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTruncateText_ShortText(t *testing.T) {
	input := "Hello"
	result := TruncateText(input, 10)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestTruncateText_LongText(t *testing.T) {
	input := "Hello World, this is a test"
	result := TruncateText(input, 11)
	expected := "Hello World..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTruncateText_ExactLength(t *testing.T) {
	input := "Hello"
	result := TruncateText(input, 5)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}
