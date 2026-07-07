package main

import (
	"fmt"
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

	// 定义主布局外壳
	mainLayout := container.NewStack()

	refreshPracticeUI = func() {
		if state.Index < 0 || state.Index >= len(state.CurrentList) {
			return
		}
		q := state.CurrentList[state.Index]

		// -------------------------------------------------------------------
		// 任务 1: 顶部导航栏 (补齐“题目列表”右侧按钮)
		// -------------------------------------------------------------------
		backBtn := widget.NewButton("← 返回", func() { showHome(w, state) })

		// ✨ 修复问题1：重新加回右侧的“题目列表”按钮
		listBtn := widget.NewButton("题目列表", func() {
			showQuestionModal(w, state, refreshPracticeUI)
		})

		titleLbl := canvas.NewText(state.Title, hexColor("#111111"))
		titleLbl.TextStyle = fyne.TextStyle{Bold: true}
		titleLbl.TextSize = 16

		// 使用 Border 布局：左放返回，右放题目列表，中间居中显示模式标题
		topNav := container.NewBorder(nil, nil, backBtn, listBtn, container.NewCenter(titleLbl))

		bankNameText := canvas.NewText("📖 "+state.CurrentFileName, hexColor("#333333"))
		bankNameText.TextSize = 13
		bankNameBg := canvas.NewRectangle(hexColor("#f7f7f7"))
		bankNameBg.CornerRadius = 6
		bankNameBg.StrokeColor = hexColor("#eeeeee")
		bankNameBg.StrokeWidth = 1
		fileNameCard := container.NewStack(
			bankNameBg,
			container.NewPadded(container.NewCenter(bankNameText)),
		)
		topArea := container.NewVBox(topNav, container.NewPadded(fileNameCard))

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

		//practiceMode, _, _ := strings.Cut(state.Title, " - ")

		//if practiceMode == "修正题目" {
		//	correctionCard := renderCorrectionCard(w, state)
		//} else {

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
		// fmt.Println("stemText: ", stemText)
		stemLbl := widget.NewLabel(stemText)
		stemLbl.TextStyle = fyne.TextStyle{Bold: true}
		stemLbl.Wrapping = fyne.TextWrapBreak

		// ==================== 选项渲染与交互逻辑 ====================
		optionsBox := container.NewVBox()

		if q.Type == "填空题" || q.Type == "问答题" {
			// ✨ 逻辑 1: 背题模式
			btnText := "点击显示答案"
			if isMemorizeRevealed {
				btnText = "点击隐藏答案"
			}

			lblA := widget.NewLabel(btnText)
			lblA.Alignment = fyne.TextAlignCenter
			lblA.TextStyle = fyne.TextStyle{Bold: true}

			bgA := canvas.NewRectangle(hexColor("#f0f0f0"))
			if isMemorizeRevealed {
				bgA.FillColor = hexColor("#e6f7ff") // 展开时给一个浅蓝色反馈
			}
			bgA.CornerRadius = 6

			cardA := NewClickableBox(container.NewStack(bgA, container.NewPadded(lblA)), func() {
				isMemorizeRevealed = !isMemorizeRevealed
				refreshPracticeUI() // 重绘，触发选项 B 的显示或隐藏
			})
			optionsBox.Add(cardA)

			// 只有点击了显示，才渲染 B 选项 (标准答案)
			if isMemorizeRevealed {

				ansStr := ""
				if len(q.Options) > 0 {
					ansStr = q.Options[0].Text // 取出带有顿号的完整文本
				}

				lblB := widget.NewLabel(ansStr)
				if q.Type == "填空题" {
					lblB.Alignment = fyne.TextAlignCenter
				}
				lblB.Wrapping = fyne.TextWrapBreak

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
			for i, opt := range q.Options {
				optIndex := i
				optStr := opt.Text
				if optStr == "" {
					continue
				}

				optLetter := string(rune('A' + optIndex))
				optPrefix := optLetter + "."

				prefixLbl := widget.NewLabel(optPrefix)
				prefixLbl.TextStyle = fyne.TextStyle{Bold: true}

				contentLbl := widget.NewLabel(optStr)
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
							// 只要是正确答案，背景一律变为正确色 (#5aeb55)
							bgColorStr = "#E8F5E8"

							if multiSelected[optIndex] {
								// 1. 已选择的正确选项
								rightIconText.Text = "✓"
								iconColorStr = greenColor   // 绿勾
								strokeColorStr = greenColor // 绿边框
							} else {
								// 2. 没选择的正确选项（漏选）
								rightIconText.Text = "✕"
								iconColorStr = "#ff4d4f"   // 红叉
								strokeColorStr = "#ff4d4f" // 红边框
							}
						} else {
							if multiSelected[optIndex] {
								// 3. 已选择的错误选项（错选）
								bgColorStr = "#FFEBEE" // 错误背景色
								rightIconText.Text = "✕"
								iconColorStr = "#ff4d4f"   // 红叉
								strokeColorStr = "#ff4d4f" // 红边框
							}
							// 4. 不是正确答案且用户没选的，保持默认 bgColorStr = "#f9f9f9"
						}
					} else if multiSelected[optIndex] {
						bgColorStr = "#e6f7ff" // 未提交前的浅蓝色选定状态
					}
				} else {
					// 单选、判断
					if singleWrongClicked[optIndex] {
						bgColorStr = "#FFEBEE"
						rightIconText.Text = "✕"
						iconColorStr = "#ff4d4f"   // 红叉
						strokeColorStr = "#ff4d4f" // 红边框
					}
					if singleCorrectClicked[optIndex] {
						bgColorStr = "#E8F5E8"
						rightIconText.Text = "✓"
						iconColorStr = greenColor   // 绿勾
						strokeColorStr = greenColor // 绿边框
					}
				}

				// ✨ 将计算出来的颜色状态，精准灌入到各自的组件属性中
				rightIconText.Color = hexColor(iconColorStr)

				optLayout := container.NewBorder(nil, nil, prefixLbl, container.NewCenter(rightIconText), contentLbl)

				optBg := canvas.NewRectangle(hexColor(bgColorStr))
				optBg.CornerRadius = 6
				optBg.StrokeColor = hexColor(strokeColorStr)
				optBg.StrokeWidth = 1

				// == 封装点击事件 ==
				optCard := NewClickableBox(container.NewStack(optBg, container.NewPadded(optLayout)), func() {
					// 锁定保护：选对后或多选提交后，屏蔽点击
					if singleCorrectClicked[optIndex] || multiSubmitted {
						return
					}

					if q.Type == "多选题" {
						multiSelected[optIndex] = !multiSelected[optIndex] // 切换选中状态
						refreshPracticeUI()
					} else {
						// 单选/判断：实时判定
						handleUserSelectOption(state, q, optLetter) // 落盘

						if isCorrectAnswer {
							singleCorrectClicked[optIndex] = true
							refreshPracticeUI() // 瞬间变绿

							// 🌟 开启协程，延迟 350 毫秒后自动跳下一题
							go func() {
								time.Sleep(350 * time.Millisecond)
								if state.Index < len(state.CurrentList)-1 {
									state.Index++
								}
								refreshPracticeUI()
							}()
						} else {
							singleWrongClicked[optIndex] = true
							refreshPracticeUI() // 变红，留在本题
						}
					}
				})

				optionsBox.Add(optCard)
			}
		}
		//}

		// -------------------------------------------------------------------
		// 任务 4: 答题控制按钮组
		// -------------------------------------------------------------------
		prevBtn := widget.NewButton("上一题", func() {
			if state.Index > 0 {
				state.Index--
				refreshPracticeUI()
			}
		})
		if state.Index == 0 {
			prevBtn.Disable()
		}

		nextBtn := widget.NewButton("下一题", func() {
			if state.Index < len(state.CurrentList)-1 {
				state.Index++
				refreshPracticeUI()
			}
		})
		if state.Index == len(state.CurrentList)-1 {
			nextBtn.Disable()
		}

		// ✨ 多选题提交逻辑
		submitBtn := widget.NewButton("提交", func() {
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

		// ✨ 新增拦截：如果是修正模式，强制禁用多选题的“提交”按钮（因为不需要答题判卷）
		if q.Type != "多选题" || strings.Contains(state.Title, "修正题目") {
			submitBtn.Disable()
		}

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
		centerScroll := container.NewScroll(container.NewPadded(cardStack))

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
		statsLbl.TextSize = 14

		statsBg := canvas.NewRectangle(hexColor("#f7f7f7"))
		statsRow := container.NewStack(statsBg, container.NewCenter(statsLbl))

		returnTypeBtn := widget.NewButton("返回题型", func() {
			backFunc(w, state)
		})

		// ✨ 新增：“删除本题”按钮
		delCurrentDataBtn := widget.NewButton("删除本题", func() {
			dialog.ShowConfirm("提示", "确定要将本题从当前练习集中移除吗？", func(confirm bool) {
				if confirm {
					// 这里的具体删除逻辑一会讨论（例如从 state.CurrentList 中移除当前 Index 题目）
					refreshPracticeUI()
				}
			}, w)
		})

		// 💡 核心控制：只有在“错题练习”或“修正题目”模式下才激活该按钮
		// 判定标准根据你传入 state.Title 的文本或专门的模式状态字段
		if !strings.Contains(state.Title, "错题练习") && !strings.Contains(state.Title, "修正题目") {
			delCurrentDataBtn.Disable() // 其他普通模式下自动变灰、锁死无法点击
		}

		delRecordBtn := widget.NewButton("删除记录", func() {
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
		pageBg := canvas.NewRectangle(hexColor("#f0f2f5"))

		// 使用 Border 布局：Top 放导航栏，Bottom 放底栏，中间 Center 让题目卡片自适应填充并滚动
		mainContent := container.NewBorder(topArea, bottomArea, nil, nil, centerScroll)

		mainLayout.Objects = []fyne.CanvasObject{pageBg, mainContent}
		mainLayout.Refresh()
	}

	// 触发首次渲染
	refreshPracticeUI()

	w.SetContent(mainLayout)
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
