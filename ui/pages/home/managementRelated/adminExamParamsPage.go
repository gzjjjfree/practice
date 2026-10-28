package managementRelated

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
)


// buildExamParamsPage 构建考试参数设置页面，包含考试名称、时长、时间及各题型数量输入框。
//
// 功能说明：
// 构建考试创建向导的第二步页面，允许用户设置考试名称、考试时长、开始/结束时间，以及各题型的数量。
// 页面包含模板选择按钮和确认考试参数按钮，确认后生成统计信息并更新页面。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：考试参数设置页面的UI容器对象。
func buildExamParamsPage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	if len(ws.localBankKeys) == 0 && len(ws.serverBankIDs) == 0 {
		noSourceData := canvas.NewText(core.AdminSourceNotSetMsg, core.HexColor(core.TextMutedColor))
		noSourceData.Alignment = fyne.TextAlignCenter
		noSourceData.TextSize = 30
		return container.NewPadded(container.NewCenter(
			noSourceData,
		))
	}

	makeFormLabel := func(text string) *widget.Label {
		lbl := widget.NewLabel(text)
		lbl.TextStyle = fyne.TextStyle{Bold: true}
		return lbl
	}

	ws.nameEntry = widget.NewEntry()
	ws.nameEntry.SetPlaceHolder(core.AdminExamParamsNamePlaceholder)
	if ws.examName != "" {
		ws.nameEntry.SetText(ws.examName)
	}
	ws.nameEntry.OnChanged = func(s string) {
		ws.examName = s
	}
	nameRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminExamParamsNameLabel), nil, ws.nameEntry)

	ws.durationEntry = widget.NewEntry()
	ws.durationEntry.SetPlaceHolder(core.AdminExamDurationPlaceholder)
	if ws.durationMin > 0 {
		ws.durationEntry.SetText(fmt.Sprintf("%d", ws.durationMin))
	}
	ws.durationEntry.OnChanged = func(s string) {
		val, _ := strconv.Atoi(s)
		ws.durationMin = val
	}
	durationRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminExamDurationLabel), nil, ws.durationEntry)

	startTimeDefault := time.Now().Format("2006-01-02 15:04")
	if ws.startTimeStr != "" {
		startTimeDefault = ws.startTimeStr
	}
	endTimeDefault := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04")
	if ws.endTimeStr != "" {
		endTimeDefault = ws.endTimeStr
	}

	ws.startTimeLbl = widget.NewLabel(startTimeDefault)
	ws.endTimeLbl = widget.NewLabel(endTimeDefault)
	ws.startTimeStr = startTimeDefault
	ws.endTimeStr = endTimeDefault

	// 设置时间按钮（标识符：setTimeBtn返回的自定义按钮对象，名称："修改"）：点击后弹出时间选择对话框。
	setTimeBtn := func(lbl *widget.Label, defaultOffset time.Duration) *customElements.CustomButton {
		return customElements.CreateButton(
			core.AdminTimeEditBtnText,
			core.BtnSmallWidth, core.BtnSmallHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1, 12, true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				// 获取当前标签显示的时间作为默认值
				defaultTime := time.Now().Add(defaultOffset)
				if lbl != nil && lbl.Text != "" {
					t, err := time.Parse("2006-01-02 15:04", lbl.Text)
					if err == nil {
						defaultTime = t
					}
				}
				customElements.ShowTimePickerDialog(w, defaultTime, func(result customElements.TimePickerResult) {
					formatted := customElements.TimeResultToString(result)
					lbl.SetText(formatted)
					if lbl == ws.startTimeLbl {
						ws.startTimeStr = formatted
					} else if lbl == ws.endTimeLbl {
						ws.endTimeStr = formatted
					}
				})
			},
		)
	}

	startRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminStartTimeLabel), nil, container.NewHBox(ws.startTimeLbl, setTimeBtn(ws.startTimeLbl, 0)))
	endRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminEndTimeLabel), nil, container.NewHBox(ws.endTimeLbl, setTimeBtn(ws.endTimeLbl, 2*time.Hour)))

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

	typeLabels := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	typeEntries := make(map[string]*widget.Entry)
	typeScoreEntries := make(map[string]*widget.Entry)
	var typeCards []fyne.CanvasObject

	for _, t := range typeLabels {
		tName := t
		maxCount := bankCounts[tName]
		label := widget.NewLabel(fmt.Sprintf("%s (共%d题)", tName, maxCount))
		label.TextStyle = fyne.TextStyle{Bold: true}

		countEntry := widget.NewEntry()
		countEntry.SetPlaceHolder(core.AdminTypeEntryPlaceholder)
		if ws.typeCounts != nil {
			if count, ok := ws.typeCounts[tName]; ok && count > 0 {
				countEntry.SetText(fmt.Sprintf("%d", count))
			}
		}
		countEntry.OnChanged = func(s string) {
			val, _ := strconv.Atoi(s)
			if ws.typeCounts == nil {
				ws.typeCounts = make(map[string]int)
			}
			ws.typeCounts[tName] = val
		}
		typeEntries[tName] = countEntry

		scoreEntry := widget.NewEntry()
		scoreEntry.SetPlaceHolder("分值")
		defaultScore := 1.0
		if ws.typeScores != nil {
			if sc, ok := ws.typeScores[tName]; ok {
				defaultScore = sc
			}
		}
		scoreEntry.SetText(fmt.Sprintf("%g", defaultScore))
		scoreEntry.OnChanged = func(s string) {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				val = 1.0
			}
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			ws.typeScores[tName] = val
		}
		typeScoreEntries[tName] = scoreEntry

		compactCount := container.NewGridWrap(fyne.NewSize(55, core.TypeEntryBoxSizeH), countEntry)
		compactScore := container.NewGridWrap(fyne.NewSize(55, core.TypeEntryBoxSizeH), scoreEntry)

		itemBox := container.NewHBox(
			label,
			layout.NewSpacer(),
			widget.NewLabel("题数:"),
			compactCount,
			widget.NewLabel("分值:"),
			compactScore,
		)

		typeCard := container.NewGridWrap(fyne.NewSize(390, core.TypeCardGridWrapHeight), itemBox)
		typeCards = append(typeCards, typeCard)
	}

	ws.typeEntries = typeEntries
	ws.typeScoreEntries = typeScoreEntries

	typeCountsSection := container.NewVBox(
		makeFormLabel(core.AdminTypeSettingsTitle),
		container.NewVBox(typeCards...),
	)

	templateBtn := buildTemplateSelectBtn(ws, state, w)

	var mainContainer *fyne.Container

	// 确认考试参数按钮（标识符：confirmParamsBtn，名称："确认考试参数"）：验证并确认考试名称、时长及各题型数量设置，生成统计信息并更新页面。
	var confirmParamsBtn *customElements.CustomButton
	confirmParamsBtn = customElements.CreateButton(
		core.AdminConfirmParamsBtnText,
		core.ExamParamsConfirmBtnWidth, core.ExamParamsConfirmBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.ConfirmParamsBtnStrokeWidth,
		core.ConfirmParamsBtnFontSize,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if ws.nameEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminNameRequiredErr, w)
				return
			}
			if ws.durationEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminDurationInvalidErr, w)
				return
			}
			_, err := fmt.Sscanf(ws.durationEntry.Text, "%d", &ws.durationMin)
			if err != nil || ws.durationMin <= 0 {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminDurationInvalidErr, w)
				return
			}

			startTime, err1 := time.Parse("2006-01-02 15:04", ws.startTimeLbl.Text)
			endTime, err2 := time.Parse("2006-01-02 15:04", ws.endTimeLbl.Text)
			if err1 != nil || err2 != nil || endTime.Sub(startTime) < 30*time.Minute {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminEndTimeInvalidErr, w)
				return
			}

			ws.examName = ws.nameEntry.Text
			ws.startTimeStr = ws.startTimeLbl.Text
			ws.endTimeStr = ws.endTimeLbl.Text

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

			totalQuestions := 0
			totalScore := 0.0
			ws.typeCounts = make(map[string]int)
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			for _, t := range typeLabels {
				val := typeEntries[t].Text
				count := 0
				if val != "" {
					count, _ = strconv.Atoi(val)
				}

				maxAvailable := bankCounts[t]
				if count > maxAvailable {
					customElements.ShowCustomInformation(core.AdminErrorText, fmt.Sprintf("【%s】设置的数量（%d）超过了题库来源中的总数量（%d）", t, count, maxAvailable), w)
					return
				}

				ws.typeCounts[t] = count

				scoreVal := typeScoreEntries[t].Text
				sc := 1.0
				if scoreVal != "" {
					if parsed, err := strconv.ParseFloat(scoreVal, 64); err == nil && parsed >= 0 {
						sc = parsed
					}
				}
				ws.typeScores[t] = sc

				totalQuestions += count
				totalScore += float64(count) * sc
			}

			if totalQuestions == 0 {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminTypeCountRequiredErr, w)
				return
			}

			ws.examParamsConfirmed = true

			if ws.wizard != nil && len(ws.wizard.Items) > 2 {
				qSelectPage := buildQuestionSelectPage(ws)
				ws.wizard.Items[2] = container.NewTabItem(ws.wizard.Items[2].Text, qSelectPage)
				ws.wizard.Refresh()
			}
			if ws.wizard != nil && len(ws.wizard.Items) > 3 {
				userPage := buildUserSelectPage(ws, state)
				ws.wizard.Items[3] = container.NewTabItem(ws.wizard.Items[3].Text, userPage)
				ws.wizard.Refresh()
			}

			typeDetails := ""
			for _, t := range typeLabels {
				if ws.typeCounts[t] > 0 {
					typeDetails += fmt.Sprintf("%s: %d题(%.1f分/题)  |  ", t, ws.typeCounts[t], ws.typeScores[t])
				}
			}
			typeDetails = strings.TrimSuffix(typeDetails, "  |  ")

			typeDetailsLbl := widget.NewLabel(fmt.Sprintf(core.AdminTypeDistributionLabel, typeDetails))
			typeDetailsLbl.Wrapping = fyne.TextWrapWord

			summaryContent := container.NewVBox(
				canvas.NewText(core.AdminSuccessSetConfirmed, core.HexColor(core.ColorSelectedBorder)),
				container.NewGridWrap(fyne.NewSize(1, 4)),
				canvas.NewText(fmt.Sprintf("📋 考试名称：%s", ws.nameEntry.Text), core.HexColor(core.TextPrimaryColor)),
				canvas.NewText(fmt.Sprintf("⏱️ 考试时长：%d 分钟", ws.durationMin), core.HexColor(core.TextBodyColor)),
				canvas.NewText(fmt.Sprintf("🕒 考试时间：%s 至 %s", ws.startTimeLbl.Text, ws.endTimeLbl.Text), core.HexColor(core.TextBodyColor)),
				typeDetailsLbl,
				canvas.NewText(fmt.Sprintf("🎯 试卷总题数：%d 题，总分：%.1f 分", totalQuestions, totalScore), core.HexColor(core.ColorSelectedBorder)),
			)

			summaryCard := container.NewPadded(summaryContent)

			bottomBtnRow := container.NewHBox(
				layout.NewSpacer(),
				templateBtn,
				container.NewGridWrap(fyne.NewSize(20, 1)),
				confirmParamsBtn,
				layout.NewSpacer(),
			)

			mainContainer.Objects = []fyne.CanvasObject{
				nameRow,
				durationRow,
				startRow,
				endRow,
				typeCountsSection,
				summaryCard,
				bottomBtnRow,
			}
			mainContainer.Refresh()
		})

	bottomBtnRow := container.NewHBox(
		layout.NewSpacer(),
		templateBtn,
		container.NewGridWrap(fyne.NewSize(20, 1)),
		confirmParamsBtn,
		layout.NewSpacer(),
	)

	var initialObjects []fyne.CanvasObject
	initialObjects = append(initialObjects, nameRow, durationRow, startRow, endRow, typeCountsSection)

	if ws.examParamsConfirmed {
		totalQuestions := 0
		totalScore := 0.0
		typeDetails := ""
		if ws.typeCounts != nil {
			for _, t := range typeLabels {
				count := ws.typeCounts[t]
				sc := 1.0
				if ws.typeScores != nil {
					if s, ok := ws.typeScores[t]; ok {
						sc = s
					}
				}
				totalQuestions += count
				totalScore += float64(count) * sc
				if count > 0 {
					typeDetails += fmt.Sprintf("%s: %d题(%.1f分/题)  |  ", t, count, sc)
				}
			}
		}
		typeDetails = strings.TrimSuffix(typeDetails, "  |  ")
		typeDetailsLbl := widget.NewLabel(fmt.Sprintf(core.AdminTypeDistributionLabel, typeDetails))
		typeDetailsLbl.Wrapping = fyne.TextWrapWord

		summaryContent := container.NewVBox(
			canvas.NewText(core.AdminSuccessSetConfirmed, core.HexColor(core.ColorSelectedBorder)),
			container.NewGridWrap(fyne.NewSize(1, 4)),
			canvas.NewText(fmt.Sprintf("📋 考试名称：%s", ws.examName), core.HexColor(core.TextPrimaryColor)),
			canvas.NewText(fmt.Sprintf("⏱️ 考试时长：%d 分钟", ws.durationMin), core.HexColor(core.TextBodyColor)),
			canvas.NewText(fmt.Sprintf("🕒 考试时间：%s 至 %s", ws.startTimeStr, ws.endTimeStr), core.HexColor(core.TextBodyColor)),
			typeDetailsLbl,
			canvas.NewText(fmt.Sprintf("🎯 试卷总题数：%d 题，总分：%.1f 分", totalQuestions, totalScore), core.HexColor(core.ColorSelectedBorder)),
		)
		summaryCard := container.NewPadded(summaryContent)
		initialObjects = append(initialObjects, summaryCard)
	} else {
		initialObjects = append(initialObjects, container.NewGridWrap(fyne.NewSize(1, 8)))
	}

	initialObjects = append(initialObjects, bottomBtnRow)
	mainContainer = container.NewVBox(initialObjects...)

	ws.examParamsContent = mainContainer
	return container.NewPadded(mainContainer)
}
