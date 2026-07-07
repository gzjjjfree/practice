package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// ==================== 辅助工具 ====================
func hexColor(hex string) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	return color.Transparent
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeElement(slice []string, item string) []string {
	var res []string
	for _, s := range slice {
		if s != item {
			res = append(res, s)
		}
	}
	return res
}

// 辅助转换函数：将后端 Parser 的 QuestionItem 同步回全局 UI 的状态层中
func syncQuestionsToState(state *AppState, bankData *BankData) {
	state.Questions = make([]Question, len(bankData.Questions))
	for idx, srcQ := range bankData.Questions {
		var convertedOpts []Option
		for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
			if srcQ.Options[k] != "" {
				convertedOpts = append(convertedOpts, Option{Label: k, Text: srcQ.Options[k]})
			}
		}
		var answers []string
		cleanAns := strings.NewReplacer("、", "", ",", "", " ", "").Replace(srcQ.Answer)
		for _, char := range cleanAns {
			answers = append(answers, string(char))
		}
		if len(answers) == 0 && srcQ.Answer != "" {
			answers = append(answers, srcQ.Answer)
		}

		state.Questions[idx] = Question{
			ID:         strconv.Itoa(srcQ.ID),
			Type:       srcQ.Type,
			Content:    srcQ.Content,
			Options:    convertedOpts,
			Answers:    answers,
			Difficulty: srcQ.Difficulty,
		}
	}

	if state.ModeIndices == nil {
		state.ModeIndices = make(map[string]int)
	}
}

// 专门用于统计当前 CurrentList 里的答题数据
func calculateCurrentModeStats(state *AppState) (correctCount int, wrongCount int) {
	for _, q := range state.CurrentList {
		historyAns, exists := state.PracticeRecords[q.ID]
		if !exists || historyAns == "null" || historyAns == "" {
			continue // 没做过的题跳过
		}

		isRight := false
		if q.Type == "填空题" || q.Type == "问答题" {
			isRight = true // 背题模式只要看了就算绿
		} else if q.Type == "多选题" {
			userAnsSlice := strings.Split(historyAns, ",")
			for j := range userAnsSlice {
				userAnsSlice[j] = strings.TrimSpace(userAnsSlice[j])
			}
			sort.Strings(userAnsSlice)

			correctAnsSlice := make([]string, len(q.Answers))
			copy(correctAnsSlice, q.Answers)
			sort.Strings(correctAnsSlice)

			if len(userAnsSlice) == len(correctAnsSlice) {
				matchCount := 0
				for j := range userAnsSlice {
					if userAnsSlice[j] == correctAnsSlice[j] {
						matchCount++
					}
				}
				if matchCount == len(correctAnsSlice) {
					isRight = true
				}
			}
		} else {
			for _, ans := range q.Answers {
				if historyAns == ans {
					isRight = true
					break
				}
			}
		}

		if isRight {
			correctCount++
		} else {
			wrongCount++
		}
	}
	return correctCount, wrongCount
}

// deleteBank 执行删除题库并智能切换游标
func deleteBank(w fyne.Window, state *AppState) {
	targetStorageKey := state.CurrentStorageKey
	// 1. 找到要删除的题库 Key 在 StoredFiles 列表中的索引
	targetIndex := -1
	for i, key := range state.StoredFiles {
		if key == targetStorageKey {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return // 没找到，安全拦截
	}

	// 2. 执行物理删除（从本地存储中删除对应的 JSON 文件）
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	err := os.Remove(filepath.Join(storageDir, targetStorageKey+".json"))
	if err != nil && !os.IsNotExist(err) {
		dialog.ShowInformation("[DEBUG ❌]", fmt.Sprintf("删除题库错误: %v\n", err), w)
		return
	}

	// 3. 从内存的 StoredFiles 列表中彻底移除该 Key
	state.StoredFiles = append(state.StoredFiles[:targetIndex], state.StoredFiles[targetIndex+1:]...)

	// ==================== 4. 智能选择下一个题库 ====================
	if len(state.StoredFiles) > 0 {
		var nextBankKey string

		if targetIndex < len(state.StoredFiles) {
			// 情况 A：后面还有题库，原 targetIndex+1 的元素已经补位到了 targetIndex
			// 游标不动，就自动选中了“下一个”
			nextBankKey = state.StoredFiles[targetIndex]
		} else {
			// 情况 B：删除的是列表中最后一个题库，游标 targetIndex 已越界
			// 强制将游标上移，选中当前新的最后一个题库
			nextBankKey = state.StoredFiles[len(state.StoredFiles)-1]
		}

		// 触发加载算出来的下一个题库
		// (需要结合你现有的本地读取函数，例如从文件读取 JSON 并解析)
		loadAndRenderBank(w, state, nextBankKey)

	} else {
		// 情况 C：题库被全部删光了，清空所有当前浏览状态
		state.CurrentStorageKey = ""
		state.CurrentFileName = "未选择题库"
		state.Questions = nil
		state.CurrentList = nil

		// 务必同步清空偏好记录，防止下次打开 APP 时报错
		fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", "")

		// 刷新主页 UI，显示空状态（请替换为你实际刷新 UI 的函数）
		// refreshHomeUI()
	}
}

// 辅助函数：加载指定的题库并刷新界面
func loadAndRenderBank(w fyne.Window, state *AppState, targetKey string) {

	state.CurrentStorageKey = targetKey
	state.CurrentFileName = extractFileName(targetKey)

	// 1. 拼装手机/PC通用的沙盒绝对路径
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absoluteReadPath := filepath.Join(storageDir, targetKey+".json")

	// 2. 从本地读取出 .json 缓存字节流
	bytesData, err := os.ReadFile(absoluteReadPath)
	if err == nil {
		// 3. 将缓存数据反序列化为最初的 BankData 结构体
		var bankData BankData
		if err := json.Unmarshal(bytesData, &bankData); err == nil {

			// 🔥【核心修复点】：必须把解出来的题目真正灌入到当前的全局 state 中！
			// 这样后续的 len(state.Questions) 检查才不会变成 0
			syncQuestionsToState(state, &bankData)

			// 🔥【核心注入】：在这里一键载入该题库专属的错题集与收藏集！
			loadRecordsForCurrentBank(state)

			// 🔥【核心注入】：切换题库时同步加载上一次未完成的练习记录
			loadPracticeRecordsFromLocal(state)

			// 记录最后一次打开的题库 Key
			fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", state.CurrentStorageKey)

			// dialog.ShowInformation("切换成功", "已载入题库: "+state.CurrentFileName, w)
		} else {
			dialog.ShowInformation("[DEBUG ❌]", fmt.Sprintf("JSON反序列化失败: %v\n", err), w)
			//fmt.Printf("[DEBUG ❌] JSON反序列化失败: %v\n", err)
		}
	} else {
		dialog.ShowInformation("[DEBUG ❌]", fmt.Sprintf("读取沙盒题库文件失败: %v\n", err), w)
		//fmt.Printf("[DEBUG ❌] 读取沙盒题库文件失败: %v\n", err)
	}
	// 2. 更新偏好设置
	//fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", targetKey)
}
