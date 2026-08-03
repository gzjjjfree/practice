package pages

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/gz_storage"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/ui/widgets"
)

// ShowPractice renders the main practice/quiz page UI.
func ShowPractice(w fyne.Window, state *core.AppState, backFunc func(fyne.Window, *core.AppState)) {
	var refreshPracticeUI func()

	var currentQID string
	var isMemorizeRevealed bool
	var multiSelected = make(map[int]bool)
	var singleWrongClicked = make(map[int]bool)
	var singleCorrectClicked = make(map[int]bool)
	var multiSubmitted bool

	topShell := container.NewStack()
	bottomShell := container.NewStack()
	centerShell := container.NewStack()

	centerScroll := container.NewVScroll(nil)
	touchArea := widgets.NewTouchInterceptor(centerShell, centerScroll,
		func() {
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		},
		func() {
			if state.Index < len(state.CurrentList)-1 {
				state.Index++
				refreshPracticeUI()
			}
		},
	)
	centerScroll.Content = touchArea

	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	mainContent := container.NewBorder(topShell, bottomShell, nil, nil, centerScroll)
	mainLayout := container.NewStack(pageBg, mainContent)

	marginBox := canvas.NewRectangle(color.Transparent)
	marginBox.SetMinSize(fyne.NewSize(1, core.PageMarginHeight))

	rootLayout := container.NewBorder(widgets.GetTitle(), marginBox, nil, nil, mainLayout)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, rootLayout)
	w.SetContent(wRootLayout)

	refreshPracticeUI = func() {
		if state.Index < 0 || state.Index >= len(state.CurrentList) {
			return
		}
		q := state.CurrentList[state.Index]

		backBtn := widgets.GetBox("← 返回", 90, 35, &widgets.BoxColor{TextColor: core.TextPrimaryColor, BgColor: core.SectionBgColor,
			StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, func() { backFunc(w, state) })

		titleLbl := canvas.NewText(state.Title, core.HexColor(core.TextPrimaryColor))
		titleLbl.TextStyle = fyne.TextStyle{Bold: true}
		titleLbl.TextSize = 16
		titleCenter := container.NewCenter(titleLbl)
		topNav := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
		topNavPadded := container.NewPadded(topNav)

		bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
		bankNameText.Alignment = fyne.TextAlignCenter
		bankNameText.Wrapping = fyne.TextWrapBreak
		bankNameText.SizeName = theme.SizeNameWindowButtonHeight

		bankNameBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))
		bankNameBg.CornerRadius = 6
		bankNameBg.StrokeColor = core.HexColor(core.BorderLightColor)
		bankNameBg.StrokeWidth = 1
		fileNameCard := container.NewPadded(container.NewStack(bankNameBg, container.NewPadded(bankNameText)))

		topArea := container.NewVBox(topNavPadded, fileNameCard)

		globalIndexStr := "0"
		if q.ID != "" {
			globalIndexStr = q.ID
		}
		metaLeftStr := fmt.Sprintf("总%s题  %s", globalIndexStr, q.Type)
		metaLeft := canvas.NewText(metaLeftStr, core.HexColor(core.TextMutedColor))
		metaLeft.TextSize = 13

		isFav := state.FavSet[q.ID]
		starChar, starColor := "☆", core.StarColor
		if isFav {
			starChar, starColor = "⭐", core.StarColor
		}
		starText := canvas.NewText(starChar, core.HexColor(starColor))
		starText.TextSize = 18
		starBtn := widgets.NewClickableBox(container.NewCenter(starText), func() {
			state.FavSet[q.ID] = !state.FavSet[q.ID]
			gz_storage.SaveSetToLocal(state, "收藏集", state.FavSet)
			refreshPracticeUI()
		})

		diffText := canvas.NewText("难度: 普通", core.HexColor(core.TextMutedColor))
		diffText.TextSize = 12
		diffBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))
		diffBg.CornerRadius = 4
		diffTag := container.NewStack(diffBg, container.NewPadded(diffText))
		metaRight := container.NewHBox(starBtn, diffTag)
		metaRow := container.NewBorder(nil, nil, metaLeft, metaRight, nil)

		resetQuestionState := func() {
			isMemorizeRevealed = false
			multiSelected = make(map[int]bool)
			singleWrongClicked = make(map[int]bool)
			singleCorrectClicked = make(map[int]bool)
			multiSubmitted = false
		}

		if q.ID != currentQID {
			currentQID = q.ID
			resetQuestionState()

			state.PracticeRecordsMu.Lock()
			historyAns, exists := state.PracticeRecords[q.ID]
			state.PracticeRecordsMu.Unlock()

			if exists && historyAns != "null" {
				if q.Type == "填空题" || q.Type == "问答题" {
					isMemorizeRevealed = true
				} else if q.Type == "多选题" {
					historyLetters := strings.Split(historyAns, ",")
					for _, letter := range historyLetters {
						letter = strings.TrimSpace(letter)
						if len(letter) == 1 {
							idx := int(letter[0] - 'A')
							if idx >= 0 && idx < len(q.Options) {
								multiSelected[idx] = true
							}
						}
					}
					multiSubmitted = true
				} else {
					if len(historyAns) == 1 {
						clickedIdx := int(historyAns[0] - 'A')
						isHistoryCorrect := false
						for _, correctLetter := range q.Answers {
							if correctLetter == historyAns {
								isHistoryCorrect = true
								break
							}
						}
						if isHistoryCorrect {
							singleCorrectClicked[clickedIdx] = true
						} else {
							singleWrongClicked[clickedIdx] = true
							for _, correctLetter := range q.Answers {
								if len(correctLetter) == 1 {
									cIdx := int(correctLetter[0] - 'A')
									singleCorrectClicked[cIdx] = true
								}
							}
						}
					}
				}
			}
		}

		stemText := fmt.Sprintf("%d. %s", state.Index+1, q.Content)
		stemLbl := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleSubHeading,
				Text:  stemText,
			},
		)
		stemLbl.Wrapping = fyne.TextWrapBreak

		ruler := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleSubHeading,
				Text:  "第一行\n第二行\n第三行",
			},
		)
		minHeightSpacer := canvas.NewRectangle(color.Transparent)
		minHeightSpacer.SetMinSize(fyne.NewSize(1, ruler.MinSize().Height))
		stemBox := container.NewStack(minHeightSpacer, stemLbl)

		optionsBox := container.NewVBox()
		marginBox2 := canvas.NewRectangle(color.Transparent)
		marginBox2.SetMinSize(fyne.NewSize(1, core.OptionSpacing))
		optionsBox.Add(marginBox2)

		if q.Type == "填空题" || q.Type == "问答题" {
			btnText := "点击显示答案"
			if isMemorizeRevealed {
				btnText = "点击隐藏答案"
			}
			lblA := canvas.NewText(btnText, core.HexColor(core.TextSecondaryColor))
			lblA.TextSize = 18
			lblA.Alignment = fyne.TextAlignCenter
			lblA.TextStyle = fyne.TextStyle{Bold: true}
			bgA := canvas.NewRectangle(core.HexColor(core.PageBgColor))
			if isMemorizeRevealed {
				bgA.FillColor = core.HexColor(core.AnswerRevealBg)
			}
			bgA.CornerRadius = 6
			cardA := widgets.NewClickableBox(container.NewStack(bgA, container.NewPadded(container.NewCenter(lblA))), func() {
				isMemorizeRevealed = !isMemorizeRevealed
				refreshPracticeUI()
			})
			optionsBox.Add(cardA)

			if isMemorizeRevealed {
				ansStr := ""
				if len(q.Options) > 0 {
					ansStr = q.Options[0].Text
				}
				lblB := core.CreateOptionLabel(ansStr)
				bgB := canvas.NewRectangle(core.HexColor(core.ColorCorrectBg))
				bgB.StrokeColor = core.HexColor(core.ColorCorrectBorder)
				bgB.StrokeWidth = 2
				bgB.CornerRadius = 6
				cardB := container.NewStack(bgB, container.NewPadded(lblB))
				optionsBox.Add(cardB)
				gz_storage.HandleUserSelectOption(state, q, "已查看")
			}
		} else {
			hasSingleAnswered := len(singleCorrectClicked) > 0 || len(singleWrongClicked) > 0

			for i, opt := range q.Options {
				optIndex := i
				optStr := opt.Text
				if optStr == "" {
					continue
				}

				optLetter := string(rune('A' + optIndex))
				optPrefix := optLetter + ". "

				prefixLbl := widget.NewLabel(optPrefix)
				prefixLbl.TextStyle = fyne.TextStyle{Bold: true}
				prefixLbl.SizeName = theme.SizeNameSubHeadingText

				contentLbl := core.CreateOptionLabel(optStr)
				contentLbl.Alignment = fyne.TextAlignCenter
				contentLbl.Wrapping = fyne.TextWrapBreak

				rightIconText := canvas.NewText("", core.HexColor(core.CardBgColor))
				rightIconText.TextSize = 16
				rightIconText.TextStyle = fyne.TextStyle{Bold: true}

				bgColorStr := core.OptionDefaultBg
				strokeColorStr := core.BorderLightColor
				iconColorStr := core.CardBgColor
				greenColor := core.ColorCorrectBorder

				isCorrectAnswer := false
				for _, a := range q.Answers {
					if a == optLetter {
						isCorrectAnswer = true
						break
					}
				}

				if q.Type == "多选题" {
					if multiSubmitted {
						if isCorrectAnswer {
							bgColorStr = core.ColorCorrectBg
							if multiSelected[optIndex] {
								rightIconText.Text = "✓"
								iconColorStr = greenColor
								strokeColorStr = greenColor
							} else {
								rightIconText.Text = "✕"
								iconColorStr = core.ColorWrongBorder
								strokeColorStr = core.ColorWrongBorder
							}
						} else {
							if multiSelected[optIndex] {
								bgColorStr = core.ColorWrongBg
								rightIconText.Text = "✕"
								iconColorStr = core.ColorWrongBorder
								strokeColorStr = core.ColorWrongBorder
							}
						}
					} else if multiSelected[optIndex] {
						bgColorStr = core.ColorSelectedBg
					}
				} else {
					if singleWrongClicked[optIndex] {
						bgColorStr = core.ColorWrongBg
						rightIconText.Text = "✕"
						iconColorStr = core.ColorWrongBorder
						strokeColorStr = core.ColorWrongBorder
					}
					if singleCorrectClicked[optIndex] {
						bgColorStr = core.ColorCorrectBg
						rightIconText.Text = "✓"
						iconColorStr = greenColor
						strokeColorStr = greenColor
					}
				}

				rightIconText.Color = core.HexColor(iconColorStr)
				fixedIconBox := container.NewGridWrap(fyne.NewSize(core.IconFixedBoxSize, core.IconFixedBoxSize), container.NewCenter(rightIconText))
				optLayout := container.NewBorder(nil, nil, prefixLbl, fixedIconBox, contentLbl)

				optBg := canvas.NewRectangle(core.HexColor(bgColorStr))
				optBg.CornerRadius = core.CardCornerRadius
				optBg.StrokeColor = core.HexColor(strokeColorStr)
				optBg.StrokeWidth = 1

				optCard := widgets.NewClickableBox(container.NewStack(optBg, container.NewPadded(optLayout)), func() {
					if q.Type == "多选题" && multiSubmitted {
						return
					}
					if q.Type != "多选题" && hasSingleAnswered {
						return
					}

					if q.Type == "多选题" {
						multiSelected[optIndex] = !multiSelected[optIndex]
						refreshPracticeUI()
					} else {
						gz_storage.HandleUserSelectOption(state, q, optLetter)

						if isCorrectAnswer {
							singleCorrectClicked[optIndex] = true
							refreshPracticeUI()

							go func() {
								time.Sleep(time.Duration(core.SingleCorrectDelay) * time.Millisecond)
								state.PracticeRecordsMu.Lock()
								if state.Index < len(state.CurrentList)-1 {
									state.Index++
								}
								state.PracticeRecordsMu.Unlock()
								fyne.Do(func() {
									refreshPracticeUI()
								})
							}()
						} else {
							singleWrongClicked[optIndex] = true
							for _, correctLetter := range q.Answers {
								if len(correctLetter) == 1 {
									cIdx := int(correctLetter[0] - 'A')
									singleCorrectClicked[cIdx] = true
								}
							}
							refreshPracticeUI()
						}
					}
				})

				optionsBox.Add(optCard)
				optionsBox.Add(marginBox2)
			}
		}

		prevBtn := widgets.GetBox("上一题", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnPrimaryBg, StrokeColor: core.BtnPrimaryBg, TextSize: 18}, true, state.Index == 0, func() {
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		})

		nextBtn := widgets.GetBox("下一题", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnPrimaryBg, StrokeColor: core.BtnPrimaryBg, TextSize: 18}, true, state.Index >= len(state.CurrentList)-1, func() {
			if state.Index < len(state.CurrentList)-1 {
				state.Index++
				refreshPracticeUI()
			}
		})

		submitBtn := widgets.GetBox("提交", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnPrimaryBg, StrokeColor: core.BtnPrimaryBg, TextSize: 18}, true, q.Type != "多选题" || strings.Contains(state.Title, "修正题目"), func() {
			if multiSubmitted || len(multiSelected) == 0 {
				return
			}
			var userAnsList []string
			for i := range q.Options {
				if multiSelected[i] {
					userAnsList = append(userAnsList, string(rune('A'+i)))
				}
			}
			userAnsStr := strings.Join(userAnsList, ",")
			gz_storage.HandleUserSelectOption(state, q, userAnsStr)
			multiSubmitted = true
			isAllCorrect := core.IsAnswerCorrect(userAnsStr, q)
			refreshPracticeUI()
			if isAllCorrect {
				go func() {
					time.Sleep(time.Duration(core.MultiCorrectDelay) * time.Millisecond)
					state.PracticeRecordsMu.Lock()
					if state.Index < len(state.CurrentList)-1 {
						state.Index++
					}
					state.PracticeRecordsMu.Unlock()
					fyne.Do(func() {
						refreshPracticeUI()
					})
				}()
			}
		})

		navGrid := container.NewGridWithColumns(3, prevBtn, submitBtn, nextBtn)

		var questionContent *fyne.Container
		if strings.Contains(state.Title, "修正题目") {
			editForm := RenderCorrectionCard(w, state, q, refreshPracticeUI)
			questionContent = container.NewVBox(metaRow, widget.NewSeparator(), editForm, layout.NewSpacer(), navGrid)
		} else {
			questionContent = container.NewVBox(metaRow, widget.NewSeparator(), stemBox, optionsBox, layout.NewSpacer(), navGrid)
		}

		cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		cardBg.CornerRadius = 8
		cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
		cardBg.StrokeWidth = 1
		cardStack := container.NewStack(cardBg, container.NewPadded(questionContent))

		wCount := len(state.CurrentList)
		mCorrect, nWrong := parser.CalculateCurrentModeStats(state)

		statsText := fmt.Sprintf("✓: %d             ✕: %d", mCorrect, nWrong)
		statsLbl := canvas.NewText(statsText, core.HexColor(core.TextHintColor))
		statsLbl.TextSize = 16

		stateBtn := widgets.GetBox(fmt.Sprintf("总数: %d", wCount), 90, 36, &widgets.BoxColor{TextColor: core.TextHintColor, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 16}, false, false, func() { ShowQuestionModal(w, state, refreshPracticeUI) })

		gap := canvas.NewRectangle(color.Transparent)
		gap.SetMinSize(fyne.NewSize(35, 1))
		stateBox := container.NewHBox(stateBtn, gap, container.NewCenter(statsLbl))

		statsBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))
		sizeBox := container.NewGridWrap(fyne.NewSize(400, 35))
		statsRow := container.NewStack(statsBg, sizeBox, container.NewCenter(stateBox))

		returnTypeBtn := widgets.GetBox("返回题型", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnSecondaryBg, StrokeColor: core.BtnSecondaryBg, TextSize: 18}, true, false, func() { backFunc(w, state) })
		delCurrentDataBtn := widgets.GetBox("删除本题", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnSecondaryBg, StrokeColor: core.BtnSecondaryBg, TextSize: 18}, true, !strings.Contains(state.Title, "错题练习") && !strings.Contains(state.Title, "修正题目"), func() {
			core.ShowCustomConfirm("提示", "确定要将本题从当前练习集中移除吗？", func(confirm bool) {
				if confirm {
					refreshPracticeUI()
				}
			}, w)
		})
		delRecordBtn := widgets.GetBox("删除记录", 100, 35, &widgets.BoxColor{TextColor: core.CardBgColor, BgColor: core.BtnSecondaryBg, StrokeColor: core.BtnSecondaryBg, TextSize: 18}, true, false, func() {
			core.ShowCustomConfirm("提示", "确定要清空本次练习的所有答题记录和对错统计吗？", func(confirm bool) {
				if confirm {
					gz_storage.ClearPracticeRecords(state)
					resetQuestionState()
					refreshPracticeUI()
				}
			}, w)
		})

		actionGrid := container.NewGridWithColumns(3, returnTypeBtn, delCurrentDataBtn, delRecordBtn)
		bottomArea := container.NewVBox(statsRow, actionGrid)

		topShell.Objects = []fyne.CanvasObject{topArea}
		fyne.Do(func() { topShell.Refresh() })
		bottomShell.Objects = []fyne.CanvasObject{bottomArea}
		fyne.Do(func() { bottomShell.Refresh() })
		centerShell.Objects = []fyne.CanvasObject{container.NewPadded(cardStack)}
		fyne.Do(func() { centerShell.Refresh() })
	}

	refreshPracticeUI()
}

// ShowQuestionModal shows a pop-up grid of all questions with color-coded status.
func ShowQuestionModal(w fyne.Window, state *core.AppState, onNavigate func()) {
	grid := container.NewGridWrap(fyne.NewSize(50, 50))

	var modal *widget.PopUp

	for i := 0; i < len(state.CurrentList); i++ {
		idx := i
		q := state.CurrentList[idx]

		bg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		bg.CornerRadius = 6
		bg.StrokeWidth = 2
		bg.StrokeColor = core.HexColor(core.BorderMediumColor)

		isCurrent := (idx == state.Index)
		state.PracticeRecordsMu.Lock()
		historyAns, exists := state.PracticeRecords[q.ID]
		state.PracticeRecordsMu.Unlock()
		isAnswered := exists && historyAns != "null" && historyAns != ""

		isRight := false
		if isAnswered {
			isRight = core.IsAnswerCorrect(historyAns, q)
		}

		if isCurrent {
			bg.FillColor = core.HexColor(core.ColorSelectedBg)
			bg.StrokeColor = core.HexColor(core.ColorSelectedBorder)
		} else if isAnswered {
			if isRight {
				bg.FillColor = core.HexColor(core.ColorCorrectBg)
				bg.StrokeColor = core.HexColor(core.ColorCorrectBorder)
			} else {
				bg.FillColor = core.HexColor(core.ColorWrongBg)
				bg.StrokeColor = core.HexColor(core.ColorWrongBorder)
			}
		}

		shortType := ""
		switch q.Type {
		case "单选题":
			shortType = "单"
		case "多选题":
			shortType = "多"
		case "判断题":
			shortType = "判"
		case "填空题":
			shortType = "填"
		case "问答题", "简答题", "案例分析":
			shortType = "答"
		default:
			if len(q.Type) > 0 {
				shortType = string([]rune(q.Type)[0])
			}
		}

		lblText := fmt.Sprintf("%d\n%s", idx+1, shortType)
		lbl := widget.NewLabel(lblText)
		lbl.Alignment = fyne.TextAlignCenter

		box := widgets.NewClickableBox(container.NewStack(bg, lbl), func() {
			state.Index = idx
			onNavigate()
			if modal != nil {
				modal.Hide()
			}
		})
		grid.Add(box)
	}

	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(320, 360))

	titleText := canvas.NewText("题目导航", core.HexColor(core.TextPrimaryColor))
	titleText.TextSize = 16
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewCenter(titleText)

	closeBtnColor := &widgets.BoxColor{TextColor: "#FFFFFF", BgColor: core.BtnPrimaryBg, StrokeColor: core.BtnPrimaryBg, TextSize: 14}
	btnClose := widgets.GetBox("关闭", 120, 36, closeBtnColor, true, false, func() {
		if modal != nil {
			modal.Hide()
		}
	})

	modalContent := container.NewVBox(
		container.NewPadded(titleContainer),
		scroll,
		container.NewPadded(container.NewCenter(btnClose)),
	)

	modalBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	modalBg.CornerRadius = 10
	modalBg.StrokeWidth = 1
	modalBg.StrokeColor = core.HexColor(core.BorderLightColor)

	styledModal := container.NewStack(modalBg, container.NewPadded(modalContent))

	modal = widget.NewModalPopUp(styledModal, w.Canvas())
	modal.Show()
}
