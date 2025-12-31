package tts

import (
	"testing"
)

func TestNewOpenAITTS(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectedVoice  string
		expectedRate   float64
		expectedFormat string
	}{
		{
			name:           "デフォルト設定",
			config:         &Config{},
			expectedVoice:  "alloy",
			expectedRate:   1.0,
			expectedFormat: "mp3",
		},
		{
			name: "カスタム設定",
			config: &Config{
				Voice:  "nova",
				Rate:   1.5,
				Format: "opus",
			},
			expectedVoice:  "nova",
			expectedRate:   1.5,
			expectedFormat: "opus",
		},
		{
			name: "音声のみ指定",
			config: &Config{
				Voice: "shimmer",
			},
			expectedVoice:  "shimmer",
			expectedRate:   1.0,
			expectedFormat: "mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tts := NewOpenAITTS(tt.config, "test-api-key")

			if tts.config.Voice != tt.expectedVoice {
				t.Errorf("expected Voice=%s, got %s", tt.expectedVoice, tts.config.Voice)
			}
			if tts.config.Rate != tt.expectedRate {
				t.Errorf("expected Rate=%f, got %f", tt.expectedRate, tts.config.Rate)
			}
			if tts.config.Format != tt.expectedFormat {
				t.Errorf("expected Format=%s, got %s", tt.expectedFormat, tts.config.Format)
			}
		})
	}
}

func TestOpenAITTS_GetProvider(t *testing.T) {
	tts := NewOpenAITTS(&Config{}, "test-api-key")
	if tts.GetProvider() != "openai_tts" {
		t.Errorf("expected provider 'openai_tts', got %s", tts.GetProvider())
	}
}

func TestOpenAITTS_GetCharactersUsed(t *testing.T) {
	tts := NewOpenAITTS(&Config{}, "test-api-key")

	// 初期状態は0
	if tts.GetCharactersUsed() != 0 {
		t.Errorf("expected initial characters used = 0, got %d", tts.GetCharactersUsed())
	}

	// 内部的に文字数を設定してテスト
	tts.charactersUsed = 1000
	if tts.GetCharactersUsed() != 1000 {
		t.Errorf("expected characters used = 1000, got %d", tts.GetCharactersUsed())
	}
}

func TestOpenAITTS_EstimateCost(t *testing.T) {
	tests := []struct {
		name           string
		charactersUsed int
		provider       string
		expectedCost   float64
	}{
		{
			name:           "0文字",
			charactersUsed: 0,
			provider:       "tts-1",
			expectedCost:   0.0,
		},
		{
			name:           "1000文字 tts-1",
			charactersUsed: 1000,
			provider:       "tts-1",
			expectedCost:   0.015, // 1000 * ($15/1M chars) = $0.015
		},
		{
			name:           "1000000文字 tts-1",
			charactersUsed: 1_000_000,
			provider:       "tts-1",
			expectedCost:   15.0,
		},
		{
			name:           "1000000文字 tts-1-hd",
			charactersUsed: 1_000_000,
			provider:       "tts-1-hd",
			expectedCost:   30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{Provider: tt.provider}
			tts := NewOpenAITTS(config, "test-api-key")
			tts.charactersUsed = tt.charactersUsed

			cost := tts.EstimateCost()

			// 浮動小数点の比較は誤差を許容
			if cost < tt.expectedCost*0.99 || cost > tt.expectedCost*1.01 {
				t.Errorf("expected cost ≈ %f, got %f", tt.expectedCost, cost)
			}
		})
	}
}

func TestMapVoiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Kyoko", "nova"},
		{"Otoya", "onyx"},
		{"Samantha", "nova"},
		{"Alex", "onyx"},
		{"alloy", "alloy"},   // そのまま
		{"echo", "echo"},     // そのまま
		{"shimmer", "shimmer"}, // そのまま
		{"unknown", "unknown"}, // 未知の音声もそのまま
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MapVoiceName(tt.input)
			if result != tt.expected {
				t.Errorf("MapVoiceName(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetAvailableOpenAIVoices(t *testing.T) {
	voices := GetAvailableOpenAIVoices()

	expectedVoices := []string{"alloy", "echo", "fable", "onyx", "nova", "shimmer"}

	if len(voices) != len(expectedVoices) {
		t.Errorf("expected %d voices, got %d", len(expectedVoices), len(voices))
	}

	for _, expected := range expectedVoices {
		found := false
		for _, voice := range voices {
			if voice == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected voice '%s' not found in available voices", expected)
		}
	}
}
