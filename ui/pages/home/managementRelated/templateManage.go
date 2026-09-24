package managementRelated

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)

// ShowTemplateManage 渲染考试模板管理页面。
// 该函数负责展示所有考试模板的列表，并提供刷新、编辑、删除、取消考试和导出结果等功能。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框等UI元素。
//   - state: *core.AppState 类型，表示全局应用状态，用于获取模板列表、模板详情等操作。
//   - backToHome: func() 类型，返回主页的回调函数。
func ShowTemplateManage(w fyne.Window, state *core.AppState, backToHome func()) {
	// 返回主页按钮：标识符为"backBtn"，功能为点击后返回到主页（调用 backToHome 回调）。
	backBtn := customElements.CreateButton(
		core.AdminBackBtnText,
		core.BackBtnWidth,
		core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() { backToHome() },
	)

	titleText := customElements.CreateLabel(core.TemplateManageTitleText, core.HexColor(core.TextPrimaryColor), core.FontSizePageTitle, true, true, false)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleText)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// Template list
	var templateListVBox *fyne.Container
	var refreshTemplateList func()

	refreshTemplateList = func() {
		if !state.IsLoggedIn {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.TemplateManagePleaseLoginMsg, w)
			backToHome()
			return
		}

		state.FetchTemplates(100, 0, func(success bool, templates []network.TemplateItem, total int, msg string) {
			if !success {
				customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
				return
			}

			if templateListVBox == nil {
				templateListVBox = container.NewVBox()
			}

			if len(templates) == 0 {
				noDataText := customElements.CreateLabel(core.TemplateManageNoDataText, core.HexColor(core.TextMutedColor), core.FontSizeSubtitle, false, true, false)
				templateListVBox.Objects = []fyne.CanvasObject{container.NewPadded(noDataText)}
				templateListVBox.Refresh()
				return
			}

			// Fetch details for each template and render cards preserving order
			var mu sync.Mutex
			fetchedCount := 0

			type templateDetailData struct {
				Item   network.TemplateItem
				Detail *network.TemplateDetailItem
			}
			results := make([]templateDetailData, len(templates))

			for i, tmpl := range templates {
				go func(idx int, item network.TemplateItem) {
					state.GetTemplateDetail(item.TemplateID, func(success bool, detail *network.TemplateDetailItem, detailMsg string) {
						mu.Lock()
						results[idx] = templateDetailData{Item: item, Detail: detail}
						fetchedCount++
						isDone := (fetchedCount == len(templates))
						mu.Unlock()

						if isDone {
							var objects []fyne.CanvasObject
							for _, res := range results {
								resItem := res.Item // copy for closures
								tmplCard := buildTemplateCardWithDetail(&resItem, res.Detail, state, func() {
									showUpdateTemplateDialog(w, state, &resItem, func() {
										refreshTemplateList()
									})
								}, func() {
									showDeleteTemplateConfirm(w, state, &resItem, func() {
										refreshTemplateList()
									})
								}, func() {
									showCancelExamConfirm(w, state, &resItem, func() {
										refreshTemplateList()
									})
								}, func() {
									showExportResultsDialog(w, state, &resItem)
								})
								objects = append(objects, tmplCard, container.NewGridWrap(fyne.NewSize(1, 8)))
							}

							// Safely update the container objects on completion
							templateListVBox.Objects = objects
							fyne.Do(func() { templateListVBox.Refresh() })
						}
					})
				}(i, tmpl)
			}
		})
	}

	// Initial render
	refreshTemplateList()

	scrollContent := container.NewVScroll(templateListVBox)
	scrollContent.SetMinSize(fyne.NewSize(0, 400))

	mainLayout := container.NewBorder(topBar, nil, nil, nil, scrollContent)
	wRootLayout := container.NewStack(pageBg, mainLayout)
	w.SetContent(wRootLayout)
}

// buildTemplateCardWithDetail 为模板项创建一个卡片，包含各题型的问题数量。
//
// 参数:
//   - tmpl: *network.TemplateItem 类型，表示模板项的基本信息。
//   - detail: *network.TemplateDetailItem 类型，表示模板项的详细信息（各题型数量）。
//   - state: *core.AppState 类型，表示全局应用状态（当前未直接使用，但作为参数保留以保持一致性）。
//   - onEdit: func() 类型，编辑按钮的回调函数。
//   - onDelete: func() 类型，删除按钮的回调函数。
//   - onCancel: func() 类型，取消考试按钮的回调函数。
//   - onExport: func() 类型，导出结果按钮的回调函数。
//
// 返回值:
//   - fyne.CanvasObject 类型，返回构建好的模板卡片对象。
func buildTemplateCardWithDetail(tmpl *network.TemplateItem, detail *network.TemplateDetailItem, state *core.AppState, onEdit func(), onDelete func(), onCancel func(), onExport func()) fyne.CanvasObject {
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
	infoParts := []string{
		fmt.Sprintf(core.TemplateInfoQuestionCountLabel, tmpl.QuestionCount),
		fmt.Sprintf(core.TemplateInfoDurationLabel, tmpl.DurationMin),
		fmt.Sprintf(core.TemplateInfoSourceLabel, tmpl.BankSource),
	}

	// Add per-type counts if available
	if detail != nil {
		var typeParts []string
		if detail.SingleCount > 0 {
			typeParts = append(typeParts, fmt.Sprintf(core.TemplateTypeSingleChoice, detail.SingleCount))
		}
		if detail.MultiCount > 0 {
			typeParts = append(typeParts, fmt.Sprintf(core.TemplateTypeMultiChoice, detail.MultiCount))
		}
		if detail.JudgeCount > 0 {
			typeParts = append(typeParts, fmt.Sprintf(core.TemplateTypeJudge, detail.JudgeCount))
		}
		if detail.BlankCount > 0 {
			typeParts = append(typeParts, fmt.Sprintf(core.TemplateTypeFillIn, detail.BlankCount))
		}
		if detail.EssayCount > 0 {
			typeParts = append(typeParts, fmt.Sprintf(core.TemplateTypeEssay, detail.EssayCount))
		}
		if len(typeParts) > 0 {
			infoParts = append(infoParts, core.TemplateInfoTypePrefix+strings.Join(typeParts, "/"))
		}
	}

	infoText := canvas.NewText(strings.Join(infoParts, "  |  "), core.HexColor(core.TextSecondaryColor))
	infoText.TextSize = core.TemplateInfoTextFontSize

	// Time info
	timeText := canvas.NewText(fmt.Sprintf("开始: %s\n结束: %s", tmpl.StartTime, tmpl.EndTime), core.HexColor(core.TextMutedColor))
	timeText.TextSize = core.TemplateTimeTextFontSize

	// 按钮行
	// 编辑按钮：标识符为"editBtn"，功能为点击后打开编辑模板对话框（调用 onEdit 回调）。
	editBtn := customElements.CreateButton(
		core.TemplateEditBtnText,
		core.TemplateEditBtnWidth,
		core.TemplateBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onEdit,
	)

	// 取消考试按钮：标识符为"cancelBtn"，功能为点击后弹出确认对话框以取消当前考试（调用 onCancel 回调）。该按钮在模板状态不为"active"时禁用。
	cancelBtn := customElements.CreateButton(
		core.TemplateCancelExamBtnText,
		core.TemplateCancelExamBtnWidth,
		core.TemplateBtnHeight,
		core.HexColor(core.TextDisabledColor),
		core.HexColor(core.ColorWarningBg),
		core.HexColor(core.ColorWarningBorder),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		tmpl.Status != "active",
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onCancel,
	)

	// 导出结果按钮：标识符为"exportBtn"，功能为点击后打开导出考试结果的对话框（调用 onExport 回调）。
	exportBtn := customElements.CreateButton(
		core.TemplateExportResultsBtnText,
		core.TemplateExportResultsBtnWidth,
		core.TemplateBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnSecondaryBg),
		core.HexColor(core.BtnSecondaryBg),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onExport,
	)

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

	btnRow := container.NewHBox(editBtn, cancelBtn, exportBtn, deleteBtn)

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, statusBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		timeText,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		btnRow,
	)

	return container.NewStack(
		bg,
		container.NewPadded(cardContent),
	)
}

// showUpdateTemplateDialog 显示用于更新模板的对话框。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于获取和更新模板详情及执行更新操作。
//   - tmpl: *network.TemplateItem 类型，表示要更新的模板项的基本信息。
//   - onUpdated: func() 类型，更新成功后的回调函数，用于刷新模板列表。
func showUpdateTemplateDialog(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onUpdated func()) {
	examNameEntry := widget.NewEntry()
	examNameEntry.SetText(tmpl.ExamName)

	durationEntry := widget.NewEntry()
	durationEntry.SetText(fmt.Sprintf("%d", tmpl.DurationMin))

	// Per-type count entries
	typeLabels := []string{core.QTypeSingleChoice, core.QTypeMultiChoice, core.QTypeJudge, core.QTypeFillIn, core.QTypeEssay}
	typeEntries := make(map[string]*widget.Entry)
	var typeRows []fyne.CanvasObject

	for _, t := range typeLabels {
		entry := widget.NewEntry()
		entry.SetPlaceHolder(core.TemplateCountEntryPlaceholder)
		typeEntries[t] = entry

		countLabel := canvas.NewText(t+":", core.HexColor(core.TextSecondaryColor))
		countLabel.TextSize = core.TemplateTypeLabelFontSize

		row := container.NewBorder(nil, nil, countLabel, nil, entry)
		typeRows = append(typeRows, row)
	}

	// Fetch template detail to populate per-type counts
	state.GetTemplateDetail(tmpl.TemplateID, func(success bool, detail *network.TemplateDetailItem, msg string) {
		if !success || detail == nil {
			return
		}
		typeEntries[core.QTypeSingleChoice].SetText(fmt.Sprintf("%d", detail.SingleCount))
		typeEntries[core.QTypeMultiChoice].SetText(fmt.Sprintf("%d", detail.MultiCount))
		typeEntries[core.QTypeJudge].SetText(fmt.Sprintf("%d", detail.JudgeCount))
		typeEntries[core.QTypeFillIn].SetText(fmt.Sprintf("%d", detail.BlankCount))
		typeEntries[core.QTypeEssay].SetText(fmt.Sprintf("%d", detail.EssayCount))
	})

	updateBtn := widget.NewButton(core.TemplateUpdateBtnText, func() {
		examName := examNameEntry.Text

		durationMin, err := strconv.Atoi(durationEntry.Text)
		if err != nil || durationMin <= 0 {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.TemplateDurationInvalidMsg, w)
			return
		}

		// Build per-type counts from input
		typeCounts := make(map[string]int)
		for _, t := range typeLabels {
			if typeEntries[t].Text != "" {
				if v, err := strconv.Atoi(typeEntries[t].Text); err == nil {
					typeCounts[t] = v
				}
			}
		}

		// Build request with per-type counts
		var req network.TemplateUpdateReq
		req.ExamName = examName
		req.DurationMin = durationMin

		// Only include non-zero type counts
		if c, ok := typeCounts[core.QTypeSingleChoice]; ok && c > 0 {
			v := c
			req.SingleCount = &v
		}
		if c, ok := typeCounts[core.QTypeMultiChoice]; ok && c > 0 {
			v := c
			req.MultiCount = &v
		}
		if c, ok := typeCounts[core.QTypeJudge]; ok && c > 0 {
			v := c
			req.JudgeCount = &v
		}
		if c, ok := typeCounts[core.QTypeFillIn]; ok && c > 0 {
			v := c
			req.BlankCount = &v
		}
		if c, ok := typeCounts[core.QTypeEssay]; ok && c > 0 {
			v := c
			req.EssayCount = &v
		}

		state.UpdateTemplate(tmpl.TemplateID, req, func(success bool, msg string) {
			if success {
				customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, core.TemplateUpdateSuccessMsg, w)
				if onUpdated != nil {
					onUpdated()
				}
			} else {
				customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
			}
		})
	})

	var dlg dialog.Dialog
	cancelBtn := widget.NewButton(core.TemplateCancelBtnText, func() {
		if dlg != nil {
			dlg.Hide()
		}
	})

	btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), updateBtn)

	// Build type counts section
	typeCountsTitle := canvas.NewText(core.TemplateEditTypeCountsTitle, core.HexColor(core.TextPrimaryColor))
	typeCountsTitle.TextSize = core.TemplateEditTitleFontSize
	typeCountsTitle.TextStyle = fyne.TextStyle{Bold: true}
	typeCountsTitle.Alignment = fyne.TextAlignCenter

	typeCountsForm := container.NewVBox(typeRows...)

	content := container.NewVBox(
		widget.NewLabel(core.TemplateEditDialogTitle),
		container.NewGridWrap(fyne.NewSize(1, 10)),
		widget.NewLabel(core.TemplateEditExamNameLabel),
		examNameEntry,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		widget.NewLabel(core.TemplateEditExamDurationLabel),
		durationEntry,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		typeCountsTitle,
		container.NewGridWrap(fyne.NewSize(1, 8)),
		typeCountsForm,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		container.NewCenter(btnRow),
	)

	dlg = dialog.NewCustomWithoutButtons(core.TemplateEditDialogTitle, content, w)
	dlg.Show()
}

// showDeleteTemplateConfirm 显示用于删除模板的确认对话框。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行删除模板操作。
//   - tmpl: *network.TemplateItem 类型，表示要删除的模板项的基本信息。
//   - onDeleted: func() 类型，删除成功后的回调函数，用于刷新模板列表。
func showDeleteTemplateConfirm(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onDeleted func()) {
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

// showCancelExamConfirm 显示用于取消考试的确认对话框。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行取消考试操作。
//   - tmpl: *network.TemplateItem 类型，表示要取消考试的模板项的基本信息。
//   - onCanceled: func() 类型，取消成功后的回调函数，用于刷新模板列表。
func showCancelExamConfirm(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onCanceled func()) {
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

// showExportResultsDialog 显示用于导出考试结果的对话框。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行导出考试结果操作。
//   - tmpl: *network.TemplateItem 类型，表示要导出结果的模板项的基本信息。
func showExportResultsDialog(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem) {
	formatSelect := widget.NewSelect([]string{"JSON", "CSV", "Excel"}, func(s string) {})
	formatSelect.SetSelected("JSON")

	var dlg dialog.Dialog
	exportBtn := widget.NewButton(core.TemplateUpdateBtnText, func() {
		format := formatSelect.Selected
		state.ExportResults(tmpl.TemplateID, 0, format, func(success bool, results []network.ExportResultItem, msg string) {
			if success {
				summary := fmt.Sprintf("导出成功！共 %d 条记录\n", len(results))
				for _, r := range results {
					var percentage float64
					if r.TotalScore > 0 {
						percentage = (r.Score / r.TotalScore) * 100
					}
					summary += fmt.Sprintf("%s: %.1f/%.1f (%.1f%%)\n", r.Username, r.Score, r.TotalScore, percentage)
				}
				if dlg != nil {
					dlg.Hide()
				}
				customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, summary, w)
			} else {
				customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
			}
		})
	})

	cancelBtn := widget.NewButton(core.TemplateCancelBtnText, func() {
		if dlg != nil {
			dlg.Hide()
		}
	})

	btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), exportBtn)

	content := container.NewVBox(
		widget.NewLabel(core.TemplateExportResultsDialogTitle),
		container.NewGridWrap(fyne.NewSize(1, 10)),
		widget.NewLabel(core.TemplateExportResultsTemplateLabel+tmpl.ExamName),
		container.NewGridWrap(fyne.NewSize(1, 5)),
		widget.NewLabel(core.TemplateExportFormatLabel),
		formatSelect,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		container.NewCenter(btnRow),
	)

	dlg = dialog.NewCustomWithoutButtons(core.TemplateExportResultsDialogTitle, content, w)
	dlg.Show()
}
