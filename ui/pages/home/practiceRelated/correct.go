package practiceRelated

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

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/storageRelated"

	"github.com/gzjjjfree/practice/parser"
)

// SyncStateToDisk 将内存中的 Question 对象转换回 BankData，序列化为 JSON，并写入本地存储。
//
// 功能说明：
// 该函数负责将 AppState 中当前加载的题目数据（state.Questions）转换为 parser.BankData 结构，
// 并将其序列化为 JSON 格式后保存到本地存储目录中。保存的文件名基于 state.CurrentStorageKey。
//
// 参数说明：
//   - state: *core.AppState，包含当前题库文件名、存储键值以及题目列表等全局状态信息。
//
// 返回值：
//   - err: error，如果在序列化或写入文件过程中发生错误，则返回相应的错误信息；否则返回 nil。
//
// 核心业务逻辑：
//  1. 使用 defer 和 recover 机制捕获可能的 panic 并转换为错误返回。
//  2. 构建 parser.BankData 对象，设置 DisplayName 和 StorageKey，并为 Questions 切片分配容量。
//  3. 遍历 state.Questions，将每个 core.Question 转换为 parser.QuestionItem：
//     - 将题目 ID 转换为整数。
//     - 将选项列表转换为 map[string]string 格式（键为选项标签，值为选项文本）。
//     - 根据题目类型（填空题或问答题）使用 "、" 连接答案，否则直接拼接答案字符串。
//  4. 将 BankData 对象序列化为 JSON 数据。
//  5. 获取存储目录路径，并构造完整的保存文件路径（{StorageKey}.json）。
//  6. 使用 os.WriteFile 将 JSON 数据写入本地文件，权限设置为 0644。
func SyncStateToDisk(state *core.AppState) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("syncStateToDisk panic: %v", r)
			fmt.Printf("[ERROR] sync state to disk crashed: %v\n", err)
		}
	}()

	bankData := &parser.BankData{
		DisplayName: state.CurrentFileName,
		StorageKey:  state.CurrentStorageKey,
		Questions:   make([]parser.QuestionItem, len(state.Questions)),
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

		bankData.Questions[i] = parser.QuestionItem{
			ID:         idInt,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     ansStr,
			Options:    optMap,
			RawIndex:   i + 1,
			Difficulty: q.Difficulty,
		}
	}

	jsonData, err := json.Marshal(bankData)
	if err != nil {
		return err
	}

	storageDir := storageRelated.GetStorageDir()
	absoluteSavePath := filepath.Join(storageDir, state.CurrentStorageKey+".json")
	return os.WriteFile(absoluteSavePath, jsonData, 0644)
}

// CloneQuestionForEdit 深度克隆一道题目，以避免在编辑过程中修改全局状态。
//
// 功能说明：
// 该函数用于创建 core.Question 对象的深拷贝，确保在编辑题目时不会影响到原始的全局状态（如 state.Questions）。
//
// 参数说明：
//   - q: core.Question，需要被克隆的原始题目对象。
//
// 返回值：
//   - core.Question，返回一个全新的、包含相同数据的题目对象副本。
//
// 核心业务逻辑：
//  1. 创建一个新的 core.Question 对象 newQ，复制其 ID、Type、Content 和 Difficulty 字段。
//  2. 深拷贝 Options 切片：如果原始题目的 Options 不为 nil，则分配新切片并逐个复制 core.Option 对象的 Label 和 Text 字段。
//  3. 深拷贝 Answers 切片：如果原始题目的 Answers 不为 nil，则分配新切片并使用 copy 函数复制字符串数组。
//  4. 返回克隆后的题目对象 newQ。
func CloneQuestionForEdit(q core.Question) core.Question {
	newQ := core.Question{
		ID:         q.ID,
		Type:       q.Type,
		Content:    q.Content,
		Difficulty: q.Difficulty,
	}

	// Deep copy Options slice
	if q.Options != nil {
		newQ.Options = make([]core.Option, len(q.Options))
		for i, opt := range q.Options {
			newQ.Options[i] = core.Option{
				Label: opt.Label,
				Text:  opt.Text,
			}
		}
	}

	// Deep copy Answers slice
	if q.Answers != nil {
		newQ.Answers = make([]string, len(q.Answers))
		copy(newQ.Answers, q.Answers)
	}

	return newQ
}

// RenderCorrectionCard 构建单个题目的完整编辑 UI。
//
// 功能说明：
// 该函数负责为给定的题目创建并返回一个完整的编辑界面（fyne.CanvasObject），包括题干内容输入框、选项/答案编辑区域以及操作按钮面板。
//
// 参数说明：
//   - w: fyne.Window，当前 Fyne 窗口对象，用于显示对话框等交互。
//   - state: *core.AppState，包含当前题库文件名、存储键值以及题目列表等全局状态信息。
//   - q: core.Question，需要被编辑的原始题目对象。
//   - onRefresh: func()，保存或删除操作完成后的刷新回调函数。
//
// 返回值：
//   - fyne.CanvasObject，返回构建好的编辑界面 UI 组件。
//
// 核心业务逻辑：
//  1. 克隆原始题目对象 editQ，避免直接修改全局状态。
//  2. 创建四个操作按钮：btnAddOption（增加选项）、btnAddQuestion（新增题目）、btnSave（保存本题）、btnDelete（删除本题）。
//  3. 将按钮排列成两行并放入 buttonPanel 中。
//  4. 创建题干内容输入框 contentEntry，并设置其文本为 editQ.Content。
//  5. 根据题目类型（填空题/问答题、判断题、单选/多选题）动态构建选项/答案编辑区域 optionsContainer，并为 btnSave 绑定相应的保存逻辑。
//  6. 为 btnDelete 绑定删除确认对话框及执行删除的逻辑。
//  7. 将题干内容、选项/答案区域和按钮面板组合成 mainContent 并返回。
func RenderCorrectionCard(w fyne.Window, state *core.AppState, q core.Question, onRefresh func()) fyne.CanvasObject {
	editQ := CloneQuestionForEdit(q)

	// btnAddOption：增加选项按钮。用于在单选/多选题中动态添加新的选项（如 E, F, G 等），最多支持到 L 选项。
	btnAddOption := customElements.CreateButton(
		"增加选项", 90, 35,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.ColorWarningBg),
		core.HexColor(core.ColorWarningBg),
		1.5, 18,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		nil,
	)

	// btnAddQuestion：新增题目按钮。点击后弹出提示信息，准备进入新增题目的流程。
	btnAddQuestion := customElements.CreateButton(
		"新增题目", 90, 35,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.ColorCorrectBorder),
		core.HexColor(core.ColorCorrectBorder),
		1.5, 18,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			showAddQuestionDialog(w, state, onRefresh)
		})

	// btnSave：保存本题按钮。用于将当前编辑的题目内容、选项和答案保存到内存状态及本地存储中。
	btnSave := customElements.CreateButton(
		"保存本题", 90, 35,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1.5, 18,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		nil,
	)

	// btnDelete：删除本题按钮。点击后弹出确认对话框，确认后永久删除当前题目并更新本地存储。
	btnDelete := customElements.CreateButton(
		"删除本题", 90, 35,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.ColorWrongBorder),
		core.HexColor(core.ColorWrongBorder),
		1.5, 18,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		nil,
	)

	// 4. 将这 4 个按钮组合成一个整齐的面板（例如 2x2 网格或左右 HBox），并用 container.NewCenter 包裹实现整体居中
	btnRow1 := container.NewHBox(btnAddOption, container.NewGridWrap(fyne.NewSize(15, 1)), btnAddQuestion)
	btnRow2 := container.NewHBox(btnSave, container.NewGridWrap(fyne.NewSize(15, 1)), btnDelete)

	actionPanel := container.NewVBox(
		btnRow1,
		btnRow2,
	)

	buttonPanel := container.NewCenter(actionPanel)
	//btnRow1 := container.NewGridWithColumns(2, btnAddOption, btnAddQuestion)
	//btnRow2 := container.NewGridWithColumns(2, btnSave, btnDelete)
	//buttonPanel := container.NewVBox(btnRow1, btnRow2)

	contentEntry := widget.NewMultiLineEntry()
	contentEntry.SetText(editQ.Content)
	contentEntry.Wrapping = fyne.TextWrapBreak

	contentEntry.OnChanged = func(s string) {
		editQ.Content = s
	}

	dynamicFixedContentContainer := container.New(
		&customElements.DynamicFixedSizeLayout{Entry: contentEntry},
		contentEntry,
	)

	optionsContainer := container.NewVBox()

	if q.Type == "填空题" || q.Type == "问答题" {
		btnAddOption.Disable()

		ansEntry := widget.NewMultiLineEntry()
		ansEntry.SetText(strings.Join(editQ.Answers, "、"))
		ansEntry.Wrapping = fyne.TextWrapBreak // 开启自动换行

		// 根据题型设置初始行数：填空题1行，问答题3行
		initialLines := 1
		if q.Type == "问答题" {
			initialLines = 3
		}

		dynamicFixedAnsContainer := container.New(
			&customElements.DynamicFixedSizeLayout{Entry: ansEntry, InitialLines: initialLines},
			ansEntry,
		)

		optionsContainer.Add(widget.NewLabel("答案 (多个填空用、隔开):"))
		optionsContainer.Add(dynamicFixedAnsContainer)

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			editQ.Answers = strings.Split(strings.TrimSpace(ansEntry.Text), "、")
			ExecuteSave(w, state, editQ, onRefresh)
		}

	} else if q.Type == "判断题" {
		btnAddOption.Disable()

		radio := widget.NewRadioGroup([]string{"A. 正确", "B. 错误"}, func(val string) {
			if val == "" {
				// 捕获取消选择的动作，清空答案数组
				editQ.Answers = []string{}
			} else if strings.HasPrefix(val, "A") {
				editQ.Answers = []string{"A"}
			} else if strings.HasPrefix(val, "B") {
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
			ExecuteSave(w, state, editQ, onRefresh)
		}

	} else {
		btnAddOption.Enable()

		var refreshOptionsUI func()

		refreshOptionsUI = func() {
			optionsContainer.Objects = nil

			for i := range editQ.Options {
				idx := i
				currOpt := editQ.Options[idx]

				optEntry := widget.NewMultiLineEntry()
				optEntry.SetText(currOpt.Text)
				optEntry.Wrapping = fyne.TextWrapBreak // 开启自动换行

				optEntry.OnChanged = func(s string) {
					editQ.Options[idx].Text = s
				}

				// 👈 使用 dynamicFixedSizeLayout 包裹，根据实际行数动态调整高度
				dynamicFixedOptContainer := container.New(
					&customElements.DynamicFixedSizeLayout{Entry: optEntry},
					optEntry,
				)

				isAns := false
				for _, a := range editQ.Answers {
					if a == currOpt.Label {
						isAns = true
						break
					}
				}

				btnText := " " + currOpt.Label + " "
				if isAns {
					btnText = "[" + currOpt.Label + "]"
				}

				lblBtn := widget.NewButton(btnText, func() {
					if q.Type == "单选题" {
						editQ.Answers = []string{currOpt.Label}
					} else {
						found := false
						var newAns []string
						for _, a := range editQ.Answers {
							if a == currOpt.Label {
								found = true
							} else {
								newAns = append(newAns, a)
							}
						}
						if !found {
							newAns = append(newAns, currOpt.Label)
						}
						editQ.Answers = newAns
					}
					refreshOptionsUI()
				})

				if isAns {
					lblBtn.Importance = widget.HighImportance
				}

				// 👈 将原本的 optEntry 替换为动态高度容器 dynamicFixedOptContainer
				row := container.NewBorder(nil, nil, lblBtn, nil, dynamicFixedOptContainer)
				optionsContainer.Add(container.NewPadded(row))
			}
			optionsContainer.Refresh()
		}

		refreshOptionsUI()

		btnAddOption.OnTapped = func() {
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
			if len(editQ.Options) < len(labels) {
				newLabel := labels[len(editQ.Options)]
				editQ.Options = append(editQ.Options, core.Option{Label: newLabel, Text: ""})
				refreshOptionsUI()
			} else {
				customElements.ShowCustomInformation("提示", "选项数量已达系统上限", w)
			}
		}

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

			var newValidOptions []core.Option
			oldToNewMap := make(map[string]string)
			newLabelIdx := 0
			hasEmptyCorrectAnswer := false

			for _, opt := range editQ.Options {
				isEmpty := strings.TrimSpace(opt.Text) == ""

				for _, a := range editQ.Answers {
					if a == opt.Label && isEmpty {
						hasEmptyCorrectAnswer = true
						break
					}
				}

				if !isEmpty && newLabelIdx < len(labels) {
					newLabel := labels[newLabelIdx]
					newValidOptions = append(newValidOptions, core.Option{
						Label: newLabel,
						Text:  strings.TrimSpace(opt.Text),
					})
					oldToNewMap[opt.Label] = newLabel
					newLabelIdx++
				}
			}

			if len(newValidOptions) == 0 {
				customElements.ShowCustomInformation("无法保存", "单选/多选题至少需要保留一个有效的选项内容", w)
				return
			}
			editQ.Options = newValidOptions

			var updatedAnswers []string
			for _, oldAns := range editQ.Answers {
				if newL, exists := oldToNewMap[oldAns]; exists {
					updatedAnswers = append(updatedAnswers, newL)
				}
			}

			if q.Type == "单选题" {
				if hasEmptyCorrectAnswer || len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			} else if q.Type == "多选题" {
				if len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			}

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

			if len(absoluteFinalAns) == 0 {
				absoluteFinalAns = []string{"A"}
			}
			editQ.Answers = absoluteFinalAns

			ExecuteSave(w, state, editQ, onRefresh)
		}
	}

	btnDelete.OnTapped = func() {
		customElements.ShowCustomConfirm(
			"警告",
			"确定",
			"取消",
			customElements.NewCenterRichText("确定要永久删除这道题目吗？"),
			func(confirm bool) bool {
				if confirm {
					ExecuteDelete(w, state, q.ID, onRefresh)
				}
				return true
			}, w)
	}

	// 1. 将上方需要滚动或展示的主体内容放入一个 VBox 中
	topContent := container.NewVBox(
		widget.NewLabelWithStyle("题干内容:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dynamicFixedContentContainer,
		widget.NewSeparator(),
		optionsContainer,
		widget.NewSeparator(),
	)

	//scrollableContent := container.NewVScroll(topContent)

	mainContent := container.NewBorder(
		topContent,  // top
		buttonPanel, // bottom: 固定在底部
		nil,         // left
		nil,         // right
		nil,         // center: 中间主体内容区域，会自动填满剩余空间
	)

	return mainContent
}

// ExecuteSave 更新 state.Questions 和 state.CurrentList 中的题目，并调用 SyncStateToDisk 保存到本地存储。
//
// 功能说明：
// 该函数用于将编辑后的题目数据同步到 AppState 的全局状态中（state.Questions 和 state.CurrentList），
// 然后调用 SyncStateToDisk 将修改持久化到本地 JSON 文件中。如果保存失败，则显示错误对话框；否则显示成功提示并触发 UI 刷新。
//
// 参数说明：
//   - w: fyne.Window，当前 Fyne 窗口对象，用于显示错误对话框或成功提示信息。
//   - state: *core.AppState，包含当前题库文件名、存储键值以及题目列表等全局状态信息。
//   - editQ: core.Question，编辑后的题目对象。
//   - onRefresh: func()，保存操作完成后的刷新回调函数。
//
// 核心业务逻辑：
//  1. 遍历 state.Questions，找到与 editQ.ID 相同的原始题目并替换为 editQ。
//  2. 更新 state.CurrentList 中当前索引位置（state.Index）的题目为 editQ。
//  3. 调用 SyncStateToDisk(state) 将修改同步到本地存储。
//  4. 如果发生错误，则通过 dialog.ShowError 显示保存失败的错误信息并返回。
//  5. 如果成功，则通过 customElements.ShowCustomInformation 显示成功提示，并调用 onRefresh() 刷新 UI。
func ExecuteSave(w fyne.Window, state *core.AppState, editQ core.Question, onRefresh func()) {
	for i, origQ := range state.Questions {
		if origQ.ID == editQ.ID {
			state.Questions[i] = editQ
			break
		}
	}

	state.CurrentList[state.Index] = editQ

	if err := SyncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("保存失败: %v", err), w)
		return
	}
	customElements.ShowCustomInformation("成功", "本题修改已生效并保存至题库", w)
	onRefresh()
}

// ExecuteDelete 从 state.Questions 和 state.CurrentList 中移除指定的题目，并调用 SyncStateToDisk 更新本地存储。
//
// 功能说明：
// 该函数用于永久删除指定 ID 的题目，从 AppState 的全局状态（state.Questions 和 state.CurrentList）中过滤掉该题目，
// 然后调用 SyncStateToDisk 将修改持久化到本地 JSON 文件中。如果删除后当前索引超出范围，则调整索引值。如果保存失败，则显示错误对话框；否则显示成功提示并触发 UI 刷新。
//
// 参数说明：
//   - w: fyne.Window，当前 Fyne 窗口对象，用于显示错误对话框或成功提示信息。
//   - state: *core.AppState，包含当前题库文件名、存储键值以及题目列表等全局状态信息。
//   - targetID: string，需要被删除的题目的 ID。
//   - onRefresh: func()，删除操作完成后的刷新回调函数。
//
// 核心业务逻辑：
//  1. 遍历 state.Questions，将所有 ID 不等于 targetID 的题目收集到 newQuestions 切片中，并更新 state.Questions。
//  2. 遍历 state.CurrentList，将所有 ID 不等于 targetID 的题目收集到 newList 切片中，并更新 state.CurrentList。
//  3. 检查 state.Index 是否超出新的 state.CurrentList 长度范围，如果是，则将其调整为最后一个有效索引（或 0）。
//  4. 调用 SyncStateToDisk(state) 将修改同步到本地存储。
//  5. 如果发生错误，则通过 dialog.ShowError 显示删除失败的错误信息并返回。
//  6. 如果成功，则通过 customElements.ShowCustomInformation 显示成功提示，并调用 onRefresh() 刷新 UI。
func ExecuteDelete(w fyne.Window, state *core.AppState, targetID string, onRefresh func()) {
	var newQuestions []core.Question
	for _, q := range state.Questions {
		if q.ID != targetID {
			newQuestions = append(newQuestions, q)
		}
	}
	state.Questions = newQuestions

	var newList []core.Question
	for _, q := range state.CurrentList {
		if q.ID != targetID {
			newList = append(newList, q)
		}
	}
	state.CurrentList = newList

	if state.Index >= len(state.CurrentList) {
		state.Index = len(state.CurrentList) - 1
		if state.Index < 0 {
			state.Index = 0
		}
	}

	if err := SyncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("删除失败: %v", err), w)
		return
	}
	customElements.ShowCustomInformation("成功", "本题已从题库中永久删除", w)
	onRefresh()
}

// showAddQuestionDialog 显示新增题目对话框（简化版：直接创建完整题目）
func showAddQuestionDialog(w fyne.Window, state *core.AppState, onRefresh func()) {
	// 临时存储新增题目的数据
	type addQuestionData struct {
		selectedType string
		content      string
		options      []core.Option
		answers      []string
	}

	qData := &addQuestionData{
		selectedType: "单选题",
		content:      "",
		options:      nil,
		answers:      nil,
	}

	// 创建题型选择下拉框
	typeOptions := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	typeRadio := widget.NewRadioGroup(typeOptions, func(val string) {
		qData.selectedType = val
	})

	// 设置默认选中的题型为"单选题"
	if len(typeOptions) > 0 {
		typeRadio.SetSelected("单选题")
	}

	// 创建题干内容输入框
	contentEntry := widget.NewMultiLineEntry()
	contentEntry.Wrapping = fyne.TextWrapBreak

	// 使用 dynamicFixedSizeLayout 包裹，根据实际行数动态调整高度
	dynamicFixedContentContainer := container.New(
		&customElements.DynamicFixedSizeLayout{Entry: contentEntry},
		contentEntry,
	)

	// 【新增】提前声明外部的主对话框容器，以便能够在输入变化时触发整体重新布局
	var dialogContent *fyne.Container

	contentEntry.OnChanged = func(s string) {
		qData.content = s
		// 内容改变时刷新局部容器
		dynamicFixedContentContainer.Refresh()
		// 【新增】同步刷新外部主容器，促使整个对话框向下推移，扩大输入框实际展示高度
		if dialogContent != nil {
			dialogContent.Refresh()
		}
	}

	// 创建选项/答案配置容器
	optionsConfigContainer := container.NewVBox()

	// 定义更新选项UI的函数
	var updateOptionsUI func()

	// 根据题型动态更新选项配置UI
	updateOptionsUI = func() {
		optionsConfigContainer.Objects = nil

		if qData.selectedType == "单选题" || qData.selectedType == "多选题" {
			optionsConfigContainer.Add(widget.NewLabelWithStyle("选项配置:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			// 初始化选项（如果还没有）
			if qData.options == nil {
				qData.options = []core.Option{
					{Label: "A", Text: ""},
					{Label: "B", Text: ""},
					{Label: "C", Text: ""},
					{Label: "D", Text: ""},
				}
				qData.answers = []string{}
			}

			// 添加/删除选项按钮面板
			btnAddOpt := customElements.CreateButton(
				"添加选项", 80, 30,
				core.HexColor(core.TextPrimaryColor),
				core.HexColor(core.ColorCorrectBorder),
				core.HexColor(core.ColorCorrectBorder),
				1.5, 16,
				true, false,
				fyne.TextAlignCenter,
				fyne.TextWrapOff,
				fyne.TextTruncateOff,
				nil,
			)

			btnAddOpt.OnTapped = func() {
				labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
				if len(qData.options) < len(labels) {
					newLabel := labels[len(qData.options)]
					qData.options = append(qData.options, core.Option{Label: newLabel, Text: ""})
					updateOptionsUI()
				} else {
					customElements.ShowCustomInformation("提示", "选项数量已达系统上限", w)
				}
			}

			btnPanel := container.NewHBox(btnAddOpt)
			optionsConfigContainer.Add(container.NewPadded(btnPanel))

			for i := range qData.options {
				idx := i
				currOpt := qData.options[idx]

				isAns := false
				for _, a := range qData.answers {
					if a == currOpt.Label {
						isAns = true
						break
					}
				}

				optEntry := widget.NewMultiLineEntry()
				optEntry.SetText(currOpt.Text)
				optEntry.Wrapping = fyne.TextWrapBreak
				optEntry.OnChanged = func(s string) {
					qData.options[idx].Text = s
				}

				// 👈 使用 dynamicFixedSizeLayout 包裹，根据实际行数动态调整高度
				dynamicFixedOptContainer := container.New(
					&customElements.DynamicFixedSizeLayout{Entry: optEntry},
					optEntry,
				)

				btnText := " " + currOpt.Label + " "
				if isAns {
					btnText = "[" + currOpt.Label + "]"
				}

				lblBtn := widget.NewButton(btnText, func() {
					if qData.selectedType == "单选题" {
						qData.answers = []string{currOpt.Label}
					} else {
						found := false
						var newAns []string
						for _, a := range qData.answers {
							if a == currOpt.Label {
								found = true
							} else {
								newAns = append(newAns, a)
							}
						}
						if !found {
							newAns = append(newAns, currOpt.Label)
						}
						qData.answers = newAns
					}
					updateOptionsUI()
				})

				if isAns {
					lblBtn.Importance = widget.HighImportance
				}

				btnDelOpt := customElements.CreateButton(
					"删除", 50, 30,
					core.HexColor(core.TextPrimaryColor),
					core.HexColor(core.ColorWrongBorder),
					core.HexColor(core.ColorWrongBorder),
					1.5, 16,
					true, false,
					fyne.TextAlignCenter,
					fyne.TextWrapOff,
					fyne.TextTruncateOff,
					nil,
				)

				btnDelOpt.OnTapped = func() {
					if len(qData.options) > 2 {
						qData.options = append(qData.options[:idx], qData.options[idx+1:]...)
						// 清理答案中不存在的选项标签
						var newAns []string
						for _, a := range qData.answers {
							found := false
							for _, opt := range qData.options {
								if a == opt.Label {
									found = true
									break
								}
							}
							if found {
								newAns = append(newAns, a)
							}
						}
						qData.answers = newAns
						updateOptionsUI()
					} else {
						customElements.ShowCustomInformation("提示", "单选题/多选题至少需要保留2个选项", w)
					}
				}

				row := container.NewBorder(nil, nil, lblBtn, btnDelOpt, dynamicFixedOptContainer)
				optionsConfigContainer.Add(container.NewPadded(row))
			}

		} else if qData.selectedType == "判断题" {
			optionsConfigContainer.Add(widget.NewLabelWithStyle("设定正确答案:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			if qData.options == nil {
				qData.options = []core.Option{
					{Label: "A", Text: "正确"},
					{Label: "B", Text: "错误"},
				}
				qData.answers = []string{}
			}

			radio := widget.NewRadioGroup([]string{"A. 正确", "B. 错误"}, func(val string) {
				if val == "" {
					// 捕获取消选择的动作，清空答案数组
					qData.answers = []string{}
				} else if strings.HasPrefix(val, "A") {
					qData.answers = []string{"A"}
				} else if strings.HasPrefix(val, "B") {
					qData.answers = []string{"B"}
				}
			})

			if len(qData.answers) > 0 {
				if qData.answers[0] == "A" {
					radio.SetSelected("A. 正确")
				} else if qData.answers[0] == "B" {
					radio.SetSelected("B. 错误")
				}
			}

			optionsConfigContainer.Add(radio)

		} else if qData.selectedType == "填空题" || qData.selectedType == "问答题" {
			optionsConfigContainer.Add(widget.NewLabelWithStyle("答案 (多个填空用、隔开):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

			if qData.answers == nil {
				qData.answers = []string{}
			}

			ansEntry := widget.NewMultiLineEntry()
			if len(qData.answers) > 0 {
				ansEntry.SetText(strings.Join(qData.answers, "、"))
			}
			ansEntry.Wrapping = fyne.TextWrapBreak
			ansEntry.OnChanged = func(s string) {
				qData.answers = strings.Split(strings.TrimSpace(s), "、")
			}

			// 根据题型设置初始行数：填空题1行，问答题3行
			initialLines := 1
			if qData.selectedType == "问答题" {
				initialLines = 3
			}

			dynamicFixedAnsContainer := container.New(
				&customElements.DynamicFixedSizeLayout{Entry: ansEntry, InitialLines: initialLines},
				ansEntry,
			)

			optionsConfigContainer.Add(dynamicFixedAnsContainer)
		}

		optionsConfigContainer.Refresh()
	}

	// 监听题型变化，更新选项UI
	typeRadio.OnChanged = func(val string) {
		qData.selectedType = val
		qData.options = nil // 重置选项
		qData.answers = nil // 重置答案
		updateOptionsUI()
	}

	// 初始化选项UI
	updateOptionsUI()

	// 【修改】构建对话框内容（注意这里是赋值给上面提前声明好的 dialogContent 变量，去掉 := 里的冒号）
	dialogContent = container.NewVBox(
		widget.NewLabelWithStyle("题型:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewPadded(typeRadio),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("题干内容:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		dynamicFixedContentContainer,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("选项/答案配置:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		optionsConfigContainer,
	)

	// 验证新增题目数据的函数
	validateAddQuestion := func() (bool, string) {
		qData.content = strings.TrimSpace(contentEntry.Text)

		if qData.content == "" {
			return false, "题干内容不能为空"
		}

		// 根据题型验证和设置答案
		if qData.selectedType == "单选题" || qData.selectedType == "多选题" {
			// 检查是否有空选项
			hasEmptyOpt := false
			for _, opt := range qData.options {
				if strings.TrimSpace(opt.Text) == "" {
					hasEmptyOpt = true
					break
				}
			}
			if hasEmptyOpt {
				return false, "所有选项内容不能为空"
			}

			if len(qData.answers) == 0 {
				return false, "请至少选择一个正确答案"
			}
		} else if qData.selectedType == "判断题" {
			if len(qData.answers) == 0 {
				return false, "请选择正确答案"
			}
		} else if qData.selectedType == "填空题" || qData.selectedType == "问答题" {
			if len(qData.answers) == 0 || (len(qData.answers) == 1 && qData.answers[0] == "") {
				return false, "请输入答案"
			}
		}

		return true, ""
	}

	// 保存新题目的逻辑
	saveNewQuestion := func() bool {
		// 创建新题目
		newQ := core.Question{
			ID:      generateNewQuestionID(state),
			Type:    qData.selectedType,
			Content: qData.content,
			Options: qData.options,
			Answers: qData.answers,
		}

		// 将新题目添加到状态中并保存
		found := false
		for i, origQ := range state.Questions {
			if origQ.ID == newQ.ID {
				// 如果ID已存在，则更新（理论上不应该发生）
				state.Questions[i] = newQ
				found = true
				break
			}
		}

		if !found {
			state.Questions = append(state.Questions, newQ)
		}

		// 更新CurrentList
		state.CurrentList = state.Questions

		// 保存到本地存储
		if err := SyncStateToDisk(state); err != nil {
			customElements.ShowCustomInformation("错误", fmt.Sprintf("保存失败: %v", err), w)
			return false // 保存失败，保持对话框打开
		}

		customElements.ShowCustomInformation("成功", "新题目已添加并保存至题库", w)
		onRefresh()
		return true // 保存成功，关闭对话框
	}

	// 显示对话框（由ShowCustomConfirm统一创建"保存新增"和"取消"按钮）
	customElements.ShowCustomConfirm("新增题目", "保存新增", "取消", dialogContent, func(confirm bool) bool {
		if confirm {
			// 先验证数据
			valid, errMsg := validateAddQuestion()
			if !valid {
				customElements.ShowCustomInformation("提示", errMsg, w)
				return false // 验证失败，不关闭对话框，让用户继续编辑
			}
			// 验证通过，执行保存
			return saveNewQuestion()
		}
		return true // 取消操作，关闭对话框
	}, w)
}

// generateNewQuestionID 生成新的题目ID
func generateNewQuestionID(state *core.AppState) string {
	if len(state.Questions) == 0 {
		return "1"
	}

	// 找到最大的ID
	maxID := 0
	for _, q := range state.Questions {
		id, err := strconv.Atoi(q.ID)
		if err == nil && id > maxID {
			maxID = id
		}
	}

	return strconv.Itoa(maxID + 1)
}
