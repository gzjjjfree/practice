package main

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ==================== 辅助函数：关键字标红高亮 ====================
func highlightText(text, keyword string) *widget.RichText {
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
			// 使用主题的 Error 颜色（通常是红色）实现高亮，并加粗
			Style: widget.RichTextStyle{Inline: true, ColorName: theme.ColorNameError, TextStyle: fyne.TextStyle{Bold: true}},
			Text:  text[idx : idx+len(keyword)],
		})
		start = idx + len(keyword)
	}

	rt := widget.NewRichText(segments...)
	rt.Wrapping = fyne.TextWrapBreak
	return rt
}

// ==================== 全新搜索页 ====================
func showSearchPage(w fyne.Window, state *AppState, goBack func()) {
	// ==================== 1. 页面基础状态 ====================
	pageBg := canvas.NewRectangle(hexColor("#f5f5f5"))

	//backBtn := widget.NewButton("← 返回", func() { goBack() })
	backBtn := getBox("← 返回", 90, 35, &boxColor{textColor: "#000000", bgColor: "#F0F2F5", strokeColor: "#F0F2F5", textSize: 18}, false, func() { goBack() })
	navBar := container.NewBorder(getTitle(), nil, backBtn, widget.NewLabelWithStyle("搜索本题库", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	keyword := ""
	selectedType := "全部题型"
	var searchTimer *time.Timer

	// ==================== 2. UI 容器声明 ====================
	resultsContainer := container.NewVBox() // 用于动态存放搜索结果卡片
	scrollResults := container.NewScroll(resultsContainer)

	// 核心检索与渲染逻辑
	executeSearch := func() {
		// 1. 过滤逻辑
		var searchResults []Question
		for _, q := range state.Questions {
			if selectedType != "全部题型" && q.Type != selectedType {
				continue
			}

			if keyword != "" {
				lowerKeyword := strings.ToLower(keyword)
				matchContent := strings.Contains(strings.ToLower(q.Content), lowerKeyword)

				matchOption := false
				// 遍历 []Option 结构体切片
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

		// 2. 清空并重绘结果区
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

				// ==================== 渲染搜索出的题目卡片 ====================
				for _, q := range searchResults {
					cardBg := canvas.NewRectangle(hexColor("#ffffff"))
					cardBg.CornerRadius = 8

					// 头部：第几题 + 题型
					headerLeft := widget.NewLabel(fmt.Sprintf("第%s题", q.ID)) // 假设 ID 是 string，若是 int 请用 %d
					headerLeft.TextStyle = fyne.TextStyle{Bold: true}
					headerRight := widget.NewLabel(q.Type)
					headerRight.TextStyle = fyne.TextStyle{Bold: true}
					//headerRight.Color = theme.DisabledColor()
					header := container.NewBorder(nil, nil, headerLeft, headerRight)

					// 题干内容（支持关键字红字高亮）
					contentRichText := highlightText(q.Content, keyword)

					// 选项网格 (2列排布)
					optionsGrid := container.NewGridWithColumns(2)

					// 顺序遍历 []Option 切片，天然保证 A、B、C、D 顺序
					for _, opt := range q.Options {
						k := opt.Label
						optText := opt.Text

						if optText == "" {
							continue
						}

						// 判定是否是正确答案（标绿）
						isCorrect := false
						if q.Type == "填空题" || q.Type == "问答题" {
							isCorrect = true // 填空问答全标绿
						} else {
							for _, ans := range q.Answers {
								if ans == k {
									isCorrect = true
									break
								}
							}
						}

						// 选项内容同样支持红字高亮
						optStr := fmt.Sprintf("%s. %s", k, optText)
						optLabel := highlightText(optStr, keyword)

						optBg := canvas.NewRectangle(hexColor("#f9f9f9"))
						optBg.CornerRadius = 4
						optBg.StrokeColor = hexColor("#e8e8e8")
						optBg.StrokeWidth = 1

						if isCorrect {
							optBg.FillColor = hexColor("#5aeb55")   // 正确答案绿底
							optBg.StrokeColor = hexColor("#5aeb55") // 绿边
						}

						optBox := container.NewStack(optBg, container.NewPadded(optLabel))
						optionsGrid.Add(optBox)
					}

					// 组装单个卡片
					cardContent := container.NewVBox(header, contentRichText, optionsGrid)
					cardStack := container.NewStack(cardBg, container.NewPadded(cardContent))
					resultsContainer.Add(container.NewPadded(cardStack))
				}
			}
		}
		resultsContainer.Refresh()
	}

	// ==================== 3. 顶部输入与过滤区 ====================
	// 题库标题
	bankName := state.CurrentFileName
	if bankName == "" {
		bankName = "未命名题库"
	}
	titleCard := container.NewStack(
		canvas.NewRectangle(hexColor("#ffffff")),
		container.NewPadded(widget.NewLabelWithStyle("📖 "+bankName, fyne.TextAlignCenter, fyne.TextStyle{})),
	)

	// 搜索框与防抖逻辑
	input := widget.NewEntry()
	input.SetPlaceHolder("请输入题目关键词")
	input.OnChanged = func(s string) {
		keyword = strings.TrimSpace(s)
		if searchTimer != nil {
			searchTimer.Stop()
		}
		// 300ms 防抖
		searchTimer = time.AfterFunc(300*time.Millisecond, func() {
			executeSearch()
		})
	}

	searchBtn := widget.NewButton("搜索", func() {
		executeSearch()
	})
	searchBtn.Importance = widget.HighImportance

	searchCard := container.NewStack(
		canvas.NewRectangle(hexColor("#ffffff")),
		container.NewPadded(container.NewBorder(nil, nil, nil, searchBtn, input)),
	)

	// 题型过滤网格
	types := []string{"全部题型", "单选题", "多选题", "判断题", "填空题", "问答题"}
	var filterContainer *fyne.Container
	var refreshFilters func()

	refreshFilters = func() {
		grid := container.NewGridWithColumns(3)
		for _, t := range types {
			currT := t
			bg := canvas.NewRectangle(hexColor("#ffffff"))
			bg.StrokeColor = hexColor("#e8e8e8")
			bg.StrokeWidth = 1
			bg.CornerRadius = 4

			text := canvas.NewText(currT, hexColor("#333333"))
			text.Alignment = fyne.TextAlignCenter
			text.TextSize = 14

			if currT == selectedType {
				bg.FillColor = hexColor("#007AFF")
				bg.StrokeColor = hexColor("#007AFF")
				text.Color = hexColor("#ffffff")
				text.Text = currT + " ✓"
			}

			box := NewClickableBox(container.NewStack(bg, container.NewPadded(text)), func() {
				selectedType = currT
				refreshFilters()
				executeSearch() // 切换题型后立即执行搜索
			})
			grid.Add(box)
		}
		filterContainer.Objects = []fyne.CanvasObject{grid}
		filterContainer.Refresh()
	}

	filterContainer = container.NewStack()
	refreshFilters()
	filterCard := container.NewStack(canvas.NewRectangle(hexColor("#ffffff")), container.NewPadded(filterContainer))

	// ==================== 4. 页面总装 ====================
	topBox := container.NewVBox(titleCard, searchCard, filterCard)

	mainLayout := container.NewBorder(
		navBar, nil, nil, nil,
		container.NewStack(pageBg, container.NewBorder(topBox, nil, nil, nil, scrollResults)),
	)

	//mainLayout := container.NewBorder(getTitle(), mainLayoutBox, nil, nil)
	w.SetContent(mainLayout)

	// 初始化如果本身带有状态，可以执行一次空检索
	executeSearch()
}
