// 解析文件

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/shakinm/xlsReader/xls" // 旧版 .xls 专用库
	"github.com/xuri/excelize/v2"      // 新版 .xlsx 专用库
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type OptionMap map[string]string

type QuestionItem struct {
	ID         int       `json:"id"`
	Type       string    `json:"type"`
	Content    string    `json:"content"`
	Answer     string    `json:"answer"`
	Options    OptionMap `json:"options"`
	SheetName  string    `json:"sheetName,omitempty"`
	RawIndex   int       `json:"rawIndex"`
	Difficulty string    `json:"difficulty,omitempty"`
}

type BankData struct {
	DisplayName string         `json:"displayName"`
	StorageKey  string         `json:"storageKey"`
	Questions   []QuestionItem `json:"questions"`
	//QuestionTypes []string       `json:"questionTypes"`
}

// ==================== 数据清理区 ====================

func cleanXlsData(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
}

func cleanHeaderMatch(text string) string {
	t := cleanXlsData(text)
	t = strings.ReplaceAll(t, " ", "")
	t = strings.ReplaceAll(t, " ", "") // 去除全角空格
	t = strings.ReplaceAll(t, "\n", "")
	t = strings.ReplaceAll(t, "\r", "")
	return strings.ToLower(t)
}

// ==================== 核心分流引擎 ====================

func ParseBytesToBank(fileBytes []byte, fileName string, storageKey string) (bank *BankData, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("读取异常(崩溃拦截): %v", r)
			err = fmt.Errorf("读取异常(崩溃拦截): %v", r)
		}
	}()

	// 🔥【核心绝杀】：放弃依赖后缀名！直接读取二进制文件头（Magic Bytes）
	// ZIP 格式特征 (Excel 2007+ .xlsx)
	isXLSX := bytes.HasPrefix(fileBytes, []byte{0x50, 0x4B, 0x03, 0x04}) || strings.HasSuffix(strings.ToLower(fileName), ".xlsx")
	// 纯文本特征保底
	isTXT := strings.HasSuffix(strings.ToLower(fileName), ".txt") && len(fileBytes) > 0

	isJSON := strings.HasSuffix(strings.ToLower(fileName), ".json") && len(fileBytes) > 0

	// ---------------- 1. XLSX 解析链路 ----------------
	if isXLSX {
		fmt.Println("in isXLSX")
		f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
		if err != nil {
			fmt.Println(fmt.Errorf("读取异常(XLSX解码): %w", err))
			return nil, fmt.Errorf("读取异常(XLSX解码): %w", err)
		}
		defer f.Close()
		return parseExcelFromOpenedFile(f, storageKey)
	}

	// ---------------- 3. TXT 解析链路 ----------------
	if isTXT {
		fmt.Println("in isTXT")
		rawData, err := readTxtFileWithEncodingFix(fileBytes)
		if err != nil {
			return nil, fmt.Errorf("TXT 文件读取失败！")
		}
		return parseTxtContentDirect(rawData, storageKey)
	}

	if isJSON {
		var parsedBank *BankData
		err := json.Unmarshal(fileBytes, &parsedBank)
		if err != nil {
			return nil, fmt.Errorf("JSON 解析失败，文件可能已损坏或格式不匹配: %v", err)
		}

		if len(parsedBank.Questions) == 0 {
			return nil, fmt.Errorf("该 JSON 文件未包含任何有效题目")
		}

		// 🔥 关键细节：强制覆盖 DisplayName 和 StorageKey！
		// 因为用户可能是把别人导出的 JSON 导入自己电脑，或者重命名了文件。
		// 我们必须使用当前读取引擎新生成的 fileName 和 storageKey，防止本地缓存主键冲突。
		parsedBank.DisplayName = fileName
		parsedBank.StorageKey = storageKey

		// 直接返回这个被完美复原的 BankData
		return parsedBank, nil
	}

	return nil, fmt.Errorf("读取异常：不支持的文件格式（未能从二进制头识别出 xls/xlsx 特征）")
}

// ==================== 智能表头搜索 ====================

func parseExcelHeaders(rows [][]string) map[string]int {
	headers := map[string]int{
		"questionNumber": -1, "questionType": -1, "questionContent": -1, "difficulty": -1, "correctAnswer": -1,
		"optionA": -1, "optionB": -1, "optionC": -1, "optionD": -1, "optionE": -1, "optionF": -1, "optionG": -1, "optionH": -1, "optionI": -1,
		"start": 0,
	}

	keywords := map[string][]string{
		"questionNumber":  {"题号", "序号", "编号", "number", "id"},
		"questionType":    {"题型", "试题题型", "题型名", "题目类型", "类型", "题类"},
		"questionContent": {"题目", "题干", "内容", "试题内容", "问题", "question", "试题"},
		"difficulty":      {"难度", "难易度", "difficulty"},
		"correctAnswer":   {"答案", "标准答案", "正确答案", "正确选项", "参考答案", "answer"},
		"optionA":         {"选项a", "a选项", "a"},
		"optionB":         {"选项b", "b选项", "b"},
		"optionC":         {"选项c", "c选项", "c"},
		"optionD":         {"选项d", "d选项", "d"},
		"optionE":         {"选项e", "e选项", "e"},
		"optionF":         {"选项f", "f选项", "f"},
		"optionG":         {"选项g", "g选项", "g"},
		"optionH":         {"选项h", "h选项", "h"},
		"optionI":         {"选项i", "i选项", "i"},
	}

	maxHeaderSearchRows := 35
	if len(rows) < maxHeaderSearchRows {
		maxHeaderSearchRows = len(rows)
	}

	for i := 0; i < maxHeaderSearchRows; i++ {
		row := rows[i]
		hasContentKey := false
		hasAnswerKey := false
		hasNumberKey := false

		for _, cell := range row {
			cellText := cleanHeaderMatch(cell)
			if cellText == "" {
				continue
			}

			if !hasContentKey {
				for _, k := range keywords["questionContent"] {
					if strings.Contains(cellText, k) {
						hasContentKey = true
						break
					}
				}
			}
			if !hasAnswerKey {
				for _, k := range keywords["correctAnswer"] {
					if strings.Contains(cellText, k) {
						hasAnswerKey = true
						break
					}
				}
			}
			if !hasNumberKey {
				for _, k := range keywords["questionNumber"] {
					if strings.Contains(cellText, k) {
						hasNumberKey = true
						break
					}
				}
			}
		}

		if hasContentKey && (hasAnswerKey || hasNumberKey) {
			headers["start"] = i
			for key, kwList := range keywords {
				headers[key] = findColumnIndex(row, kwList)
			}
			return headers
		}
	}
	return nil
}

func findColumnIndex(row []string, keywords []string) int {
	for idx, cell := range row {
		cellText := cleanHeaderMatch(cell)
		for _, k := range keywords {
			if cellText == k {
				return idx
			}
		}
	}
	for idx, cell := range row {
		cellText := cleanHeaderMatch(cell)
		for _, k := range keywords {
			if strings.Contains(cellText, k) {
				return idx
			}
		}
	}
	return -1
}

// ==================== XLSX 及 TXT 子解析器 ====================

func parseExcelFromOpenedFile(f *excelize.File, storageKey string) (*BankData, error) {
	bank := &BankData{
		Questions: []QuestionItem{},
	}
	globalIdCounter := 1

	sheetNames := f.GetSheetList()
	for _, sheetName := range sheetNames {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) == 0 {
			continue
		}

		headers := parseExcelHeaders(rows)
		if headers == nil || headers["questionContent"] == -1 {
			continue
		}

		startRow := headers["start"]
		for i := startRow + 1; i < len(rows); i++ {
			row := rows[i]

			if len(row) <= headers["questionContent"] || strings.TrimSpace(row[headers["questionContent"]]) == "" {
				continue
			}

			qType := sheetName
			if headers["questionType"] != -1 && len(row) > headers["questionType"] && strings.TrimSpace(row[headers["questionType"]]) != "" {
				qType = strings.TrimSpace(row[headers["questionType"]])
			}

			qObj := QuestionItem{
				ID:         globalIdCounter,
				Type:       qType,
				Content:    strings.TrimSpace(row[headers["questionContent"]]),
				Options:    make(OptionMap),
				SheetName:  sheetName,
				RawIndex:   i + 1,
				Difficulty: "普通",
			}

			if headers["correctAnswer"] != -1 && len(row) > headers["correctAnswer"] {
				qObj.Answer = strings.TrimSpace(row[headers["correctAnswer"]])
			}

			optionKeys := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
			hasFoundOriginalOptions := false

			for _, key := range optionKeys {
				headerKey := "option" + key
				colIdx := headers[headerKey]
				if colIdx != -1 && len(row) > colIdx && strings.TrimSpace(row[colIdx]) != "" {
					qObj.Options[key] = strings.TrimSpace(row[colIdx])
					hasFoundOriginalOptions = true
				} else {
					qObj.Options[key] = ""
				}
			}

			if !hasFoundOriginalOptions && qObj.Answer != "" {
				qObj.Options["A"] = qObj.Answer
			}

			if strings.Contains(qObj.Type, "判断") {
				isTrue := qObj.Answer == "正确" || qObj.Answer == "√" || qObj.Answer == "对" || qObj.Answer == "A"
				qObj.Options["A"] = "正确"
				qObj.Options["B"] = "错误"
				if isTrue {
					qObj.Answer = "A"
				} else {
					qObj.Answer = "B"
				}
			}

			bank.Questions = append(bank.Questions, qObj)
			globalIdCounter++
		}
	}

	if len(bank.Questions) == 0 {
		return nil, fmt.Errorf("未能从 XLSX 文件中解析出有效题目")
	}

	bank.StorageKey = storageKey
	bank.DisplayName = extractFileName(storageKey)
	return bank, nil
}

// 解析 TXT 文件
func parseTxtContentDirect(content string, storageKey string) (*BankData, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	bank := &BankData{Questions: []QuestionItem{}}
	lines := strings.Split(content, "\n")

	var currentType string
	var currentQuestion *QuestionItem
	idCounter := 1

	questionReg := regexp.MustCompile(`^(\d+)\s*[\.、](.*)`)

	// ✨ 修复 1：彻底移除 (?=...) 断言！改为仅寻找选项的前缀标识（如 "A."、" B、"）
	optionsReg := regexp.MustCompile(`(?:^|\s+)([A-I])[\.、]`)

	ansReg := regexp.MustCompile(`[（(]\s*([A-I√✓xX✗×对错])\s*[）)]`)

	// ✨ 修复 2：填空题正则的 (?=\s+) 也去掉了，后面用纯代码替代判断，防止二次崩溃
	blankReg := regexp.MustCompile(`\s+([\x{4e00}-\x{9fa5}a-zA-Z0-9]+)`)

	endNumReg := regexp.MustCompile(`(\d+[\d\-\.~/]*\d*)$`)

	// ✨ 新增：专门匹配形如 "( 答案 )" 或 "（\n答案 ）" 的填空题
	// 特征：左括号 + 至少一个空白符 + 答案内容 + 至少一个空白符 + 右括号
	parenBlankReg := regexp.MustCompile(`[（(]\s+([^（()）]+?)\s+[）)]`)

	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "填空题") {
			currentType = "填空题"
			continue
		}
		if strings.Contains(line, "单项选择题") || strings.Contains(line, "选择题") || strings.Contains(line, "多项选择题") {
			currentType = "单选题"
			continue
		}
		if strings.Contains(line, "判断题") {
			currentType = "判断题"
			continue
		}
		if strings.Contains(line, "问答题") || strings.Contains(line, "简答题") || strings.Contains(line, "案例分析") {
			currentType = "问答题"
			continue
		}

		if match := questionReg.FindStringSubmatch(line); match != nil {
			if currentQuestion != nil {
				bank.Questions = append(bank.Questions, *currentQuestion)
			}
			qContent := match[2]
			currentQuestion = &QuestionItem{
				ID: idCounter, Type: currentType, Content: qContent,
				Options: make(OptionMap), RawIndex: idx + 1, Difficulty: "普通",
			}
			idCounter++
			for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
				currentQuestion.Options[k] = ""
			}

			if currentType == "单选题" || currentType == "判断题" {
				if ansMatch := ansReg.FindStringSubmatch(qContent); ansMatch != nil {
					rawAns := strings.TrimSpace(ansMatch[1])
					switch rawAns {
					case "√", "✓", "对":
						currentQuestion.Answer = "A"
					case "x", "X", "✗", "×", "错":
						currentQuestion.Answer = "B"
					default:
						currentQuestion.Answer = rawAns
					}
					currentQuestion.Content = ansReg.ReplaceAllString(qContent, " ( )")
				}
				if currentType == "判断题" {
					currentQuestion.Options["A"] = "正确"
					currentQuestion.Options["B"] = "错误"
				}
			} else if currentType == "填空题" {
				var ansList []string
				newContent := qContent

				// ✨ 优先匹配：适配 "( 答案 )" 或 "(\n答案 )" 格式
				parenMatches := parenBlankReg.FindAllStringSubmatch(qContent, -1)

				if len(parenMatches) > 0 {
					for _, m := range parenMatches {
						fullMatch := m[0]                  // 完整的匹配文本，例如 "( \n76~84 )"
						ansText := strings.TrimSpace(m[1]) // 提取出的纯答案，例如 "76~84"

						ansList = append(ansList, ansText)

						// 将原文本中的括号及答案替换为规范的填空横线
						newContent = strings.Replace(newContent, fullMatch, "( ______ )", 1)
					}

					currentQuestion.Content = newContent
					ansFinal := strings.Join(ansList, "、")
					currentQuestion.Options["A"] = ansFinal
					currentQuestion.Answer = ansFinal

				} else {
					// 兜底逻辑：如果不是带空格的括号格式，继续使用原来的纯空格匹配法
					matches := blankReg.FindAllStringSubmatchIndex(qContent, -1)
					for _, m := range matches {
						wordEnd := m[3]
						if wordEnd == len(qContent) || strings.ContainsRune(" \t\n\r", rune(qContent[wordEnd])) {
							word := qContent[m[2]:m[3]]
							ansList = append(ansList, word)
							newContent = strings.Replace(newContent, word, "______", 1)
						}
					}

					ansText := ""
					if len(ansList) > 0 {
						ansText = strings.Join(ansList, "、")
						currentQuestion.Content = newContent
					} else if endNumMatch := endNumReg.FindStringSubmatch(qContent); endNumMatch != nil {
						ansText = endNumMatch[1]
						currentQuestion.Content = endNumReg.ReplaceAllString(qContent, "______")
					}

					if ansText != "" {
						currentQuestion.Options["A"] = ansText
						currentQuestion.Answer = ansText
					} else {
						currentQuestion.Options["A"] = "（未识别到答案）"
					}
				}

				// 调试输出
				//fmt.Println("currentQuestion.Options[A]: ", currentQuestion.Options["A"])
				//fmt.Println("currentQuestion.Answer: ", currentQuestion.Answer)
			}
		} else if currentType == "单选题" && currentQuestion != nil {
			// ✨ 修复 1：利用 Index 截取法完美实现多选项解析，彻底绕过正则断言限制
			matches := optionsReg.FindAllStringSubmatchIndex(line, -1)
			if len(matches) > 0 {
				for i, match := range matches {
					optLetter := line[match[2]:match[3]] // 获取当前的字母 A-I

					startIdx := match[1] // 当前选项内容开始的位置
					var endIdx int
					if i+1 < len(matches) {
						endIdx = matches[i+1][0] // 提取到下一个选项字母的前面为止
					} else {
						endIdx = len(line) // 如果是最后一个选项，直达行尾
					}

					optContent := strings.TrimSpace(line[startIdx:endIdx])
					currentQuestion.Options[optLetter] = optContent
				}
			} else {
				currentQuestion.Content += "\n" + line
			}
		} else if currentType == "问答题" && currentQuestion != nil {
			if strings.HasPrefix(line, "答:") || strings.HasPrefix(line, "答：") {
				cleanAns := strings.TrimSpace(strings.Replace(strings.Replace(line, "答:", "", 1), "答：", "", 1))
				currentQuestion.Options["A"] = cleanAns
				currentQuestion.Answer = cleanAns
			} else if currentQuestion.Options["A"] != "" {
				currentQuestion.Options["A"] += "\n" + line
			} else {
				currentQuestion.Content += "\n" + line
			}
		}
	}

	if currentQuestion != nil {
		bank.Questions = append(bank.Questions, *currentQuestion)
	}
	if len(bank.Questions) == 0 {
		return nil, fmt.Errorf("未能解析出符合规范的TXT题目")
	}

	bank.StorageKey = storageKey
	bank.DisplayName = extractFileName(storageKey)
	return bank, nil
}

func getFileExtension(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractFileName(fullName string) string {
	reg := regexp.MustCompile(`^(xlsData|xlsxData|txtData|txtsData|jsonData)_`)
	res := reg.ReplaceAllString(fullName, "")
	regEnd := regexp.MustCompile(`\.(xls|xlsx|txt|txts|json)(_\d+)?$`)
	return regEnd.ReplaceAllString(res, "")
}

// 判断字节流是否为 UTF-8 编码
func isUTF8(data []byte) bool {
	i := 0
	for i < len(data) {
		if data[i] < 0x80 {
			i++
			continue
		} else if data[i] >= 0xc0 && data[i] <= 0xdf {
			if i+1 >= len(data) || data[i+1] < 0x80 || data[i+1] > 0xbf {
				return false
			}
			i += 2
			continue
		} else if data[i] >= 0xe0 && data[i] <= 0xef {
			if i+2 >= len(data) || data[i+1] < 0x80 || data[i+1] > 0xbf || data[i+2] < 0x80 || data[i+2] > 0xbf {
				return false
			}
			i += 3
			continue
		} else if data[i] >= 0xf0 && data[i] <= 0xf7 {
			if i+3 >= len(data) || data[i+1] < 0x80 || data[i+1] > 0xbf || data[i+2] < 0x80 || data[i+2] > 0xbf || data[i+3] < 0x80 || data[i+3] > 0xbf {
				return false
			}
			i += 4
			continue
		}
		return false
	}
	return true
}

// ✨ 核心函数：读取 TXT 路径并自动修复 GBK 乱码
func readTxtFileWithEncodingFix(rawBytes []byte) (string, error) {

	// 2. 如果本身就是 UTF-8 编码，直接转换返回，不做额外处理
	if isUTF8(rawBytes) {
		return string(rawBytes), nil
	}

	// 3. 🔥 如果不是 UTF-8，则判定为 GBK，启用硬核解码器将其转换为 UTF-8
	reader := transform.NewReader(bytes.NewReader(rawBytes), simplifiedchinese.GBK.NewDecoder())
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		// 如果解码失败，作为兜底屏障，尝试使用另一种国标编码 GB18030
		reader2 := transform.NewReader(bytes.NewReader(rawBytes), simplifiedchinese.GB18030.NewDecoder())
		utf8Bytes, err = io.ReadAll(reader2)
		if err != nil {
			return string(rawBytes), err // 实在不行返回原字符串
		}
	}

	return string(utf8Bytes), nil
}

// 将 xls 字节流直接转换为 xlsx 字节流
func ConvertXlsToXlsxBytes(xlsBytes []byte) ([]byte, error) {
	// 使用更健壮的读取器
	file, err := xls.OpenReader(bytes.NewReader(xlsBytes))
	if err != nil {
		return nil, err
	}

	xlsxFile := excelize.NewFile()
	defer xlsxFile.Close()

	for _, sheet := range file.GetSheets() {
		xlsxFile.NewSheet(sheet.GetName())

		// 循环所有行
		for i := 0; i < sheet.GetNumberRows(); i++ {
			row, err := sheet.GetRow(i)
			if err != nil {
				continue
			}

			// 获取这一行所有的列对象
			cols := row.GetCols()
			for j := 0; j < len(cols); j++ {
				// 获取特定列的值
				cell, err := row.GetCol(j)
				if err != nil {
					continue
				}

				// 写入 xlsx
				pos, _ := excelize.CoordinatesToCellName(j+1, i+1)
				xlsxFile.SetCellValue(sheet.GetName(), pos, cell.GetString())
			}
		}
	}

	// 将 xlsx 写入到字节缓存
	buf, err := xlsxFile.WriteToBuffer()
	return buf.Bytes(), err
}
