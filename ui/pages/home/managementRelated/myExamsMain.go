package managementRelated

import (
	"fmt"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
				}, func() {
					// View exam detail
					showExamDetailPage(w, state, &examItem, func() {
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
func buildMyExamCard(w fyne.Window, exam *network.MyExamItem, state *core.AppState, onStart func(), onContinue func(), onDetail func()) fyne.CanvasObject {
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
	var buttonsRow fyne.CanvasObject

	if exam.Status == "active" && exam.Score == nil {
		// 进行中的考试且未提交：显示"开始考试"按钮并居中
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

		buttonsRow = container.NewCenter(startBtn)
	} else if exam.Status == "in_progress" && exam.Score == nil {
		// 考试中且未提交：显示"继续考试"按钮并居中
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

		buttonsRow = container.NewCenter(continueBtn)
	} else if exam.Score != nil || exam.Status == "completed" {
		// 已交卷/有得分/已结束：显示"考试详情"按钮
		detailBtn := customElements.CreateButton(
			"考试详情",
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			1.5, 18,
			true, false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,
			fyne.TextTruncateOff,
			onDetail,
		)
		buttonsRow = container.NewCenter(detailBtn)
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

	vboxItems := []fyne.CanvasObject{
		container.NewBorder(nameText, nil, statusBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		scoreText,
	}
	if buttonsRow != nil {
		vboxItems = append(vboxItems,
			container.NewGridWrap(fyne.NewSize(1, 10)),
			buttonsRow,
		)
	}
	cardContent := container.NewVBox(vboxItems...)

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
			if strings.Contains(msg, "考试已超时") || strings.Contains(msg, "exam has timeout") || strings.Contains(msg, "timeout") {
				exam.Status = "timeout"
				for i := range state.MyExams {
					if state.MyExams[i].ExamID == exam.ExamID {
						state.MyExams[i].Status = "timeout"
					}
				}
				customElements.ShowCustomInformation("错误", "考试已超时", w)
				onBack()
			} else {
				customElements.ShowCustomInformation("错误", msg, w)
			}
			return
		}

		// 将服务器题目转换为本地格式
		localQuestions := make([]core.Question, len(questions))
		for i, q := range questions {
			var convertedOpts []core.Option
			for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
				if text, ok := q.Options[k]; ok && text != "" {
					convertedOpts = append(convertedOpts, core.Option{Label: k, Text: text})
				}
			}
			if len(convertedOpts) == 0 {
				for label, text := range q.Options {
					if text != "" {
						convertedOpts = append(convertedOpts, core.Option{Label: label, Text: text})
					}
				}
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
