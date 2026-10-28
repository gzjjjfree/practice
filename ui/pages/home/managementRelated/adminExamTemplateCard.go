package managementRelated

import (
	"fmt"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)


// buildTemplateCardForPushExam 为模板项创建一个卡片，只保留删除按钮。
//
// 参数:
//   - tmpl: *network.TemplateItem 类型，表示模板项的基本信息。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onDelete: func() 类型，删除按钮的回调函数。
//
// 返回值:
//   - fyne.CanvasObject 类型，返回构建好的模板卡片对象。
func buildTemplateCardForPushExam(tmpl *network.TemplateItem, state *core.AppState, onDelete func()) fyne.CanvasObject {
	// Card background
	bg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	bg.CornerRadius = core.TemplateCardCornerRadius

	// Template name
	nameText := customElements.CreateLabel("📋 "+tmpl.ExamName, core.HexColor(core.TextPrimaryColor), core.FontSizeHeading, true, true, false)

	// Status badge
	statusColor := core.TextMutedColor
	statusText := tmpl.Status
	switch tmpl.Status {
	case "draft":
		statusColor = core.TextHintColor
		statusText = core.TemplateStatusDraftText
	case "active":
		statusColor = core.BtnPrimaryBg
		statusText = core.TemplateStatusActiveText
	case "expired":
		statusColor = core.ColorWrongBorder
		statusText = core.TemplateStatusExpiredText
	}
	statusBadge := canvas.NewText("● "+statusText, core.HexColor(statusColor))
	statusBadge.TextSize = core.TemplateStatusBadgeFontSize

	// Info row
	// Convert bank source to Chinese display
	bankSourceText := "未知来源"
	switch tmpl.BankSource {
	case "local":
		bankSourceText = "本地题库"
	case "server":
		bankSourceText = "服务器题库"
	case "client":
		bankSourceText = "客户端"
	}

	infoParts := []string{
		fmt.Sprintf(core.TemplateInfoQuestionCountLabel, tmpl.QuestionCount),
		fmt.Sprintf(core.TemplateInfoDurationLabel, tmpl.DurationMin),
		fmt.Sprintf(core.TemplateInfoSourceLabel, bankSourceText),
	}

	// Add per-type counts from QuestionTypeDistribution
	if tmpl.QuestionTypeDistribution.SingleChoice > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeSingleChoice, tmpl.QuestionTypeDistribution.SingleChoice))
	}
	if tmpl.QuestionTypeDistribution.MultipleChoice > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeMultiChoice, tmpl.QuestionTypeDistribution.MultipleChoice))
	}
	if tmpl.QuestionTypeDistribution.TrueFalse > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeJudge, tmpl.QuestionTypeDistribution.TrueFalse))
	}
	if tmpl.QuestionTypeDistribution.FillBlank > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeFillIn, tmpl.QuestionTypeDistribution.FillBlank))
	}
	if tmpl.QuestionTypeDistribution.Essay > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeEssay, tmpl.QuestionTypeDistribution.Essay))
	}

	infoText := canvas.NewText(strings.Join(infoParts, "  |  "), core.HexColor(core.TextSecondaryColor))
	infoText.TextSize = core.TemplateInfoTextFontSize

	// Time info - use VBox for proper line breaks
	startTimeText := canvas.NewText(fmt.Sprintf("开始: %s", tmpl.StartTime), core.HexColor(core.TextMutedColor))
	startTimeText.TextSize = core.TemplateTimeTextFontSize

	endTimeText := canvas.NewText(fmt.Sprintf("结束: %s", tmpl.EndTime), core.HexColor(core.TextMutedColor))
	endTimeText.TextSize = core.TemplateTimeTextFontSize

	timeInfoContainer := container.NewVBox(startTimeText, endTimeText)

	// 删除按钮：标识符为"deleteBtn"，功能为点击后弹出确认对话框以删除当前模板（调用 onDelete 回调）。
	deleteBtn := customElements.CreateButton(
		core.TemplateDeleteBtnText,
		core.TemplateDeleteBtnWidth,
		core.TemplateBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.ColorErrorBg),
		core.HexColor(core.ColorErrorBorder),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onDelete,
	)

	btnRow := container.NewHBox(deleteBtn)

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, statusBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		timeInfoContainer,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		btnRow,
	)

	return container.NewStack(
		bg,
		container.NewPadded(cardContent),
	)
}

// showDeleteTemplateConfirmForPushExam 显示用于删除模板的确认对话框（用于推送考试模板列表页面）。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行删除模板操作。
//   - tmpl: *network.TemplateItem 类型，表示要删除的模板项的基本信息。
//   - onDeleted: func() 类型，删除成功后的回调函数，用于刷新模板列表。
func showDeleteTemplateConfirmForPushExam(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onDeleted func()) {
	customElements.ShowCustomConfirm(
		core.TemplateDeleteConfirmTitle,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.TemplateDeleteConfirmMsg, tmpl.ExamName)),
		func(confirm bool) bool {
			if confirm {
				state.DeleteTemplate(tmpl.TemplateID, func(success bool, msg string) {
					if success {
						customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, core.TemplateDeleteSuccessMsg, w)
						if onDeleted != nil {
							onDeleted()
						}
					} else {
						customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
					}
				})
			}
			return true // 返回 true 表示对话框可以关闭
		}, w)
}

// showCancelExamConfirmForTemplate 显示用于取消考试的确认对话框（用于模板列表页面）。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行取消考试操作。
//   - tmpl: *network.TemplateItem 类型，表示要取消考试的模板项的基本信息。
//   - onCanceled: func() 类型，取消成功后的回调函数，用于刷新模板列表。
func showCancelExamConfirmForTemplate(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onCanceled func()) {
	customElements.ShowCustomConfirm(
		core.TemplateCancelExamConfirmTitle,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.TemplateCancelExamConfirmMsg, tmpl.ExamName, 0)),
		func(confirm bool) bool {
			if confirm {
				state.CancelExam(tmpl.TemplateID, "", func(success bool, affectedCount int, msg string) {
					if success {
						customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, fmt.Sprintf(core.TemplateCancelExamSuccessMsg, affectedCount), w)
						if onCanceled != nil {
							onCanceled()
						}
					} else {
						customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
					}
				})
			}
			return true // 返回 true 表示对话框可以关闭
		}, w)
}

// CustomWizardTabs 自定义两行顶部向导选项卡组件，替代默认 AppTabs 以支持固定顶部及分两行显示。
type CustomWizardTabs struct {
	Items       []*container.TabItem
	activeIndex int
	contentArea *fyne.Container
	tabBtns     []*customElements.CustomButton
	onSelected  func(item *container.TabItem)
}

func (w *CustomWizardTabs) Refresh() {
	if w.activeIndex >= 0 && w.activeIndex < len(w.Items) {
		w.contentArea.Objects = []fyne.CanvasObject{w.Items[w.activeIndex].Content}
		w.contentArea.Refresh()
	}
	for i, btn := range w.tabBtns {
		if btn == nil {
			continue
		}
		if i == w.activeIndex {
			btn.SetColors(
				core.HexColor(core.BtnPrimaryBg),
				core.HexColor(core.CardBgColor),
				core.HexColor(core.BtnPrimaryBg),
			)
		} else {
			btn.SetColors(
				core.HexColor(core.CardBgColor),
				core.HexColor(core.TextPrimaryColor),
				core.HexColor(core.BorderLightColor),
			)
		}
	}
}

func (w *CustomWizardTabs) Select(index int) {
	if index < 0 || index >= len(w.Items) {
		return
	}
	w.activeIndex = index
	w.Refresh()
	if w.onSelected != nil {
		w.onSelected(w.Items[index])
	}
}

func NewCustomWizardTabs(items ...*container.TabItem) (*CustomWizardTabs, fyne.CanvasObject) {
	wizard := &CustomWizardTabs{
		Items:       items,
		activeIndex: 0,
		contentArea: container.NewStack(),
	}

	var tabBtns []*customElements.CustomButton
	var row1Objs []fyne.CanvasObject
	var row2Objs []fyne.CanvasObject

	for i, item := range items {
		idx := i
		it := item
		btn := customElements.CreateButton(
			it.Text,
			115, 36,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.TextPrimaryColor),
			core.HexColor(core.BorderLightColor),
			1, 14, false, false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,
			fyne.TextTruncateOff,
			func() {
				wizard.Select(idx)
			},
		)
		tabBtns = append(tabBtns, btn)
		if i < 3 {
			row1Objs = append(row1Objs, btn)
		} else {
			row2Objs = append(row2Objs, btn)
		}
	}
	wizard.tabBtns = tabBtns

	row1 := container.NewHBox(layout.NewSpacer(), row1Objs[0], row1Objs[1], row1Objs[2], layout.NewSpacer())
	var row2 fyne.CanvasObject
	if len(row2Objs) > 0 {
		row2Container := []fyne.CanvasObject{layout.NewSpacer()}
		for _, b := range row2Objs {
			row2Container = append(row2Container, b)
		}
		row2Container = append(row2Container, layout.NewSpacer())
		row2 = container.NewHBox(row2Container...)
	}

	tabBarVBox := container.NewVBox(
		container.NewGridWrap(fyne.NewSize(1, 4)),
		row1,
		container.NewGridWrap(fyne.NewSize(1, 6)),
	)
	if row2 != nil {
		tabBarVBox.Add(row2)
		tabBarVBox.Add(container.NewGridWrap(fyne.NewSize(1, 6)))
	}
	sepLine := canvas.NewRectangle(core.HexColor("#78909C"))
	sepLine.SetMinSize(fyne.NewSize(10, 3))
	tabBarVBox.Add(sepLine)
	tabBarVBox.Add(container.NewGridWrap(fyne.NewSize(1, 4)))

	wizard.Refresh()

	layoutObj := container.NewBorder(tabBarVBox, nil, nil, nil, wizard.contentArea)
	return wizard, layoutObj
}
