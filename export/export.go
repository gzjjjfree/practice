package export

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/parser"
)

// ShowExportDialog shows the export format selection dialog.
func ShowExportDialog(w fyne.Window, state *core.AppState) {
	formatRadio := widget.NewRadioGroup([]string{"Excel (.xlsx)", "JSON (.json)"}, nil)
	formatRadio.SetSelected("Excel (.xlsx)")

	content := container.NewVBox(
		widget.NewLabel("请选择您要导出的文件格式："),
		formatRadio,
	)

	dialog.ShowCustomConfirm("导出题库", "下一步", "取消", container.NewPadded(content), func(confirm bool) {
		if !confirm {
			return
		}

		selectedFormat := formatRadio.Selected
		ExecuteFileSave(w, state, selectedFormat)

	}, w)
}

// ExecuteFileSave invokes the system save dialog and triggers export.
func ExecuteFileSave(w fyne.Window, state *core.AppState, format string) {
	fileSaveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("保存文件对话框出错: %v", err), w)
			return
		}
		if writer == nil {
			return // user cancelled
		}
		defer writer.Close()

		if format == "JSON (.json)" {
			ExportToJSON(w, state, writer)
		} else {
			ExportToExcel(w, state, writer)
		}
	}, w)

	baseName := state.CurrentFileName
	if baseName == "" {
		baseName = "未命名题库"
	}
	baseName = strings.TrimSuffix(baseName, ".txt")

	if format == "JSON (.json)" {
		fileSaveDialog.SetFileName(baseName + "_导出.json")
		fileSaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	} else {
		fileSaveDialog.SetFileName(baseName + "_导出.xlsx")
		fileSaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx"}))
	}

	fileSaveDialog.Show()
}

// ==================== Export to JSON ====================
func ExportToJSON(w fyne.Window, state *core.AppState, writer fyne.URIWriteCloser) {
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

	bytesData, err := json.MarshalIndent(bankData, "", "  ")
	if err != nil {
		dialog.ShowError(fmt.Errorf("JSON 序列化失败: %v", err), w)
		return
	}

	_, err = writer.Write(bytesData)
	if err != nil {
		dialog.ShowError(fmt.Errorf("写入 JSON 文件失败: %v", err), w)
		return
	}

	core.ShowCustomInformation("导出成功", "JSON 格式题库已成功导出！", w)
}

// ==================== Export to Excel ====================
func ExportToExcel(w fyne.Window, state *core.AppState, writer fyne.URIWriteCloser) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "Sheet1"

	title := state.CurrentFileName
	if title == "" {
		title = "导出题库"
	}
	title = strings.TrimSuffix(title, ".txt")
	title = strings.TrimSuffix(title, ".json")
	title = strings.TrimSuffix(title, ".xlsx")

	f.SetCellValue(sheetName, "A1", title)
	f.MergeCell(sheetName, "A1", "I1")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "I1", titleStyle)
	f.SetRowHeight(sheetName, 1, 30)

	headers := []string{"题型", "题目内容", "难易度", "正确答案", "答案A", "答案B", "答案C", "答案D", "答案E", "答案F", "答案G", "答案H", "答案I"}

	borderStyle := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borderStyle,
	})

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    borderStyle,
	})

	for i, q := range state.Questions {
		row := i + 3

		var answersText string
		if q.Type == "填空题" || q.Type == "问答题" {
			answersText = strings.Join(q.Answers, "、")
		} else {
			answersText = strings.Join(q.Answers, "")
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), q.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), q.Content)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), q.Difficulty)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), answersText)

		for _, opt := range q.Options {
			if opt.Text == "" {
				continue
			}
			if len(opt.Label) > 0 {
				colOffset := int(opt.Label[0] - 'A')
				if colOffset >= 0 && colOffset <= 8 {
					cell, _ := excelize.CoordinatesToCellName(5+colOffset, row)
					f.SetCellValue(sheetName, cell, opt.Text)
				}
			}
		}

		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("I%d", row), bodyStyle)
	}

	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "B", 50)
	f.SetColWidth(sheetName, "C", "C", 8)
	f.SetColWidth(sheetName, "D", "D", 10)
	f.SetColWidth(sheetName, "E", "I", 15)

	if err := f.Write(writer); err != nil {
		dialog.ShowError(fmt.Errorf("生成 Excel 文件失败: %v", err), w)
		return
	}

	core.ShowCustomInformation("导出成功", "Excel 格式题库已成功导出！", w)
}
