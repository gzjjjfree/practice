package storageRelated

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gzjjjfree/practice/core"
)

// SaveWrongSetToLocal saves the wrong-set as a JSON array of integers to a file.
func SaveWrongSetToLocal(state *core.AppState) {
	saveSetNoLock(state, "错题集", state.WrongSet)
}

// SaveFavSetToDisk saves the favorite set to disk.
func SaveFavSetToDisk(state *core.AppState) {
	if state.CurrentFileName == "" {
		return
	}
	saveSetNoLock(state, "收藏集", state.FavSet)
}

// saveSetNoLock is an internal helper that writes the wrong-set/fav-set without acquiring lock.
func saveSetNoLock(state *core.AppState, suffix string, currentSet map[string]bool) {
	if state.CurrentFileName == "" {
		return
	}

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

// HandleUserSelectOption is the central answer-handling gate.
func HandleUserSelectOption(state *core.AppState, q core.Question, selectedAnswer string) {
	state.PracticeRecordsMu.Lock()

	if state.PracticeRecords == nil {
		state.PracticeRecords = make(map[string]string)
	}

	_, alreadyAnswered := state.PracticeRecords[q.ID]
	if !alreadyAnswered || state.PracticeRecords[q.ID] == "null" {
		state.PracticeRecords[q.ID] = selectedAnswer

		isCorrect := core.IsAnswerCorrect(selectedAnswer, q)

		if !isCorrect {
			state.WrongSet[q.ID] = true
		}
	}

	// Copy ModeRecords for saving under lock, then unlock before I/O
	var modeCopy map[string]map[string]string
	if state.Title != "" && state.ModeRecords != nil {
		modeCopy = make(map[string]map[string]string, len(state.ModeRecords))
		for k, v := range state.ModeRecords {
			vCopy := make(map[string]string, len(v))
			for kk, vv := range v {
				vCopy[kk] = vv
			}
			modeCopy[k] = vCopy
		}
	}
	state.PracticeRecordsMu.Unlock()

	// Save outside lock (I/O doesn't need the lock since we copied data above)
	SaveWrongSetToLocal(state)
	savePracticeRecordsNoLock(state, modeCopy)
}

// savePracticeRecordsNoLock writes ModeRecords data to disk without acquiring lock.
func savePracticeRecordsNoLock(state *core.AppState, modeRecordsCopy map[string]map[string]string) {
	if state.CurrentFileName == "" {
		return
	}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := GetStorageDir()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := json.MarshalIndent(modeRecordsCopy, "", "  ")
	if err != nil {
		fmt.Printf("[WARN] marshal records %s failed: %v\n", fileName, err)
		return
	}
	if err := os.WriteFile(absolutePath, bytesData, 0644); err != nil {
		fmt.Printf("[WARN] write records %s failed: %v\n", fileName, err)
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
