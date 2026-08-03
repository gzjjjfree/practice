package main

import (
	"testing"

	"github.com/gzjjjfree/practice/core"
)

// ==================== IsAnswerCorrect 单元测试 ====================

func TestIsAnswerCorrect_SingleChoice(t *testing.T) {
	q := core.Question{
		Type:    "单选题",
		Answers: []string{"B"},
	}

	tests := []struct {
		name       string
		userAnswer string
		want       bool
	}{
		{"正确选项", "B", true},
		{"错误选项", "A", false},
		{"空答案", "", false},
		{"null标记", "null", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsAnswerCorrect(tt.userAnswer, q)
			if got != tt.want {
				t.Errorf("IsAnswerCorrect(%q, q) = %v, want %v", tt.userAnswer, got, tt.want)
			}
		})
	}
}

func TestIsAnswerCorrect_MultipleChoice(t *testing.T) {
	q := core.Question{
		Type:    "多选题",
		Answers: []string{"A", "C"},
	}

	tests := []struct {
		name       string
		userAnswer string
		want       bool
	}{
		{"正确顺序 A,C", "A,C", true},
		{"反序 C,A（应通过排序匹配）", "C,A", true},
		{"部分正确 A", "A", false},
		{"部分正确 A,B", "A,B", false},
		{"错误选项 D", "D", false},
		{"空答案", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsAnswerCorrect(tt.userAnswer, q)
			if got != tt.want {
				t.Errorf("IsAnswerCorrect(%q, q) = %v, want %v", tt.userAnswer, got, tt.want)
			}
		})
	}
}

func TestIsAnswerCorrect_TrueFalse(t *testing.T) {
	q := core.Question{
		Type:    "判断题",
		Answers: []string{"A"}, // A = 正确
	}

	tests := []struct {
		name       string
		userAnswer string
		want       bool
	}{
		{"正确", "A", true},
		{"错误", "B", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsAnswerCorrect(tt.userAnswer, q)
			if got != tt.want {
				t.Errorf("IsAnswerCorrect(%q, q) = %v, want %v", tt.userAnswer, got, tt.want)
			}
		})
	}
}

func TestIsAnswerCorrect_FillBlank(t *testing.T) {
	q := core.Question{
		Type:    "填空题",
		Answers: []string{"25~30"},
	}

	tests := []struct {
		name       string
		userAnswer string
		want       bool
	}{
		{"已查看（非空）", "已查看", true},
		{"具体答案", "25~30", true},
		{"空答案", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsAnswerCorrect(tt.userAnswer, q)
			if got != tt.want {
				t.Errorf("IsAnswerCorrect(%q, q) = %v, want %v", tt.userAnswer, got, tt.want)
			}
		})
	}
}

func TestIsAnswerCorrect_Essay(t *testing.T) {
	q := core.Question{
		Type:    "问答题",
		Answers: []string{"参考答案"},
	}

	tests := []struct {
		name       string
		userAnswer string
		want       bool
	}{
		{"已查看（非空）", "已查看", true},
		{"具体答案", "参考答案", true},
		{"空答案", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := core.IsAnswerCorrect(tt.userAnswer, q)
			if got != tt.want {
				t.Errorf("IsAnswerCorrect(%q, q) = %v, want %v", tt.userAnswer, got, tt.want)
			}
		})
	}
}

// ==================== normalizeAnswerSlice 单元测试 ====================

func TestNormalizeAnswerSlice(t *testing.T) {
	tests := []struct {
		name       string
		userAnswer string
		qType      string
		answers    []string
		want       []string
	}{
		{
			name:       "多选题-反序",
			userAnswer: "C,A",
			qType:      "多选题",
			answers:    []string{"A", "C"},
			want:       []string{"A", "C"},
		},
		{
			name:       "单选题",
			userAnswer: "B",
			qType:      "单选题",
			answers:    []string{"B"},
			want:       []string{"B"},
		},
		{
			name:       "填空题-顿号分隔",
			userAnswer: "25、30",
			qType:      "填空题",
			answers:    []string{"25~30"},
			want:       []string{"25", "30"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := core.Question{Type: tt.qType, Answers: tt.answers}
			got := core.NormalizeAnswerSlice(tt.userAnswer, q)
			if len(got) != len(tt.want) {
				t.Errorf("normalizeAnswerSlice(%q, q) 长度 = %d, want %d", tt.userAnswer, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("normalizeAnswerSlice(%q, q)[%d] = %q, want %q", tt.userAnswer, i, got[i], tt.want[i])
				}
			}
		})
	}
}
