package gz_storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"github.com/gzjjjfree/practice/core"
)

// SaveSetToLocal saves a set (wrong/fav) as a JSON array of integers to a file.
func SaveSetToLocal(state *core.AppState, suffix string, currentSet map[string]bool) {
	if state.CurrentFileName == "" {
		return
	}

	state.PracticeRecordsMu.Lock()
	defer state.PracticeRecordsMu.Unlock()

	fileName := state.CurrentFileName + "_" + suffix + ".json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)

	var idList []int
	for idStr, active := range currentSet {
		if active {
			if id, err := strconv.Atoi(idStr); err == nil {
				idList = append(idList, id)
			}
		}
	}

	jsonData, err := json.Marshal(idList)
	if err != nil {
		fmt.Printf("[WARN] marshal %s failed: %v\n", fileName, err)
		return
	}
	if err := os.WriteFile(absolutePath, jsonData, 0644); err != nil {
		fmt.Printf("[WARN] write %s failed: %v\n", fileName, err)
	}
}

// LoadSetFromLocal loads a set from the corresponding JSON file.
func LoadSetFromLocal(state *core.AppState, suffix string) map[string]bool {
	resultSet := make(map[string]bool)
	if state.CurrentFileName == "" {
		return resultSet
	}

	fileName := state.CurrentFileName + "_" + suffix + ".json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := os.ReadFile(absolutePath)
	if err != nil {
		return resultSet
	}

	var idList []int
	if err := json.Unmarshal(bytesData, &idList); err != nil {
		fmt.Printf("[WARN] parse %s failed, file may be corrupted: %v\n", fileName, err)
	} else {
		for _, id := range idList {
			resultSet[strconv.Itoa(id)] = true
		}
	}
	return resultSet
}

// LoadRecordsForCurrentBank loads both wrong set and favorite set for the current bank.
func LoadRecordsForCurrentBank(state *core.AppState) {
	state.WrongSet = LoadSetFromLocal(state, "错题集")
	state.FavSet = LoadSetFromLocal(state, "收藏集")
}

// SavePracticeRecordsToLocal saves the entire ModeRecords map to disk.
func SavePracticeRecordsToLocal(state *core.AppState) {
	if state.CurrentFileName == "" {
		return
	}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := json.MarshalIndent(state.ModeRecords, "", "  ")
	if err != nil {
		fmt.Printf("[WARN] marshal records %s failed: %v\n", fileName, err)
		return
	}
	if err := os.WriteFile(absolutePath, bytesData, 0644); err != nil {
		fmt.Printf("[WARN] write records %s failed: %v\n", fileName, err)
	}
}

// LoadPracticeRecordsFromLocal loads ModeRecords from disk.
func LoadPracticeRecordsFromLocal(state *core.AppState) {
	if state.CurrentFileName == "" {
		state.ModeRecords = make(map[string]map[string]string)
		return
	}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := os.ReadFile(absolutePath)
	if err != nil {
		state.ModeRecords = make(map[string]map[string]string)
		return
	}

	err = json.Unmarshal(bytesData, &state.ModeRecords)
	if err != nil {
		fmt.Printf("[WARN] parse records %s failed: %v, using empty records\n", fileName, err)
		state.ModeRecords = make(map[string]map[string]string)
	} else if state.ModeRecords == nil {
		state.ModeRecords = make(map[string]map[string]string)
	}

	if state.Title != "" && state.ModeRecords[state.Title] != nil {
		state.PracticeRecordsMu.Lock()
		state.PracticeRecords = state.ModeRecords[state.Title]
		state.PracticeRecordsMu.Unlock()
	}
}

// ClearPracticeRecords clears all practice records for the current mode.
func ClearPracticeRecords(state *core.AppState) {
	state.PracticeRecordsMu.Lock()
	defer state.PracticeRecordsMu.Unlock()

	for key := range state.PracticeRecords {
		delete(state.PracticeRecords, key)
	}

	if state.Title != "" && state.ModeRecords != nil {
		delete(state.ModeRecords, state.Title)
	}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)
	_ = os.Remove(absolutePath)
}

// HandleUserSelectOption is the central answer-handling gate.
func HandleUserSelectOption(state *core.AppState, q core.Question, selectedAnswer string) {
	state.PracticeRecordsMu.Lock()
	defer state.PracticeRecordsMu.Unlock()

	if state.PracticeRecords == nil {
		state.PracticeRecords = make(map[string]string)
	}

	_, alreadyAnswered := state.PracticeRecords[q.ID]
	if alreadyAnswered && state.PracticeRecords[q.ID] != "null" {
		return
	}

	state.PracticeRecords[q.ID] = selectedAnswer

	isCorrect := core.IsAnswerCorrect(selectedAnswer, q)

	if !isCorrect {
		state.WrongSet[q.ID] = true
		SaveSetToLocal(state, "错题集", state.WrongSet)
	}

	SavePracticeRecordsToLocal(state)
}

// GetStorageDir returns the app's storage directory path.
// It can be overridden via SetStorageDir for testing.
func GetStorageDir() string {
	if customStorageDir != "" {
		return customStorageDir
	}
	return fyne.CurrentApp().Storage().RootURI().Path()
}

var customStorageDir = ""

// SetStorageDir overrides the storage directory path (called once from main.go).
func SetStorageDir(path string) {
	customStorageDir = path
}
