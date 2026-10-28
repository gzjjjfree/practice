package managementRelated

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
)


// showTemplateDetailDialog 显示模板详情选择对话框，允许用户选择并应用模板参数。
//
// 功能说明：
// 显示模板详情选择对话框，包含模板列表的单选按钮组，以及"应用模板"和"取消"按钮。用户选择模板后点击"应用模板"按钮可将模板参数应用到考试设置中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - templateInfos: []tplInfo，模板信息及详情列表。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
func showTemplateDetailDialog(ws *AdminExamCreateState, templateInfos []tplInfo, w fyne.Window) {
	var displayNames []string
	for _, ti := range templateInfos {
		displayNames = append(displayNames, formatTemplateInfo(ti))
	}

	radioGroup := widget.NewRadioGroup(displayNames, func(s string) {
		for _, ti := range templateInfos {
			if formatTemplateInfo(ti) == s {
				ws.selectedTemplateID = ti.Item.TemplateID
				if ti.Detail != nil {
					ws.templateDetail = ti.Detail
					ws.typeCounts = map[string]int{
						"单选题": ti.Detail.SingleCount,
						"多选题": ti.Detail.MultiCount,
						"判断题": ti.Detail.JudgeCount,
						"填空题": ti.Detail.BlankCount,
						"问答题": ti.Detail.EssayCount,
					}
					if ws.nameEntry != nil {
						ws.nameEntry.SetText(ti.Item.ExamName)
					}
					if ws.durationEntry != nil {
						ws.durationEntry.SetText(fmt.Sprintf("%d", ti.Item.DurationMin))
					}
					if ws.typeEntries != nil {
						ws.typeEntries["单选题"].SetText(fmt.Sprintf("%d", ti.Detail.SingleCount))
						ws.typeEntries["多选题"].SetText(fmt.Sprintf("%d", ti.Detail.MultiCount))
						ws.typeEntries["判断题"].SetText(fmt.Sprintf("%d", ti.Detail.JudgeCount))
						ws.typeEntries["填空题"].SetText(fmt.Sprintf("%d", ti.Detail.BlankCount))
						ws.typeEntries["问答题"].SetText(fmt.Sprintf("%d", ti.Detail.EssayCount))
					}
				}
				break
			}
		}
	})

	if len(templateInfos) > 0 {
		radioGroup.SetSelected(displayNames[0])
	}

	selectLabel := widget.NewLabel(core.AdminTemplateSelectPrompt)
	selectLabel.Wrapping = fyne.TextWrapWord
	selectLabel.TextStyle = fyne.TextStyle{Bold: true}
	radioGroup.Horizontal = false

	var dlg dialog.Dialog
	// 应用模板按钮（标识符：applyBtn，名称："应用模板"）：确认并应用选中的模板参数到考试设置中，包括考试名称、时长及各题型数量。
	applyBtn := customElements.CreateButton(
		core.AdminApplyTemplateBtn,
		core.DialogActionBtnWidth, core.DialogActionBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1, 16, true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if ws.selectedTemplateID > 0 && ws.templateDetail != nil {
				applyTemplate(ws)
				customElements.ShowCustomInformation(core.AdminSuccessText, fmt.Sprintf("已从模板【%s】快速设置考试参数", ws.templateDetail.ExamName), w)
			}
			if dlg != nil {
				dlg.Hide()
			}
		},
	)
	// 取消按钮（标识符：cancelBtn，名称："取消"）：关闭模板选择对话框，不应用任何模板参数。
	cancelBtn := customElements.CreateButton(
		core.AdminCancelText,
		core.DialogActionBtnWidth, core.DialogActionBtnHeight,
		core.HexColor(core.TextBodyColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		1.5, core.AdminDialogBtnFontSize, false, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if dlg != nil {
				dlg.Hide()
			}
		},
	)

	btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), applyBtn)
	content := container.NewVBox(
		selectLabel,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		radioGroup,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		container.NewCenter(btnRow),
	)

	dlg = dialog.NewCustomWithoutButtons(core.AdminTemplateDialogTitle, content, w)
	dlg.Show()
}

// applyTemplate 将选中的模板参数应用到考试设置页面（名称、时长、各题型数量）。
//
// 功能说明：
// 将选中的模板参数（考试名称、考试时长、各题型数量）应用到考试设置页面的输入框中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func applyTemplate(ws *AdminExamCreateState) {
	if ws.templateDetail == nil {
		return
	}
	ws.examName = ws.templateDetail.ExamName
	ws.durationMin = ws.templateDetail.DurationMin

	if ws.nameEntry != nil {
		ws.nameEntry.SetText(ws.templateDetail.ExamName)
	}
	if ws.durationEntry != nil {
		ws.durationEntry.SetText(fmt.Sprintf("%d", ws.templateDetail.DurationMin))
	}

	bankCounts := make(map[string]int)
	if ws.sourceTab == 0 {
		for _, q := range ws.localQuestions {
			bankCounts[q.Type]++
		}
	} else {
		for _, sq := range ws.serverQuestions {
			qLocal := serverQuestionToLocal(&sq)
			bankCounts[qLocal.Type]++
		}
	}

	singleCount := ws.templateDetail.SingleCount
	if singleCount > bankCounts["单选题"] {
		singleCount = bankCounts["单选题"]
	}
	multiCount := ws.templateDetail.MultiCount
	if multiCount > bankCounts["多选题"] {
		multiCount = bankCounts["多选题"]
	}
	judgeCount := ws.templateDetail.JudgeCount
	if judgeCount > bankCounts["判断题"] {
		judgeCount = bankCounts["判断题"]
	}
	blankCount := ws.templateDetail.BlankCount
	if blankCount > bankCounts["填空题"] {
		blankCount = bankCounts["填空题"]
	}
	essayCount := ws.templateDetail.EssayCount
	if essayCount > bankCounts["问答题"] {
		essayCount = bankCounts["问答题"]
	}

	if ws.typeCounts == nil {
		ws.typeCounts = make(map[string]int)
	}
	ws.typeCounts["单选题"] = singleCount
	ws.typeCounts["多选题"] = multiCount
	ws.typeCounts["判断题"] = judgeCount
	ws.typeCounts["填空题"] = blankCount
	ws.typeCounts["问答题"] = essayCount

	if ws.typeEntries != nil {
		ws.typeEntries["单选题"].SetText(fmt.Sprintf("%d", singleCount))
		ws.typeEntries["多选题"].SetText(fmt.Sprintf("%d", multiCount))
		ws.typeEntries["判断题"].SetText(fmt.Sprintf("%d", judgeCount))
		ws.typeEntries["填空题"].SetText(fmt.Sprintf("%d", blankCount))
		ws.typeEntries["问答题"].SetText(fmt.Sprintf("%d", essayCount))
	}
	if ws.typeScoreEntries != nil {
		for _, t := range []string{"单选题", "多选题", "判断题", "填空题", "问答题"} {
			if ws.typeScoreEntries[t] != nil {
				ws.typeScoreEntries[t].SetText("1")
			}
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			ws.typeScores[t] = 1.0
		}
	}
}
