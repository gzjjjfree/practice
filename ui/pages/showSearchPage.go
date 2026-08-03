package pages

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/ui/widgets"
)

// HighlightText creates rich text with keyword matches highlighted in red/bold.
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

// ShowSearchPage builds search UI with keyword highlighting and type filtering.
func ShowSearchPage(w fyne.Window, state *core.AppState, goBack func()) {
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	backBtn := widgets.GetBox("← 返回", 90, 35, &widgets.BoxColor{TextColor: core.TextPrimaryColor, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, func() {
		if state.ActiveSearchTimer != nil {
			state.ActiveSearchTimer.Stop()
			state.ActiveSearchTimer = nil
		}
		state.SearchGeneration++ // 递增代数以失效所有待执行的 timer
		goBack()
	})
	navBar := container.NewBorder(widgets.GetTitle(), nil, backBtn, widget.NewLabelWithStyle("搜索本题库", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	keyword := ""
	selectedType := "全部题型"

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
				emptyLbl := widget.NewLabel("未找到相关题目")
				emptyLbl.Alignment = fyne.TextAlignCenter
				resultsContainer.Add(container.NewPadded(emptyLbl))
			} else {
				countLbl := widget.NewLabel(fmt.Sprintf("找到 %d 个结果", len(searchResults)))
				countLbl.Alignment = fyne.TextAlignCenter
				resultsContainer.Add(container.NewPadded(countLbl))

				for _, q := range searchResults {
					cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
					cardBg.CornerRadius = 8

					headerLeft := widget.NewLabel(fmt.Sprintf("第%s题", q.ID))
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
						if q.Type == "填空题" || q.Type == "问答题" {
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
						optBg.CornerRadius = 4
						optBg.StrokeColor = core.HexColor(core.BorderLightColor)
						optBg.StrokeWidth = 1

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
		bankName = "未命名题库"
	}
	titleCard := container.NewStack(
		canvas.NewRectangle(core.HexColor(core.CardBgColor)),
		container.NewPadded(widget.NewLabelWithStyle("📖 "+bankName, fyne.TextAlignCenter, fyne.TextStyle{})),
	)

	input := widget.NewEntry()
	input.SetPlaceHolder("请输入题目关键词")
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

	searchBtn := widget.NewButton("搜索", func() {
		executeSearch()
	})
	searchBtn.Importance = widget.HighImportance

	searchCard := container.NewStack(
		canvas.NewRectangle(core.HexColor(core.CardBgColor)),
		container.NewPadded(container.NewBorder(nil, nil, nil, searchBtn, input)),
	)

	types := []string{"全部题型", "单选题", "多选题", "判断题", "填空题", "问答题"}
	var filterContainer *fyne.Container
	var refreshFilters func()

	refreshFilters = func() {
		grid := container.NewGridWithColumns(3)
		for _, t := range types {
			currT := t
			bg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
			bg.StrokeColor = core.HexColor(core.BorderLightColor)
			bg.StrokeWidth = 1
			bg.CornerRadius = 4

			text := canvas.NewText(currT, core.HexColor(core.TextSecondaryColor))
			text.Alignment = fyne.TextAlignCenter
			text.TextSize = 14

			if currT == selectedType {
				bg.FillColor = core.HexColor(core.ColorSelectedBorder)
				bg.StrokeColor = core.HexColor(core.ColorSelectedBorder)
				text.Color = core.HexColor(core.CardBgColor)
				text.Text = currT + " ✓"
			}

			box := widgets.NewClickableBox(container.NewStack(bg, container.NewPadded(text)), func() {
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
