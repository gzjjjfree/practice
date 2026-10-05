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
	"github.com/gzjjjfree/practice/storageRelated"
)

// ShowPractice 渲染主要的练习/考试页面 UI。
// 这是用户进行题目练习的核心页面，支持顺序练习、随机练习、错题练习、收藏练习等多种模式。
// 参数:
//   - w: Fyne 窗口引用，用于创建弹窗和设置内容
//   - state: 应用状态指针，包含题目列表、用户答案、练习记录等所有运行时数据
//   - backFunc: 返回函数，用于返回上一页（题型选择页面）
func ShowPractice(w fyne.Window, state *core.AppState, backFunc func(fyne.Window, *core.AppState)) {
	var refreshPracticeUI func() // 用于刷新练习页面 UI 的闭包函数

	// ========== 题目状态跟踪变量 ==========
	var currentQID string                         // 当前显示的题目 ID，用于检测题目切换
	var isMemorizeRevealed bool                   // 填空题/问答题：是否已显示答案
	var multiSelected = make(map[int]bool)        // 多选题：已选中的选项索引集合
	var singleWrongClicked = make(map[int]bool)   // 单选题：错误点击的选项索引
	var singleCorrectClicked = make(map[int]bool) // 单选题：正确点击的选项索引
	var multiSubmitted bool                       // 多选题：是否已提交

	// ========== 页面布局结构 ==========
	// 页面分为上、中、下三个区域：顶部导航、中间题目内容、底部统计和操作
	topShell := container.NewStack()
	bottomShell := container.NewStack()
	centerShell := container.NewStack()
	navShell := container.NewStack() // 👈 [新增] 用于承载固定在底部的导航按钮

	marginBox2 := canvas.NewRectangle(color.Transparent)
	marginBox2.SetMinSize(fyne.NewSize(1, core.OptionSpacing))

	// 中间滚动区域，使用触摸拦截器实现滑动翻页
	centerScroll := container.NewVScroll(nil)
	touchArea := customElements.NewTouchInterceptor(centerShell, centerScroll,
		// 左滑（向左滑动）：进入下一题
		func() {
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		},
		// 右滑（向右滑动）：进入上一题
		func() {
			if state.Index < len(state.CurrentList)-1 {
				state.Index++
				refreshPracticeUI()
			}
		},
	)
	centerScroll.Content = touchArea

	// 👈 [核心修改] 将白色卡片背景移到外层，使其能向下撑满整个屏幕剩余空间
	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.CardCornerRadius
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	navShellCenter := container.NewVBox(navShell, marginBox2)
	// 👈 [核心修改] 使用 Border 布局：导航按钮放在 Bottom 固定，滚动题目填满中间
	cardContent := container.NewBorder(nil, navShellCenter, nil, nil, centerScroll)
	cardStack := container.NewStack(cardBg, cardContent)

	// 页面背景色
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// 中间主内容区：将完整的卡片（包含底色、导航和滚动区）包裹 Padding 后放入
	mainContent := container.NewBorder(topShell, bottomShell, nil, nil, container.NewPadded(cardStack))
	mainLayout := container.NewStack(pageBg, mainContent)

	// 顶部标题栏和底部边距
	marginBox := canvas.NewRectangle(color.Transparent)

	rootLayout := container.NewBorder(customElements.GetTitle(), marginBox, nil, nil, mainLayout)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, rootLayout)
	w.SetContent(wRootLayout)

	// ========== UI 刷新函数 ==========
	// 每次状态变化时调用此函数重新渲染页面
	refreshPracticeUI = func() {
		// 边界检查：确保索引在有效范围内
		if state.Index < 0 || state.Index >= len(state.CurrentList) {
			return
		}
		q := state.CurrentList[state.Index]

		// --- 顶部导航栏 ---
		// 返回按钮
		backBtn := customElements.CreateButton(
			core.AdminBackBtnText,
			core.BackBtnWidth, core.BackBtnHeight,
			core.HexColor(core.TextPrimaryColor),
			core.HexColor(core.SectionBgColor),
			core.HexColor(core.SectionBgColor),
			core.StrokeMedium, core.FontSizeBody,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				backFunc(w, state)
			},
		)

		// 练习标题（如"错题练习 - 单选题"）
		titleLbl := canvas.NewText(state.Title, core.HexColor(core.TextPrimaryColor))
		titleLbl.TextStyle = fyne.TextStyle{Bold: true}
		titleLbl.TextSize = core.FontSizeBody
		titleCenter := container.NewCenter(titleLbl)
		topNav := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
		topNavPadded := container.NewPadded(topNav)

		// 题库名称显示
		bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
		bankNameText.Alignment = fyne.TextAlignCenter
		bankNameText.Wrapping = fyne.TextWrapBreak
		bankNameText.SizeName = theme.SizeNameWindowButtonHeight

		bankNameBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))
		bankNameBg.CornerRadius = core.SmallButtonCorner
		bankNameBg.StrokeColor = core.HexColor(core.BorderLightColor)
		bankNameBg.StrokeWidth = core.StrokeThin
		fileNameCard := container.NewPadded(container.NewStack(bankNameBg, container.NewPadded(bankNameText)))

		topArea := container.NewVBox(topNavPadded, fileNameCard)

		// --- 元信息行（题目编号、题型、收藏、难度）---
		globalIndexStr := "0"
		if q.ID != "" {
			globalIndexStr = q.ID
		}
		metaLeftStr := fmt.Sprintf("总%s题  %s", globalIndexStr, q.Type)
		metaLeft := canvas.NewText(metaLeftStr, core.HexColor(core.TextMutedColor))
		metaLeft.TextSize = core.FontSizeSubtitle

		// 收藏按钮（⭐/☆），点击切换收藏状态并持久化
		isFav := state.FavSet[q.ID]
		starChar := "☆"
		if isFav {
			starChar = "⭐"
		}
		starBtn := customElements.CreateButton(
			starChar,
			core.StarBtnSize, core.StarBtnSize,
			core.HexColor(core.StarColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.CardBgColor),
			core.StrokeMedium, core.FontSizeButton,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				state.FavSet[q.ID] = !state.FavSet[q.ID]
				storageRelated.SaveFavSetToDisk(state) // 将收藏集保存到磁盘
				refreshPracticeUI()
			},
		)

		// 难度标签（目前固定显示"普通"）
		diffText := canvas.NewText(core.PracticeDifficultyLabel, core.HexColor(core.TextMutedColor))
		diffText.TextSize = core.FontSizeSubtitle
		diffBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		diffBg.CornerRadius = core.SmallButtonCorner
		diffTag := container.NewStack(diffBg, container.NewPadded(diffText))
		metaRight := container.NewHBox(starBtn, diffTag)
		metaRow := container.NewBorder(nil, nil, metaLeft, metaRight, nil)

		// --- 题目状态重置函数（每次切换题目时调用）---
		resetQuestionState := func() {
			isMemorizeRevealed = false
			multiSelected = make(map[int]bool)
			singleWrongClicked = make(map[int]bool)
			singleCorrectClicked = make(map[int]bool)
			multiSubmitted = false
		}

		// 检测题目 ID 是否变化，变化则重置状态并加载历史记录
		if q.ID != currentQID {
			currentQID = q.ID
			resetQuestionState()

			// 从存储中读取历史答题记录，恢复之前的状态
			state.PracticeRecordsMu.Lock()
			historyAns, exists := state.PracticeRecords[q.ID]
			state.PracticeRecordsMu.Unlock()

			if exists && historyAns != "null" {
				if q.Type == "填空题" || q.Type == "问答题" {
					isMemorizeRevealed = true
				} else if q.Type == "多选题" {
					// 解析历史答案（逗号分隔的字母）
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

		// --- 题干显示 ---
		stemText := fmt.Sprintf("%d. %s", state.Index+1, q.Content)
		stemLbl := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleSubHeading,
				Text:  stemText,
			},
		)
		stemLbl.Wrapping = fyne.TextWrapBreak // 支持文本自动换行

		ruler := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleSubHeading,
				Text:  "第一行\n第二行\n第三行",
			},
		)
		minHeightSpacer := canvas.NewRectangle(color.Transparent)
		minHeightSpacer.SetMinSize(fyne.NewSize(1, ruler.MinSize().Height))
		stemBox := container.NewStack(minHeightSpacer, stemLbl)

		// --- 选项区域 ---
		optionsBox := container.NewVBox()
		marginBox2 := canvas.NewRectangle(color.Transparent)
		marginBox2.SetMinSize(fyne.NewSize(1, core.OptionSpacing))
		optionsBox.Add(marginBox2)

		// 填空题/问答题：点击显示/隐藏答案
		if q.Type == "填空题" || q.Type == "问答题" {
			btnText := core.PracticeShowAnswerBtnText
			if isMemorizeRevealed {
				btnText = core.PracticeHideAnswerBtnText
			}
			cardAColor := core.PageBgColor
			if isMemorizeRevealed {
				cardAColor = core.AnswerRevealBg // 显示答案时使用特殊背景色
			}

			// 填空题居中，问答题左对齐
			btnAlign := fyne.TextAlignCenter
			lblAlign := fyne.TextAlignCenter
			if q.Type == "问答题" {
				btnAlign = fyne.TextAlignCenter
				lblAlign = fyne.TextAlignLeading
			}

			cardA := customElements.CreateButton(
				btnText,
				core.BtnAutoSizeWidth, core.BtnAutoSizeHeight,
				core.HexColor(core.TextSecondaryColor),
				core.HexColor(cardAColor),
				core.HexColor(core.BorderLightColor),
				core.StrokeMedium, core.FontSizeButton,
				true, false,
				btnAlign,             // 👈 填空题居中，问答题左对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					isMemorizeRevealed = !isMemorizeRevealed
					refreshPracticeUI()
				},
			)

			optionsBox.Add(cardA)

			if isMemorizeRevealed {
				ansStr := ""
				if len(q.Options) > 0 {
					ansStr = q.Options[0].Text
				}
				lblB := widget.NewLabel(ansStr)
				lblB.Alignment = lblAlign
				lblB.Wrapping = fyne.TextWrapBreak
				lblB.SizeName = theme.SizeNameSubHeadingText

				bgB := canvas.NewRectangle(core.HexColor(core.ColorCorrectBg))
				bgB.StrokeColor = core.HexColor(core.ColorCorrectBorder)
				bgB.StrokeWidth = core.StrokeMedium
				bgB.CornerRadius = core.SmallButtonCorner
				cardB := container.NewStack(bgB, container.NewPadded(lblB))
				optionsBox.Add(marginBox2)
				optionsBox.Add(cardB)
				storageRelated.HandleUserSelectOption(state, q, "已查看")
			}
		} else {
			// 单选题/多选题渲染
			hasSingleAnswered := len(singleCorrectClicked) > 0 || len(singleWrongClicked) > 0

			for i, opt := range q.Options {
				optIndex := i
				optStr := opt.Text
				if optStr == "" {
					continue
				}

				optLetter := string(rune('A' + optIndex))
				optPrefix := optLetter + "."

				// 1. 左侧序号：使用 canvas.NewText 统一设置字号
				prefixLbl := canvas.NewText(optPrefix, core.HexColor(core.TextBodyColor))
				prefixLbl.TextStyle = fyne.TextStyle{Bold: true}
				prefixLbl.TextSize = core.FontSizeButton // 👈 设置按钮字号

				leftPadding := canvas.NewRectangle(color.Transparent)
				leftPadding.SetMinSize(fyne.NewSize(12, 1))
				leftBox := container.NewHBox(leftPadding, container.NewCenter(prefixLbl))

				// 3. 右侧状态图标 (✓ / ✕)
				rightIconText := canvas.NewText("", core.HexColor(core.CardBgColor))
				rightIconText.TextSize = core.FontSizeButton // 👈 设置按钮字号
				rightIconText.TextStyle = fyne.TextStyle{Bold: true}

				bgColorStr := core.OptionDefaultBg
				strokeColorStr := core.BorderLightColor
				iconColorStr := core.CardBgColor
				greenColor := core.ColorCorrectBorder

				// 判断该选项是否为正确答案
				isCorrectAnswer := false
				for _, a := range q.Answers {
					if a == optLetter {
						isCorrectAnswer = true
						break
					}
				}

				// 根据题型和状态设置选项样式及图标
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

				rightPadding := canvas.NewRectangle(color.Transparent)
				rightPadding.SetMinSize(fyne.NewSize(12, 1))
				rightBox := container.NewHBox(container.NewCenter(rightIconText), rightPadding)

				// 4. 将左中右三部分组装成 Border 布局
				optionLayout := container.NewBorder(
					nil, nil,
					leftBox,  // 左侧：序号
					rightBox, // 右侧：状态图标
				)

				// 5. 外层透明/背景响应卡片
				btnBg := customElements.CreateButton(
					optStr,
					-1, core.FileListItemHeight,
					core.HexColor(core.TextBodyColor),
					core.HexColor(bgColorStr),
					core.HexColor(strokeColorStr),
					core.StrokeMedium, core.FontSizeButton,
					false, false,
					fyne.TextAlignCenter,
					fyne.TextWrapOff,
					fyne.TextTruncateOff,
					func() {
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
							storageRelated.HandleUserSelectOption(state, q, optLetter)

							if isCorrectAnswer {
								singleCorrectClicked[optIndex] = true

								state.PracticeRecordsMu.Lock()
								currentGen := state.AnswerGenCounter + 1
								state.AnswerGenCounter = currentGen
								state.PracticeRecordsMu.Unlock()

								refreshPracticeUI()

								go func(gen int) {
									time.Sleep(time.Duration(core.SingleCorrectDelay) * time.Millisecond)
									state.PracticeRecordsMu.Lock()
									skip := gen != state.AnswerGenCounter
									if !skip && state.Index < len(state.CurrentList)-1 {
										state.Index++
									}
									state.PracticeRecordsMu.Unlock()

									if !skip {
										fyne.Do(func() {
											refreshPracticeUI()
										})
									}
								}(currentGen)
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
					},
				)

				// 6. 叠层合并
				optCard := container.NewStack(btnBg, optionLayout)

				optionsBox.Add(optCard)
				optionsBox.Add(marginBox2)
			}
		}

		// --- 导航按钮（上一题/下一题/提交）---
		prevBtn := customElements.CreateButton(
			core.PracticePrevBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true, state.Index == 0,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				state.PracticeRecordsMu.Lock() // 加锁
				if state.Index > 0 {
					state.AnswerGenCounter++
					state.Index--
					state.PracticeRecordsMu.Unlock() // 解锁
					refreshPracticeUI()
				} else {
					state.PracticeRecordsMu.Unlock() // 条件不满足时也要解锁
				}
			})

		nextBtn := customElements.CreateButton(
			core.PracticeNextBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true, state.Index >= len(state.CurrentList)-1,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				state.PracticeRecordsMu.Lock() // 加锁
				if state.Index < len(state.CurrentList)-1 {
					state.AnswerGenCounter++
					state.Index++
					state.PracticeRecordsMu.Unlock() // 解锁
					refreshPracticeUI()
				} else {
					state.PracticeRecordsMu.Unlock() // 条件不满足时也要解锁
				}
			})

		// --- 提交按钮（仅多选题显示）---
		submitBtn := customElements.CreateButton(
			core.PracticeSubmitBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true,
			q.Type != "多选题" || strings.Contains(state.Title, "修正题目"),
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if multiSubmitted || len(multiSelected) == 0 {
					return // 已提交或未选择时不可提交
				}
				var userAnsList []string
				for i := range q.Options {
					if multiSelected[i] {
						userAnsList = append(userAnsList, string(rune('A'+i)))
					}
				}
				userAnsStr := strings.Join(userAnsList, ",")
				storageRelated.HandleUserSelectOption(state, q, userAnsStr) // 记录用户答案
				multiSubmitted = true
				isAllCorrect := core.IsAnswerCorrect(userAnsStr, q) // 判断答案是否正确
				refreshPracticeUI()
				if isAllCorrect {
					// 1. 获取并递增生成号（加锁）
					state.PracticeRecordsMu.Lock()
					currentGen := state.AnswerGenCounter + 1
					state.AnswerGenCounter = currentGen
					state.PracticeRecordsMu.Unlock()

					go func(gen int) {
						time.Sleep(time.Duration(core.MultiCorrectDelay) * time.Millisecond)

						state.PracticeRecordsMu.Lock()
						// 2. 校验生成号是否匹配（若用户在这期间点了上下题，此处会被置为 true）
						skip := gen != state.AnswerGenCounter
						if !skip && state.Index < len(state.CurrentList)-1 {
							state.Index++
						}
						state.PracticeRecordsMu.Unlock()

						// 3. 只有未被跳过才刷新 UI
						if !skip {
							fyne.Do(func() {
								refreshPracticeUI()
							})
						}
					}(currentGen) // 将 currentGen 传入闭包
				}
			})

		// 1. 将三个按钮放入标准的水平 HBox 中，并设置适当的按钮间距
		btnSpacing := canvas.NewRectangle(color.Transparent)
		btnSpacing.SetMinSize(fyne.NewSize(12, 1)) // 按钮之间的间距，可根据需要调整

		btnSpacing2 := canvas.NewRectangle(color.Transparent)
		btnSpacing2.SetMinSize(fyne.NewSize(12, 1))

		buttonsRow := container.NewHBox(prevBtn, btnSpacing, submitBtn, btnSpacing2, nextBtn)

		// 2. 用 container.NewCenter 将整个按钮行在父容器中完美居中
		navGrid := container.NewCenter(buttonsRow)

		// --- 题目内容区域 ---
		var questionContent *fyne.Container
		if strings.Contains(state.Title, "修正题目") {
			editForm := RenderCorrectionCard(w, state, q, refreshPracticeUI)
			// 1. 将顶部的元信息和分割线组合为一个 Header 容器
			metaHeader := container.NewVBox(metaRow, widget.NewSeparator())

			// 2. 使用 NewBorder：将 Header 放在 Top，editForm 放在 Center
			// 这样 editForm 才能分到剩余的全部高度，从而把里面的按钮推到底部
			questionContent = container.NewBorder(
				metaHeader, // top: 顶栏元信息
				nil,        // bottom
				nil,        // left
				nil,        // right
				editForm,   // center: 占满下方剩余全部空间
			)
		} else {
			// 👈 同理，移除 spacer 和 navGrid
			questionContent = container.NewVBox(metaRow, widget.NewSeparator(), stemBox, optionsBox)
		}

		// (注：原本在这里声明的 cardBg 和 cardStack 已经被我们移到外层初始化了，请直接删除它们)

		// --- 底部统计信息 ---
		wCount := len(state.CurrentList)

		mCorrect, nWrong := core.CalculateCurrentModeStats(state)

		statsText := fmt.Sprintf("✓: %d             ✕: %d", mCorrect, nWrong)
		statsLbl := canvas.NewText(statsText, core.HexColor(core.TextHintColor))
		statsLbl.TextSize = core.FontSizeDialogMsg

		// 题目导航按钮：点击弹出题目导航弹窗
		stateBtn := customElements.CreateButton(
			fmt.Sprintf("总数: %d", wCount),
			core.ModalCloseBtnWidth, core.NavButtonHeight,
			core.HexColor(core.TextHintColor),
			core.HexColor(core.SectionBgColor),
			core.HexColor(core.SectionBgColor),
			core.StrokeMedium, core.FontSizeDialogMsg,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() { ShowQuestionModal(w, state, refreshPracticeUI) },
		)

		gap := canvas.NewRectangle(color.Transparent)
		gap.SetMinSize(fyne.NewSize(core.NavButtonHeight, 1))
		stateBox := container.NewHBox(stateBtn, gap, container.NewCenter(statsLbl))

		statsBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))
		sizeBox := container.NewGridWrap(fyne.NewSize(400, core.SpacingLarge))
		statsRow := container.NewStack(statsBg, sizeBox, container.NewCenter(stateBox))

		// --- 底部操作按钮 ---
		// 判断是否为考试详情页面
		isExamDetailPage := strings.Contains(state.Title, "考试详情")
		
		returnTypeBtn := customElements.CreateButton(
			core.PracticeReturnTypeBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true, isExamDetailPage,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() { backFunc(w, state) },
		)
		// 删除本题：仅错题练习和收藏练习和修正题目模式下可用，考试详情页面中不可用
		delCurrentDataBtn := customElements.CreateButton(
			core.PracticeDeleteCurrentBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true,
			isExamDetailPage ||
				!strings.Contains(state.Title, "错题练习") &&
				!strings.Contains(state.Title, "修正题目") &&
				!strings.Contains(state.Title, "收藏练习"),
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				customElements.ShowCustomConfirm(
					core.PracticeConfirmDialogTitle,
					"确定",
					"取消",
					customElements.NewCenterRichText(core.PracticeDeleteCurrentConfirmMsg),
					func(confirm bool) bool {
						if confirm {
							q := state.CurrentList[state.Index] // 获取当前题目

							if strings.Contains(state.Title, "修正题目") {
								// 修正题目模式：从 state.Questions 和 state.CurrentList 中删除，并保存到磁盘
								var newQuestions []core.Question
								for _, qq := range state.Questions {
									if qq.ID != q.ID {
										newQuestions = append(newQuestions, qq)
									}
								}
								state.Questions = newQuestions

								var newList []core.Question
								for _, qq := range state.CurrentList {
									if qq.ID != q.ID {
										newList = append(newList, qq)
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
									customElements.ShowCustomInformation("错误", fmt.Sprintf("删除失败: %v", err), w)
								} else {
									customElements.ShowCustomInformation("成功", "本题已从题库中永久删除", w)
									refreshPracticeUI()
								}
							} else if strings.Contains(state.Title, "错题练习") {
								// 错题练习模式：从 WrongSet 中删除，并从 CurrentList 中移除
								delete(state.WrongSet, q.ID)
								storageRelated.SaveWrongSetToLocal(state)

								var newList []core.Question
								for _, qq := range state.CurrentList {
									if qq.ID != q.ID {
										newList = append(newList, qq)
									}
								}
								state.CurrentList = newList

								if state.Index >= len(state.CurrentList) {
									state.Index = len(state.CurrentList) - 1
									if state.Index < 0 {
										state.Index = 0
									}
								}
								refreshPracticeUI()
							} else if strings.Contains(state.Title, "收藏练习") {
								// 收藏练习模式：从 FavSet 中删除，并从 CurrentList 中移除
								delete(state.FavSet, q.ID)
								storageRelated.SaveFavSetToDisk(state)

								var newList []core.Question
								for _, qq := range state.CurrentList {
									if qq.ID != q.ID {
										newList = append(newList, qq)
									}
								}
								state.CurrentList = newList

								if state.Index >= len(state.CurrentList) {
									state.Index = len(state.CurrentList) - 1
									if state.Index < 0 {
										state.Index = 0
									}
								}
								refreshPracticeUI()
							} else {
								// 其他情况，刷新 UI
								refreshPracticeUI()
							}
						}
						return true
					}, w)
			})
		// 删除记录：清空本次练习的所有答题记录和对错统计，考试详情页面中不可用
		delRecordBtn := customElements.CreateButton(
			core.PracticeDeleteRecordBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true,
			isExamDetailPage,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				customElements.ShowCustomConfirm(
					core.PracticeConfirmDialogTitle,
					"确定",
					"取消",
					customElements.NewCenterRichText(core.PracticeDeleteRecordConfirmMsg),
					func(confirm bool) bool {
						if confirm {
							storageRelated.ClearPracticeRecords(state)
							resetQuestionState()
							refreshPracticeUI()
						}
						return true
					}, w)
			})

		actionGrid := container.NewGridWithColumns(3, returnTypeBtn, delCurrentDataBtn, delRecordBtn)
		bottomArea := container.NewVBox(statsRow, actionGrid)

		// 统一更新各个 Shell 区域的内容
		topShell.Objects = []fyne.CanvasObject{topArea}
		fyne.Do(func() { topShell.Refresh() })
		bottomShell.Objects = []fyne.CanvasObject{bottomArea}
		fyne.Do(func() { bottomShell.Refresh() })

		// 👈 [新增] 单独刷新并渲染被固定在底部的导航按钮
		navShell.Objects = []fyne.CanvasObject{container.NewPadded(navGrid)}
		fyne.Do(func() { navShell.Refresh() })

		// 👈 [修改] 滚动区域的 Stack 仅存放纯粹的题目本身
		centerShell.Objects = []fyne.CanvasObject{container.NewPadded(questionContent)}
		fyne.Do(func() { centerShell.Refresh() })
	}

	// 初始渲染
	refreshPracticeUI()
}

// ShowQuestionModal 显示题目导航弹窗，以网格形式展示所有题目并标记状态。
// 题目状态用颜色编码：绿色=正确，红色=错误，蓝色=当前题目，灰色=未作答。
// 点击题目可跳转到对应题号。
// ShowQuestionModal 显示题目导航弹窗，以网格形式展示所有题目并标记状态。
func ShowQuestionModal(w fyne.Window, state *core.AppState, onNavigate func()) {
	grid := container.NewGridWrap(fyne.NewSize(core.ModalGridItemSize, core.ModalGridItemSize))

	var modal *widget.PopUp

	for i := 0; i < len(state.CurrentList); i++ {
		idx := i
		q := state.CurrentList[idx]

		// 1. 计算状态
		isCurrent := (idx == state.Index)

		state.PracticeRecordsMu.Lock()
		historyAns, exists := state.PracticeRecords[q.ID]
		state.PracticeRecordsMu.Unlock()

		isAnswered := exists && historyAns != "null" && historyAns != ""
		isRight := false
		if isAnswered {
			isRight = core.IsAnswerCorrect(historyAns, q)
		}

		// 2. 根据状态决定动态颜色
		bgColor := core.HexColor(core.CardBgColor)
		borderColor := core.HexColor(core.BorderMediumColor)

		if isCurrent {
			bgColor = core.HexColor(core.ColorSelectedBg)
			borderColor = core.HexColor(core.ColorSelectedBorder)
		} else if isAnswered {
			if isRight {
				bgColor = core.HexColor(core.ColorCorrectBg)
				borderColor = core.HexColor(core.ColorCorrectBorder)
			} else {
				bgColor = core.HexColor(core.ColorWrongBg)
				borderColor = core.HexColor(core.ColorWrongBorder)
			}
		}

		// 3. 计算题型简称
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
			runes := []rune(q.Type)
			if len(runes) > 0 {
				shortType = string(runes[0])
			}
		}

		// 4. 创建按状态渲染颜色的按钮（注意允许换行 TextWrapWord）
		lblText := fmt.Sprintf("%d\n%s", idx+1, shortType)
		fmt.Println("Creating button for question index:", idx, "with label:", lblText, "bgColor:", bgColor, "borderColor:", borderColor)
		box := customElements.CreateButton(
			"", // 👈 不传文字，避免底层单行组件干扰
			core.ModalGridItemSize, core.ModalGridItemSize,
			core.HexColor(core.TextBodyColor),
			bgColor,
			borderColor,
			core.StrokeMedium, core.FontSizeSubtitle,
			false, false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,
			fyne.TextTruncateOff,
			func() {
				state.PracticeRecordsMu.Lock()
				state.Index = idx
				state.PracticeRecordsMu.Unlock()

				onNavigate()
				if modal != nil {
					modal.Hide()
				}
			},
		)

		// 5. 用绝对布局或 Stack 在按钮正上方精确绘制“数字”和“题型”两行文字
		contentBox := container.NewWithoutLayout()
		contentBox.Resize(fyne.NewSize(core.ModalGridItemSize, core.ModalGridItemSize))

		// 上面的数字
		numText := canvas.NewText(fmt.Sprintf("%d", idx+1), core.HexColor(core.TextBodyColor))
		numText.TextSize = 14 // 可以写死一个绝对安全的字号
		numText.TextStyle = fyne.TextStyle{Bold: true}
		numText.Resize(numText.MinSize())

		// 下面的题型
		typeText := canvas.NewText(shortType, core.HexColor(core.TextBodyColor))
		typeText.TextSize = 12 // 比数字稍微小一点
		typeText.Resize(typeText.MinSize())

		// 计算居中坐标与紧凑间距（总高 50）
		totalH := numText.MinSize().Height + typeText.MinSize().Height + 1 // 1像素间距
		startY := (core.ModalGridItemSize - totalH) / 2

		numX := (core.ModalGridItemSize - numText.MinSize().Width) / 2
		numText.Move(fyne.NewPos(numX, startY))

		typeX := (core.ModalGridItemSize - typeText.MinSize().Width) / 2
		typeText.Move(fyne.NewPos(typeX, startY+numText.MinSize().Height+1))

		contentBox.Add(numText)
		contentBox.Add(typeText)

		// 6. 将底层带颜色的按钮和上层的两行文字组合在一起
		cellContainer := container.NewStack(box, contentBox)
		grid.Add(cellContainer)
	}

	// 滚动容器
	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(320, 360))

	// 标题
	titleText := canvas.NewText(core.PracticeQuestionModalTitle, core.HexColor(core.TextPrimaryColor))
	titleText.TextSize = core.FontSizeDialogMsg
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewCenter(titleText)

	// 关闭按钮
	btnClose := customElements.CreateButton(
		core.PracticeModalCloseBtnText,
		core.ModalCloseBtnWidth, core.ModalCloseBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium, core.FontSizeSubtitle,
		true, false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		func() {
			if modal != nil {
				modal.Hide()
			}
		},
	)

	// 使用 Border 布局：顶部标题，底部按钮，中间 Scroll 自动撑满
	modalContent := container.NewBorder(
		container.NewPadded(titleContainer),
		container.NewPadded(container.NewCenter(btnClose)),
		nil, nil,
		scroll,
	)

	modalBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	modalBg.CornerRadius = core.LargeCardCorner
	modalBg.StrokeWidth = core.StrokeThin
	modalBg.StrokeColor = core.HexColor(core.BorderLightColor)

	styledModal := container.NewStack(modalBg, container.NewPadded(modalContent))

	modal = widget.NewModalPopUp(styledModal, w.Canvas())
	modal.Show()
}
