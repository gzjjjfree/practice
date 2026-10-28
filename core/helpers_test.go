package core

import (
	"testing"
)

func TestNormalizeAnswerLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a", "A"},
		{"Ａ", "A"},
		{"  b  ", "B"},
		{"ａ", "A"},
	}

	for _, tc := range tests {
		got := NormalizeAnswerLabel(tc.input)
		if got != tc.expected {
			t.Errorf("NormalizeAnswerLabel(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsAnswerCorrect(t *testing.T) {
	qSingle := Question{
		Type:    "单选题",
		Answers: []string{"A"},
	}

	qMultiple := Question{
		Type:    "多选题",
		Answers: []string{"A", "B"},
	}

	qBlank := Question{
		Type:    "填空题",
		Answers: []string{},
	}

	tests := []struct {
		name     string
		userAns  string
		q        Question
		expected bool
	}{
		{"Single correct", "A", qSingle, true},
		{"Single case insensitivity", "a", qSingle, true},
		{"Single incorrect", "B", qSingle, false},
		{"Multiple correct order", "A,B", qMultiple, true},
		{"Multiple reverse order", "B,A", qMultiple, true},
		{"Multiple incorrect", "A", qMultiple, false},
		{"Blank non-empty", "some answer", qBlank, true},
		{"Blank empty", "", qBlank, false},
		{"Null answer", "null", qSingle, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAnswerCorrect(tc.userAns, tc.q)
			if got != tc.expected {
				t.Errorf("IsAnswerCorrect(%q, %+v) = %v, expected %v", tc.userAns, tc.q, got, tc.expected)
			}
		})
	}
}

func TestTruncateUsername(t *testing.T) {
	if TruncateUsername("short") != "short" {
		t.Errorf("Expected 'short', got %q", TruncateUsername("short"))
	}
	if TruncateUsername("verylongusername") != "veryl..." {
		t.Errorf("Expected 'veryl...', got %q", TruncateUsername("verylongusername"))
	}
}

func TestGetCanvasSizeAndResponsive(t *testing.T) {
	// Test with nil window (should fallback gracefully)
	size := GetCanvasSize(nil)
	if size.Width != 400 || size.Height != 750 {
		t.Errorf("Expected default size (400, 750), got %+v", size)
	}

	isSmall := IsSmallScreen(nil)
	if !isSmall {
		t.Errorf("Expected IsSmallScreen(nil) to be true since 400x750 < 550x700")
	}

	w := GetResponsiveWidth(nil, 380, 10)
	if w != 380 {
		t.Errorf("Expected responsive width 380, got %f", w)
	}
}
