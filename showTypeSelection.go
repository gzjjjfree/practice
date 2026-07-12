package main

import (
	"fmt"
	"math/rand"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ==================== 题型选择中间页 (带模式标签版) ====================
func showTypeSelection(w fyne.Window, state *AppState, practiceMode string) {
	// practiceMode 传入 "顺序练习" 或 "随机练习"

	// 1. 顶部导航栏组件
	//backBtn := widget.NewButton("← 返回", func() {
	//	showHome(w, state)
	//})
	//backBtnText := canvas.NewText("返回", hexColor("#333333"))
	//backBtnText.TextSize = 18
	//backBtnText.TextStyle = fyne.TextStyle{Bold: true}
	//
	//backBtnBg := canvas.NewRectangle(hexColor("#f0f0f0"))
	//backBtnBg.CornerRadius = 4
	//backBtnBg.StrokeColor = hexColor("#d9d9d9")
	//backBtnBg.StrokeWidth = 1
	//
	//// ✨ 修正点：使用 container.NewGridWrap 强行固定包裹框的尺寸
	//// 这样文字居中后，四周就会被稳稳地撑开自定义的 Padding 呼吸感
	//backBtnCustomLayout := container.NewStack(
	//	backBtnBg,
	//	container.NewGridWrap(fyne.NewSize(60, 30), container.NewCenter(backBtnText)),
	//)
	//
	//backBtn := NewClickableBox(backBtnCustomLayout, func() {
	//	showHome(w, state)
	//})

	backBtn := getBox("← 返回", 90, 35, &boxColor{textColor: "#000000", bgColor: "#F7F7F7", strokeColor: "#F7F7F7", textSize: 18}, false, func() { showHome(w, state) })

	// ✨ 核心修复 1：放弃 widget.NewLabel，改用 canvas.NewText
	// 这样可以手动锁死字体颜色（如深灰色 #333333），确保在 WSL 暗色端和手机端都能清晰可见，不会隐形
	modeText := canvas.NewText(practiceMode, hexColor("#333333"))
	modeText.TextSize = 18
	modeText.TextStyle = fyne.TextStyle{Bold: true}
	modeText.Alignment = fyne.TextAlignTrailing // 文字靠右对齐

	// 为了给靠右的文字四周留有一点点舒适的内边距，用 Padded 包裹一下
	rightBox := container.NewPadded(modeText)

	// ✨ 核心修复 2：使用 NewBorder 布局
	// 参数顺序：Top, Bottom, Left, Right, Center
	// 把返回按钮卡在左边（Left），模式文本卡在右边（Right），中间（Center）塞一个隐形占位符
	topBar := container.NewBorder(nil, nil, backBtn, rightBox, widget.NewLabel(""))

	// 2. 当前题库名称显示区
	//bankNameText := canvas.NewText("📖 "+state.CurrentFileName, hexColor("#333333"))
	//bankNameText.TextSize = 16
	//bankNameText.Alignment = fyne.TextAlignCenter
	bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
	bankNameText.Alignment = fyne.TextAlignCenter // 🎯 保证文字在两行时也是居中的
	bankNameText.Wrapping = fyne.TextWrapBreak    // 允许换行
	bankNameText.SizeName = theme.SizeNameWindowButtonHeight

	bankNameBg := canvas.NewRectangle(hexColor("#ffffff"))
	bankNameBg.CornerRadius = 6
	bankNameBg.StrokeColor = hexColor("#dddddd")
	bankNameBg.StrokeWidth = 1

	bankNameCard := container.NewStack(
		bankNameBg,
		container.NewPadded(bankNameText),
	)

	// 3. 统计当前题库中真实存在哪些题型
	typeSet := make(map[string]bool)
	for _, q := range state.Questions {
		if q.Type != "" {
			typeSet[q.Type] = true
		}
	}

	// 定义全局固定的标准题型顺序
	standardOrder := []string{"填空题", "单选题", "多选题", "判断题", "问答题"}

	// 始终将 "全部题型" 放在第一位
	availableTypes := []string{"全部题型"}

	// 严格按照标准顺序检查
	for _, stdType := range standardOrder {
		if typeSet[stdType] {
			availableTypes = append(availableTypes, stdType)
		}
	}

	// 防御性检查
	for t := range typeSet {
		isStandard := false
		for _, stdType := range standardOrder {
			if t == stdType {
				isStandard = true
				break
			}
		}
		if !isStandard {
			availableTypes = append(availableTypes, t)
		}
	}

	// 4. 渲染题型列表
	listVBox := container.NewVBox()

	for i, t := range availableTypes {
		typeStr := t

		rowBg := canvas.NewRectangle(hexColor("#ffffff"))
		rowBg.CornerRadius = 4
		rowBg.StrokeColor = hexColor("#eeeeee")
		rowBg.StrokeWidth = 1

		lblText := fmt.Sprintf("题目%d. (%s)", i+1, typeStr)
		rowLbl := canvas.NewText(lblText, hexColor("#222222"))
		rowLbl.TextSize = 18

		rowContent := container.NewBorder(
			layout.NewSpacer(), layout.NewSpacer(), nil, nil,
			container.NewPadded(rowLbl),
		)
		rowStack := container.NewStack(rowBg, rowContent)

		clickableRow := NewClickableBox(rowStack, func() {
			var tempList []Question
			tempList = make([]Question, 0)

			// 按练习模式分类
			switch practiceMode {
			case "错题练习":
				for _, q := range state.Questions {
					// ✨ 核心过滤：检查当前题目的 ID 是否存在于错题集 map 中
					if isWrong, exists := state.WrongSet[q.ID]; exists && isWrong {
						tempList = append(tempList, q)
					}
				}
			case "收藏练习":
				for _, q := range state.Questions {
					// ✨ 核心过滤：检查当前题目的 ID 是否存在于错题集 map 中
					if isFav, exists := state.FavSet[q.ID]; exists && isFav {
						tempList = append(tempList, q)
					}
				}
			default:
				tempList = state.Questions
			}

			var filteredList []Question

			// 按题型分类
			if typeStr == "全部题型" {
				filteredList = make([]Question, len(tempList))
				copy(filteredList, tempList)
			} else {
				for _, q := range tempList {
					if q.Type == typeStr {
						filteredList = append(filteredList, q)
					}
				}
			}

			if practiceMode == "随机练习" {
				rand.Shuffle(len(filteredList), func(i, j int) {
					filteredList[i], filteredList[j] = filteredList[j], filteredList[i]
				})
			}

			// 1. 保存【当前离开前】模式的进度
			if state.Title != "" {
				state.ModeIndices[state.Title] = state.Index
			}

			state.CurrentList = filteredList
			state.Title = fmt.Sprintf("%s - %s", practiceMode, typeStr)
			//state.Index = 0

			if len(state.CurrentList) == 0 {
				dialog.ShowInformation("提示", "该题型下没有题目", w)
				return
			}

			// 4. 读取【目标模式】的历史进度
			state.Index = state.ModeIndices[state.Title]

			// 5. 安全越界检查（比如错题本里有些题被移除了，导致列表变短）
			if state.Index >= len(state.CurrentList) {
				state.Index = 0
				// 列表为空时的防御
				if len(state.CurrentList) > 0 {
					state.Index = len(state.CurrentList) - 1
				}
			}

			// 初始化大管家
			if state.ModeRecords == nil {
				state.ModeRecords = make(map[string]map[string]string)
			}
			// 确保当前模式的子 Map 存在
			if state.ModeRecords[state.Title] == nil {
				state.ModeRecords[state.Title] = make(map[string]string)
			}

			// 🔥 将快捷引用直接指向当前模式的子 Map
			// 因为 Go 的 Map 是引用类型，之后所有的 state.PracticeRecords[q.ID] = "A"
			// 都会自动、实时地写进 state.ModeRecords[state.CurrentMode] 里面！
			state.PracticeRecords = state.ModeRecords[state.Title]

			showPractice(w, state, func(win fyne.Window, st *AppState) {
				showTypeSelection(win, st, practiceMode)
			})
		})

		listVBox.Add(clickableRow)
	}

	// 5. 整体布局组装
	pageBg := canvas.NewRectangle(hexColor("#f7f7f7"))

	contentBox := container.NewVBox(
		container.NewPadded(bankNameCard),
		widget.NewSeparator(),
		container.NewPadded(listVBox),
	)

	mainLayoutBox := container.NewStack(
		pageBg,
		container.NewBorder(topBar, nil, nil, nil, contentBox),
	)

	mainLayout := container.NewBorder(getTitle(), nil, nil, nil, mainLayoutBox)

	w.SetContent(mainLayout)
}
