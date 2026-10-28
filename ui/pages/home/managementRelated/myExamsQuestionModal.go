package managementRelated

import (
	"fmt"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/ui/pages/home/practiceRelated"
)


// ShowExamQuestionModal 显示考试题目导航弹窗，以网格形式展示所有题目并标记答题状态（已答/未答/当前）。
// 题目状态用颜色编码：蓝色=当前题目，绿色=已答题目，灰色/默认=未作答。
// 点击题目可跳转到对应题号。
func ShowExamQuestionModal(w fyne.Window, state *core.AppState, onNavigate func()) {
	grid := container.NewGridWrap(fyne.NewSize(core.ModalGridItemSize, core.ModalGridItemSize))

	var modal *widget.PopUp

	for i := 0; i < len(state.CurrentList); i++ {
		idx := i
		q := state.CurrentList[idx]

		// 1. 计算状态
		isCurrent := (idx == state.Index)
		ansVal, hasAns := state.ExamAnswers[q.ID]
		isAnswered := hasAns && ansVal != "" && ansVal != "null"

		// 2. 根据状态决定动态颜色（两种答题状态：已答 vs 未答，以及当前题目高亮）
		bgColor := core.HexColor(core.CardBgColor)
		borderColor := core.HexColor(core.BorderMediumColor)

		if isCurrent {
			bgColor = core.HexColor(core.ColorSelectedBg)
			borderColor = core.HexColor(core.ColorSelectedBorder)
		} else if isAnswered {
			bgColor = core.HexColor(core.ColorCorrectBg)
			borderColor = core.HexColor(core.ColorCorrectBorder)
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

		// 4. 创建底层按钮
		box := customElements.CreateButton(
			"",
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
				state.Index = idx
				onNavigate()
				if modal != nil {
					modal.Hide()
				}
			},
		)

		// 5. 用绝对布局在按钮正上方绘制数字和题型
		contentBox := container.NewWithoutLayout()
		contentBox.Resize(fyne.NewSize(core.ModalGridItemSize, core.ModalGridItemSize))

		numText := canvas.NewText(fmt.Sprintf("%d", idx+1), core.HexColor(core.TextBodyColor))
		numText.TextSize = 14
		numText.TextStyle = fyne.TextStyle{Bold: true}
		numText.Resize(numText.MinSize())

		typeText := canvas.NewText(shortType, core.HexColor(core.TextBodyColor))
		typeText.TextSize = 12
		typeText.Resize(typeText.MinSize())

		totalH := numText.MinSize().Height + typeText.MinSize().Height + 1
		startY := (core.ModalGridItemSize - totalH) / 2

		numX := (core.ModalGridItemSize - numText.MinSize().Width) / 2
		numText.Move(fyne.NewPos(numX, startY))

		typeX := (core.ModalGridItemSize - typeText.MinSize().Width) / 2
		typeText.Move(fyne.NewPos(typeX, startY+numText.MinSize().Height+1))

		contentBox.Add(numText)
		contentBox.Add(typeText)

		cellContainer := container.NewStack(box, contentBox)
		grid.Add(cellContainer)
	}

	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(320, 360))

	titleText := canvas.NewText("答题卡 (题目列表)", core.HexColor(core.TextPrimaryColor))
	titleText.TextSize = core.FontSizeDialogMsg
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleContainer := container.NewCenter(titleText)

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

// showExamDetailPage 获取考试结果详情，并复用 ShowPractice 渲染考试详情页面（显示用户选择和正确结果，类同顺序练习）。
func showExamDetailPage(w fyne.Window, state *core.AppState, exam *network.MyExamItem, onBack func()) {
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
		localQuestions := make([]core.Question, len(result.Items))
		state.PracticeRecords = make(map[string]string)
		for i, item := range result.Items {
			var convertedOpts []core.Option
			for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
				if text, ok := item.Options[k]; ok && text != "" {
					convertedOpts = append(convertedOpts, core.Option{Label: k, Text: text})
				}
			}
			if len(convertedOpts) == 0 {
				for label, text := range item.Options {
					convertedOpts = append(convertedOpts, core.Option{Label: label, Text: text})
				}
			}
			var answers []string
			cleanAns := strings.NewReplacer("、", "", ",", "", " ", "").Replace(item.CorrectAnswer)
			for _, char := range cleanAns {
				answers = append(answers, string(char))
			}
			if len(answers) == 0 && item.CorrectAnswer != "" {
				answers = append(answers, item.CorrectAnswer)
			}
			localQuestions[i] = core.Question{
				ID:      item.QuestionID,
				Type:    item.QuestionType,
				Content: item.Content,
				Options: convertedOpts,
				Answers: answers,
				Score:   1.0,
			}
			if item.UserAnswer != "" && item.UserAnswer != "null" {
				state.PracticeRecords[item.QuestionID] = item.UserAnswer
			}
		}
		state.CurrentList = localQuestions
		state.Index = 0
		state.Title = exam.BankName + " - 考试详情"
		state.CurrentFileName = exam.BankName
		practiceRelated.ShowPractice(w, state, func(win fyne.Window, s *core.AppState) {
			onBack()
		})
	})
}
