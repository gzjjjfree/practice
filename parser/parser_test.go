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
