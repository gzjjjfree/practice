// 测试写入
package practiceRelated

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
)

// HighlightText 创建富文本，将关键词匹配部分高亮为红色/粗体。
// 用于搜索结果中突出显示关键词。
//
// 功能说明：
//   - 该函数接收一段文本和关键词，返回一个富文本对象（*widget.RichText）。
//   - 如果关键词为空，则直接返回原始文本的富文本对象；
//   - 否则，将文本中与关键词匹配的部分高亮显示为红色且加粗，其余部分保持默认样式。
//
// 参数说明：
//   - text: 原始文本内容，需要进行关键词高亮处理的字符串。
//   - keyword: 需要高亮的关键词字符串。如果为空字符串，则不进行高亮处理。
//
// 返回值：
//   - *widget.RichText: 包含高亮处理后的富文本对象，支持自动换行（fyne.TextWrapBreak）。
func HighlightText(text, keyword string) *widget.RichText {
	if keyword == "" {
		rt := widget.NewRichText(&widget.TextSegment{Style: widget.RichTextStyle{Inline: true}, Text: text})
		rt.Wrapping = fyne.TextWrapBreak
		return rt
	}

	var segments []widget.RichTextSegment
	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)
	start := 0

	for {
		idx := strings.Index(lowerText[start:], lowerKeyword)
		if idx == -1 {
			segments = append(segments, &widget.TextSegment{Style: widget.RichTextStyle{Inline: true}, Text: text[start:]})
			break
		}
		idx += start
		if idx > start {
			segments = append(segments, &widget.TextSegment{Style: widget.RichTextStyle{Inline: true}, Text: text[start:idx]})
		}
		segments = append(segments, &widget.TextSegment{
			Style: widget.RichTextStyle{Inline: true, ColorName: theme.ColorNameError, TextStyle: fyne.TextStyle{Bold: true}},
			Text:  text[idx : idx+len(keyword)],
		})
		start = idx + len(keyword)
	}

	rt := widget.NewRichText(segments...)
	rt.Wrapping = fyne.TextWrapBreak
	return rt
}

// ShowSearchPage 构建带有关键词高亮和题型过滤的搜索 UI。
// 支持在题库中搜索题目，关键词在题干和选项中以红色粗体高亮显示。
// 支持按题型过滤搜索结果。
//
// 功能说明：
//   - 该函数用于创建并展示一个搜索页面，允许用户通过输入关键词在题库中搜索题目。
//   - 搜索结果中的题干和选项部分，匹配的关键词将以红色粗体高亮显示。
//   - 支持按题型（如全部题型、单选题、多选题、判断题、填空题、简答题等）过滤搜索结果。
//
// 参数说明：
//   - w: fyne.Window 类型的窗口对象，用于展示搜索页面 UI。
//   - state: *core.AppState 类型的全局状态对象，包含题库数据、搜索定时器、当前文件名等信息。
//   - goBack: func() 类型的回调函数，在用户点击返回按钮时调用，用于退出搜索页面并返回上一页。
func ShowSearchPage(w fyne.Window, state *core.AppState, goBack func()) {
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// 返回按钮（backBtn）：用于退出搜索页面并返回上一页。
	// 功能与业务用途：
	//   - 点击该按钮时，会停止当前正在执行的搜索定时器（ActiveSearchTimer），并将其置为 nil。
	//   - 递增 SearchGeneration 的值，使所有待执行的搜索定时器失效，防止在页面返回后仍有旧的搜索任务被触发。
	//   - 调用 goBack() 回调函数，退出搜索页面并返回上一页。
	backBtn := customElements.CreateButton(
		core.AdminBackBtnText, core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.PageBgColor),
		core.HexColor(core.PageBgColor),
		core.StrokeMedium, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if state.ActiveSearchTimer != nil {
				state.ActiveSearchTimer.Stop()
				state.ActiveSearchTimer = nil
			}
			state.SearchGeneration++ // 递增代数以失效所有待执行的 timer
			goBack()
		})
	// 1. 用 canvas.NewText 替代 widget.NewLabelWithStyle，并精准控制字号
	titleText := canvas.NewText(core.SearchPageTitleText, core.HexColor(core.TextPrimaryColor))
	titleText.TextSize = core.FontSizeBody // 👈 统一使用核心字号
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	// 2. 如果需要它在标题栏居中，可以用 container.NewCenter 包裹
	titleCenter := container.NewCenter(titleText)
	rightSpace := canvas.NewRectangle(color.Transparent)
	rightSpace.SetMinSize(fyne.NewSize(16, 1)) // 16 像素的右侧边距

	// 把文字和右侧空白拼成一个 HBox，再放入 Border 的 Right 槽位
	rightBox := container.NewHBox(titleCenter, rightSpace)

	// 3. 放入 Border 布局中
	navBar := container.NewBorder(customElements.GetTitle(), nil, backBtn, rightBox, nil)
	keyword := ""
	selectedType := core.FilterTypeAllText

	resultsContainer := container.NewVBox()
	scrollResults := container.NewScroll(resultsContainer)

	executeSearch := func() {
		var searchResults []core.Question
		for _, q := range state.Questions {
			if selectedType != "全部题型" && q.Type != selectedType {
				continue
			}

			if keyword != "" {
				lowerKeyword := strings.ToLower(keyword)
				matchContent := strings.Contains(strings.ToLower(q.Content), lowerKeyword)

				matchOption := false
				for _, opt := range q.Options {
					if strings.Contains(strings.ToLower(opt.Text), lowerKeyword) {
						matchOption = true
						break
					}
				}

				if !matchContent && !matchOption {
					continue
				}
			}
			searchResults = append(searchResults, q)
		}

		resultsContainer.Objects = nil

		if keyword != "" {
			if len(searchResults) == 0 {
				emptyLbl := widget.NewLabel(core.SearchEmptyResultMsg)
				emptyLbl.Alignment = fyne.TextAlignCenter
				resultsContainer.Add(container.NewPadded(emptyLbl))
			} else {
				countLbl := widget.NewLabel(fmt.Sprintf(core.SearchResultCountMsgFormat, len(searchResults)))
				countLbl.Alignment = fyne.TextAlignCenter
				resultsContainer.Add(container.NewPadded(countLbl))

				for _, q := range searchResults {
					cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
					cardBg.CornerRadius = core.CardCornerRadius

					headerLeft := widget.NewLabel(fmt.Sprintf(core.QuestionIDHeaderFormat, q.ID))
					headerLeft.TextStyle = fyne.TextStyle{Bold: true}
					headerRight := widget.NewLabel(q.Type)
					headerRight.TextStyle = fyne.TextStyle{Bold: true}
					header := container.NewBorder(nil, nil, headerLeft, headerRight)

					contentRichText := HighlightText(q.Content, keyword)

					optionsGrid := container.NewGridWithColumns(2)
					for _, opt := range q.Options {
						k := opt.Label
						optText := opt.Text

						if optText == "" {
							continue
						}

						isCorrect := false
						if q.Type == core.QTypeFillIn || q.Type == core.QTypeEssay {
							isCorrect = true
						} else {
							for _, ans := range q.Answers {
								if ans == k {
									isCorrect = true
									break
								}
							}
						}

						optStr := fmt.Sprintf("%s. %s", k, optText)
						optLabel := HighlightText(optStr, keyword)

						optBg := canvas.NewRectangle(core.HexColor(core.OptionDefaultBg))
						optBg.CornerRadius = core.SmallButtonCorner
						optBg.StrokeColor = core.HexColor(core.BorderLightColor)
						optBg.StrokeWidth = core.StrokeThin

						if isCorrect {
							optBg.FillColor = core.HexColor(core.SearchHighlightBg)
							optBg.StrokeColor = core.HexColor(core.SearchHighlightBg)
						}

						optBox := container.NewStack(optBg, container.NewPadded(optLabel))
						optionsGrid.Add(optBox)
					}

					cardContent := container.NewVBox(header, contentRichText, optionsGrid)
					cardStack := container.NewStack(cardBg, container.NewPadded(cardContent))
					resultsContainer.Add(container.NewPadded(cardStack))
				}
			}
		}
		fyne.Do(func() { resultsContainer.Refresh() })
	}

	bankName := state.CurrentFileName
	if bankName == "" {
		bankName = core.DefaultBankNameText
	}
	titleCard := container.NewStack(
		canvas.NewRectangle(core.HexColor(core.CardBgColor)),
		container.NewPadded(widget.NewLabelWithStyle(core.BankTitlePrefix+bankName, fyne.TextAlignCenter, fyne.TextStyle{})),
	)

	input := widget.NewEntry()
	input.SetPlaceHolder(core.SearchInputPlaceholderText)
	input.OnChanged = func(s string) {
		keyword = strings.TrimSpace(s)
		if state.ActiveSearchTimer != nil {
			state.ActiveSearchTimer.Stop()
		}
		gen := state.SearchGeneration + 1
		state.SearchGeneration = gen
		state.ActiveSearchTimer = time.AfterFunc(time.Duration(core.SearchDebounceMs)*time.Millisecond, func() {
			if state.SearchGeneration != gen {
				return // 已过期，跳过本次回调
			}
			executeSearch()
		})
	}

	searchBtn := widget.NewButton(core.SearchBtnText, func() {
		executeSearch()
	})
	searchBtn.Importance = widget.HighImportance

	searchCard := container.NewStack(
		canvas.NewRectangle(core.HexColor(core.CardBgColor)),
		container.NewPadded(container.NewBorder(nil, nil, nil, searchBtn, input)),
	)

	types := []string{core.FilterTypeAllText, core.FilterTypeSingleChoiceText, core.FilterTypeMultiChoiceText, core.FilterTypeJudgeText, core.FilterTypeFillInText, core.FilterTypeEssayText}
	var filterContainer *fyne.Container
	var refreshFilters func()

	refreshFilters = func() {
		grid := container.NewGridWithColumns(3)
		for _, t := range types {
			currT := t
			btnColor := &struct {
				TextColor   string
				BgColor     string
				StrokeColor string
			}{
				TextColor: core.TextSecondaryColor,
				BgColor:   core.CardBgColor, StrokeColor: core.BorderLightColor,
			}
			text := currT

			if currT == selectedType {
				btnColor.BgColor = core.ColorSelectedBorder
				btnColor.StrokeColor = core.ColorSelectedBorder
				btnColor.TextColor = core.CardBgColor
				text = currT + core.FilterSelectedSuffixText
			}

			box := customElements.CreateButton(
				text, 0, 0,
				core.HexColor(btnColor.TextColor),
				core.HexColor(btnColor.BgColor),
				core.HexColor(btnColor.StrokeColor),
				core.StrokeMedium, core.FontSizeSubtitle,
				false, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					selectedType = currT
					refreshFilters()
					executeSearch()
				})
			grid.Add(box)
		}
		filterContainer.Objects = []fyne.CanvasObject{grid}
		fyne.Do(func() { filterContainer.Refresh() })
	}

	filterContainer = container.NewStack()
	refreshFilters()
	filterCard := container.NewStack(canvas.NewRectangle(core.HexColor(core.CardBgColor)), container.NewPadded(filterContainer))

	topBox := container.NewVBox(titleCard, searchCard, filterCard)

	mainLayout := container.NewBorder(
		navBar, nil, nil, nil,
		container.NewStack(pageBg, container.NewBorder(topBox, nil, nil, nil, scrollResults)),
	)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, mainLayout)
	w.SetContent(wRootLayout)

	executeSearch()
}
