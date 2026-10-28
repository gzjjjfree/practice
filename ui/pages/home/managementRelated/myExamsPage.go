package managementRelated

import (
	"fmt"
	"strings"
	"image/color"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)


// renderMyExamPage 渲染考试题目页面。
//
// 功能说明：
// 支持填空题/问答题的显示答案功能，以及单选题/多选题的答题和反馈。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用程序窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态对象。
//   - onBack: func() 类型，返回考试列表页面的回调函数（交卷后调用）。
//
// 核心业务逻辑：
// 1. 维护题目状态跟踪变量（当前题目 ID、是否显示答案、多选题已选索引、单选错误/正确点击状态、多选题是否已提交）。
// 2. 定义 refreshExamUI 闭包函数，用于刷新考试 UI。
// 3. 创建交卷按钮（红色），点击后弹出确认对话框，确认后提交考试并显示得分结果。
// 4. 根据题目类型（填空题/问答题 vs 单选题/多选题）渲染不同的选项交互逻辑。
// 5. 创建上一题、下一题导航按钮，支持题目切换。
// 6. 使用 TouchInterceptor 实现滑动切换题目的功能。
func renderMyExamPage(w fyne.Window, state *core.AppState, onBack func()) {
	// 题目状态跟踪变量
	var currentQID string                  // 当前题目 ID
	var isMemorizeRevealed bool            // 是否已显示答案
	var multiSelected = make(map[int]bool) // 多选题已选索引

	var refreshExamUI func() // 刷新考试 UI 的闭包函数
	refreshExamUI = func() {
		if state.Index < 0 || state.Index >= len(state.CurrentList) {
			return
		}
		q := state.CurrentList[state.Index]

		// 返回按钮
		backBtn := customElements.CreateButton(
			core.AdminBackBtnText,
			core.BackBtnWidth, core.BackBtnHeight,
			core.HexColor(core.TextPrimaryColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BorderLightColor),
			core.StrokeMedium, core.FontSizeBody,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				onBack()
			},
		)

		// 交卷按钮（红色）
		// submitBtn: 交卷按钮，标识符为 core.MyExamsSubmitBtnText。
		// 功能与业务用途：点击该按钮可弹出确认提交对话框，确认后调用 state.SubmitMyExamSubmission 提交考试答案，显示得分结果并返回考试列表页面（通过调用 onBack 回调函数实现）。
		submitBtn := customElements.CreateButton(
			core.MyExamsSubmitBtnText, core.ModalCloseBtnWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.ColorErrorBg),
			core.HexColor(core.ColorErrorBorder),
			core.StrokeMedium, core.FontSizeBody,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				customElements.ShowCustomConfirm(
					core.MyExamsConfirmSubmitTitle,
					"确定",
					"取消",
					customElements.NewCenterRichText(core.MyExamsConfirmSubmitMsg),
					func(confirm bool) bool {
						if confirm {
							state.SubmitMyExamSubmission(func(success bool, result *network.ExamResult, msg string) {
								if !success {
									customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
									return
								}

								scoreText := fmt.Sprintf(core.MyExamsScoreResultFormat, result.Score, result.CorrectCount, result.TotalCount)
								customElements.ShowCustomInformation(core.MyExamsResultDialogTitle, scoreText, w)
								onBack()
							})
						}
						return true // 返回 true 表示对话框可以关闭
					},
					w,
				)
			},
		)

		titleLbl := canvas.NewText(state.Title, core.HexColor(core.TextPrimaryColor))
		titleLbl.TextStyle = fyne.TextStyle{Bold: true}
		titleLbl.TextSize = core.ExamTitleLabelFontSize
		titleCenter := container.NewCenter(titleLbl)

		score := q.Score
		if score <= 0 {
			score = 1.0
		}
		metaLeftStr := fmt.Sprintf("第 %d/%d 题  %s（每题 %g 分）", state.Index+1, len(state.CurrentList), q.Type, score)
		metaLeft := canvas.NewText(metaLeftStr, core.HexColor(core.TextMutedColor))
		metaLeft.TextSize = 13

		answeredCount := 0
		for _, qItem := range state.CurrentList {
			if val, ok := state.ExamAnswers[qItem.ID]; ok && val != "" && val != "null" {
				answeredCount++
			}
		}
		summaryBtnText := fmt.Sprintf("答题卡 (%d/%d)", answeredCount, len(state.CurrentList))
		summaryBtn := customElements.CreateButton(
			summaryBtnText,
			core.ModalCloseBtnWidth, core.NavButtonHeight,
			core.HexColor(core.TextHintColor),
			core.HexColor(core.SectionBgColor),
			core.HexColor(core.SectionBgColor),
			core.StrokeMedium, core.FontSizeDialogMsg,
			false, false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,
			fyne.TextTruncateOff,
			func() {
				ShowExamQuestionModal(w, state, refreshExamUI)
			},
		)

		topNav := container.NewBorder(nil, nil, backBtn, submitBtn, titleCenter)
		metaRow := container.NewBorder(nil, nil, metaLeft, summaryBtn, nil)

		resetQuestionState := func() {
			isMemorizeRevealed = false
			multiSelected = make(map[int]bool)
		}

		if q.ID != currentQID {
			currentQID = q.ID
			resetQuestionState()
			if ans, ok := state.ExamAnswers[q.ID]; ok && ans != "" && ans != "null" {
				if q.Type == "多选题" {
					for _, char := range ans {
						charStr := string(char)
						for idx, opt := range q.Options {
							if opt.Label == charStr {
								multiSelected[idx] = true
								break
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

		optionsBox := container.NewVBox()
		marginBox2 := canvas.NewRectangle(color.Transparent)
		marginBox2.SetMinSize(fyne.NewSize(1, core.OptionSpacing))
		optionsBox.Add(marginBox2)

		if q.Type == "填空题" || q.Type == "问答题" {
			btnText := core.MyExamsShowAnswerBtnText
			if isMemorizeRevealed {
				btnText = core.MyExamsHideAnswerBtnText
			}
			cardAColor := core.PageBgColor
			if isMemorizeRevealed {
				cardAColor = core.AnswerRevealBg
			}
			// answerToggleBtn: 显示答案/隐藏答案按钮，标识符为 core.MyExamsShowAnswerBtnText 或 core.MyExamsHideAnswerBtnText。
			// 功能与业务用途：针对填空题和问答题，点击该按钮可切换是否显示题目答案。当未显示答案时显示"显示答案"，点击后变为"隐藏答案"并展示答案卡片。
			cardA := customElements.CreateButton(
				btnText, 0, 0,
				core.HexColor(core.TextSecondaryColor),
				core.HexColor(cardAColor),
				core.HexColor(core.BorderLightColor),
				1.5, 18,
				true, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {

					isMemorizeRevealed = !isMemorizeRevealed
					refreshExamUI()
				},
			)
			optionsBox.Add(cardA)

			if isMemorizeRevealed {
				ansStr := ""
				if len(q.Options) > 0 {
					ansStr = q.Options[0].Text
				}
				lblB := core.CreateOptionLabel(ansStr)
				bgB := canvas.NewRectangle(core.HexColor(core.ColorCorrectBg))
				bgB.StrokeColor = core.HexColor(core.ColorCorrectBorder)
				bgB.StrokeWidth = core.AnswerRevealStrokeWidth
				bgB.CornerRadius = core.AnswerRevealCornerRadius
				cardB := container.NewStack(bgB, container.NewPadded(lblB))
				optionsBox.Add(cardB)
			}
		} else {
			for i, opt := range q.Options {
				optIndex := i
				optStr := opt.Text
				if optStr == "" {
					continue
				}

				optLetter := opt.Label
				optPrefix := optLetter + ". "

				prefixLbl := widget.NewLabel(optPrefix)
				prefixLbl.TextStyle = fyne.TextStyle{Bold: true}
				prefixLbl.SizeName = theme.SizeNameSubHeadingText

				contentLbl := core.CreateOptionLabel(optStr)
				contentLbl.Alignment = fyne.TextAlignCenter
				contentLbl.Wrapping = fyne.TextWrapBreak

				rightIconText := canvas.NewText("", core.HexColor(core.CardBgColor))
				rightIconText.TextSize = core.RightIconTextFontSize
				rightIconText.TextStyle = fyne.TextStyle{Bold: true}
				rightIconText.Color = core.HexColor(core.CardBgColor)

				bgColorStr := core.OptionDefaultBg
				strokeColorStr := core.BorderLightColor

				if q.Type == "多选题" {
					if multiSelected[optIndex] {
						bgColorStr = core.ColorSelectedBg
					}
				} else {
					if state.ExamAnswers[q.ID] == optLetter {
						bgColorStr = core.ColorSelectedBg
					}
				}

				fixedIconBox := container.NewGridWrap(fyne.NewSize(core.IconFixedBoxSize, core.IconFixedBoxSize), container.NewCenter(rightIconText))
				_ = container.NewBorder(nil, nil, prefixLbl, fixedIconBox, contentLbl)

				// optCard: 选项按钮，标识符为选项文本内容 optStr。
				// 功能与业务用途：针对单选题和多选题，点击该按钮可选择或取消选择对应选项。对于多选题，支持多次切换选择状态；对于单选题，选择后可更改。
				optCard := customElements.CreateButton(
					optStr, 0, 0,
					core.HexColor(core.TextBodyColor),
					core.HexColor(bgColorStr),
					core.HexColor(strokeColorStr),
					1.5, 16,
					false, false,
					fyne.TextAlignCenter, // 👈 居中对齐
					fyne.TextWrapOff,     // 👈 不换行
					fyne.TextTruncateOff, // 👈 不换行（截断）
					func() {
						if q.Type == "多选题" {
							multiSelected[optIndex] = !multiSelected[optIndex]
							var sel []string
							for idx, o := range q.Options {
								if multiSelected[idx] {
									sel = append(sel, o.Label)
								}
							}
							if len(sel) > 0 {
								state.ExamAnswers[q.ID] = strings.Join(sel, "")
							} else {
								delete(state.ExamAnswers, q.ID)
							}
							refreshExamUI()
						} else {
							state.ExamAnswers[q.ID] = optLetter
							refreshExamUI()
						}
					})

				optionsBox.Add(optCard)
				optionsBox.Add(marginBox2)
			}
		}

		// prevBtn: 上一题按钮，标识符为 core.MyExamsPrevBtnText。
		// 功能与业务用途：点击该按钮可切换到上一道题目（当当前题目索引大于0时可用），并刷新考试 UI。
		prevBtn := customElements.CreateButton(
			core.MyExamsPrevBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true, state.Index == 0,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if state.Index > 0 {
					state.Index--
					refreshExamUI()
				}
			},
		)

		// nextBtn: 下一题按钮，标识符为 core.MyExamsNextBtnText。
		// 功能与业务用途：点击该按钮可切换到下一道题目（当当前题目索引小于总题目数减1时可用），并刷新考试 UI。
		nextBtn := customElements.CreateButton(
			core.MyExamsNextBtnText, core.NavButtonWidth, core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium, core.FontSizeBody,
			true, state.Index >= len(state.CurrentList)-1,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if state.Index < len(state.CurrentList)-1 {
					state.Index++
					refreshExamUI()
				}
			})

		btnSpacing := canvas.NewRectangle(color.Transparent)
		btnSpacing.SetMinSize(fyne.NewSize(20, 1))
		buttonsRow := container.NewHBox(prevBtn, btnSpacing, nextBtn)
		navGrid := container.NewCenter(buttonsRow)

		questionContent := container.NewVBox(topNav, metaRow, widget.NewSeparator(), stemLbl, optionsBox, layout.NewSpacer(), navGrid)

		cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		cardBg.CornerRadius = 8
		cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
		cardBg.StrokeWidth = 1
		cardStack := container.NewStack(cardBg, container.NewPadded(questionContent))

		touchArea := customElements.NewTouchInterceptor(
			container.NewStack(pageBgForExam(), cardStack),
			nil,
			func() {
				if state.Index > 0 {
					state.Index--
					refreshExamUI()
				}
			},
			func() {
				if state.Index < len(state.CurrentList)-1 {
					state.Index++
					refreshExamUI()
				}
			},
		)

		wRootLayout := container.NewBorder(customElements.GetTitle(), nil, nil, nil, touchArea)
		w.SetContent(wRootLayout)
	}

	// Initial render
	refreshExamUI()
}

// showExamDetailDialog 显示考试详情对话框，包含暂停/继续和结果展示功能。
//
// 功能说明：
// 从服务器获取考试详情和得分信息，并在对话框中展示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
// 如果考试状态为"进行中"或"考试中"，则显示"暂停考试"按钮。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用程序窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态对象。
//   - exam: *network.MyExamItem 类型，表示考试项的数据结构，包含考试 ID、名称、状态等信息。
func showExamDetailDialog(w fyne.Window, state *core.AppState, exam *network.MyExamItem) {
	// Get exam result
	sessionID := exam.ExamSessionID
	if sessionID == "" {
		sessionID = state.ExamSessionID
	}
	if sessionID == "" {
		// 如果没有 exam_session_id，则无法获取考试详情
		// 客户端在调用获取考试详情/事后回看接口 (GET /api/v1/my-exams/result/{session_id}) 时，
		// URL 路径中传递的应当是调用 StartExam 返回的 exam_session_id 字符串（例如 "2-1-1708512345678901234"），
		// 而非考试本身的数字 exam_id。
		customElements.ShowCustomInformation(core.BankManageErrorMsgType, "考试会话ID不存在，无法获取考试详情", w)
		return
	}
	state.GetExamResult(sessionID, func(success bool, result *network.ExamDetailResult, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		// Build detail text
		detailText := fmt.Sprintf("考试名称: %s\n\n", exam.BankName)
		detailText += fmt.Sprintf("状态: %s\n", exam.Status)
		detailText += fmt.Sprintf("题目数: %d\n", exam.QuestionCount)
		detailText += fmt.Sprintf("时长: %d分钟\n", exam.DurationMin)
		detailText += fmt.Sprintf("开始时间: %s\n", exam.StartTime)
		detailText += fmt.Sprintf("结束时间: %s\n", exam.EndTime)
		detailText += fmt.Sprintf("\n用户得分: %.1f\n", result.Score)
		detailText += fmt.Sprintf("正确题数: %d / %d\n", result.CorrectCount, result.TotalCount)

		// Show pause/resume buttons if status is active or in_progress
		var buttons []fyne.CanvasObject
		if exam.Status == "active" || exam.Status == "in_progress" {
			// pauseBtn: 暂停考试按钮，标识符为 core.MyExamsPauseExamBtnText。
			// 功能与业务用途：当考试状态为"进行中"或"考试中"时显示，点击该按钮可调用 state.PauseExam 暂停当前考试会话，并在成功后刷新详情对话框。
			pauseBtn := customElements.CreateButton(
				core.MyExamsPauseExamBtnText, core.ActionBtnWidth, core.NavButtonHeight,
				core.HexColor(core.CardBgColor),
				core.HexColor(core.ColorWarningBg),
				core.HexColor(core.ColorWarningBorder),
				core.StrokeMedium, core.FontSizeSmall,
				true, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					state.PauseExam(fmt.Sprintf("%d", exam.ExamID), func(success bool, data *network.PauseResumeData, msg string) {
						if success {
							customElements.ShowCustomInformation(core.BankManageSuccessMsgType, core.MyExamsPauseSuccessMsg, w)
							showExamDetailDialog(w, state, exam)
						} else {
							customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
						}
					})
				})
			buttons = append(buttons, pauseBtn)
		}

		// Create content
		content := container.NewVBox(
			widget.NewLabel(core.MyExamsDetailDialogTitle),
			container.NewGridWrap(fyne.NewSize(1, 10)),
			widget.NewRichText(
				&widget.TextSegment{
					Style: widget.RichTextStyle{},
					Text:  detailText,
				},
			),
		)

		if len(buttons) > 0 {
			content = container.NewVBox(
				widget.NewLabel(core.MyExamsDetailDialogTitle),
				container.NewGridWrap(fyne.NewSize(1, 10)),
				widget.NewRichText(
					&widget.TextSegment{
						Style: widget.RichTextStyle{},
						Text:  detailText,
					},
				),
				container.NewGridWrap(fyne.NewSize(1, 10)),
				container.NewHBox(buttons...),
			)
		}

		dialog.NewCustomWithoutButtons(core.MyExamsDetailDialogTitle, content, w).Show()
	})
}

// pageBgForExam 创建考试页面的背景矩形。
//
// 功能说明：
// 创建一个使用页面背景颜色的矩形对象，用于考试题目页面的背景层。
//
// 返回值说明：
//   - *canvas.Rectangle 类型，返回构建好的背景矩形对象。
func pageBgForExam() *canvas.Rectangle {
	bg := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	return bg
}
