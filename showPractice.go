package main

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ==================== 答题/练习主界面 ====================
func showPractice(w fyne.Window, state *AppState, backFunc func(fyne.Window, *AppState)) {
	// 用于存放临时收藏状态的 map（未来可集成到 state 中存入本地持久化）

	var refreshPracticeUI func()

	// ==================== 题目的瞬态 UI 状态 ====================
	var currentQID string
	var isMemorizeRevealed bool           // 填空/问答：是否点击了显示答案
	var multiSelected map[int]bool        // 多选题：记录哪些选项被点亮了 (浅蓝色)
	var singleWrongClicked map[int]bool   // 单选/判断：记录点错了的选项 (标红)
	var singleCorrectClicked map[int]bool // 单选/判断：记录点对了的选项 (标绿)
	var multiSubmitted bool               // 多选题：记录是否按下了提交按钮

	// ==================== ✨ 核心优化 1：在闭包外定义三个固定的“外壳容器” ====================
	topShell := container.NewStack()
	bottomShell := container.NewStack()
	centerShell := container.NewStack()

	// ✨ 提前组装好带滑动探测功能的智能滚动条 (只创建这一次！)
	centerScroll := container.NewVScroll(nil)
	touchArea := NewTouchInterceptor(centerShell, centerScroll,
		func() { // 左滑上一题
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		},
		func() { // 右滑下一题
			if state.Index < len(state.CurrentList)-1 {
				state.Index++
				refreshPracticeUI()
			}
		},
	)
	centerScroll.Content = touchArea

	// ✨ 提前组装好整个页面的大框架，并直接设置给 Window
	pageBg := canvas.NewRectangle(hexColor("#f0f2f5"))
	mainContent := container.NewBorder(topShell, bottomShell, nil, nil, centerScroll)
	mainLayout := container.NewStack(pageBg, mainContent)

	marginBox := canvas.NewRectangle(color.Transparent)
	marginBox.SetMinSize(fyne.NewSize(1, 12)) // 💡 12 是间距高度，如果你想要更大可以改成 16 或 20
	//maincontent := container.NewBorder(getTitle(), marginBox, nil, nil, mainLayout)

	// 整个页面的根节点
	rootLayout := container.NewBorder(getTitle(), marginBox, nil, nil, mainLayout)
	w.SetContent(rootLayout) // 🎯 整个页面生命周期中，SetContent 只执行这绝无仅有的一次
	// 定义主布局外壳
	//mainLayout := container.NewStack()

	refreshPracticeUI = func() {
		//fmt.Println("start refreshPracticeUI")
		if state.Index < 0 || state.Index >= len(state.CurrentList) {
			return
		}
		q := state.CurrentList[state.Index]

		// -------------------------------------------------------------------
		// 任务 1: 顶部导航栏 (补齐“题目列表”右侧按钮)
		// -------------------------------------------------------------------

		backBtn := getBox("← 返回", 90, 35, &boxColor{textColor: "#000000", bgColor: "#F0F2F5", strokeColor: "#F0F2F5", textSize: 18}, false, func() { showHome(w, state) })

		// ==================== 2. 题目列表按钮调整 ====================
		listBtn := getBox("题目列表", 90, 35, &boxColor{textColor: "#ffffff", bgColor: "#5D5D5D", strokeColor: "#5D5D5D", textSize: 18},
			false, func() { showQuestionModal(w, state, refreshPracticeUI) })

		titleLbl := canvas.NewText(state.Title, hexColor("#111111"))
		titleLbl.TextStyle = fyne.TextStyle{Bold: true}
		titleLbl.TextSize = 16

		// 1. ✨ 给标题本身加一个基础盒子，确保它不会受两侧按钮挤压，居中更稳固
		titleCenter := container.NewCenter(titleLbl)
		//titleCenter := container.NewGridWrap(fyne.NewSize(100, 30), container.NewCenter(titleLbl))

		// 使用 Border 布局：左放返回，右放题目列表，中间居中显示模式标题
		topNav := container.NewBorder(nil, nil, backBtn, listBtn, titleCenter)

		// 2. ✨ 给整个导航条（topNav）包裹一层标准的 Padded 边距，拉开四周呼吸感
		topNavPadded := container.NewPadded(topNav)

		// 1. 设置 Label：允许换行，且文字始终保持中心对齐
		bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
		bankNameText.Alignment = fyne.TextAlignCenter // 🎯 保证文字在两行时也是居中的
		bankNameText.Wrapping = fyne.TextWrapBreak    // 允许换行
		bankNameText.SizeName = theme.SizeNameWindowButtonHeight

		bankNameBg := canvas.NewRectangle(hexColor("#f7f7f7"))
		bankNameBg.CornerRadius = 6
		bankNameBg.StrokeColor = hexColor("#eeeeee")
		bankNameBg.StrokeWidth = 1

		// 2. 用 Padded 撑开文字和卡片边缘的距离（上下左右的呼吸感）
		cardContent := container.NewPadded(bankNameText)

		// 3. 组合背景和内容
		rawFileNameCard := container.NewStack(
			bankNameBg,
			cardContent,
		)

		// 4. ✨ 核心修正：取消 container.NewCenter！
		// 直接再套一层 Padded 即可。
		// 因为放在 VBox 里，它会自动撑满屏幕宽度，加上 Padded 就会在屏幕左右两边留出好看的间距。
		fileNameCard := container.NewPadded(rawFileNameCard)

		// 终极组装 topArea
		topArea := container.NewVBox(
			topNavPadded,
			fileNameCard,
		)

		// -------------------------------------------------------------------
		// 任务 2: 题目标签行 (全局序号优化版：使用 Question.ID)
		// -------------------------------------------------------------------
		// ✨ 优化：直接使用题目自带的 ID 作为全局序号，避免遍历匹配
		globalIndexStr := "0"
		if q.ID != "" {
			globalIndexStr = q.ID
		}

		metaLeftStr := fmt.Sprintf("总%s题  %s", globalIndexStr, q.Type)
		metaLeft := canvas.NewText(metaLeftStr, hexColor("#666666"))
		metaLeft.TextSize = 13

		// 收藏按钮 (☆ -> ⭐)
		isFav := state.FavSet[q.ID]
		starChar, starColor := "☆", "#999999"
		if isFav {
			starChar, starColor = "⭐", "#999999" //"#fadb14"
		}
		starText := canvas.NewText(starChar, hexColor(starColor))
		starText.TextSize = 18

		starBtn := NewClickableBox(container.NewCenter(starText), func() {
			// 切换状态
			state.FavSet[q.ID] = !state.FavSet[q.ID]

			// 🔥 核心逻辑：触发自动写盘保存！
			saveSetToLocal(state, "收藏集", state.FavSet)

			refreshPracticeUI() // 点击后立即重绘更新状态
		})

		// 难度标签 (假设默认是"普通")
		diffText := canvas.NewText("难度: 普通", hexColor("#666666"))
		diffText.TextSize = 12
		diffBg := canvas.NewRectangle(hexColor("#f0f0f0"))
		diffBg.CornerRadius = 4
		diffTag := container.NewStack(diffBg, container.NewPadded(diffText))

		metaRight := container.NewHBox(starBtn, diffTag)
		metaRow := container.NewBorder(nil, nil, metaLeft, metaRight, nil)

		// -------------------------------------------------------------------
		// 🌟 状态重置与历史记录恢复机制 (解决翻页或重新打开后痕迹消失的问题)
		// -------------------------------------------------------------------
		break_record := func() {
			// 1. 初始化背题模式状态
			isMemorizeRevealed = false

			// 2. 初始化单选/多选的临时控制状态
			multiSelected = make(map[int]bool)
			singleWrongClicked = make(map[int]bool)
			singleCorrectClicked = make(map[int]bool)
			multiSubmitted = false
		}

		if q.ID != currentQID {
			currentQID = q.ID

			break_record()

			// 3. ✨【核心新增】：从已存的本地记录中提取本题的历史作答状态
			if historyAns, exists := state.PracticeRecords[q.ID]; exists && historyAns != "null" {
				if q.Type == "填空题" || q.Type == "问答题" {
					// 填空/问答：如果历史状态是已查看，直接展开标准答案
					isMemorizeRevealed = true
				} else if q.Type == "多选题" {
					// 多选题：解开逗号分隔的历史答案字符串 (如 "A,B")，还原点亮状态
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
					// 历史做过的题，直接进入“已提交”的红绿结算渲染状态
					multiSubmitted = true
				} else {
					// 单选/判断题：历史选择的字母 (如 "A")
					if len(historyAns) == 1 {
						clickedIdx := int(historyAns[0] - 'A')

						// 校验这个历史选择是否属于标准答案中的任意一个
						isHistoryCorrect := false
						for _, correctLetter := range q.Answers {
							if correctLetter == historyAns {
								isHistoryCorrect = true
								break
							}
						}

						// 根据对错，把历史痕迹直接注入渲染映射集中
						if isHistoryCorrect {
							singleCorrectClicked[clickedIdx] = true
						} else {
							singleWrongClicked[clickedIdx] = true

							// 💡 优化：既然用户以前答错了，为了良好体验，顺便把正确的那个选项也标绿指引出来
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

		// -------------------------------------------------------------------
		// 任务 3: 题干区 (清洗隐藏换行符，彻底消除脱节)
		// -------------------------------------------------------------------
		stemText := fmt.Sprintf("%d. %s", state.Index+1, q.Content)

		// ✨ 终极正解：使用 Fyne 富文本自带的“副标题”样式
		// RichTextStyleSubHeading 在 Fyne 的底层主题中被硬编码为“大号字体 + 加粗”
		stemLbl := widget.NewRichText(
			&widget.TextSegment{
				Style: widget.RichTextStyleSubHeading, // 🎯 调大字号且加粗的核心
				Text:  stemText,
			},
		)

		// 🎯 开启富文本的强制换行保护，防止长题目被截断
		stemLbl.Wrapping = fyne.TextWrapBreak

		// ==================== 选项渲染与交互逻辑 ====================
		optionsBox := container.NewVBox()

		marginBox := canvas.NewRectangle(color.Transparent)
		marginBox.SetMinSize(fyne.NewSize(1, 15)) // 💡 12 是间距高度，如果你想要更大可以改成 16 或 20
		optionsBox.Add(marginBox)
		marginBox.SetMinSize(fyne.NewSize(1, 5))

		if q.Type == "填空题" || q.Type == "问答题" {
			// ✨ 逻辑 1: 背题模式
			btnText := "点击显示答案"
			if isMemorizeRevealed {
				btnText = "点击隐藏答案"
			}

			lblA := canvas.NewText(btnText, hexColor("#333333"))
			lblA.TextSize = 18 // 调大字体
			lblA.Alignment = fyne.TextAlignCenter
			lblA.TextStyle = fyne.TextStyle{Bold: true}

			bgA := canvas.NewRectangle(hexColor("#f0f0f0"))
			if isMemorizeRevealed {
				bgA.FillColor = hexColor("#e6f7ff") // 展开时给一个浅蓝色反馈
			}
			bgA.CornerRadius = 6

			// ✨ 注意：canvas.Text 没有 Alignment 属性，必须通过包裹 container.NewCenter 来居中
			cardA := NewClickableBox(container.NewStack(bgA, container.NewPadded(container.NewCenter(lblA))), func() {
				isMemorizeRevealed = !isMemorizeRevealed
				refreshPracticeUI()
			})
			optionsBox.Add(cardA)

			// 只有点击了显示，才渲染 B 选项 (标准答案)
			if isMemorizeRevealed {

				ansStr := ""
				if len(q.Options) > 0 {
					ansStr = q.Options[0].Text // 取出带有顿号的完整文本
				}

				lblB := createOptionLabel(ansStr)

				bgB := canvas.NewRectangle(hexColor("#E8F5E8"))
				bgB.StrokeColor = hexColor("#06a050") // 绿色边框高亮
				bgB.StrokeWidth = 2
				bgB.CornerRadius = 6

				cardB := container.NewStack(bgB, container.NewPadded(lblB))
				optionsBox.Add(cardB)

				// 顺便记录一下“已查看”，确保答题记录文件有进度
				handleUserSelectOption(state, q, "已查看")
			}

		} else {
			// ✨ 逻辑 2 & 3: 单选、多选、判断题

			// 🎯 新增核心逻辑：判断当前单选/判断题是否已经作答（只要有对/错记录就算已答）
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

				contentLbl := createOptionLabel(optStr)
				contentLbl.Alignment = fyne.TextAlignCenter
				contentLbl.Wrapping = fyne.TextWrapBreak

				rightIconText := canvas.NewText("", hexColor("#ffffff"))
				rightIconText.TextSize = 16
				rightIconText.TextStyle = fyne.TextStyle{Bold: true}

				// ==================== 选项颜色与边框动态渲染核心 ====================
				bgColorStr := "#f9f9f9"     // 默认底色
				strokeColorStr := "#e8e8e8" // 默认边框色（淡灰色）
				iconColorStr := "#ffffff"   // 默认图标色
				greenColor := "#06a050"

				// 检查本选项是否在标准答案中
				isCorrectAnswer := false
				for _, a := range q.Answers {
					if a == optLetter {
						isCorrectAnswer = true
						break
					}
				}

				// 根据状态变更颜色和图标
				if q.Type == "多选题" {
					if multiSubmitted {
						if isCorrectAnswer {
							// 只要是正确答案，背景一律变为正确色
							bgColorStr = "#E8F5E8"

							if multiSelected[optIndex] {
								rightIconText.Text = "✓"
								iconColorStr = greenColor
								strokeColorStr = greenColor
							} else {
								rightIconText.Text = "✕"
								iconColorStr = "#ff4d4f"
								strokeColorStr = "#ff4d4f"
							}
						} else {
							if multiSelected[optIndex] {
								bgColorStr = "#FFEBEE"
								rightIconText.Text = "✕"
								iconColorStr = "#ff4d4f"
								strokeColorStr = "#ff4d4f"
							}
						}
					} else if multiSelected[optIndex] {
						bgColorStr = "#e6f7ff" // 未提交前的浅蓝色选定状态
					}
				} else {
					// 单选、判断
					if singleWrongClicked[optIndex] {
						bgColorStr = "#FFEBEE"
						rightIconText.Text = "✕"
						iconColorStr = "#ff4d4f"
						strokeColorStr = "#ff4d4f"
					}
					if singleCorrectClicked[optIndex] {
						bgColorStr = "#E8F5E8"
						rightIconText.Text = "✓"
						iconColorStr = greenColor
						strokeColorStr = greenColor
					}
				}

				rightIconText.Color = hexColor(iconColorStr)

				optLayout := container.NewBorder(nil, nil, prefixLbl, container.NewCenter(rightIconText), contentLbl)

				optBg := canvas.NewRectangle(hexColor(bgColorStr))
				optBg.CornerRadius = 6
				optBg.StrokeColor = hexColor(strokeColorStr)
				optBg.StrokeWidth = 1

				// == 封装点击事件 ==
				optCard := NewClickableBox(container.NewStack(optBg, container.NewPadded(optLayout)), func() {

					// 🎯 核心锁定拦截
					if q.Type == "多选题" && multiSubmitted {
						return // 多选题一旦提交，全盘锁定
					}
					if q.Type != "多选题" && hasSingleAnswered {
						return // 单选/判断一旦点过（无论对错），全盘锁定
					}

					if q.Type == "多选题" {
						multiSelected[optIndex] = !multiSelected[optIndex]
						refreshPracticeUI()
					} else {
						// 单选/判断：实时判定
						handleUserSelectOption(state, q, optLetter)

						if isCorrectAnswer {
							singleCorrectClicked[optIndex] = true
							refreshPracticeUI() // 瞬间变绿

							// 🌟 开启协程，延迟后自动跳下一题
							go func() {
								time.Sleep(350 * time.Millisecond)
								if state.Index < len(state.CurrentList)-1 {
									state.Index++
								}
								refreshPracticeUI()
							}()
						} else {
							singleWrongClicked[optIndex] = true

							// 💡 优化：既然用户答错了且选项被锁定，直接把正确的答案标绿提示出来
							for _, correctLetter := range q.Answers {
								if len(correctLetter) == 1 {
									cIdx := int(correctLetter[0] - 'A')
									singleCorrectClicked[cIdx] = true
								}
							}

							refreshPracticeUI() // 变红，留在本题
						}
					}
				})

				optionsBox.Add(optCard)

				optionsBox.Add(marginBox)
			}
		}

		// -------------------------------------------------------------------
		// 任务 4: 答题控制按钮组
		// -------------------------------------------------------------------

		// 渲染“上一题”：直接将 state.Index == 0 作为禁用条件传入
		prevBtn := getBox("上一题", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#418BEC", strokeColor: "#418BEC", textSize: 18}, state.Index == 0, func() {
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		})

		//nextBtn := widget.NewButton("下一题", func() {
		//	if state.Index < len(state.CurrentList)-1 {
		//		state.Index++
		//		refreshPracticeUI()
		//	}
		//})
		//if state.Index == len(state.CurrentList)-1 {
		//	nextBtn.Disable()
		//}
		// 渲染“下一题”：将 state.Index == 最后一题 作为禁用条件传入
		nextBtn := getBox("下一题", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#418BEC", strokeColor: "#418BEC", textSize: 18},
			state.Index >= len(state.CurrentList)-1, func() {
				if state.Index < len(state.CurrentList)-1 {
					state.Index++
					refreshPracticeUI()
				}
			})

		submitBtn := getBox("提交", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#418BEC", strokeColor: "#418BEC", textSize: 18},
			q.Type != "多选题" || strings.Contains(state.Title, "修正题目"), func() {
				if multiSubmitted || len(multiSelected) == 0 {
					return
				}
				// 1. 收集用户点了哪些选项
				var userAnsList []string
				for i := range q.Options {
					if multiSelected[i] {
						userAnsList = append(userAnsList, string(rune('A'+i)))
					}
				}
				// 2. 将数组连成 "A,B,C" 落盘打分
				userAnsStr := strings.Join(userAnsList, ",")
				handleUserSelectOption(state, q, userAnsStr)
				// 3. 标记为已提交，并判定是否全对
				multiSubmitted = true
				sort.Strings(userAnsList)
				correctList := make([]string, len(q.Answers))
				copy(correctList, q.Answers)
				sort.Strings(correctList)
				isAllCorrect := len(userAnsList) == len(correctList)
				if isAllCorrect {
					for i := range userAnsList {
						if userAnsList[i] != correctList[i] {
							isAllCorrect = false
							break
						}
					}
				}
				refreshPracticeUI() // 瞬间重绘，显示红绿校验结果
				// 4. 如果全对，延迟翻页
				if isAllCorrect {
					go func() {
						time.Sleep(500 * time.Millisecond) // 多选题看结果的时间稍长一点
						if state.Index < len(state.CurrentList)-1 {
							state.Index++
						}
						refreshPracticeUI()
					}()
				}
			})

		navGrid := container.NewGridWithColumns(3, prevBtn, submitBtn, nextBtn)

		// ==================== 核心接入点 ====================
		var questionContent *fyne.Container

		if strings.Contains(state.Title, "修正题目") {
			// 调用我们写好的专属修正表单，传入刷新回调
			editForm := renderCorrectionCard(w, state, q, refreshPracticeUI)

			// 组装修正卡片
			questionContent = container.NewVBox(
				metaRow, // 依然保留顶部的：总第X题、收藏⭐、难度
				widget.NewSeparator(),
				editForm, // 塞入修正表单
				layout.NewSpacer(),
				navGrid, // 依然保留底部的：上一题、下一题
			)
		} else {
			// 原有的普通刷题内容组装
			questionContent = container.NewVBox(
				metaRow,
				widget.NewSeparator(),
				stemLbl,
				optionsBox,
				layout.NewSpacer(), // 隔开选项和按钮
				navGrid,
			)
		}

		// 给中间卡片整体加一个白底圆角框 (还原小程序视觉)
		cardBg := canvas.NewRectangle(hexColor("#ffffff"))
		cardBg.CornerRadius = 8
		cardBg.StrokeColor = hexColor("#e8e8e8")
		cardBg.StrokeWidth = 1
		cardStack := container.NewStack(cardBg, container.NewPadded(questionContent))

		// 允许长题目内容滚动
		//centerShell = container.NewPadded(cardStack)
		//centerScroll := container.NewScroll(container.NewPadded(cardStack))
		// ✨ 核心接入：使用智能滑动滚动条替换普通的 container.NewScroll
		// 1. ✨ 先创建一个空的外层垂直滚动条
		//centerScroll := container.NewVScroll(nil)
		//touchArea := NewTouchInterceptor(container.NewPadded(cardStack), centerScroll,
		//	//centerScroll := NewSwipeScroll(container.NewPadded(cardStack),
		//	// 👈 触发向左滑动时的动作（上一题）
		//	func() {
		//		if state.Index > 0 {
		//			state.Index--
		//			refreshPracticeUI()
		//		}
		//	},
		//	// 👉 触发向右滑动时的动作（下一题）
		//	func() {
		//		if state.Index < len(state.CurrentList)-1 {
		//			state.Index++
		//			refreshPracticeUI()
		//		}
		//	},
		//)

		// 3. ✨ 将配置好翻页逻辑的探测器，正式塞入滚动条中
		//centerScroll.Content = touchArea

		// -------------------------------------------------------------------
		// 任务 5: 底部统计与操作区 (全自动化持久化版)
		// -------------------------------------------------------------------
		wCount := len(state.CurrentList) // 当前题型总数
		// mCorrect := state.PracticeCorrectCount // 本次练习做对的题数
		// nWrong := state.PracticeWrongCount     // 本次练习做错的题数
		// 动态获取当前模式下的对错总数
		mCorrect, nWrong := calculateCurrentModeStats(state)

		statsText := fmt.Sprintf("总数: %d             ✓: %d             ✕: %d", wCount, mCorrect, nWrong)
		statsLbl := canvas.NewText(statsText, hexColor("#555555"))
		statsLbl.TextSize = 16

		statsBg := canvas.NewRectangle(hexColor("#f7f7f7"))
		sizeBox := container.NewGridWrap(fyne.NewSize(400, 36))
		statsRow := container.NewStack(statsBg, sizeBox, container.NewCenter(statsLbl))

		returnTypeBtn := getBox("返回题型", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#5D5D5D", strokeColor: "#5D5D5D", textSize: 18}, false, func() {
			backFunc(w, state)
		})
		// ✨ 新增：“删除本题”按钮
		delCurrentDataBtn := getBox("删除本题", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#5D5D5D", strokeColor: "#5D5D5D", textSize: 18},
			!strings.Contains(state.Title, "错题练习") && !strings.Contains(state.Title, "修正题目"), func() {
				dialog.ShowConfirm("提示", "确定要将本题从当前练习集中移除吗？", func(confirm bool) {
					if confirm {
						// 这里的具体删除逻辑一会讨论（例如从 state.CurrentList 中移除当前 Index 题目）
						refreshPracticeUI()
					}
				}, w)
			})

		// 💡 核心控制：只有在“错题练习”或“修正题目”模式下才激活该按钮
		// 判定标准根据你传入 state.Title 的文本或专门的模式状态字段
		delRecordBtn := getBox("删除记录", 100, 35, &boxColor{textColor: "#ffffff", bgColor: "#5D5D5D", strokeColor: "#5D5D5D", textSize: 18}, false, func() {
			dialog.ShowConfirm("提示", "确定要清空本次练习的所有答题记录和对错统计吗？", func(confirm bool) {
				if confirm {
					clearPracticeRecords(state)
					break_record()
					refreshPracticeUI()
				}
			}, w)
		})

		// ✨ 修正为 3 列网格，将三个按钮并排规整显示
		actionGrid := container.NewGridWithColumns(3, returnTypeBtn, delCurrentDataBtn, delRecordBtn)
		bottomArea := container.NewVBox(statsRow, actionGrid)

		// -------------------------------------------------------------------
		// 最终组装
		// -------------------------------------------------------------------
		//pageBg := canvas.NewRectangle(hexColor("#f0f2f5"))

		topShell.Objects = []fyne.CanvasObject{topArea}
		topShell.Refresh()

		bottomShell.Objects = []fyne.CanvasObject{bottomArea}
		bottomShell.Refresh()

		centerShell.Objects = []fyne.CanvasObject{container.NewPadded(cardStack)}
		centerShell.Refresh()

		// 使用 Border 布局：Top 放导航栏，Bottom 放底栏，中间 Center 让题目卡片自适应填充并滚动
		//mainContent := container.NewBorder(topArea, bottomArea, nil, nil, centerScroll)
		//
		//mainLayout.Objects = []fyne.CanvasObject{pageBg, mainContent}
		//mainLayout.Refresh()
		//fmt.Println("finish refreshPracticeUI")
	}

	// ✨ 核心接入：用滑动容器包裹主内容，并注入翻页逻辑
	//swipeableContent := NewSwipeableBox(maincontent,
	//	// 👈 触发向左滑动时的动作（按你的要求：上一题）
	//	func() {
	//		if state.Index > 0 {
	//			state.Index--
	//			refreshPracticeUI()
	//		}
	//	},
	//	// 👉 触发向右滑动时的动作（按你的要求：下一题）
	//	func() {
	//		if state.Index < len(state.CurrentList)-1 {
	//			state.Index++
	//			refreshPracticeUI()
	//		}
	//	},
	//)

	// 触发首次渲染
	refreshPracticeUI()
	//w.SetContent(maincontent)
}

// ==================== 题目导航列表弹窗 ====================
func showQuestionModal(w fyne.Window, state *AppState, onNavigate func()) {
	grid := container.NewGridWrap(fyne.NewSize(50, 50))

	var d dialog.Dialog

	for i := 0; i < len(state.CurrentList); i++ {
		idx := i
		q := state.CurrentList[idx]

		bg := canvas.NewRectangle(theme.InputBackgroundColor())
		bg.CornerRadius = 6
		bg.StrokeWidth = 2
		bg.StrokeColor = theme.DisabledColor()

		isCurrent := (idx == state.Index)

		// 从最新的历史记录映射中提取作答情况
		historyAns, exists := state.PracticeRecords[q.ID]
		isAnswered := exists && historyAns != "null" && historyAns != ""

		isRight := false
		if isAnswered {
			if q.Type == "填空题" || q.Type == "问答题" {
				// 填空/问答题在我们的逻辑中，点开“已查看”即视为完成（算作正确/绿色）
				isRight = true
			} else if q.Type == "多选题" {
				// 多选题需要严谨的切片排序比对
				userAnsSlice := strings.Split(historyAns, ",")
				for j := range userAnsSlice {
					userAnsSlice[j] = strings.TrimSpace(userAnsSlice[j])
				}
				sort.Strings(userAnsSlice)

				correctAnsSlice := make([]string, len(q.Answers))
				copy(correctAnsSlice, q.Answers)
				sort.Strings(correctAnsSlice)

				if len(userAnsSlice) == len(correctAnsSlice) {
					matchCount := 0
					for j := range userAnsSlice {
						if userAnsSlice[j] == correctAnsSlice[j] {
							matchCount++
						}
					}
					if matchCount == len(correctAnsSlice) {
						isRight = true
					}
				}
			} else {
				// 单选题、判断题直接比对即可
				for _, ans := range q.Answers {
					if historyAns == ans {
						isRight = true
						break
					}
				}
			}
		}

		// ✨ 按照优先级应用界面颜色
		if isCurrent {
			bg.FillColor = hexColor("#e6f7ff")   // 当前所在题目底色
			bg.StrokeColor = hexColor("#1890ff") // 当前题目赋予深蓝色边框使其醒目
		} else if isAnswered {
			if isRight {
				bg.FillColor = hexColor("#E8F5E8")   // 答对的底色
				bg.StrokeColor = hexColor("#5aeb55") // 绿边框
			} else {
				bg.FillColor = hexColor("#FFEBEE")   // 答错的底色
				bg.StrokeColor = hexColor("#ff4d4f") // 红边框
			}
		}

		// ---------------------------------------------------------
		// ✨ 新增：提取题型简称
		// ---------------------------------------------------------
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
				shortType = string([]rune(q.Type)[0]) // 兜底：取第一个字
			}
		}

		// ✨ 修改：将题号和题型简称用换行符 \n 拼接在一块显示
		lblText := fmt.Sprintf("%d\n%s", idx+1, shortType)
		lbl := widget.NewLabel(lblText)
		lbl.Alignment = fyne.TextAlignCenter

		box := NewClickableBox(container.NewStack(bg, lbl), func() {
			state.Index = idx
			onNavigate()
			d.Hide()
		})
		grid.Add(box)
	}

	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(320, 400))

	d = dialog.NewCustom("题目导航", "关闭", scroll, w)
	d.Show()
}
