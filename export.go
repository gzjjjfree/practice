package main

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
)

// ==================== 1. 选择导出格式的弹窗 ====================
func showExportDialog(w fyne.Window, state *AppState) {
	formatRadio := widget.NewRadioGroup([]string{"Excel (.xlsx)", "JSON (.json)"}, nil)
	formatRadio.SetSelected("Excel (.xlsx)") // 默认选中 Excel

	content := container.NewVBox(
		widget.NewLabel("请选择您要导出的文件格式："),
		formatRadio,
	)

	dialog.ShowCustomConfirm("导出题库", "下一步", "取消", container.NewPadded(content), func(confirm bool) {
		if !confirm {
			return
		}

		selectedFormat := formatRadio.Selected
		executeFileSave(w, state, selectedFormat)

	}, w)
}

// ==================== 2. 唤起系统保存文件对话框 ====================
func executeFileSave(w fyne.Window, state *AppState, format string) {
	fileSaveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("保存文件对话框出错: %v", err), w)
			return
		}
		if writer == nil {
			return // 用户点击了取消
		}
		defer writer.Close()

		// 根据选择的格式分发到对应的处理函数
		if format == "JSON (.json)" {
			exportToJSON(w, state, writer)
		} else {
			exportToExcel(w, state, writer)
		}
	}, w)

	// 根据当前题库名动态设置默认的保存文件名和后缀过滤器
	baseName := state.CurrentFileName
	if baseName == "" {
		baseName = "未命名题库"
	}
	baseName = strings.TrimSuffix(baseName, ".txt") // 如果是 txt 导入的，去掉原有后缀

	if format == "JSON (.json)" {
		fileSaveDialog.SetFileName(baseName + "_导出.json")
		fileSaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	} else {
		fileSaveDialog.SetFileName(baseName + "_导出.xlsx")
		fileSaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx"}))
	}

	fileSaveDialog.Show()
}

// ==================== 导出为 JSON ====================
func exportToJSON(w fyne.Window, state *AppState, writer fyne.URIWriteCloser) {
	// 1. 组装最外层的 BankData
	bankData := BankData{
		DisplayName: state.CurrentFileName,
		StorageKey:  state.CurrentStorageKey,
		Questions:   make([]QuestionItem, len(state.Questions)),
	}

	// 2. 逆向转换：将 Question 转换回 QuestionItem
	for i, q := range state.Questions {
		// 转换 ID
		idInt, _ := strconv.Atoi(q.ID)

		// 转换选项 ([]Option 还原为 map)
		optMap := make(map[string]string) // 假设你的 OptionMap 底层是 map[string]string
		for _, opt := range q.Options {
			optMap[opt.Label] = opt.Text
		}

		// 转换答案
		// 多选题 ["A", "C"] -> "AC"
		// 填空题/问答题 ["25~30", "15~30"] -> "25~30、15~30"
		var ansStr string
		if q.Type == "填空题" || q.Type == "问答题" {
			ansStr = strings.Join(q.Answers, "、")
		} else {
			ansStr = strings.Join(q.Answers, "")
		}

		// 装载回 QuestionItem
		bankData.Questions[i] = QuestionItem{
			ID:         idInt,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     ansStr,
			Options:    optMap,
			RawIndex:   i + 1,
			Difficulty: q.Difficulty,
		}
	}

	// 3. 将组装好的 BankData 序列化为 JSON
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

	dialog.ShowInformation("导出成功", "JSON 格式题库已成功导出！", w)
}

// ==================== 4. 导出为 Excel (图2 横向扩展格式) ====================
func exportToExcel(w fyne.Window, state *AppState, writer fyne.URIWriteCloser) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "Sheet1"

	// ==================== 1. 设置第一行：合并的大标题 ====================
	// 获取题库名称并去除后缀
	title := state.CurrentFileName
	if title == "" {
		title = "导出题库"
	}
	title = strings.TrimSuffix(title, ".txt")
	title = strings.TrimSuffix(title, ".json")
	title = strings.TrimSuffix(title, ".xlsx")

	f.SetCellValue(sheetName, "A1", title)
	f.MergeCell(sheetName, "A1", "I1") // 默认合并到 I 列 (根据最多选项数可调)

	// 标题样式：居中、加粗、大字号
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "I1", titleStyle)
	f.SetRowHeight(sheetName, 1, 30) // 加高标题行

	// ==================== 2. 设置第二行：动态表头 ====================
	headers := []string{"题型", "题目内容", "难易度", "正确答案", "答案A", "答案B", "答案C", "答案D", "答案E", "答案F", "答案G", "答案H", "答案I"}

	// 表头与正文的通用边框样式
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

	// ==================== 3. 遍历题库填充正文数据 ====================
	// 正文样式：带边框，支持换行
	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border:    borderStyle,
	})

	for i, q := range state.Questions {
		row := i + 3 // 第一行标题，第二行表头，数据从第三行开始

		// 格式化答案：选择题直接拼接 (如 "ACD")，填空题用顿号分隔 (如 "25、30")
		var answersText string
		if q.Type == "填空题" || q.Type == "问答题" {
			answersText = strings.Join(q.Answers, "、")
		} else {
			answersText = strings.Join(q.Answers, "")
		}

		// 填充基础列
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), q.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), q.Content)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), q.Difficulty)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), answersText)

		// 填充选项列 (横向打散)
		// 逻辑：A 对应第 5 列 (E), B 对应第 6 列 (F)...
		for _, opt := range q.Options {
			if opt.Text == "" {
				continue
			}
			if len(opt.Label) > 0 {
				// 将字符 'A' 转换为列偏移量 0, 'B' 为 1
				colOffset := int(opt.Label[0] - 'A')
				if colOffset >= 0 && colOffset <= 8 { // 确保不超过 "答案I" (总计9个选项)
					cell, _ := excelize.CoordinatesToCellName(5+colOffset, row)
					// 注意：图2中的选项只存文字，去掉了 "A. " 的前缀
					f.SetCellValue(sheetName, cell, opt.Text)
				}
			}
		}

		// 为当前行的所有已用单元格应用边框样式
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("I%d", row), bodyStyle)
	}

	// ==================== 4. 页面视觉优化 (列宽) ====================
	f.SetColWidth(sheetName, "A", "A", 10) // 题型
	f.SetColWidth(sheetName, "B", "B", 50) // 题干较长
	f.SetColWidth(sheetName, "C", "C", 8)  // 难度
	f.SetColWidth(sheetName, "D", "D", 10) // 正确答案
	f.SetColWidth(sheetName, "E", "I", 15) // 各个选项列

	// ==================== 5. 物理写入文件 ====================
	if err := f.Write(writer); err != nil {
		dialog.ShowError(fmt.Errorf("生成 Excel 文件失败: %v", err), w)
		return
	}

	dialog.ShowInformation("导出成功", "Excel 格式题库已成功导出！", w)
}
