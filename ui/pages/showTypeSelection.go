package pages

import (
	"fmt"
	"math/rand"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/ui/widgets"
)

// ShowTypeSelection renders the intermediate page for selecting question type before starting a practice mode.
func ShowTypeSelection(w fyne.Window, state *core.AppState, practiceMode string, backToHome func()) {
	backBtn := widgets.GetBox("← 返回", 90, 35, &widgets.BoxColor{TextColor: core.TextPrimaryColor, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, func() {
		if backToHome != nil {
			backToHome()
		}
	})

	modeText := canvas.NewText(practiceMode, core.HexColor(core.TextSecondaryColor))
	modeText.TextSize = 18
	modeText.TextStyle = fyne.TextStyle{Bold: true}
	modeText.Alignment = fyne.TextAlignTrailing
	rightBox := container.NewPadded(modeText)
	topBar := container.NewBorder(nil, nil, backBtn, rightBox, widget.NewLabel(""))

	bankNameText := widget.NewLabel("📖 " + state.CurrentFileName)
	bankNameText.Alignment = fyne.TextAlignCenter
	bankNameText.Wrapping = fyne.TextWrapBreak
	bankNameText.SizeName = theme.SizeNameWindowButtonHeight

	bankNameBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	bankNameBg.CornerRadius = 6
	bankNameBg.StrokeColor = core.HexColor(core.BorderMediumColor)
	bankNameBg.StrokeWidth = 1
	bankNameCard := container.NewStack(bankNameBg, container.NewPadded(bankNameText))

	typeSet := make(map[string]bool)
	for _, q := range state.Questions {
		if q.Type != "" {
			typeSet[q.Type] = true
		}
	}

	standardOrder := []string{"填空题", "单选题", "多选题", "判断题", "问答题"}
	availableTypes := []string{"全部题型"}

	for _, stdType := range standardOrder {
		if typeSet[stdType] {
			availableTypes = append(availableTypes, stdType)
		}
	}

	for t := range typeSet {
		isStandard := false
		for _, stdType := range standardOrder {
			if t == stdType {
				isStandard = true
				break
			}
		}
		if !isStandard {
			availableTypes = append(availableTypes, t)
		}
	}

	listVBox := container.NewVBox()

	for i, t := range availableTypes {
		typeStr := t

		rowBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		rowBg.CornerRadius = 4
		rowBg.StrokeColor = core.HexColor(core.BorderLightColor)
		rowBg.StrokeWidth = 1

		lblText := fmt.Sprintf("题目%d. (%s)", i+1, typeStr)
		rowLbl := canvas.NewText(lblText, core.HexColor(core.TextBodyColor))
		rowLbl.TextSize = 18

		rowContent := container.NewBorder(layout.NewSpacer(), layout.NewSpacer(), nil, nil, container.NewPadded(rowLbl))
		rowStack := container.NewStack(rowBg, rowContent)

		clickableRow := widgets.NewClickableBox(rowStack, func() {
			var tempList []core.Question
			tempList = make([]core.Question, 0)

			switch practiceMode {
			case "错题练习":
				for _, q := range state.Questions {
					if isWrong, exists := state.WrongSet[q.ID]; exists && isWrong {
						tempList = append(tempList, q)
					}
				}
			case "收藏练习":
				for _, q := range state.Questions {
					if isFav, exists := state.FavSet[q.ID]; exists && isFav {
						tempList = append(tempList, q)
					}
				}
			default:
				tempList = state.Questions
			}

			var filteredList []core.Question

			if typeStr == "全部题型" {
				filteredList = make([]core.Question, len(tempList))
				copy(filteredList, tempList)
			} else {
				for _, q := range tempList {
					if q.Type == typeStr {
						filteredList = append(filteredList, q)
					}
				}
			}

			if practiceMode == "随机练习" {
				rand.Shuffle(len(filteredList), func(i, j int) {
					filteredList[i], filteredList[j] = filteredList[j], filteredList[i]
				})
			}

			if state.Title != "" {
				state.ModeIndices[state.Title] = state.Index
			}

			state.CurrentList = filteredList
			state.Title = fmt.Sprintf("%s - %s", practiceMode, typeStr)

			if len(state.CurrentList) == 0 {
				core.ShowCustomInformation("提示", "该题型下没有题目", w)
				return
			}

			state.Index = state.ModeIndices[state.Title]

			if state.Index >= len(state.CurrentList) {
				state.Index = 0
				if len(state.CurrentList) > 0 {
					state.Index = len(state.CurrentList) - 1
				}
			}

			if state.ModeRecords == nil {
				state.ModeRecords = make(map[string]map[string]string)
			}
			if state.ModeRecords[state.Title] == nil {
				state.ModeRecords[state.Title] = make(map[string]string)
			}
			state.PracticeRecordsMu.Lock()
			state.PracticeRecords = state.ModeRecords[state.Title]
			state.PracticeRecordsMu.Unlock()

			ShowPractice(w, state, func(win fyne.Window, st *core.AppState) {
				ShowTypeSelection(win, st, practiceMode, backToHome)
			})
		})

		listVBox.Add(clickableRow)
	}

	pageBg := canvas.NewRectangle(core.HexColor(core.SectionBgColor))

	contentBox := container.NewVBox(
		container.NewPadded(bankNameCard),
		widget.NewSeparator(),
		container.NewPadded(listVBox),
	)

	mainLayoutBox := container.NewStack(
		pageBg,
		container.NewBorder(topBar, nil, nil, nil, contentBox),
	)

	mainLayout := container.NewBorder(widgets.GetTitle(), nil, nil, nil, mainLayoutBox)

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	wRootLayout := container.NewStack(wBackground, mainLayout)
	w.SetContent(wRootLayout)
}
