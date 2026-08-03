package main

import (
	"testing"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/parser"
)

// ==================== cleanXlsData 单元测试 ====================

func TestCleanXlsData(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"正常文本", "  hello world  ", "hello world"},
		{"含空字节", "hello\x00world", "helloworld"},
		{"纯空白", "   \n\t  ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.CleanXlsData(tt.input)
			if got != tt.want {
				t.Errorf("cleanXlsData(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== cleanHeaderMatch 单元测试 ====================

func TestCleanHeaderMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"正常表头", "  题号  ", "题号"},
		{"全角空格", "题 号", "题号"},
		{"含换行", "题\n号\r", "题号"},
		{"英文表头", "  Number  ", "number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.CleanHeaderMatch(tt.input)
			if got != tt.want {
				t.Errorf("cleanHeaderMatch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== getFileExtension 单元测试 ====================

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"xlsx", "test.xlsx", "xlsx"},
		{"xls", "test.xls", "xls"},
		{"txt", "test.txt", "txt"},
		{"json", "data.json", "json"},
		{"无扩展名", "noext", ""},
		{"多级扩展名", "file.tar.gz", "gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.GetFileExtension(tt.input)
			if got != tt.want {
				t.Errorf("getFileExtension(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== extractFileName 单元测试 ====================

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"带时间戳无前缀点", "xlsxData_内燃机车钳工题库_1720000000000", "内燃机车钳工题库_1720000000000"},
		{"带后缀", "txtData_test.txt_1234567890", "test"},
		{"无前缀", "普通文件名", "普通文件名"},
		{"含括号别名", "xlsxData_题库(1)_999999", "题库(1)_999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.ExtractFileName(tt.input)
			if got != tt.want {
				t.Errorf("extractFileName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== isUTF8 单元测试 ====================

func TestIsUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"纯ASCII", []byte("hello world"), true},
		{"有效UTF-8中文", []byte("\xe4\xb8\xad\xe6\x96\x87"), true},
		{"GBK编码", []byte{0xb2, 0xe2, 0xca, 0xd4}, false},
		{"空字节流", []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.IsUTF8(tt.input)
			if got != tt.want {
				t.Errorf("isUTF8(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ==================== contains / removeElement 单元测试 ====================

func TestContains(t *testing.T) {
	if core.Contains([]string{"a", "b", "c"}, "b") != true {
		t.Error("contains([a,b,c], b) = false, want true")
	}
	if core.Contains([]string{"a", "b", "c"}, "d") != false {
		t.Error("contains([a,b,c], d) = true, want false")
	}
}

func TestRemoveElement(t *testing.T) {
	input := []string{"a", "b", "c", "b"}
	got := core.RemoveElement(input, "b")
	want := []string{"a", "c"}
	if len(got) != len(want) {
		t.Fatalf("removeElement(%v, b) 长度 = %d, want %d", input, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("removeElement(%v, b)[%d] = %q, want %q", input, i, got[i], want[i])
		}
	}
}
