package pages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/gz_storage"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/ui/widgets"
)

// SyncStateToDisk converts in-memory Question objects back to BankData, serializes to JSON, and writes to local storage.
func SyncStateToDisk(state *core.AppState) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("syncStateToDisk panic: %v", r)
			fmt.Printf("[ERROR] sync state to disk crashed: %v\n", err)
		}
	}()

	bankData := &parser.BankData{
		DisplayName: state.CurrentFileName,
		StorageKey:  state.CurrentStorageKey,
		Questions:   make([]parser.QuestionItem, len(state.Questions)),
	}

	for i, q := range state.Questions {
		idInt, _ := strconv.Atoi(q.ID)
		optMap := make(map[string]string)
		for _, opt := range q.Options {
			optMap[opt.Label] = opt.Text
		}

		var ansStr string
		if q.Type == "填空题" || q.Type == "问答题" {
			ansStr = strings.Join(q.Answers, "、")
		} else {
			ansStr = strings.Join(q.Answers, "")
		}

		bankData.Questions[i] = parser.QuestionItem{
			ID:         idInt,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     ansStr,
			Options:    optMap,
			RawIndex:   i + 1,
			Difficulty: q.Difficulty,
		}
	}

	jsonData, err := json.Marshal(bankData)
	if err != nil {
		return err
	}

	storageDir := gz_storage.GetStorageDir()
	absoluteSavePath := filepath.Join(storageDir, state.CurrentStorageKey+".json")
	return os.WriteFile(absoluteSavePath, jsonData, 0644)
}

// CloneQuestionForEdit deep-clones a question to avoid mutating global state during editing.
func CloneQuestionForEdit(q core.Question) core.Question {
	newQ := core.Question{
		ID:         q.ID,
		Type:       q.Type,
		Content:    q.Content,
		Difficulty: q.Difficulty,
	}

	// Deep copy Options slice
	if q.Options != nil {
		newQ.Options = make([]core.Option, len(q.Options))
		for i, opt := range q.Options {
			newQ.Options[i] = core.Option{
				Label: opt.Label,
				Text:  opt.Text,
			}
		}
	}

	// Deep copy Answers slice
	if q.Answers != nil {
		newQ.Answers = make([]string, len(q.Answers))
		copy(newQ.Answers, q.Answers)
	}

	return newQ
}

// RenderCorrectionCard builds the full edit UI for a single question.
func RenderCorrectionCard(w fyne.Window, state *core.AppState, q core.Question, onRefresh func()) fyne.CanvasObject {
	editQ := CloneQuestionForEdit(q)

	btnAddOption := widgets.GetBox("增加选项", 90, 35, &widgets.BoxColor{TextColor: core.TextPrimaryColor, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, nil)
	btnAddQuestion := widgets.GetBox("新增题目", 90, 35, &widgets.BoxColor{TextColor: core.TextPrimaryColor, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, func() {
		core.ShowCustomInformation("提示", "进入新增题目流", w)
	})
	btnSave := widgets.GetBox("保存本题", 90, 35, &widgets.BoxColor{TextColor: core.BtnPrimaryBg, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, nil)
	btnDelete := widgets.GetBox("删除本题", 90, 35, &widgets.BoxColor{TextColor: core.ColorWrongBorder, BgColor: core.SectionBgColor, StrokeColor: core.SectionBgColor, TextSize: 18}, true, false, nil)

	btnRow1 := container.NewGridWithColumns(2, btnAddOption, btnAddQuestion)
	btnRow2 := container.NewGridWithColumns(2, btnSave, btnDelete)
	buttonPanel := container.NewVBox(btnRow1, btnRow2)

	contentEntry := widget.NewMultiLineEntry()
	contentEntry.SetText(editQ.Content)
	contentEntry.Wrapping = fyne.TextWrapBreak

	optionsContainer := container.NewVBox()

	if q.Type == "填空题" || q.Type == "问答题" {
		btnAddOption.SetDisabled(true)

		ansEntry := widget.NewMultiLineEntry()
		ansEntry.SetText(strings.Join(editQ.Answers, "、"))

		optionsContainer.Add(widget.NewLabel("答案 (多个填空用、隔开):"))
		optionsContainer.Add(ansEntry)

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			editQ.Answers = strings.Split(strings.TrimSpace(ansEntry.Text), "、")
			ExecuteSave(w, state, editQ, onRefresh)
		}

	} else if q.Type == "判断题" {
		btnAddOption.SetDisabled(true)

		radio := widget.NewRadioGroup([]string{"A. 正确", "B. 错误"}, func(val string) {
			if strings.HasPrefix(val, "A") {
				editQ.Answers = []string{"A"}
			} else {
				editQ.Answers = []string{"B"}
			}
		})

		if len(editQ.Answers) > 0 {
			if editQ.Answers[0] == "A" {
				radio.SetSelected("A. 正确")
			} else if editQ.Answers[0] == "B" {
				radio.SetSelected("B. 错误")
			}
		}

		optionsContainer.Add(widget.NewLabel("设定正确答案:"))
		optionsContainer.Add(radio)

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			ExecuteSave(w, state, editQ, onRefresh)
		}

	} else {
		btnAddOption.SetDisabled(false)

		var refreshOptionsUI func()

		refreshOptionsUI = func() {
			optionsContainer.Objects = nil

			for i := range editQ.Options {
				idx := i
				currOpt := editQ.Options[idx]

				optEntry := widget.NewMultiLineEntry()
				optEntry.SetText(currOpt.Text)
				optEntry.OnChanged = func(s string) {
					editQ.Options[idx].Text = s
				}

				isAns := false
				for _, a := range editQ.Answers {
					if a == currOpt.Label {
						isAns = true
						break
					}
				}

				btnText := " " + currOpt.Label + " "
				if isAns {
					btnText = "[" + currOpt.Label + "]"
				}

				lblBtn := widget.NewButton(btnText, func() {
					if q.Type == "单选题" {
						editQ.Answers = []string{currOpt.Label}
					} else {
						found := false
						var newAns []string
						for _, a := range editQ.Answers {
							if a == currOpt.Label {
								found = true
							} else {
								newAns = append(newAns, a)
							}
						}
						if !found {
							newAns = append(newAns, currOpt.Label)
						}
						editQ.Answers = newAns
					}
					refreshOptionsUI()
				})

				if isAns {
					lblBtn.Importance = widget.HighImportance
				}

				row := container.NewBorder(nil, nil, lblBtn, nil, optEntry)
				optionsContainer.Add(container.NewPadded(row))
			}
			optionsContainer.Refresh()
		}

		refreshOptionsUI()

		btnAddOption.OnTapped = func() {
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
			if len(editQ.Options) < len(labels) {
				newLabel := labels[len(editQ.Options)]
				editQ.Options = append(editQ.Options, core.Option{Label: newLabel, Text: ""})
				refreshOptionsUI()
			} else {
				core.ShowCustomInformation("提示", "选项数量已达系统上限", w)
			}
		}

		btnSave.OnTapped = func() {
			editQ.Content = strings.TrimSpace(contentEntry.Text)
			labels := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

			var newValidOptions []core.Option
			oldToNewMap := make(map[string]string)
			newLabelIdx := 0
			hasEmptyCorrectAnswer := false

			for _, opt := range editQ.Options {
				isEmpty := strings.TrimSpace(opt.Text) == ""

				for _, a := range editQ.Answers {
					if a == opt.Label && isEmpty {
						hasEmptyCorrectAnswer = true
						break
					}
				}

				if !isEmpty && newLabelIdx < len(labels) {
					newLabel := labels[newLabelIdx]
					newValidOptions = append(newValidOptions, core.Option{
						Label: newLabel,
						Text:  strings.TrimSpace(opt.Text),
					})
					oldToNewMap[opt.Label] = newLabel
					newLabelIdx++
				}
			}

			if len(newValidOptions) == 0 {
				core.ShowCustomInformation("无法保存", "单选/多选题至少需要保留一个有效的选项内容", w)
				return
			}
			editQ.Options = newValidOptions

			var updatedAnswers []string
			for _, oldAns := range editQ.Answers {
				if newL, exists := oldToNewMap[oldAns]; exists {
					updatedAnswers = append(updatedAnswers, newL)
				}
			}

			if q.Type == "单选题" {
				if hasEmptyCorrectAnswer || len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			} else if q.Type == "多选题" {
				if len(updatedAnswers) == 0 {
					editQ.Answers = q.Answers
				} else {
					editQ.Answers = updatedAnswers
				}
			}

			validFinalLabels := make(map[string]bool)
			for _, vo := range editQ.Options {
				validFinalLabels[vo.Label] = true
			}

			var absoluteFinalAns []string
			for _, a := range editQ.Answers {
				if validFinalLabels[a] {
					absoluteFinalAns = append(absoluteFinalAns, a)
				}
			}

			if len(absoluteFinalAns) == 0 {
				absoluteFinalAns = []string{"A"}
			}
			editQ.Answers = absoluteFinalAns

			ExecuteSave(w, state, editQ, onRefresh)
		}
	}

	btnDelete.OnTapped = func() {
		dialog.ShowConfirm("警告", "确定要永久删除这道题目吗？", func(confirm bool) {
			if confirm {
				ExecuteDelete(w, state, q.ID, onRefresh)
			}
		}, w)
	}

	mainContent := container.NewVBox(
		widget.NewLabelWithStyle("题干内容:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		contentEntry,
		widget.NewSeparator(),
		optionsContainer,
		widget.NewSeparator(),
		buttonPanel,
	)

	return mainContent
}

// ExecuteSave updates the question in both state.Questions and state.CurrentList, calls SyncStateToDisk.
func ExecuteSave(w fyne.Window, state *core.AppState, editQ core.Question, onRefresh func()) {
	for i, origQ := range state.Questions {
		if origQ.ID == editQ.ID {
			state.Questions[i] = editQ
			break
		}
	}

	state.CurrentList[state.Index] = editQ

	if err := SyncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("保存失败: %v", err), w)
		return
	}
	core.ShowCustomInformation("成功", "本题修改已生效并保存至题库", w)
	onRefresh()
}

// ExecuteDelete removes a question from state.Questions and state.CurrentList.
func ExecuteDelete(w fyne.Window, state *core.AppState, targetID string, onRefresh func()) {
	var newQuestions []core.Question
	for _, q := range state.Questions {
		if q.ID != targetID {
			newQuestions = append(newQuestions, q)
		}
	}
	state.Questions = newQuestions

	var newList []core.Question
	for _, q := range state.CurrentList {
		if q.ID != targetID {
			newList = append(newList, q)
		}
	}
	state.CurrentList = newList

	if state.Index >= len(state.CurrentList) {
		state.Index = len(state.CurrentList) - 1
		if state.Index < 0 {
			state.Index = 0
		}
	}

	if err := SyncStateToDisk(state); err != nil {
		dialog.ShowError(fmt.Errorf("删除失败: %v", err), w)
		return
	}
	core.ShowCustomInformation("成功", "本题已从题库中永久删除", w)
	onRefresh()
}
