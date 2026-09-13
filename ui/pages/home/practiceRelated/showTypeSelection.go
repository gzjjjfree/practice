package practiceRelated

import (
	"fmt"
	"image/color"
	"math/rand"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/gz_theme"
)

// ShowTypeSelection 渲染选择题型的中间页面。
//
// 功能说明：
// 用户从练习模式（顺序/随机/错题/收藏）进入此页面，选择要练习的题型（全部/单选/多选/判断/填空/问答）。
// 选择后进入练习页面开始答题。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用的窗口实例，用于设置页面内容和显示弹窗提示。
//   - state: *core.AppState 类型，表示全局应用状态，包含题目列表、错题集、收藏集、练习记录等信息。
//   - practiceMode: string 类型，表示当前的练习模式，可选值包括："顺序练习"、"随机练习"、"错题练习"、"收藏练习"等。
//   - backToHome: func() 类型，返回主页面的回调函数，用于点击返回按钮时退出当前页面并返回首页。
func ShowTypeSelection(w fyne.Window, state *core.AppState, practiceMode string, backToHome func()) {
	// 【← 返回 按钮 (backBtn)】
	backBtn := customElements.CreateButton(
		"← 返回", core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.SectionBgColor),
		core.HexColor(core.SectionBgColor),
		core.StrokeMedium, core.FontSizeButton,
		true, false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		func() {
			if backToHome != nil {
				backToHome()
			}
		})

	modeText := canvas.NewText(practiceMode, core.HexColor(core.TextSecondaryColor))
	modeText.TextSize = core.FontSizeBody
	modeText.TextStyle = fyne.TextStyle{Bold: true}
	modeText.Alignment = fyne.TextAlignTrailing

	// 创建一个10像素宽度的spacer
	rightSpacer := container.NewGridWrap(fyne.NewSize(10, 1))

	// 使用HBox将文本和spacer组合
	rightBox := container.NewHBox(modeText, rightSpacer)

	topBar := container.NewBorder(nil, nil, backBtn, rightBox, widget.NewLabel(""))

	bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
	bankNameText.Alignment = fyne.TextAlignCenter
	bankNameText.Wrapping = fyne.TextWrapBreak
	bankNameText.SizeName = theme.SizeNameWindowButtonHeight

	bankNameBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	bankNameBg.CornerRadius = core.InputCornerRadius
	bankNameBg.StrokeColor = core.HexColor(core.BorderMediumColor)
	bankNameBg.StrokeWidth = core.StrokeThin
	bankNameCard := container.NewStack(bankNameBg, container.NewPadded(bankNameText))

	typeSet := make(map[string]bool)
	for _, q := range state.Questions {
		if q.Type != "" {
			typeSet[q.Type] = true
		}
	}

	standardOrder := []string{"填空题", "单选题", "多选题", "判断题", "问答题"}
	availableTypes := []string{"全部题型"}

	for _, stdType := range standardOrder {
		if typeSet[stdType] {
			availableTypes = append(availableTypes, stdType)
		}
	}

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

	listVBox := container.NewVBox()

	for i, t := range availableTypes {
		typeStr := t
		displayText := fmt.Sprintf("🏷️ 题目%d. (%s)", i+1, typeStr)

		// 【题型选择按钮 (clickableRow)】：采用与主界面文件列表一致的尺寸与对齐样式
		clickableRow := customElements.CreateButton(
			displayText,
			-1,                      // 👈 -1 代表全宽自适应填充
			core.FileListItemHeight, // 👈 统一使用与文件列表项一样的高度
			core.HexColor(core.TextHintColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BorderLightColor),
			core.StrokeMedium,
			core.FontSizeSmall, // 👈 使用统一的小字号 FontSizeSmall
			false, false,
			fyne.TextAlignLeading, // 👈 靠左对齐
			fyne.TextWrapBreak,    // 👈 自动换行
			fyne.TextTruncateOff,  // 👈 不截断
			func() {
				var tempList []core.Question
				tempList = make([]core.Question, 0)

				switch practiceMode {
				case "错题练习":
					for _, q := range state.Questions {
						if isWrong, exists := state.WrongSet[q.ID]; exists && isWrong {
							tempList = append(tempList, q)
						}
					}
				case "收藏练习":
					for _, q := range state.Questions {
						if isFav, exists := state.FavSet[q.ID]; exists && isFav {
							tempList = append(tempList, q)
						}
					}
				default:
					tempList = state.Questions
				}

				var filteredList []core.Question

				if typeStr == "全部题型" {
					filteredList = make([]core.Question, len(tempList))
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

				if state.Title != "" {
					state.ModeIndices[state.Title] = state.Index
				}

				state.CurrentList = filteredList
				state.Title = fmt.Sprintf("%s - %s", practiceMode, typeStr)

				if len(state.CurrentList) == 0 {
					customElements.ShowCustomInformation("提示", "该题型下没有题目", w)
					return
				}

				state.Index = state.ModeIndices[state.Title]

				if state.Index >= len(state.CurrentList) {
					state.Index = 0
					if len(state.CurrentList) > 0 {
						state.Index = len(state.CurrentList) - 1
					}
				}

				if state.ModeRecords == nil {
					state.ModeRecords = make(map[string]map[string]string)
				}
				if state.ModeRecords[state.Title] == nil {
					state.ModeRecords[state.Title] = make(map[string]string)
				}
				state.PracticeRecordsMu.Lock()
				state.PracticeRecords = state.ModeRecords[state.Title]
				state.PracticeRecordsMu.Unlock()

				ShowPractice(w, state, func(win fyne.Window, st *core.AppState) {
					ShowTypeSelection(win, st, practiceMode, backToHome)
				})
			})

		// =================【控制列表项外边距，对齐文件列表】=================
		mLeft := canvas.NewRectangle(color.Transparent)
		mLeft.SetMinSize(fyne.NewSize(6, 0)) // 👈 左外边距 6px

		mRight := canvas.NewRectangle(color.Transparent)
		mRight.SetMinSize(fyne.NewSize(10, 0)) // 👈 右外边距 10px

		itemWithMargin := container.NewBorder(nil, nil, mLeft, mRight, clickableRow)
		listVBox.Add(itemWithMargin)
		// =======================================================================
	}

	// 增加滚动视图包裹，保证列表多时可以滑动，布局体验与文件列表保持一致
	scrollList := container.NewScroll(listVBox)

	customTheme := &gz_theme.NoShadowTheme{Theme: theme.DefaultTheme()}
	noShadowScrollWrapper := container.NewThemeOverride(scrollList, customTheme)

	pageBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))

	// 💡 核心改动：将 contentBox 改为 Border 布局
	// 顶部放 bankNameCard 和 分割线，Center 位置放 noShadowScrollWrapper（自动向下填充撑满）
	topHeader := container.NewVBox(
		container.NewPadded(bankNameCard),
		widget.NewSeparator(),
	)

	contentBox := container.NewBorder(
		topHeader, // Top: 放置卡片和分割线
		nil, nil, nil,
		container.NewPadded(noShadowScrollWrapper), // Center: 自动拉伸填满剩余的底部空间
	)

	mainLayoutBox := container.NewStack(
		pageBg,
		container.NewBorder(topBar, nil, nil, nil, contentBox),
	)

	mainLayout := container.NewBorder(customElements.GetTitle(), nil, nil, nil, mainLayoutBox)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, mainLayout)
	w.SetContent(wRootLayout)
}
