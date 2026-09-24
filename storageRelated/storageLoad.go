package storageRelated

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/parser"
)

// RefreshLocalFilesList 扫描本地沙盒目录中的 .json 题库文件，更新已存储文件列表。
func RefreshLocalFilesList(state *core.AppState) {
	fmt.Printf("[REFRESH_FILES] START: storageDir=%q\n", GetStorageDir())
	storageDir := GetStorageDir()
	files, err := os.ReadDir(storageDir)
	if err != nil {
		fmt.Printf("[REFRESH_FILES] ReadDir error: %v\n", err)
		return
	}

	type FileInfo struct {
		Key       string
		Timestamp int64
	}

	var fileList []FileInfo

	fmt.Printf("[REFRESH_FILES] total files in dir: %d\n", len(files))
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			// 屏蔽掉错题集和收藏集，防止它们作为独立题库被展示
			if strings.Contains(f.Name(), "错题") || strings.Contains(f.Name(), "收藏") || strings.Contains(f.Name(), "答题记录") {
				fmt.Printf("[REFRESH_FILES]   SKIP (reserved): %s\n", f.Name())
				continue
			}

			name := strings.TrimSuffix(f.Name(), ".json")

			// 只处理题库文件（以 xxxData_ 开头），排除 preferences.json 等非题库文件
			if !strings.HasPrefix(name, "xlsxData_") && !strings.HasPrefix(name, "xlsData_") &&
				!strings.HasPrefix(name, "txtData_") && !strings.HasPrefix(name, "txtsData_") &&
				!strings.HasPrefix(name, "jsonData_") {
				fmt.Printf("[REFRESH_FILES]   SKIP (not a bank file): %s\n", f.Name())
				continue
			}
			// 假设 key 格式为: 类型Data_文件名_时间戳
			parts := strings.Split(name, "_")
			fmt.Printf("[REFRESH_FILES]   FILE: name=%q, parts count=%d\n", name, len(parts))
			if len(parts) >= 3 {
				tsStr := parts[len(parts)-1]
				ts, _ := strconv.ParseInt(tsStr, 10, 64)
				fileList = append(fileList, FileInfo{Key: name, Timestamp: ts})
				fmt.Printf("[REFRESH_FILES]   INCLUDED: Key=%q, Timestamp=%d\n", name, ts)
			} else {
				// No timestamp in filename (e.g., overwritten file like xlsxData_xxx.json)
				// Use current time so it sorts correctly (treated as newest)
				fmt.Printf("[REFRESH_FILES]   INCLUDED (no timestamp): Key=%q\n", name)
				fileList = append(fileList, FileInfo{Key: name, Timestamp: time.Now().UnixNano() / int64(time.Millisecond)})
			}
		}
	}

	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Timestamp < fileList[j].Timestamp
	})

	var finalKeys []string
	for i := len(fileList) - 1; i >= 0; i-- {
		finalKeys = append(finalKeys, fileList[i].Key)
	}

	fmt.Printf("[REFRESH_FILES] FINAL StoredFiles has %d keys: %v\n", len(finalKeys), finalKeys)
	state.StoredFiles = finalKeys
}

// LoadAndRenderBank 从本地加载指定题库并刷新界面状态。
func LoadAndRenderBank(w fyne.Window, state *core.AppState, targetKey string) {
	state.CurrentStorageKey = targetKey
	state.CurrentFileName = parser.ExtractFileName(targetKey)

	storageDir := GetStorageDir()
	absoluteReadPath := filepath.Join(storageDir, targetKey+".json")

	bytesData, err := os.ReadFile(absoluteReadPath)
	if err == nil {
		var bankData parser.BankData
		if err := json.Unmarshal(bytesData, &bankData); err == nil {
			core.SyncQuestionsToState(state, &bankData)
			LoadRecordsForCurrentBank(state)
			LoadPracticeRecordsFromLocal(state)
			fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", state.CurrentStorageKey)
		} else {
			// 静默忽略JSON反序列化失败，不弹窗提示用户
			fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", "")
		}
	} else {
		// 静默忽略读取沙盒文件失败，不弹窗提示用户
		fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", "")
	}
}

// ShowBankFileOpenDialog 打开并处理题库文件导入
func ShowBankFileOpenDialog(w fyne.Window, state *core.AppState, refreshHomeUI func()) {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			if err != nil {
				dialog.ShowError(err, w)
			}
			return
		}
		defer reader.Close()

		originalFileName := reader.URI().Name()
		fileBytes, readErr := io.ReadAll(reader)
		if readErr != nil {
			customElements.ShowCustomInformation("读取失败", fmt.Sprintf("无法读取文件 %s: %v", originalFileName, readErr), w)
			return
		}

		storageDir := GetStorageDir()
		files, _ := os.ReadDir(storageDir)

		// 搜寻重复旧文件
		ext := filepath.Ext(originalFileName)
		baseName := strings.TrimSuffix(originalFileName, ext)
		searchPatterns := []string{"_" + originalFileName + "_", "_" + baseName + "_", "_" + originalFileName}

		var oldFileNames []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
				for _, p := range searchPatterns {
					if strings.Contains(f.Name(), p) {
						oldFileNames = append(oldFileNames, f.Name())
						break
					}
				}
			}
		}

		// 执行保存与解析
		processAndSave := func(isNewAlias bool) {
			fileName := originalFileName

			// 如果是作为副本新增，生成唯一文件名
			if isNewAlias {
				fileName = GenerateUniqueBankFileName(storageDir, originalFileName)
			} else { // 覆盖原有旧文件
				for _, oldName := range oldFileNames {
					_ = os.Remove(filepath.Join(storageDir, oldName))
				}
			}

			// 显示加载进度框
			progDialog := dialog.NewCustomWithoutButtons("读取中", widget.NewProgressBarInfinite(), w)
			progDialog.Show()
			defer progDialog.Hide()

			// 旧版 XLS 升级处理
			if strings.ToLower(filepath.Ext(fileName)) == ".xls" {
				if newBytes, convErr := parser.ConvertXlsToXlsxBytes(fileBytes); convErr == nil {
					fileBytes = newBytes
					fileName = baseName + ".xlsx"
				}
			}

			timestamp := time.Now().UnixMilli()
			fileExt := strings.ToLower(parser.GetFileExtension(fileName))
			storageKey := fmt.Sprintf("%sData_%s_%d", fileExt, fileName, timestamp)

			// 解析并保存
			bankData, parseErr := parser.ParseBytesToBank(fileBytes, fileName, storageKey)
			if parseErr != nil {
				customElements.ShowCustomInformation("解析失败", fmt.Sprintf("无法解析文件 %s: %v", fileName, parseErr), w)
				return
			}

			jsonData, _ := json.Marshal(bankData)
			if err := os.WriteFile(filepath.Join(storageDir, storageKey+".json"), jsonData, 0644); err != nil {
				customElements.ShowCustomInformation("存储失败", fmt.Sprintf("无法存储文件 %s: %v", fileName, err), w)
				return
			}

			// 更新全局状态与UI
			fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", storageKey)
			state.CurrentStorageKey = bankData.StorageKey
			state.CurrentFileName = bankData.DisplayName

			core.SyncQuestionsToState(state, bankData)
			RefreshLocalFilesList(state)
			refreshHomeUI()
		}

		// 文件重复弹窗确认
		if len(oldFileNames) > 0 {
			showConflictDialog(w, originalFileName, processAndSave)
		} else {
			processAndSave(false)
		}
	}, w)

	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx", ".xls", ".txt", ".json"}))
	fileDialog.Show()
}

// ConflictResolutionChoice 表示用户对文件冲突的选择
type ConflictResolutionChoice int

const (
	ResolveOverwrite ConflictResolutionChoice = iota // 覆盖原有文件
	ResolveNewAlias                                  // 作为新文件保存
)

// FindExistingBankFiles 查找本地存储中与指定文件名冲突的题库文件
func FindExistingBankFiles(storageDir string, humanName string) []string {
	files, _ := os.ReadDir(storageDir)

	ext := filepath.Ext(humanName)
	baseName := strings.TrimSuffix(humanName, ext)
	searchPatterns := []string{"_" + humanName + "_", "_" + baseName + "_", "_" + humanName}

	var oldFileNames []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			// 排除错题集、收藏集、答题记录等文件
			if strings.Contains(f.Name(), "错题") || strings.Contains(f.Name(), "收藏") || strings.Contains(f.Name(), "答题记录") {
				continue
			}

			for _, p := range searchPatterns {
				if strings.Contains(f.Name(), p) {
					// 排除带(n)后缀的文件如：测试题库.xlsx(1)_timestamp.json
					if !strings.Contains(f.Name(), "_"+baseName+"(") && !strings.Contains(f.Name(), humanName+"(") {
						oldFileNames = append(oldFileNames, f.Name())
					}
					break
				}
			}
		}
	}

	return oldFileNames
}

// ShowFileConflictDialog 显示文件冲突对话框
func ShowFileConflictDialog(w fyne.Window, fileName string, onChoice func(choice ConflictResolutionChoice)) {
	var confirmDialog dialog.Dialog

	cancelBtn := widget.NewButton(" 取消 ", func() { confirmDialog.Hide() })

	overwriteBtn := widget.NewButton(" 覆盖 ", func() {
		confirmDialog.Hide()
		onChoice(ResolveOverwrite)
	})
	overwriteBtn.Importance = widget.DangerImportance

	addBtn := widget.NewButton(" 新增 ", func() {
		confirmDialog.Hide()
		onChoice(ResolveNewAlias)
	})
	addBtn.Importance = widget.HighImportance

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, overwriteBtn, addBtn, layout.NewSpacer())
	msgLbl := widget.NewLabel(fmt.Sprintf("题库列表中已存在名为【%s】的文件。\n\n请选择接下来的操作：", fileName))
	msgLbl.Alignment = fyne.TextAlignCenter
	msgLbl.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		msgLbl,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		container.NewPadded(container.NewCenter(buttons)),
	)

	styledContent := container.NewStack(
		container.NewGridWrap(fyne.NewSize(300, 1)),
		container.NewPadded(content),
	)

	confirmDialog = dialog.NewCustomWithoutButtons("文件已存在", styledContent, w)
	confirmDialog.Show()
}

// GenerateUniqueBankFileName 根据原有文件名和存储目录，生成不重复的唯一副本文件名（如：测试(1).xlsx）
func GenerateUniqueBankFileName(storageDir string, originalFileName string) string {
	files, _ := os.ReadDir(storageDir)
	ext := filepath.Ext(originalFileName)
	baseName := strings.TrimSuffix(originalFileName, ext)
	fileName := originalFileName

	for counter := 1; ; counter++ {
		testFileName := fmt.Sprintf("%s(%d)%s", baseName, counter, ext)
		testPattern := "_" + testFileName + "_"
		exists := false
		for _, f := range files {
			if !f.IsDir() && (strings.Contains(f.Name(), testPattern) || strings.Contains(f.Name(), "_"+testFileName)) {
				exists = true
				break
			}
		}
		if !exists {
			fileName = testFileName
			break
		}
	}
	return fileName
}

// GenerateNewStorageKey 生成新的存储键（处理新增副本的情况）
func GenerateNewStorageKey(storageDir string, humanName string, fileExtPrefix string) string {
	ext := filepath.Ext(humanName)
	baseName := strings.TrimSuffix(humanName, ext)

	for counter := 1; ; counter++ {
		testFileName := fmt.Sprintf("%s(%d)%s", baseName, counter, ext)
		testKey := fmt.Sprintf("%sData_%s_%d", fileExtPrefix, testFileName, time.Now().UnixMilli())

		// 检查该存储键是否已存在
		exists := false
		files, _ := os.ReadDir(storageDir)
		for _, f := range files {
			if !f.IsDir() && f.Name() == testKey+".json" {
				exists = true
				break
			}
		}

		if !exists {
			return testKey
		}
	}
}

// OverwriteExistingBankFiles 覆盖现有的题库文件
func OverwriteExistingBankFiles(storageDir string, oldFileNames []string) error {
	for _, oldName := range oldFileNames {
		lfPath := filepath.Join(storageDir, oldName)
		if err := os.Remove(lfPath); err != nil {
			return fmt.Errorf("删除旧文件失败: %v", err)
		}
	}
	return nil
}

// 辅助函数：显示同名覆盖/新增冲突弹窗（保留向后兼容）
func showConflictDialog(w fyne.Window, fileName string, onChoice func(isNewAlias bool)) {
	var confirmDialog dialog.Dialog

	cancelBtn := widget.NewButton(" 取消 ", func() { confirmDialog.Hide() })

	overwriteBtn := widget.NewButton(" 覆盖 ", func() {
		confirmDialog.Hide()
		onChoice(false)
	})
	overwriteBtn.Importance = widget.DangerImportance

	addBtn := widget.NewButton(" 新增 ", func() {
		confirmDialog.Hide()
		onChoice(true)
	})
	addBtn.Importance = widget.HighImportance

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, overwriteBtn, addBtn, layout.NewSpacer())
	msgLbl := widget.NewLabel(fmt.Sprintf("题库列表中已存在名为【%s】的文件。\n\n请选择接下来的操作：", fileName))
	msgLbl.Alignment = fyne.TextAlignCenter
	msgLbl.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		msgLbl,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		container.NewPadded(container.NewCenter(buttons)),
	)

	styledContent := container.NewStack(
		container.NewGridWrap(fyne.NewSize(300, 1)),
		container.NewPadded(content),
	)

	confirmDialog = dialog.NewCustomWithoutButtons("文件已存在", styledContent, w)
	confirmDialog.Show()
}

// LoadRecordsForCurrentBank loads both wrong set and favorite set for the current bank.
func LoadRecordsForCurrentBank(state *core.AppState) {
	state.WrongSet = LoadSetFromLocal(state, "错题集")
	state.FavSet = LoadSetFromLocal(state, "收藏集")
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
