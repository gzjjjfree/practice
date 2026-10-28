package managementRelated

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
)


// buildGroupedSection 构建按题型分组的题目卡片区域，包含标题和所有题目卡片。
//
// 功能说明：
// 为指定题型的题目列表构建UI分组区域，包含题型统计标题（总题数、已选题数）和所有题目的详细卡片。
//
// 输入参数：
//   - qType: string，题型名称（如"单选题"、"多选题"等）。
//   - questions: []core.Question，该题型的题目列表。
//   - selected: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - onRefresh: func()，题目选择状态改变时的刷新回调函数。
//
// 返回值：
//   - fyne.CanvasObject：按题型分组的题目卡片区域UI容器对象。
func buildGroupedSection(qType string, questions []core.Question, selected map[string]bool, ws *AdminExamCreateState, onRefresh func()) fyne.CanvasObject {
	totalCount := len(questions)
	selectedCount := 0
	for _, q := range questions {
		if selected[q.ID] {
			selectedCount++
		}
	}

	headerText := fmt.Sprintf("【%s】共 %d 题，已选 %d 题", qType, totalCount, selectedCount)
	headerTextCanvas := canvas.NewText(headerText, core.HexColor(core.TextPrimaryColor))
	headerTextCanvas.TextSize = 15
	headerTextCanvas.TextStyle = fyne.TextStyle{Bold: true}
	headerTextCanvas.Alignment = fyne.TextAlignLeading

	var cardObjs []fyne.CanvasObject
	for idx, q := range questions {
		qCopy := q
		card := buildDetailedQuestionCard(&qCopy, selected, ws, onRefresh, idx+1, qType)
		cardObjs = append(cardObjs, card)
	}

	questionsVBox := container.NewVBox(cardObjs...)

	return container.NewVBox(
		headerTextCanvas,
		container.NewGridWrap(fyne.NewSize(1, 4)),
		questionsVBox,
	)
}

// enforceTypeLimit 检查并限制特定题型的选题数量是否已达到模板设定的上限。
//
// 功能说明：
// 检查指定题目的题型选择数量是否已达到模板配置的上限要求。若已达到上限且题目未被选中，则阻止继续选择。
//
// 输入参数：
//   - q: *core.Question，题目对象。
//   - selectedMap: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - bool：若未达到上限或题目已被选中，返回 true；若已达到上限且题目未被选中，返回 false。
func enforceTypeLimit(q *core.Question, selectedMap map[string]bool, ws *AdminExamCreateState) bool {
	if ws == nil || len(ws.typeCounts) == 0 {
		return true
	}

	required := ws.typeCounts[q.Type]
	if required <= 0 || selectedMap[q.ID] {
		return true
	}

	currentCount := 0
	for _, lq := range ws.localQuestions {
		if lq.Type == q.Type && selectedMap[lq.ID] {
			currentCount++
		}
	}
	for _, sq := range ws.serverQuestions {
		sqID := formatQuestionID(sq.ID)
		if sq.Type == q.Type && selectedMap[sqID] {
			currentCount++
		}
	}

	if currentCount >= required {
		fmt.Printf("[TYPE_LIMIT] %s 选择数量已达上限 (%d/%d)，阻止继续选择\n", q.Type, currentCount, required)
		return false
	}

	return true
}

// buildDetailedQuestionCard 构建单个题目的详细卡片，包含题目内容、选项、答案及选择状态。
//
// 功能说明：
// 为单个题目构建详细的UI卡片，包含题目序号与内容、选项列表（选择/判断题）、答案区域，以及支持点击切换选中状态的背景按钮。
//
// 输入参数：
//   - q: *core.Question，题目对象。
//   - selected: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - onRefresh: func()，题目选择状态改变时的刷新回调函数。
//   - serialNum: int，题目在当前题型中的序号。
//   - qType: string，题型名称。
//
// 返回值：
//   - fyne.CanvasObject：单个题目的详细卡片UI容器对象。
func buildDetailedQuestionCard(q *core.Question, selected map[string]bool, ws *AdminExamCreateState, onRefresh func(), serialNum int, qType string) fyne.CanvasObject {
	isSelected := selected[q.ID]

	bgColorStr := core.CardBgColor
	strokeColorStr := core.BorderLightColor
	checkIcon := "○"

	if isSelected {
		bgColorStr = core.ColorSelectedBg
		strokeColorStr = core.ColorSelectedBorder
		checkIcon = "●"
	}

	bgColor := core.HexColor(bgColorStr)
	strokeColor := core.HexColor(strokeColorStr)

	contentLines := []fyne.CanvasObject{}

	// 1. 构建标题
	titleStr := fmt.Sprintf("%s %d、%s", checkIcon, serialNum, q.Content)
	titleLabel := widget.NewLabel(titleStr)
	titleLabel.Wrapping = fyne.TextWrapWord
	titleLabel.TextStyle = fyne.TextStyle{Bold: false}
	contentLines = append(contentLines, titleLabel)

	// 2. 构建选项列表（选择/判断题）
	if len(q.Options) > 0 && q.Type != core.QTypeFillIn && q.Type != core.QTypeEssay {
		contentLines = append(contentLines, layout.NewSpacer())

		type optItem struct {
			Label string
			Text  string
		}
		var items []optItem
		for _, opt := range q.Options {
			if strings.TrimSpace(opt.Text) == "" {
				continue
			}
			items = append(items, optItem{Label: opt.Label, Text: opt.Text})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].Label < items[j].Label
		})

		var optionObjs []fyne.CanvasObject
		for _, item := range items {
			optText := item.Label + ". " + item.Text
			optLabel := widget.NewLabel(optText)
			optLabel.TextStyle = fyne.TextStyle{Bold: false}
			optLabel.Alignment = fyne.TextAlignLeading
			optionObjs = append(optionObjs, optLabel)
		}

		var rowObjs []fyne.CanvasObject
		for i, opt := range optionObjs {
			rowObjs = append(rowObjs, opt)
			if (i+1)%2 == 0 || i == len(optionObjs)-1 {
				contentLines = append(contentLines, container.NewHBox(rowObjs...))
				rowObjs = nil
			}
		}
	}

	contentLines = append(contentLines, layout.NewSpacer())

	// 3. 构建答案区域
	if q.Type == core.QTypeFillIn || q.Type == core.QTypeEssay {
		var fullAnswer strings.Builder
		for _, ans := range q.Answers {
			fullAnswer.WriteString(ans)
		}
		candidate := fullAnswer.String()
		if candidate == "" {
			for _, opt := range q.Options {
				if strings.TrimSpace(opt.Text) != "" {
					candidate = strings.TrimSpace(opt.Text)
					break
				}
			}
		}
		if candidate == "" {
			candidate = core.AdminNoAnswerText
		}

		ansTextLabel := widget.NewLabel(candidate)
		ansTextLabel.Wrapping = fyne.TextWrapWord
		ansTextLabel.TextStyle = fyne.TextStyle{Bold: true}

		ansHintText := canvas.NewText(core.AdminAnswerHintText, core.HexColor(core.ColorSelectedBorder))
		ansHintText.TextSize = 13
		ansHintText.TextStyle = fyne.TextStyle{Bold: true}

		contentLines = append(contentLines, ansTextLabel, ansHintText)
	} else {
		answerText := canvas.NewText(core.AdminAnswerPrefix+formatAnswer(q, q.Options), core.HexColor(core.ColorSelectedBorder))
		answerText.TextSize = 13
		answerText.TextStyle = fyne.TextStyle{Bold: true}
		contentLines = append(contentLines, answerText)
	}

	// 4. 卡片整体内容
	cardContent := container.NewVBox(contentLines...)
	cardLayout := container.NewPadded(cardContent)

	// 点击回调
	newOnToggle := func() {
		if !enforceTypeLimit(q, selected, ws) {
			return
		}
		selected[q.ID] = !selected[q.ID]
		onRefresh()
	}

	// 5. 使用 CreateButton 创建底层的背景卡片与响应对象（标识符：btnBg，名称："题目卡片背景按钮"）：提供可点击切换题目选中状态的背景按钮，内容由上层覆盖。
	btnBg := customElements.CreateButton(
		"",    // 空文本，因为内容由上层覆盖
		-1, 0, // 宽度 -1 代表宽度自适应填充，高度 0 由容器内部撑开
		core.HexColor(core.TextBodyColor),
		bgColor,
		strokeColor,
		1.0,                 // 边框宽度
		core.FontSizeButton, // 字号参数
		false, false,        // 不加粗，不禁用
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		newOnToggle,
	)

	// 6. 将自定义按钮背景与多行卡片内容通过 Stack 叠加组合
	return container.NewStack(btnBg, cardLayout)
}

func formatQuestionID(id interface{}) string {
	switch v := id.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%d", int(v))
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// serverQuestionToLocal 将服务器题库题目转换为本地题目结构体。
//
// 功能说明：
// 将服务器题库的题目数据（network.ServerQuestion）转换为本地题目结构体（core.Question），以便在UI中统一处理。
//
// 输入参数：
//   - q: *network.ServerQuestion，服务器题库题目对象。
//
// 返回值：
//   - *core.Question：转换后的本地题目对象。
func serverQuestionToLocal(q *network.ServerQuestion) *core.Question {
	var opts []core.Option
	for label, text := range q.Options {
		opts = append(opts, core.Option{Label: label, Text: text})
	}
	var answers []string
	for _, ch := range q.Answer {
		answers = append(answers, string(ch))
	}

	return &core.Question{
		ID:         formatQuestionID(q.ID),
		Type:       q.Type,
		Content:    q.Content,
		Options:    opts,
		Answers:    answers,
		Difficulty: q.Difficulty,
	}
}

// convertBankDataToQuestions 将本地题库数据转换为本地题目结构体列表。
//
// 功能说明：
// 将本地题库数据（parser.BankData）中的题目信息转换为本地题目结构体（core.Question）列表，以便在UI中统一处理。
//
// 输入参数：
//   - bankData: *parser.BankData，本地题库数据对象。
//
// 返回值：
//   - []core.Question：转换后的本地题目列表。
func convertBankDataToQuestions(bankData *parser.BankData) []core.Question {
	var questions []core.Question
	for _, q := range bankData.Questions {
		var opts []core.Option
		for label, text := range q.Options {
			opts = append(opts, core.Option{Label: label, Text: text})
		}

		var answers []string
		if q.Answer != "" {
			cleanAns := strings.NewReplacer("、", "", ",", "", " ", "").Replace(q.Answer)
			for _, ch := range cleanAns {
				norm := core.NormalizeAnswerLabel(string(ch))
				if norm != "" {
					answers = append(answers, norm)
				}
			}
			if len(answers) == 0 {
				answers = append(answers, core.NormalizeAnswerLabel(q.Answer))
			}
		}

		questions = append(questions, core.Question{
			ID:         fmt.Sprintf("%d", q.ID),
			Type:       q.Type,
			Content:    q.Content,
			Options:    opts,
			Answers:    answers,
			Difficulty: q.Difficulty,
		})
	}
	return questions
}

// randomFillQuestions 随机填满各题型缺少的题目
//
// 功能说明：
// 根据考试参数中配置的每种题型所需数量（ws.typeCounts），从未被选中的可用题目中随机挑选并标记选中，直到达到所需数量。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func randomFillQuestions(ws *AdminExamCreateState) {
	if ws == nil || len(ws.typeCounts) == 0 {
		return
	}

	var allQuestions []core.Question
	if ws.sourceTab == 0 {
		allQuestions = ws.localQuestions
	} else {
		for _, sq := range ws.serverQuestions {
			allQuestions = append(allQuestions, *serverQuestionToLocal(&sq))
		}
	}

	typeMap := make(map[string][]core.Question)
	for _, q := range allQuestions {
		typeMap[q.Type] = append(typeMap[q.Type], q)
	}

	rand.Seed(time.Now().UnixNano())

	for qType, required := range ws.typeCounts {
		if required <= 0 {
			continue
		}
		currentCount := 0
		var unselected []core.Question
		for _, q := range typeMap[qType] {
			if ws.selectedQIDs[q.ID] {
				currentCount++
			} else {
				unselected = append(unselected, q)
			}
		}

		needed := required - currentCount
		if needed <= 0 {
			continue
		}

		if len(unselected) <= needed {
			for _, q := range unselected {
				ws.selectedQIDs[q.ID] = true
			}
		} else {
			rand.Shuffle(len(unselected), func(i, j int) {
				unselected[i], unselected[j] = unselected[j], unselected[i]
			})
			for i := 0; i < needed; i++ {
				ws.selectedQIDs[unselected[i].ID] = true
			}
		}
	}
}

// buildTemplateListPage 构建考试模板列表页面，显示所有考试模板并支持取消考试操作。
//
// 功能说明：
// 构建考试创建向导的第一步页面，显示所有考试模板的列表，并提供刷新、编辑、删除、取消考试和导出结果等功能。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：考试模板列表页面的UI容器对象。
func buildTemplateListPage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	// Template list
	templateListVBox := container.NewVBox()
	var refreshTemplateList func()

	refreshTemplateList = func() {
		if !state.IsLoggedIn {
			return
		}

		state.FetchTemplates(100, 0, func(success bool, templates []network.TemplateItem, total int, msg string) {
			if !success {
				return
			}

			if templateListVBox == nil {
				templateListVBox = container.NewVBox()
			}

			if len(templates) == 0 {
				noDataText := customElements.CreateLabel("暂无考试模板", core.HexColor(core.TextMutedColor), core.FontSizeSubtitle, false, true, false)
				templateListVBox.Objects = []fyne.CanvasObject{container.NewPadded(noDataText)}
				templateListVBox.Refresh()
				return
			}

			// Render cards directly from templates since the list API now includes question_type_distribution
			var objects []fyne.CanvasObject
			for _, tmpl := range templates {
				tmplCard := buildTemplateCardForPushExam(&tmpl, state, func() {
					showDeleteTemplateConfirmForPushExam(w, state, &tmpl, func() {
						refreshTemplateList()
					})
				})
				objects = append(objects, tmplCard, container.NewGridWrap(fyne.NewSize(1, 8)))
			}

			// Safely update the container objects on completion
			templateListVBox.Objects = objects
			fyne.Do(func() { templateListVBox.Refresh() })
		})
	}

	// 保存刷新函数到状态中，供点击时调用
	ws.templateRefreshFunc = refreshTemplateList

	scrollContent := container.NewVScroll(templateListVBox)
	scrollContent.SetMinSize(fyne.NewSize(0, 400))

	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	mainLayout := container.NewBorder(nil, nil, nil, nil, scrollContent)
	wRootLayout := container.NewStack(pageBg, mainLayout)

	return wRootLayout
}
