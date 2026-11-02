package storageRelated

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gzjjjfree/practice/core"
)

func TestStorageDirAndConflict(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	SetStorageDir(tmpDir)
	if GetStorageDir() != tmpDir {
		t.Errorf("expected storage dir %q, got %q", tmpDir, GetStorageDir())
	}

	// Test FindExistingBankFiles & GenerateUniqueBankFileName
	dummyFile := filepath.Join(tmpDir, "xlsxData_testbank_123456.json")
	if err := os.WriteFile(dummyFile, []byte(`{"displayName":"testbank","questions":[]}`), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	existing := FindExistingBankFiles(tmpDir, "testbank.xlsx")
	if len(existing) != 1 || existing[0] != "xlsxData_testbank_123456.json" {
		t.Errorf("FindExistingBankFiles failed, got %v", existing)
	}

	uniqueName := GenerateUniqueBankFileName(tmpDir, "testbank.xlsx")
	if uniqueName != "testbank(1).xlsx" {
		t.Errorf("expected 'testbank(1).xlsx', got %q", uniqueName)
	}
}

func TestLocalSetsAndRecordsStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_sets_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	SetStorageDir(tmpDir)

	state := &core.AppState{
		CurrentFileName: "mybank",
		Title:           "练习模式",
		Index:           1,
		WrongSet:        map[string]bool{"1": true},
		FavSet:          map[string]bool{"2": true},
		ModeRecords:     map[string]map[string]string{"练习模式": {"1": "A"}},
		ModeIndices:     map[string]int{"练习模式": 1},
	}

	SaveWrongSetToLocal(state)
	SaveFavSetToDisk(state)
	SavePracticeProgress(state)

	// Test loading
	wrongSetLoaded := LoadSetFromLocal(state, "错题集")
	if !wrongSetLoaded["1"] {
		t.Errorf("expected wrong set to contain '1'")
	}

	favSetLoaded := LoadSetFromLocal(state, "收藏集")
	if !favSetLoaded["2"] {
		t.Errorf("expected fav set to contain '2'")
	}

	LoadPracticeRecordsFromLocal(state)
	if state.ModeRecords["练习模式"]["1"] != "A" {
		t.Errorf("expected mode record '1'='A', got %q", state.ModeRecords["练习模式"]["1"])
	}

	ClearPracticeRecords(state)
	if _, ok := state.ModeRecords["练习模式"]; ok {
		t.Errorf("expected mode records to be cleared")
	}
}
