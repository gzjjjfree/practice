package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
)

// 1. ✨ 保存数据集到本地沙盒 (支持 错题集 / 收藏集)
func saveSetToLocal(state *AppState, suffix string, currentSet map[string]bool) {
	if state.CurrentFileName == "" {
		return
	}

	// 构造标准文件名，例如: "附件2-2 内燃机车钳工题库_错题集.json"
	fileName := state.CurrentFileName + "_" + suffix + ".json"
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absolutePath := filepath.Join(storageDir, fileName)

	// 将 map 转换为整型数组 [2, 6, 7]
	var idList []int
	for idStr, active := range currentSet {
		if active {
			if id, err := strconv.Atoi(idStr); err == nil {
				idList = append(idList, id)
			}
		}
	}

	// 序列化并写入
	jsonData, _ := json.Marshal(idList)
	_ = os.WriteFile(absolutePath, jsonData, 0644)
}

// 2. ✨ 从本地沙盒加载数据集
func loadSetFromLocal(state *AppState, suffix string) map[string]bool {
	resultSet := make(map[string]bool)
	if state.CurrentFileName == "" {
		return resultSet
	}

	fileName := state.CurrentFileName + "_" + suffix + ".json"
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := os.ReadFile(absolutePath)
	if err != nil {
		return resultSet // 文件不存在直接返回空集
	}

	var idList []int
	if err := json.Unmarshal(bytesData, &idList); err == nil {
		for _, id := range idList {
			resultSet[strconv.Itoa(id)] = true
		}
	}
	return resultSet
}

// 3. ✨ 核心卡口：切换题库时，必须联动加载对应的错题和收藏
func loadRecordsForCurrentBank(state *AppState) {
	state.WrongSet = loadSetFromLocal(state, "错题集")
	state.FavSet = loadSetFromLocal(state, "收藏集")
}

// 1. ✨ 保存本次练习记录与统计到本地文件
func savePracticeRecordsToLocal(state *AppState) {
	if state.CurrentFileName == "" {
		return
	}

	// 定义存储的数据结构，把记录和当前的 ✓、✕ 计数一起打包
	//type PracticeSaveData struct {
	//	//CorrectCount int               `json:"correct_count"`
	//	//WrongCount   int               `json:"wrong_count"`
	//	Records map[string]string `json:"records"`
	//}
	//
	//data := PracticeSaveData{
	//	//CorrectCount: state.PracticeCorrectCount,
	//	//WrongCount:   state.PracticeWrongCount,
	//	Records: state.PracticeRecords,
	//}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absolutePath := filepath.Join(storageDir, fileName)

	// 🔥 修改：将外层的 ModeRecords 整体序列化
	bytesData, err := json.MarshalIndent(state.ModeRecords, "", "  ")
	if err != nil {
		fmt.Println("保存记录失败:", err)
		return
	}
	os.WriteFile(absolutePath, bytesData, 0644)

	//jsonData, _ := json.Marshal(data)
	//_ = os.WriteFile(absolutePath, jsonData, 0644)
}

// 2. ✨ 从本地加载已有的答题记录
func loadPracticeRecordsFromLocal(state *AppState) {
	if state.CurrentFileName == "" {
		//state.PracticeRecords = make(map[string]string)
		state.ModeRecords = make(map[string]map[string]string)
		//state.PracticeCorrectCount = 0
		//state.PracticeWrongCount = 0
		return
	}

	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absolutePath := filepath.Join(storageDir, fileName)

	bytesData, err := os.ReadFile(absolutePath)
	if err != nil {
		// 文件不存在，赋空值
		state.ModeRecords = make(map[string]map[string]string)
		return
	}

	// 🔥 修改：反序列化给 ModeRecords
	err = json.Unmarshal(bytesData, &state.ModeRecords)
	if err != nil || state.ModeRecords == nil {
		state.ModeRecords = make(map[string]map[string]string)
	}

	// 如果读取成功，且有当前模式，记得恢复一下指针！
	if state.Title != "" && state.ModeRecords[state.Title] != nil {
		state.PracticeRecords = state.ModeRecords[state.Title]
	}

	//bytesData, err := os.ReadFile(absolutePath)
	//if err != nil {
	//	// 文件不存在，说明是全新练习，初始化干净的数据
	//	state.PracticeRecords = make(map[string]string)
	//	//state.PracticeCorrectCount = 0
	//	//state.PracticeWrongCount = 0
	//	return
	//}

	//type PracticeSaveData struct {
	//	CorrectCount int               `json:"correct_count"`
	//	WrongCount   int               `json:"wrong_count"`
	//	Records      map[string]string `json:"records"`
	//}
	//
	//var data PracticeSaveData
	//if err := json.Unmarshal(bytesData, &data); err == nil {
	//	//state.PracticeCorrectCount = data.CorrectCount
	//	//state.PracticeWrongCount = data.WrongCount
	//	state.PracticeRecords = data.Records
	//} else {
	//	state.PracticeRecords = make(map[string]string)
	//	//state.PracticeCorrectCount = 0
	//	//state.PracticeWrongCount = 0
	//}
}

// 3. ✨ 【配合任务5】提供给“删除记录”按钮使用的清空函数
func clearPracticeRecords(state *AppState) {
	//state.PracticeCorrectCount = 0
	//state.PracticeWrongCount = 0
	state.PracticeRecords = make(map[string]string)

	// 从物理磁盘上彻底删除该记录文件
	fileName := state.CurrentFileName + "_答题记录.json"
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absolutePath := filepath.Join(storageDir, fileName)
	_ = os.Remove(absolutePath)
}

// 💡 这是当你点击某个选项（比如选了 "A"）时的处理卡口：
func handleUserSelectOption(state *AppState, q Question, selectedAnswer string) {
	// 🔥【核心硬核注入】：全自动熔断器，防止 nil map 触发 assignment 崩溃
	if state.PracticeRecords == nil {
		state.PracticeRecords = make(map[string]string)
	}

	// selectedAnswer 格式定义：如果是单选/判断为 "A" 或 "B"；如果是多选，是由逗号分隔的已选项字符串如 "A,B"

	// 如果这道题之前已经答过了，不重复计分
	_, alreadyAnswered := state.PracticeRecords[q.ID]
	if alreadyAnswered && state.PracticeRecords[q.ID] != "null" {
		return
	}

	// 1. 将用户的选择存入临时会话记录
	state.PracticeRecords[q.ID] = selectedAnswer

	// 2. ✨ 处理多选与单选的通用判定逻辑
	isCorrect := false

	// 将用户选的答案切开并排序，消除用户点击顺序带来的干扰 (如选了 B,A 转换为 ["A", "B"])
	userAnsSlice := strings.Split(selectedAnswer, ",")
	for i := range userAnsSlice {
		userAnsSlice[i] = strings.TrimSpace(userAnsSlice[i])
	}
	sort.Strings(userAnsSlice)

	// 同样将结构体里的标准答案 []string 复制一份进行排序，确保比对基准绝对一致
	correctAnsSlice := make([]string, len(q.Answers))
	copy(correctAnsSlice, q.Answers)
	sort.Strings(correctAnsSlice)

	// 比对两个切片的长度和内容是否完全相等
	if len(userAnsSlice) == len(correctAnsSlice) {
		matchCount := 0
		for i := range userAnsSlice {
			if userAnsSlice[i] == correctAnsSlice[i] {
				matchCount++
			}
		}
		if matchCount == len(correctAnsSlice) {
			isCorrect = true
		}
	}

	// 3. 执行打分状态联动
	//if isCorrect {
	//	state.PracticeCorrectCount++
	//} else {
	if !isCorrect {
		//state.PracticeWrongCount++

		// 错题集只增不减：打上钢印并自动同步到沙盒持久化
		state.WrongSet[q.ID] = true
		saveSetToLocal(state, "错题集", state.WrongSet)
	}

	// 4. 每次作答后立刻将本次答题状态落盘
	savePracticeRecordsToLocal(state)
}
