package practiceRelated

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/storageRelated"
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

// ShowSearchPage 构建带有关键词高亮、题型过滤和多选下拉题库搜索的 UI。
// 支持在多个勾选的题库中搜索题目，默认仅选择当前题库。
// 关键词在题干和选项中以红色粗体高亮显示。
//
// 参数说明：
//   - w: fyne.Window 类型的窗口对象，用于展示搜索页面 UI。
//   - state: *core.AppState 类型的全局状态对象，包含题库数据、搜索定时器、存储文件列表等信息。
//   - goBack: func() 类型的回调函数，在用户点击返回按钮时调用，用于退出搜索页面并返回上一页。
func ShowSearchPage(w fyne.Window, state *core.AppState, goBack func()) {
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	backBtn := customElements.CreateButton(
		core.AdminBackBtnText, core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.PageBgColor),
		core.HexColor(core.PageBgColor),
		core.StrokeMedium, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		func() {
			if state.ActiveSearchTimer != nil {
				state.ActiveSearchTimer.Stop()
				state.ActiveSearchTimer = nil
			}
			state.SearchGeneration++
			goBack()
		})

	titleText := canvas.NewText("题库搜索", core.HexColor(core.TextPrimaryColor))
	titleText.TextSize = core.FontSizeBody
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	titleCenter := container.NewCenter(titleText)
	rightSpace := canvas.NewRectangle(color.Transparent)
	rightSpace.SetMinSize(fyne.NewSize(16, 1))

	rightBox := container.NewHBox(titleCenter, rightSpace)
	navBar := container.NewBorder(customElements.GetTitle(), nil, backBtn, rightBox, nil)

	keyword := ""
	selectedType := core.FilterTypeAllText

	// 默认开始只选择当前题库
	selectedBanks := make(map[string]bool)
	if state.CurrentStorageKey != "" {
		selectedBanks[state.CurrentStorageKey] = true
	} else if len(state.StoredFiles) > 0 {
		selectedBanks[state.StoredFiles[0]] = true
	}

	resultsContainer := container.NewVBox()
	scrollResults := container.NewScroll(resultsContainer)

	searchAndAppend := func(q core.Question, searchResults *[]core.Question, kw string, st string) {
		if st != "全部题型" && q.Type != st {
			return
		}

		if kw != "" {
			lowerKeyword := strings.ToLower(kw)
			matchContent := strings.Contains(strings.ToLower(q.Content), lowerKeyword)

			matchOption := false
			for _, opt := range q.Options {
				if strings.Contains(strings.ToLower(opt.Text), lowerKeyword) {
					matchOption = true
					break
				}
			}

			if !matchContent && !matchOption {
				return
			}
		}
		*searchResults = append(*searchResults, q)
	}

	executeSearch := func() {
		var searchResults []core.Question
		storageDir := storageRelated.GetStorageDir()

		for _, key := range state.StoredFiles {
			if !selectedBanks[key] {
				continue
			}

			filePath := fmt.Sprintf("%s/%s.json", storageDir, key)
			bytesData, err := os.ReadFile(filePath)
			if err != nil {
				if key == state.CurrentStorageKey && len(state.Questions) > 0 {
					for _, q := range state.Questions {
						qCopy := q
						if qCopy.BankName == "" {
							qCopy.BankName = state.CurrentFileName
						}
						searchAndAppend(qCopy, &searchResults, keyword, selectedType)
					}
				}
				continue
			}

			var bankData parser.BankData
			if err := json.Unmarshal(bytesData, &bankData); err != nil {
				continue
			}

			bankName := bankData.DisplayName
			if bankName == "" {
				bankName = parser.ExtractFileName(key)
			}

			for _, srcQ := range bankData.Questions {
				var convertedOpts []core.Option
				for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
					if srcQ.Options[k] != "" {
						convertedOpts = append(convertedOpts, core.Option{Label: k, Text: srcQ.Options[k]})
					}
				}
				var answers []string
				cleanAns := strings.NewReplacer("、", "", ",", "", " ", "").Replace(srcQ.Answer)
				for _, char := range cleanAns {
					norm := core.NormalizeAnswerLabel(string(char))
					if norm != "" {
						answers = append(answers, norm)
					}
				}
				if len(answers) == 0 && srcQ.Answer != "" {
					answers = append(answers, core.NormalizeAnswerLabel(srcQ.Answer))
				}

				q := core.Question{
					ID:         fmt.Sprintf("%d", srcQ.ID),
					Type:       srcQ.Type,
					Content:    srcQ.Content,
					Options:    convertedOpts,
					Answers:    answers,
					Difficulty: srcQ.Difficulty,
					BankName:   bankName,
				}

				searchAndAppend(q, &searchResults, keyword, selectedType)
			}
		}

		hasAnySelected := false
		for _, k := range state.StoredFiles {
			if selectedBanks[k] {
				hasAnySelected = true
				break
			}
		}
		if !hasAnySelected && len(state.Questions) > 0 {
			for _, q := range state.Questions {
				qCopy := q
				if qCopy.BankName == "" {
					qCopy.BankName = state.CurrentFileName
				}
				searchAndAppend(qCopy, &searchResults, keyword, selectedType)
			}
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

					bankTag := q.BankName
					if bankTag == "" {
						bankTag = core.DefaultBankNameText
					}
					headerLeft := widget.NewLabel(fmt.Sprintf("[%s] 第%s题", bankTag, q.ID))
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

	bankSelectContentContainer := container.NewVBox()
	var refreshBankSelectSummaryDisplay func()
	refreshBankSelectSummaryDisplay = func() {
		bankSelectContentContainer.Objects = nil
		var names []string
		for _, key := range state.StoredFiles {
			if selectedBanks[key] {
				names = append(names, parser.ExtractFileName(key))
			}
		}
		if len(names) == 0 {
			lbl := widget.NewLabel("• 未选择题库 (请点击右侧按钮选择)")
			lbl.Wrapping = fyne.TextWrapWord
			bankSelectContentContainer.Add(lbl)
			return
		}
		for _, name := range names {
			lbl := widget.NewLabel("• " + name)
			lbl.Wrapping = fyne.TextWrapWord
			bankSelectContentContainer.Add(lbl)
		}
	}
	refreshBankSelectSummaryDisplay()

	var showBankSelectDialog func()
	showBankSelectDialog = func() {
		checkboxContent := container.NewVBox()

		var refreshCheckboxes func()
		refreshCheckboxes = func() {
			checkboxContent.Objects = nil
			if len(state.StoredFiles) == 0 {
				checkboxContent.Add(widget.NewLabel("暂无其他已存储题库"))
				return
			}
			grid := container.NewGridWithColumns(1)
			for _, key := range state.StoredFiles {
				ckKey := key
				showName := parser.ExtractFileName(ckKey)
				isChecked := selectedBanks[ckKey]
				cb := widget.NewCheck("📖 "+showName, func(checked bool) {
					selectedBanks[ckKey] = checked
				})
				cb.SetChecked(isChecked)
				grid.Add(cb)
			}
			checkboxContent.Add(grid)
		}
		refreshCheckboxes()

		selectAllBtn := widget.NewButton("全选", func() {
			for _, k := range state.StoredFiles {
				selectedBanks[k] = true
			}
			refreshCheckboxes()
		})
		deselectAllBtn := widget.NewButton("全不选", func() {
			for _, k := range state.StoredFiles {
				selectedBanks[k] = false
			}
			refreshCheckboxes()
		})

		var dlg dialog.Dialog
		confirmBtn := widget.NewButton("确定", func() {
			if dlg != nil {
				dlg.Hide()
			}
			refreshBankSelectSummaryDisplay()
			executeSearch()
		})
		confirmBtn.Importance = widget.HighImportance

		cancelBtn := widget.NewButton("取消", func() {
			if dlg != nil {
				dlg.Hide()
			}
		})

		actionsRow := container.NewHBox(selectAllBtn, deselectAllBtn, layout.NewSpacer(), cancelBtn, confirmBtn)

		scrollableChecks := container.NewVScroll(checkboxContent)
		scrollableChecks.SetMinSize(fyne.NewSize(380, 320))

		dialogBody := container.NewVBox(
			widget.NewLabelWithStyle("请勾选需要搜索的题库（支持多选）:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewGridWrap(fyne.NewSize(1, 10)),
			scrollableChecks,
			container.NewGridWrap(fyne.NewSize(1, 15)),
			actionsRow,
		)

		styledContent := container.NewStack(
			container.NewGridWrap(fyne.NewSize(420, 480)),
			container.NewPadded(dialogBody),
		)

		dlg = dialog.NewCustomWithoutButtons("多选搜索题库", styledContent, w)
		dlg.Show()
	}

	dropdownTriggerBtn := widget.NewButton("选择/修改题库 ▾", func() {
		showBankSelectDialog()
	})

	bankSelectHeader := container.NewHBox(
		widget.NewLabelWithStyle("搜索题库范围:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		dropdownTriggerBtn,
	)

	bankSelectContent := container.NewVBox(
		bankSelectHeader,
		container.NewGridWrap(fyne.NewSize(1, 6)),
		bankSelectContentContainer,
	)

	bankSelectCard := container.NewStack(
		canvas.NewRectangle(core.HexColor(core.CardBgColor)),
		container.NewPadded(bankSelectContent),
	)

	input := widget.NewEntry()
	input.SetPlaceHolder(core.SearchInputPlaceholderText)
	input.OnChanged = func(s string) {
		keyword = parser.SanitizeXSS(strings.TrimSpace(s))
		if state.ActiveSearchTimer != nil {
			state.ActiveSearchTimer.Stop()
		}
		gen := state.SearchGeneration + 1
		state.SearchGeneration = gen
		state.ActiveSearchTimer = time.AfterFunc(time.Duration(core.SearchDebounceMs)*time.Millisecond, func() {
			if state.SearchGeneration != gen {
				return
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
				fyne.TextAlignCenter,
				fyne.TextWrapOff,
				fyne.TextTruncateOff,
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

	topBox := container.NewVBox(bankSelectCard, searchCard, filterCard)

	mainLayout := container.NewBorder(
		navBar, nil, nil, nil,
		container.NewStack(pageBg, container.NewBorder(topBox, nil, nil, nil, scrollResults)),
	)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, mainLayout)
	w.SetContent(wRootLayout)

	executeSearch()
}
