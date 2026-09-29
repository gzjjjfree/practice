package managementRelated

import (
	"fmt"
	"image/color"
	"strings"

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

// ShowMyExams 渲染普通用户的"我的考试"页面。
//
// 功能说明：
// 从服务器加载用户被推送的考试列表，每个考试以卡片形式展示，
// 包含考试名称、题目数、时长、状态、得分等信息。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用程序窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态对象。
//   - backToHome: func() 类型，返回主页面的回调函数。
//
// 核心业务逻辑：
// 1. 创建顶部导航栏，包含返回按钮和页面标题。
// 2. 调用 state.LoadMyExams 从服务器加载用户的考试列表。
// 3. 根据加载结果渲染考试卡片列表或无数据提示。
// 4. 为每个考试项创建卡片 UI，并绑定开始考试或查看详情等交互逻辑。
func ShowMyExams(w fyne.Window, state *core.AppState, backToHome func()) {
	// backBtn: 返回按钮，标识符为 core.AdminBackBtnText。
	// 功能与业务用途：点击该按钮可退出当前"我的考试"页面，返回到上一级主页（通过调用 backToHome 回调函数实现）。
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
			backToHome()
		},
	)

	titleCenter := customElements.CreateLabel(
		core.MyExamsTitleText,
		core.HexColor(core.TextPrimaryColor),
		core.FontSizePageTitle, true, false, false,
	)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleCenter)

	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// Fetch my exams from server
	state.LoadMyExams(func(success bool, exams []network.MyExamItem, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		examListVBox := container.NewVBox()

		if len(exams) == 0 {
			noDataText := canvas.NewText(core.MyExamsNoDataText, core.HexColor(core.TextMutedColor))
			noDataText.TextSize = core.FontSizeSmall
			noDataText.Alignment = fyne.TextAlignCenter
			examListVBox.Add(container.NewPadded(noDataText))
		} else {
			for _, exam := range exams {
				examItem := exam
				examCard := buildMyExamCard(w, &examItem, state, func() {
					// Start this exam
					startExamInSession(w, state, &examItem, func() {
						// Refresh the list after returning from exam
						ShowMyExams(w, state, backToHome)
					})
				}, func() {
					// Continue this exam
					startExamInSession(w, state, &examItem, func() {
						// Refresh the list after returning from exam
						ShowMyExams(w, state, backToHome)
					})
				})
				examListVBox.Add(examCard)
				examListVBox.Add(container.NewGridWrap(fyne.NewSize(1, 8)))
			}
		}

		scrollContent := container.NewVScroll(examListVBox)
		scrollContent.SetMinSize(fyne.NewSize(0, core.MyExamsScrollMinHeight))

		mainLayout := container.NewBorder(topBar, nil, nil, nil, scrollContent)
		wRootLayout := container.NewStack(pageBg, mainLayout)
		w.SetContent(wRootLayout)
	})
}

// buildMyExamCard 为每个考试项创建卡片 UI。
//
// 功能说明：
// 根据考试状态（未开始/进行中/考试中/考试超时/已结束）显示不同的颜色标识，并生成包含考试信息、得分和交互按钮的卡片组件。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用程序窗口对象。
//   - exam: *network.MyExamItem 类型，表示考试项的数据结构，包含考试名称、题目数、时长、状态、得分等信息。
//   - state: *core.AppState 类型，表示全局应用状态对象。
//   - onStart: func() 类型，开始考试的回调函数（当考试状态为"进行中"且未提交时触发）。
//   - onContinue: func() 类型，继续考试的回调函数（当考试状态为"考试中"时触发）。
//
// 返回值说明：
//   - fyne.CanvasObject 类型，返回构建好的考试卡片 UI 组件。
//
// 核心业务逻辑：
// 1. 根据考试状态（upcoming/active/in_progress/timeout/completed）映射对应的中文状态文本和颜色标识。
// 2. 构建考试名称、信息行（题目数、时长、开始/结束时间）、得分显示和状态徽章。
// 3. 根据考试状态和是否已提交得分，动态生成不同的按钮组合（开始考试、继续考试、查看详情、未开放）。
// 4. 返回一个包含所有元素的卡片 UI 组件。
func buildMyExamCard(w fyne.Window, exam *network.MyExamItem, state *core.AppState, onStart func(), onContinue func()) fyne.CanvasObject {
	// 状态徽章颜色映射
	statusColor := core.TextMutedColor
	statusText := exam.Status
	switch exam.Status {
	case "upcoming":
		statusColor = core.TextHintColor
		statusText = "未开始"
	case "active":
		statusColor = core.BtnPrimaryBg
		statusText = "进行中"
	case "in_progress":
		statusColor = core.ColorWarningBg
		statusText = "考试中"
	case "timeout":
		statusColor = core.ColorWrongBorder
		statusText = "考试超时"
	case "completed":
		statusColor = core.ColorCorrectBorder
		statusText = "已结束"
	}

	// 考试名称
	nameText := canvas.NewText(exam.BankName, core.HexColor(core.TextPrimaryColor))
	nameText.TextSize = core.FontSizeBody
	nameText.TextStyle = fyne.TextStyle{Bold: true}

	// 信息行：题目数、时长、开始/结束时间
	infoParts := []string{
		fmt.Sprintf("题目数: %d", exam.QuestionCount),
		fmt.Sprintf("时长: %d分钟", exam.DurationMin),
	}
	if exam.StartTime != "" {
		infoParts = append(infoParts, fmt.Sprintf("开始: %s", exam.StartTime))
	}
	if exam.EndTime != "" {
		infoParts = append(infoParts, fmt.Sprintf("结束: %s", exam.EndTime))
	}

	infoText := canvas.NewText(strings.Join(infoParts, "  |  "), core.HexColor(core.TextSecondaryColor))
	infoText.TextSize = core.FontSizeSmallest

	// 得分显示（有得分时显示，否则隐藏）
	var scoreText *canvas.Text
	if exam.Score != nil {
		scoreText = canvas.NewText(fmt.Sprintf("得分: %.1f", *exam.Score), core.HexColor(core.ColorCorrectBorder))
	} else {
		scoreText = canvas.NewText("", core.HexColor(core.CardBgColor)) // hidden
	}
	scoreText.TextSize = core.FontSizeSmall

	// 状态徽章
	statusBadge := canvas.NewText("● "+statusText, core.HexColor(statusColor))
	statusBadge.TextSize = core.FontSizeSmallest

	// 按钮行：根据考试状态显示不同按钮
	var buttonsRow *fyne.Container

	if exam.Status == "active" && exam.Score == nil {
		// 进行中的考试且未提交：显示"开始考试"和"查看详情"按钮
		// startBtn: 开始考试按钮，标识符为 core.MyExamsStartExamBtnText。
		// 功能与业务用途：点击该按钮可启动当前考试，进入全屏答题会话（通过调用 onStart 回调函数实现）。
		startBtn := customElements.CreateButton(
			core.MyExamsStartExamBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			onStart,
		)

		// viewDetailsBtn: 查看详情按钮，标识符为 core.MyExamsViewDetailsBtnText。
		// 功能与业务用途：点击该按钮可弹出考试详情对话框，显示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
		viewDetailsBtn := customElements.CreateButton(
			core.MyExamsViewDetailsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showExamDetailDialog(w, state, exam)
			},
		)

		buttonsRow = container.NewHBox(startBtn, viewDetailsBtn)
	} else if exam.Status == "in_progress" && exam.Score == nil {
		// 考试中且未提交：显示"继续考试"和"查看详情"按钮
		// continueBtn: 继续考试按钮，标识符为 core.MyExamsContinueExamBtnText。
		// 功能与业务用途：点击该按钮可继续当前考试，进入全屏答题会话（通过调用 onContinue 回调函数实现）。
		continueBtn := customElements.CreateButton(
			core.MyExamsContinueExamBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.ColorWarningBg),
			core.HexColor(core.ColorWarningBorder),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			onContinue,
		)

		// viewDetailsBtn: 查看详情按钮，标识符为 core.MyExamsViewDetailsBtnText。
		// 功能与业务用途：点击该按钮可弹出考试详情对话框，显示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
		viewDetailsBtn := customElements.CreateButton(
			core.MyExamsViewDetailsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showExamDetailDialog(w, state, exam)
			},
		)

		buttonsRow = container.NewHBox(continueBtn, viewDetailsBtn)
	} else if exam.Status == "active" && exam.Score != nil {
		// 进行中的考试但已提交：只显示"查看详情"
		// viewDetailsBtn: 查看详情按钮，标识符为 core.MyExamsViewDetailsBtnText。
		// 功能与业务用途：点击该按钮可弹出考试详情对话框，显示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
		viewDetailsBtn := customElements.CreateButton(
			core.MyExamsViewDetailsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showExamDetailDialog(w, state, exam)
			},
		)

		buttonsRow = container.NewCenter(viewDetailsBtn)
	} else if exam.Status == "in_progress" && exam.Score != nil {
		// 考试中但已提交：只显示"查看详情"
		// viewDetailsBtn: 查看详情按钮，标识符为 core.MyExamsViewDetailsBtnText。
		// 功能与业务用途：点击该按钮可弹出考试详情对话框，显示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
		viewDetailsBtn := customElements.CreateButton(
			core.MyExamsViewDetailsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showExamDetailDialog(w, state, exam)
			},
		)

		buttonsRow = container.NewCenter(viewDetailsBtn)
	} else if exam.Score != nil {
		// 非进行中但有得分：显示"查看详情"
		// viewDetailsBtn: 查看详情按钮，标识符为 core.MyExamsViewDetailsBtnText。
		// 功能与业务用途：点击该按钮可弹出考试详情对话框，显示考试的详细信息（包括状态、题目数、时长、开始/结束时间、得分等）。
		viewDetailsBtn := customElements.CreateButton(
			core.MyExamsViewDetailsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showExamDetailDialog(w, state, exam)
			},
		)

		buttonsRow = container.NewCenter(viewDetailsBtn)
	} else {
		// 未开放：显示灰色"未开放"按钮
		// notOpenBtn: 未开放按钮，标识符为 core.MyExamsNotOpenBtnText。
		// 功能与业务用途：当考试未开放时显示，该按钮为禁用状态，无交互功能，仅用于提示用户当前考试尚未开放。
		buttonsRow = container.NewCenter(
			customElements.CreateButton(
				core.MyExamsNotOpenBtnText,
				core.ActionBtnWidth, core.ActionBtnHeight,
				core.HexColor(core.TextMutedColor),
				core.HexColor(core.BtnDisabledBg),
				core.HexColor(core.BtnDisabledStroke),
				1.5, 16,
				true, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				nil,
			))
	}

	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.CardCornerRadius
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, statusBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		scoreText,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		buttonsRow,
	)

	return container.NewStack(
		cardBg,
		container.NewPadded(cardContent),
	)
}

// startExamInSession 在全屏会话中启动考试。
//
// 功能说明：
// 从服务器获取题目并转换为本地格式，然后渲染考试页面。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前应用程序窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态对象。
//   - exam: *network.MyExamItem 类型，表示考试项的数据结构，包含考试 ID、名称等信息。
//   - onBack: func() 类型，返回考试列表页面的回调函数。
//
// 核心业务逻辑：
// 1. 调用 state.StartMyExam 从服务器获取考试的题目列表和会话 ID。
// 2. 将服务器返回的题目格式转换为本地 core.Question 格式。
// 3. 更新 AppState 中的当前题目列表、索引、标题、会话 ID 和答案映射。
// 4. 调用 renderMyExamPage 渲染考试题目页面。
func startExamInSession(w fyne.Window, state *core.AppState, exam *network.MyExamItem, onBack func()) {
	// 启动考试会话
	state.StartMyExam(exam.ExamID, func(success bool, sessionID string, questions []network.ServerQuestion, msg string) {
		if !success {
			customElements.ShowCustomInformation("错误", msg, w)
			return
		}

		// 将服务器题目转换为本地格式
		localQuestions := make([]core.Question, len(questions))
		for i, q := range questions {
			var convertedOpts []core.Option
			for label, text := range q.Options {
				convertedOpts = append(convertedOpts, core.Option{Label: label, Text: text})
			}
			var answers []string
			for _, char := range q.Answer {
				answers = append(answers, string(char))
			}
			if len(answers) == 0 && q.Answer != "" {
				answers = append(answers, q.Answer)
			}

			score := q.Score
			if score <= 0 {
				score = 1.0
			}

			localQuestions[i] = core.Question{
				ID:         fmt.Sprintf("%v", q.ID),
				Type:       q.Type,
				Content:    q.Content,
				Options:    convertedOpts,
				Answers:    answers,
				Difficulty: q.Difficulty,
				Score:      score,
			}
		}

		state.CurrentList = localQuestions
		state.Index = 0
		state.Title = exam.BankName
		state.ExamSessionID = sessionID
		state.ExamAnswers = make(map[string]string)

		// 渲染考试 UI
		renderMyExamPage(w, state, onBack)
	})
}

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
	var currentQID string                         // 当前题目 ID
	var isMemorizeRevealed bool                   // 是否已显示答案
	var multiSelected = make(map[int]bool)        // 多选题已选索引
	var singleWrongClicked = make(map[int]bool)   // 单选错误点击
	var singleCorrectClicked = make(map[int]bool) // 单选正确点击
	var multiSubmitted bool                       // 多选题是否已提交

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
			core.MyExamsSubmitBtnText, core.NavButtonWidth, core.NavButtonHeight,
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

		topNav := container.NewBorder(nil, nil, backBtn, submitBtn, titleCenter)
		metaRow := container.NewBorder(nil, nil, metaLeft, nil, nil)

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
				rightIconText.TextSize = core.RightIconTextFontSize
				rightIconText.TextStyle = fyne.TextStyle{Bold: true}

				bgColorStr := core.OptionDefaultBg
				strokeColorStr := core.BorderLightColor
				iconColorStr := core.CardBgColor
				greenColor := core.ColorCorrectBorder

				isCorrectAnswer := false
				for _, correctLetter := range q.Answers {
					if correctLetter == optLetter {
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
				_ = container.NewBorder(nil, nil, prefixLbl, fixedIconBox, contentLbl)

				// optCard: 选项按钮，标识符为选项文本内容 optStr。
				// 功能与业务用途：针对单选题和多选题，点击该按钮可选择或取消选择对应选项。对于多选题，支持多次切换选择状态；对于单选题，选择后不可更改，并立即显示正确/错误反馈及正确答案提示。
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

						if q.Type == "多选题" && multiSubmitted {
							return
						}
						if q.Type != "多选题" && hasSingleAnswered {
							return
						}

						if q.Type == "多选题" {
							multiSelected[optIndex] = !multiSelected[optIndex]
							refreshExamUI()
						} else {
							state.ExamAnswers[q.ID] = optLetter
							if isCorrectAnswer {
								singleCorrectClicked[optIndex] = true
								refreshExamUI()
							} else {
								singleWrongClicked[optIndex] = true
								for _, correctLetter := range q.Answers {
									if len(correctLetter) == 1 {
										cIdx := int(correctLetter[0] - 'A')
										singleCorrectClicked[cIdx] = true
									}
								}
								refreshExamUI()
							}
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
	state.GetExamResult(fmt.Sprintf("%d", exam.ExamID), func(success bool, result *network.ExamDetailResult, msg string) {
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
