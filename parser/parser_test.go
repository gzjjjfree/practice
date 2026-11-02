package parser

import (
	"encoding/json"
	"testing"
)

func TestParseBytesToBankJSON(t *testing.T) {
	bank := BankData{
		DisplayName: "Test Bank",
		StorageKey:  "test_bank",
		Questions: []QuestionItem{
			{
				ID:      1,
				Type:    "单选题",
				Content: "What is 1+1?",
				Answer:  "A",
				Options: OptionMap{
					"A": "2",
					"B": "3",
				},
			},
		},
	}

	data, err := json.Marshal(bank)
	if err != nil {
		t.Fatalf("failed to marshal test bank: %v", err)
	}

	storageKey := "test_bank"
	parsed, err := ParseBytesToBank(data, "test.json", storageKey)
	if err != nil {
		t.Fatalf("ParseBytesToBank failed: %v", err)
	}

	expectedName := ExtractFileName(storageKey)
	if parsed.DisplayName != expectedName {
		t.Errorf("expected DisplayName %q, got %q", expectedName, parsed.DisplayName)
	}
	if len(parsed.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(parsed.Questions))
	}
	if parsed.Questions[0].Content != "What is 1+1?" {
		t.Errorf("expected content 'What is 1+1?', got %q", parsed.Questions[0].Content)
	}
}

func TestParseBytesToBankInvalid(t *testing.T) {
	invalidData := []byte("not a valid format")
	_, err := ParseBytesToBank(invalidData, "invalid.txt", "invalid")
	if err == nil {
		t.Error("expected error parsing invalid bytes, got nil")
	}
}

func TestParserHelpersAndSanitize(t *testing.T) {
	// Test ExtractFileName
	if name := ExtractFileName("xlsxData_mybank.xlsx_123456"); name != "mybank" {
		t.Errorf("ExtractFileName = %q, expected 'mybank'", name)
	}
	if name := ExtractFileName("txtData_hello_111"); name != "hello" {
		t.Errorf("ExtractFileName = %q, expected 'hello'", name)
	}

	// Test GetFileExtension
	if ext := GetFileExtension("test.XLSX"); ext != "XLSX" {
		t.Errorf("GetFileExtension = %q, expected 'XLSX'", ext)
	}

	// Test IsUTF8 & FixEncodingBytes
	if !IsUTF8([]byte("hello utf8")) {
		t.Errorf("expected IsUTF8 to be true")
	}
	fixed := FixEncodingBytes([]byte("hello"))
	if string(fixed) != "hello" {
		t.Errorf("FixEncodingBytes failed")
	}

	// Test SanitizeBankData & SanitizeXSS
	dirtyBank := &BankData{
		DisplayName: "<script>alert(1)</script>SafeName",
		StorageKey:  "safe_key",
		Questions: []QuestionItem{
			{
				ID:      1,
				Content: "Question <script>evil()</script> content",
				Answer:  "A",
				Options: OptionMap{"A": "Option <img src=x onerror=alert(1)> A"},
			},
		},
	}
	SanitizeBankData(dirtyBank)
	if dirtyBank.DisplayName != "SafeName" {
		t.Errorf("SanitizeBankData failed for DisplayName: %q", dirtyBank.DisplayName)
	}
	if dirtyBank.Questions[0].Content != "Question  content" {
		t.Errorf("SanitizeXSS failed for Content: %q", dirtyBank.Questions[0].Content)
	}
}

func TestParseTxtContentDirectEdgeCases(t *testing.T) {
	txtContent := `选择题
1. 下列哪个是正确的？ (A)
A. 选项A
B. 选项B

选择题
2. 下列哪些是正确的？ (B)
A. 选项A
B. 选项B
`
	bank, err := parseTxtContentDirect(txtContent, "txtData_test_txt")
	if err != nil {
		t.Fatalf("parseTxtContentDirect failed: %v", err)
	}
	if len(bank.Questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(bank.Questions))
	}
	if bank.Questions[0].Type != "单选题" {
		t.Errorf("expected type '单选题', got %q", bank.Questions[0].Type)
	}
	if bank.Questions[1].Type != "单选题" {
		t.Errorf("expected type '单选题', got %q", bank.Questions[1].Type)
	}
}
