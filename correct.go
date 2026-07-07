package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// 将当前内存中的 state.Questions 强行逆向转化为 BankData 并覆写本地文件
func syncStateToDisk(state *AppState) error {
	bankData := BankData{
		DisplayName: state.CurrentFileName,
		StorageKey:  state.CurrentStorageKey,
		Questions:   make([]QuestionItem, len(state.Questions)),
	}

	for i, q := range state.Questions {
		idInt, _ := strconv.Atoi(q.ID)
		optMap := make(map[string]string)
		for _, opt := range q.Options {
			optMap[opt.Label] = opt.Text
		}

		var ansStr string
		if q.Type == "填空题" || q.Type == "问答题" {
			ansStr = strings.Join(q.Answers, "、")
		} else {
			ansStr = strings.Join(q.Answers, "")
		}

		bankData.Questions[i] = QuestionItem{
			ID:         idInt,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     ansStr,
			Options:    optMap,
			RawIndex:   i + 1, // 重建索引
			Difficulty: q.Difficulty,
		}
	}

	jsonData, err := json.Marshal(bankData)
	if err != nil {
		return err
	}

	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
	absoluteSavePath := filepath.Join(storageDir, state.CurrentStorageKey+".json")
	return os.WriteFile(absoluteSavePath, jsonData, 0644)
}

// 深度克隆题目，防止在点击“保存”前污染全局状态
func cloneQuestionForEdit(q Question) Question {
	newQ := q
	newQ.Options = make([]Option, len(q.Options))
	copy(newQ.Options, q.Options)
	newQ.Answers = make([]string, len(q.Answers))
	copy(newQ.Answers, q.Answers)
	return newQ
}

// 构建修正题目模式的界面
func renderCorrectionCard(w fyne.Window, state *AppState, q Question, onRefresh func()) fyne.CanvasObject {
	// 1. 克隆出用于编辑的副本
	editQ := cloneQuestionForEdit(q)

	// ==================== 问题 2: 按钮区布局 ====================
	btnAddOption := widget.NewButton("增加选项", nil)
	btnAddQuestion := widget.NewButton("新增题目", func() {
		// 预留给后续新增题目逻辑
		dialog.ShowInformation("提示", "进入新增题目流", w)
	})
	btnSave := widget.NewButton("保存本题", nil)
	btnSave.Importance = widget.HighImportance // 设为主色调
	btnDelete := widget.NewButton("删除本题", nil)

	btnRow1 := container.NewGridWithColumns(2, btnAddOption, btnAddQuestion)
	btnRow2 := container.NewGridWithColumns(2, btnSave, btnDelete)
	buttonPanel := container.NewVBox(btnRow1, btnRow2)

	// ==================== 公共区域：题干编辑 ====================
	contentEntry := widget.NewMultiLineEntry()
	contentEntry.SetText(editQ.Content)
	contentEntry.Wrapping = fyne.TextWrapBreak

	// 动态选项区容器
	optionsContainer := container.NewVBox()

	// ==================== 问题 3 & 4: 题型逻辑 ====================
	if q.Type == "填空题" || q.Type == "问答题" {
		btnAddOption.Disable() // 不可选

		ansEntry := widget.NewMultiLineEntry()
		ansEntry.SetText(strings.Join(editQ.Answers, "、"))

		optionsContainer.Add(widget.NewLabel("答案 (多个填空用、隔开):"))
		optionsContainer.Add(ansEntry)

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			editQ.Answers = strings.Split(strings.TrimSpace(ansEntry.Text), "、")
			executeSave(w, state, editQ, onRefresh)
		}

	} else if q.Type == "判断题" {
		btnAddOption.Disable() // 不可选

		radio := widget.NewRadioGroup([]string{"A. 正确", "B. 错误"}, func(val string) {
			if strings.HasPrefix(val, "A") {
				editQ.Answers = []string{"A"}
			} else {
				editQ.Answers = []string{"B"}
			}
		})

		if len(editQ.Answers) > 0 {
			if editQ.Answers[0] == "A" {
				radio.SetSelected("A. 正确")
			} else if editQ.Answers[0] == "B" {
				radio.SetSelected("B. 错误")
			}
		}

		optionsContainer.Add(widget.NewLabel("设定正确答案:"))
		optionsContainer.Add(radio)

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			executeSave(w, state, editQ, onRefresh)
		}

	} else {
		// ==================== 问题 5 & 6: 单选题、多选题 ====================
		btnAddOption.Enable()

		var refreshOptionsUI func()

		// 动态渲染选项列表
		refreshOptionsUI = func() {
			optionsContainer.Objects = nil

			for i := range editQ.Options {
				idx := i // 捕获闭包变量
				currOpt := editQ.Options[idx]

				// 1. 选项内容输入框
				optEntry := widget.NewMultiLineEntry()
				optEntry.SetText(currOpt.Text)
				optEntry.OnChanged = func(s string) {
					editQ.Options[idx].Text = s // 实时同步输入到内存
				}

				// 2. 判定当前选项是否为正确答案
				isAns := false
				for _, a := range editQ.Answers {
					if a == currOpt.Label {
						isAns = true
						break
					}
				}

				// 3. 字母选项按钮（作为设置正确答案的开关）
				btnText := " " + currOpt.Label + " "
				if isAns {
					btnText = "[" + currOpt.Label + "]" // 视觉提示
				}

				lblBtn := widget.NewButton(btnText, func() {
					if q.Type == "单选题" {
						// 单选题：排他性设为正确答案
						editQ.Answers = []string{currOpt.Label}
					} else {
						// 多选题：切换状态
						found := false
						var newAns []string
						for _, a := range editQ.Answers {
							if a == currOpt.Label {
								found = true // 原本有，说明是要取消
							} else {
								newAns = append(newAns, a)
							}
						}
						if !found { // 原本没有，追加进去
							newAns = append(newAns, currOpt.Label)
						}
						editQ.Answers = newAns
					}
					refreshOptionsUI() // 刷新 UI 高亮状态
				})

				// 选中的答案按钮高亮显示为主题色（如蓝色）
				if isAns {
					lblBtn.Importance = widget.HighImportance
				}

				// 使用 Border 布局：左侧字母按钮，中间填充输入框
				row := container.NewBorder(nil, nil, lblBtn, nil, optEntry)
				optionsContainer.Add(container.NewPadded(row))
			}
			optionsContainer.Refresh()
		}

		// 初始加载渲染
		refreshOptionsUI()

		// 响应“增加选项”动作
		btnAddOption.OnTapped = func() {
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
			if len(editQ.Options) < len(labels) {
				newLabel := labels[len(editQ.Options)]
				editQ.Options = append(editQ.Options, Option{Label: newLabel, Text: ""})
				refreshOptionsUI()
			} else {
				dialog.ShowInformation("提示", "选项数量已达系统上限", w)
			}
		}

		// ==================== 数据清洗与保存防空逻辑 ====================
		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

			var newValidOptions []Option
			oldToNewMap := make(map[string]string) // 记录旧标签到新标签的映射
			newLabelIdx := 0
			hasEmptyCorrectAnswer := false

			// 1. 过滤空选项，并重新按序分配 ABCD 标签
			for _, opt := range editQ.Options {
				isEmpty := strings.TrimSpace(opt.Text) == ""

				// 检查这个空选项是否被勾选为了答案
				for _, a := range editQ.Answers {
					if a == opt.Label && isEmpty {
						hasEmptyCorrectAnswer = true
						break
					}
				}

				if !isEmpty && newLabelIdx < len(labels) {
					newLabel := labels[newLabelIdx]
					newValidOptions = append(newValidOptions, Option{
						Label: newLabel,
						Text:  strings.TrimSpace(opt.Text),
					})
					oldToNewMap[opt.Label] = newLabel
					newLabelIdx++
				}
			}

			if len(newValidOptions) == 0 {
				dialog.ShowInformation("无法保存", "单选/多选题至少需要保留一个有效的选项内容", w)
				return
			}
			editQ.Options = newValidOptions

			// 2. 处理正确答案的映射与回退
			var updatedAnswers []string
			for _, oldAns := range editQ.Answers {
				if newL, exists := oldToNewMap[oldAns]; exists {
					updatedAnswers = append(updatedAnswers, newL)
				}
			}

			if q.Type == "单选题" {
				// 如果选择的答案由于是空行被删除了，或者根本没选答案，还原为原始答案
				if hasEmptyCorrectAnswer || len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			} else if q.Type == "多选题" {
				// 如果多选题剔除无效选项后，完全没有正确答案了，还原为原始答案
				if len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			}

			// 3. 终极越界保护
			// （极端情况：原答案包含D，但由于编辑把CD删了只剩两项，回退原答案会导致系统崩溃。必须将超出范围的答案抹除）
			validFinalLabels := make(map[string]bool)
			for _, vo := range editQ.Options {
				validFinalLabels[vo.Label] = true
			}

			var absoluteFinalAns []string
			for _, a := range editQ.Answers {
				if validFinalLabels[a] {
					absoluteFinalAns = append(absoluteFinalAns, a)
				}
			}

			// 如果由于各种极端的删除操作导致绝对没有合法答案，强制选A保底
			if len(absoluteFinalAns) == 0 {
				absoluteFinalAns = []string{"A"}
			}
			editQ.Answers = absoluteFinalAns

			// 4. 发送给持久化执行器
			executeSave(w, state, editQ, onRefresh)
		}
	}

	// ==================== 删除逻辑 ====================
	btnDelete.OnTapped = func() {
		dialog.ShowConfirm("警告", "确定要永久删除这道题目吗？", func(confirm bool) {
			if confirm {
				executeDelete(w, state, q.ID, onRefresh)
			}
		}, w)
	}

	// 拼装卡片并返回
	mainContent := container.NewVBox(
		widget.NewLabelWithStyle("题干内容:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		contentEntry,
		widget.NewSeparator(),
		optionsContainer,
		widget.NewSeparator(),
		buttonPanel,
	)

	return mainContent // ✨ 直接返回核心容器，交给外层统一套滚动条
}

// ==================== 保存与删除执行器 ====================
func executeSave(w fyne.Window, state *AppState, editQ Question, onRefresh func()) {
	// 更新全局 Questions 切片
	for i, origQ := range state.Questions {
		if origQ.ID == editQ.ID {
			state.Questions[i] = editQ
			break
		}
	}

	// 更新当前浏览的列表
	state.CurrentList[state.Index] = editQ

	if err := syncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("保存失败: %v", err), w)
		return
	}
	dialog.ShowInformation("成功", "本题修改已生效并保存至题库", w)
	onRefresh()
}

func executeDelete(w fyne.Window, state *AppState, targetID string, onRefresh func()) {
	// 从全局列表剔除
	var newQuestions []Question
	for _, q := range state.Questions {
		if q.ID != targetID {
			newQuestions = append(newQuestions, q)
		}
	}
	state.Questions = newQuestions

	// 从当前列表剔除
	var newList []Question
	for _, q := range state.CurrentList {
		if q.ID != targetID {
			newList = append(newList, q)
		}
	}
	state.CurrentList = newList

	// 游标防越界
	if state.Index >= len(state.CurrentList) {
		state.Index = len(state.CurrentList) - 1
		if state.Index < 0 {
			state.Index = 0
		}
	}

	if err := syncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("删除失败: %v", err), w)
		return
	}
	dialog.ShowInformation("成功", "本题已从题库中永久删除", w)
	onRefresh()
}
