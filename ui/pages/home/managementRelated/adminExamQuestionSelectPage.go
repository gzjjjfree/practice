package managementRelated

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)


// ==================== Step 2: 筛选条件 ====================

// ==================== Step 3: 题目选择列表 ====================

// buildQuestionSelectPage 构建题目选择列表页面，显示根据筛选条件过滤后的题目卡片。
//
// 功能说明：
// 构建考试创建向导的第四步页面，显示根据用户设置的筛选条件（题型、难度）和题库来源过滤后的题目卡片列表。
// 若考试参数尚未确认，则显示提示信息；否则渲染本地或服务器题目列表。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - fyne.CanvasObject：题目选择列表页面的UI容器对象。
func buildQuestionSelectPage(ws *AdminExamCreateState) fyne.CanvasObject {
	if !ws.examParamsConfirmed {
		noParamsData := canvas.NewText(core.AdminParamsNotSetMsg, core.HexColor(core.TextMutedColor))
		noParamsData.Alignment = fyne.TextAlignCenter
		noParamsData.TextSize = 30
		return container.NewPadded(container.NewCenter(
			noParamsData,
		))
	}

	if ws.questionFilterTypes == nil {
		ws.questionFilterTypes = make(map[string]bool)
	}

	ws.qListVBox = container.NewVBox()
	ws.qListScroll = container.NewVScroll(ws.qListVBox)

	ws.titleLabel = buildQuestionSelectTitle(ws)

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	var checkObjs []fyne.CanvasObject
	checkBoxes := make(map[string]*widget.Check)
	throttledRefresh := throttleFunc(func() {
		refreshQuestionList(ws)
	}, 150*time.Millisecond)

	var updating bool
	for _, qType := range typeOrder {
		t := qType
		cb := widget.NewCheck(t, func(checked bool) {
			if updating {
				return
			}
			updating = true
			defer func() { updating = false }()

			if checked {
				for otherType, otherCb := range checkBoxes {
					if otherType != t {
						otherCb.SetChecked(false)
						ws.questionFilterTypes[otherType] = false
					}
				}
				ws.questionFilterTypes[t] = true
			} else {
				ws.questionFilterTypes[t] = false
			}
			throttledRefresh()
		})
		cb.SetChecked(ws.questionFilterTypes[t])
		checkBoxes[t] = cb
		checkObjs = append(checkObjs, cb)
	}
	filterBox := container.NewGridWrap(fyne.NewSize(130, core.LayoutSpacingH20), checkObjs...)

	// 随机填满按钮（标识符：randomFillBtn，名称："🎲 随机填满"）：一键随机填满各题型所需数量。
	randomFillBtn := customElements.CreateButton(
		"🎲 随机填满",
		110, 32,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1, 14, true, false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		throttleFunc(func() {
			randomFillQuestions(ws)
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
		}, 200*time.Millisecond),
	)

	filterRow := container.NewBorder(nil, nil, filterBox, randomFillBtn)

	topContainer := container.NewVBox(
		ws.titleLabel,
		container.NewGridWrap(fyne.NewSize(1, 6)),
		filterRow,
		container.NewGridWrap(fyne.NewSize(1, 6)),
	)

	if ws.sourceTab == 0 && len(ws.localQuestions) > 0 {
		renderLocalQuestions(ws)
	} else if ws.sourceTab == 1 && len(ws.serverQuestions) > 0 {
		renderServerQuestions(ws)
	} else {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		ws.qListVBox.Add(noData)
	}

	return container.NewBorder(topContainer, nil, nil, nil, ws.qListScroll)
}

// buildQuestionSelectTitle 构建题目选择页面的标题标签，显示各题型已选/所需数量。
//
// 功能说明：
// 根据当前已选题目数量和模板配置的各题型所需数量，生成题目选择页面的标题文本，格式为"选择题目（单选题：x/y, 多选题：a/b, ...）"。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - *widget.Label：题目选择页面的标题标签对象。
func buildQuestionSelectTitle(ws *AdminExamCreateState) *widget.Label {
	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	var parts []string

	for _, qType := range typeOrder {
		count := 0
		for _, q := range ws.localQuestions {
			if q.Type == qType && ws.selectedQIDs[q.ID] {
				count++
			}
		}
		for _, q := range ws.serverQuestions {
			qLocal := serverQuestionToLocal(&q)
			if qLocal.Type == qType && ws.selectedQIDs[qLocal.ID] {
				count++
			}
		}
		required := 0
		if ws.typeCounts != nil {
			required = ws.typeCounts[qType]
		}
		if required > 0 {
			parts = append(parts, fmt.Sprintf("%s：%d/%d", qType, count, required))
		}
	}

	titleText := "选择题目（" + strings.Join(parts, ", ") + "）"
	title := widget.NewLabel(titleText)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Wrapping = fyne.TextWrapWord

	return title
}

// updateQuestionSelectTitle 更新题目选择页面的标题标签。
//
// 功能说明：
// 重新构建题目选择页面的标题标签，并更新UI中的标题文本、样式和换行属性。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func updateQuestionSelectTitle(ws *AdminExamCreateState) {
	if ws.titleLabel != nil {
		newTitle := buildQuestionSelectTitle(ws)
		ws.titleLabel.Text = newTitle.Text
		ws.titleLabel.TextStyle = newTitle.TextStyle
		ws.titleLabel.Wrapping = newTitle.Wrapping
		ws.titleLabel.Refresh()
	}
}

// renderLocalQuestions 渲染本地题库中的题目列表，按题型分组。
//
// 功能说明：
// 从本地题库中过滤题目，并按题型分组渲染题目卡片列表到UI容器中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func renderLocalQuestions(ws *AdminExamCreateState) {
	ws.qListVBox.Objects = nil

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	groupedMap := make(map[string][]core.Question)
	for _, q := range ws.localQuestions {
		groupedMap[q.Type] = append(groupedMap[q.Type], q)
	}

	for _, qType := range typeOrder {
		if !ws.questionFilterTypes[qType] {
			continue
		}
		questions, ok := groupedMap[qType]
		if !ok || len(questions) == 0 {
			continue
		}

		section := buildGroupedSection(qType, questions, ws.selectedQIDs, ws, func() {
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		})
		ws.qListVBox.Add(section)
		ws.qListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.AdminListSpacingH8)))
	}

	if len(ws.qListVBox.Objects) == 0 {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = core.AdminInfoTextSize
		ws.qListVBox.Add(noData)
	}
}

// renderServerQuestions 渲染服务器题库中的题目列表，按题型分组。
//
// 功能说明：
// 从服务器题库中过滤题目，并按题型分组渲染题目卡片列表到UI容器中。
// 仅显示模板配置中所需数量大于0的题型对应的题目。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func renderServerQuestions(ws *AdminExamCreateState) {
	ws.qListVBox.Objects = nil

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	groupedMap := make(map[string][]core.Question)

	for _, q := range ws.serverQuestions {
		localQ := serverQuestionToLocal(&q)
		required := 0
		if ws.typeCounts != nil {
			required = ws.typeCounts[q.Type]
		}
		if required == 0 {
			continue
		}
		groupedMap[q.Type] = append(groupedMap[q.Type], *localQ)
	}

	for _, qType := range typeOrder {
		if !ws.questionFilterTypes[qType] {
			continue
		}
		questions, ok := groupedMap[qType]
		if !ok || len(questions) == 0 {
			continue
		}

		section := buildGroupedSection(qType, questions, ws.selectedQIDs, ws, func() {
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		})
		ws.qListVBox.Add(section)
		ws.qListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.ListWrapHeight)))
	}

	if len(ws.qListVBox.Objects) == 0 {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = core.AdminInfoTextSize
		ws.qListVBox.Add(noData)
	}
}

// ==================== Step 4: 考试参数设置 ====================

// buildTemplateSelectBtn 创建模板选择按钮：点击后弹出模板选择对话框，从服务器获取模板列表。
//
// 功能说明：
// 构建并返回"选择模板"按钮，点击该按钮会弹出模板选择对话框，从服务器获取可用模板列表及详情供用户选择。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - state: *core.AppState，全局应用状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//
// 返回值：
//   - *customElements.CustomButton：模板选择按钮对象。
func buildTemplateSelectBtn(ws *AdminExamCreateState, state *core.AppState, w fyne.Window) *customElements.CustomButton {
	// 模板选择按钮（标识符：返回的自定义按钮对象，名称："选择模板"）：点击后弹出模板选择对话框，从服务器获取可用模板列表及详情供用户选择。

	return customElements.CreateButton(
		core.AdminTemplateBtnText,
		core.BtnMediumWideWidth, core.BtnMediumWideHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnSuccessBg),
		core.HexColor(core.BtnSuccessStroke),
		1, core.AdminConfirmBtnFontSize, true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			showTemplateSelectDialog(ws, state, w)
		},
	)
}

type tplInfo struct {
	Item      network.TemplateItem
	Detail    *network.TemplateDetailItem
	HasDetail bool
}

// formatTemplateInfo 格式化模板信息字符串，包含考试名称、总题数及各题型题数分布。
//
// 功能说明：
// 将模板信息格式化为可读的字符串，格式为"考试名称（共 X 题）[单选题X/多选题Y/判断题Z/填空题W/问答题V]"。
//
// 输入参数：
//   - ti: tplInfo，包含模板基本信息、详情信息及是否有详情的结构体。
//
// 返回值：
//   - string：格式化后的模板信息字符串。
func formatTemplateInfo(ti tplInfo) string {
	info := fmt.Sprintf("%s（共 %d 题）", ti.Item.ExamName, ti.Item.QuestionCount)
	if ti.HasDetail && ti.Detail != nil {
		var parts []string
		if ti.Detail.SingleCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeSingleChoice, ti.Detail.SingleCount))
		}
		if ti.Detail.MultiCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeMultiChoice, ti.Detail.MultiCount))
		}
		if ti.Detail.JudgeCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeJudge, ti.Detail.JudgeCount))
		}
		if ti.Detail.BlankCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeFillIn, ti.Detail.BlankCount))
		}
		if ti.Detail.EssayCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeEssay, ti.Detail.EssayCount))
		}
		if len(parts) > 0 {
			info += " [" + strings.Join(parts, "/") + "]"
		}
	}
	return info
}

// showTemplateSelectDialog 显示模板选择对话框，从服务器获取模板列表及详情供用户选择。
//
// 功能说明：
// 从服务器获取可用模板列表，并并发获取每个模板的详细信息，然后显示模板详情选择对话框供用户选择并应用模板参数。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - state: *core.AppState，全局应用状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
func showTemplateSelectDialog(ws *AdminExamCreateState, state *core.AppState, w fyne.Window) {
	state.FetchTemplates(100, 0, func(success bool, templates []network.TemplateItem, total int, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.AdminErrorText, msg, w)
			return
		}

		if len(templates) == 0 {
			customElements.ShowCustomInformation(core.AdminErrorText, core.AdminNoQuestionsAvailableMsg, w)
			return
		}

		tplInfos := make([]tplInfo, len(templates))
		for i, tmpl := range templates {
			tplInfos[i] = tplInfo{Item: tmpl, HasDetail: false}
		}

		var fetchedCount int
		var mu sync.Mutex

		for i := range tplInfos {
			tmplID := tplInfos[i].Item.TemplateID
			go func(idx int, id int) {
				state.GetTemplateDetail(id, func(success bool, detail *network.TemplateDetailItem, detailMsg string) {
					mu.Lock()
					defer mu.Unlock()
					if success && detail != nil {
						tplInfos[idx].Detail = detail
						tplInfos[idx].HasDetail = true
					}
					fetchedCount++
					if fetchedCount == len(tplInfos) {
						showTemplateDetailDialog(ws, tplInfos, w)
					}
				})
			}(i, tmplID)
		}
	})
}
