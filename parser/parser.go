package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"

	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// OptionMap maps option labels (A-I) to text.
type OptionMap map[string]string

// QuestionItem is an internal parsed question struct used for JSON serialization.
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

// BankData is the top-level container for a question bank.
type BankData struct {
	DisplayName string         `json:"displayName"`
	StorageKey  string         `json:"storageKey"`
	Questions   []QuestionItem `json:"questions"`
}

// ==================== Data Cleaning ====================

// CleanXlsData removes whitespace and null bytes from parsed text.
func CleanXlsData(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
}

// CleanHeaderMatch normalizes header text for matching.
func CleanHeaderMatch(text string) string {
	t := CleanXlsData(text)
	t = strings.ReplaceAll(t, " ", "")
	t = strings.ReplaceAll(t, "\u3000", "") // full-width space (U+3000)
	t = strings.ReplaceAll(t, "\n", "")
	t = strings.ReplaceAll(t, "\r", "")
	return strings.ToLower(t)
}

// stripLeadingNumber removes common leading question number formats.
var leadingNumberRegexes []*regexp.Regexp

func init() {
	patterns := []string{
		`^[0-9]+[\.、\)]`,
		`^[（\(][0-9]+[）\)]`,
		`^[\x{2460}-\x{24FF}]`,
		`^[\x{3251}-\x{32BF}]`,
		`^[一二三四五六七八九十]+`,
		`^[IVXivx]+[\.、\)]`,
	}
	for _, pat := range patterns {
		re := regexp.MustCompile(pat)
		leadingNumberRegexes = append(leadingNumberRegexes, re)
	}
}

func stripLeadingNumber(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return t
	}

	for _, re := range leadingNumberRegexes {
		if re.MatchString(t) {
			t = re.ReplaceAllString(t, "")
			return strings.TrimSpace(t)
		}
	}
	return t
}

// batchStripLeadingNumbers strips leading numbers from consecutive questions of the same type.
func batchStripLeadingNumbers(questions []QuestionItem, threshold int) {
	if len(questions) == 0 || threshold < 2 {
		return
	}

	typeIndexMap := make(map[string][]int)
	for i, q := range questions {
		typeIndexMap[q.Type] = append(typeIndexMap[q.Type], i)
	}

	for _, indices := range typeIndexMap {
		if len(indices) < threshold {
			continue
		}

		consecutiveStart := -1
		consecutiveCount := 0

		for w := 0; w < len(indices); w++ {
			idx := indices[w]
			stripped := stripLeadingNumber(questions[idx].Content)
			if stripped != "" && stripped != questions[idx].Content {
				if consecutiveStart == -1 {
					consecutiveStart = w
					consecutiveCount = 1
				} else {
					consecutiveCount++
				}

				if consecutiveCount >= threshold {
					for k := consecutiveStart; k <= w; k++ {
						questions[indices[k]].Content = stripLeadingNumber(questions[indices[k]].Content)
					}
				}
			} else {
				consecutiveStart = -1
				consecutiveCount = 0
			}
		}

		if consecutiveCount >= threshold {
			for k := consecutiveStart; k < len(indices); k++ {
				questions[indices[k]].Content = stripLeadingNumber(questions[indices[k]].Content)
			}
		}
	}
}

// ==================== Core Parsing Engine ====================

// ParseBytesToBank detects file format via magic bytes and dispatches to the appropriate parser.
func ParseBytesToBank(fileBytes []byte, fileName string, storageKey string) (bank *BankData, err error) {
	defer func() {
		if r := recover(); r != nil {
			stackBuf := make([]byte, 4096)
			n := runtime.Stack(stackBuf, false)
			stackTrace := string(stackBuf[:n])
			fmt.Printf("[PANIC] parse error (caught): %v\n%s\n", r, stackTrace)
			err = fmt.Errorf("parse error (caught): %v\nstack: %s", r, stackTrace)
		}
	}()

	isXLSX := bytes.HasPrefix(fileBytes, []byte{0x50, 0x4B, 0x03, 0x04}) || strings.HasSuffix(strings.ToLower(fileName), ".xlsx")
	isTXT := strings.HasSuffix(strings.ToLower(fileName), ".txt") && len(fileBytes) > 0
	isJSON := strings.HasSuffix(strings.ToLower(fileName), ".json") && len(fileBytes) > 0

	// XLSX parsing
	if isXLSX {
		fmt.Println("in isXLSX")
		f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
		if err != nil {
			return nil, fmt.Errorf("XLSX decode error: %w", err)
		}
		defer f.Close()
		return parseExcelFromOpenedFile(f, storageKey)
	}

	// TXT parsing
	if isTXT {
		fmt.Println("in isTXT")
		rawData, err := readTxtFileWithEncodingFix(fileBytes)
		if err != nil {
			return nil, fmt.Errorf("TXT file read failed")
		}
		return parseTxtContentDirect(rawData, storageKey)
	}

	if isJSON {
		var parsedBank *BankData
		err := json.Unmarshal(fileBytes, &parsedBank)
		if err != nil {
			return nil, fmt.Errorf("JSON parse failed, file may be corrupted or mismatched: %v", err)
		}

		if len(parsedBank.Questions) == 0 {
			return nil, fmt.Errorf("this JSON file contains no valid questions")
		}

		parsedBank.DisplayName = ExtractFileName(storageKey)
		parsedBank.StorageKey = storageKey
		return parsedBank, nil
	}

	return nil, fmt.Errorf("unsupported file format (could not recognize xls/xlsx magic bytes)")
}

// ==================== Smart Header Search ====================

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
			cellText := CleanHeaderMatch(cell)
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
		cellText := CleanHeaderMatch(cell)
		for _, k := range keywords {
			if cellText == k {
				return idx
			}
		}
	}
	for idx, cell := range row {
		cellText := CleanHeaderMatch(cell)
		for _, k := range keywords {
			if strings.Contains(cellText, k) {
				return idx
			}
		}
	}
	return -1
}

// ==================== XLSX and TXT Sub-Parsers ====================

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
		return nil, fmt.Errorf("no valid questions parsed from XLSX file")
	}

	batchStripLeadingNumbers(bank.Questions, 3)

	bank.StorageKey = storageKey
	bank.DisplayName = ExtractFileName(storageKey)
	return bank, nil
}

func parseTxtContentDirect(content string, storageKey string) (*BankData, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	bank := &BankData{Questions: []QuestionItem{}}
	lines := strings.Split(content, "\n")

	var currentType string
	var currentQuestion *QuestionItem
	idCounter := 1

	questionReg := regexp.MustCompile(`^(\d+)\s*[\.、](.*)`)
	optionsReg := regexp.MustCompile(`(?:^|\s+)([A-I])[\.、]`)
	ansReg := regexp.MustCompile(`[（(]\s*([A-I√✓xX✗×对错])\s*[）)]`)
	blankReg := regexp.MustCompile(`\s+([\x{4e00}-\x{9fa5}a-zA-Z0-9]+)`)
	endNumReg := regexp.MustCompile(`(\d+[\d\-\.~/]*\d*)$`)
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

				parenMatches := parenBlankReg.FindAllStringSubmatch(qContent, -1)

				if len(parenMatches) > 0 {
					for _, m := range parenMatches {
						fullMatch := m[0]
						ansText := strings.TrimSpace(m[1])
						ansList = append(ansList, ansText)
						newContent = strings.Replace(newContent, fullMatch, "( ______ )", 1)
					}

					currentQuestion.Content = newContent
					ansFinal := strings.Join(ansList, "、")
					currentQuestion.Options["A"] = ansFinal
					currentQuestion.Answer = ansFinal

				} else {
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
			}
		} else if currentType == "单选题" && currentQuestion != nil {
			matches := optionsReg.FindAllStringSubmatchIndex(line, -1)
			if len(matches) > 0 {
				for i, match := range matches {
					optLetter := line[match[2]:match[3]]

					startIdx := match[1]
					var endIdx int
					if i+1 < len(matches) {
						endIdx = matches[i+1][0]
					} else {
						endIdx = len(line)
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
		return nil, fmt.Errorf("no valid TXT questions parsed")
	}

	batchStripLeadingNumbers(bank.Questions, 3)

	bank.StorageKey = storageKey
	bank.DisplayName = ExtractFileName(storageKey)
	return bank, nil
}

func GetFileExtension(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func ExtractFileName(fullName string) string {
	reg := regexp.MustCompile(`^(xlsData|xlsxData|txtData|txtsData|jsonData)_`)
	res := reg.ReplaceAllString(fullName, "")
	// Strip trailing _<digits> (timestamp) whether or not preceded by a file extension.
	// Handles both "xxx.xlsx_12345" and "xxx_12345" (no extension).
	regEnd := regexp.MustCompile(`(?:\.(?:xls|xlsx|txt|txts|json))?(?:_\d+)?$`)
	return regEnd.ReplaceAllString(res, "")
}

// IsUTF8 checks whether the given byte slice is valid UTF-8.
func IsUTF8(data []byte) bool {
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

func readTxtFileWithEncodingFix(rawBytes []byte) (string, error) {
	if IsUTF8(rawBytes) {
		return string(rawBytes), nil
	}

	reader := transform.NewReader(bytes.NewReader(rawBytes), simplifiedchinese.GBK.NewDecoder())
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		reader2 := transform.NewReader(bytes.NewReader(rawBytes), simplifiedchinese.GB18030.NewDecoder())
		utf8Bytes, err = io.ReadAll(reader2)
		if err != nil {
			return string(rawBytes), err
		}
	}

	return string(utf8Bytes), nil
}

// ConvertXlsToXlsxBytes converts legacy .xls bytes to .xlsx bytes.
func ConvertXlsToXlsxBytes(xlsBytes []byte) ([]byte, error) {
	file, err := xls.OpenReader(bytes.NewReader(xlsBytes))
	if err != nil {
		return nil, err
	}

	xlsxFile := excelize.NewFile()
	defer xlsxFile.Close()

	for _, sheet := range file.GetSheets() {
		xlsxFile.NewSheet(sheet.GetName())

		for i := 0; i < sheet.GetNumberRows(); i++ {
			row, err := sheet.GetRow(i)
			if err != nil {
				continue
			}

			cols := row.GetCols()
			for j := 0; j < len(cols); j++ {
				cell, err := row.GetCol(j)
				if err != nil {
					continue
				}

				pos, _ := excelize.CoordinatesToCellName(j+1, i+1)
				xlsxFile.SetCellValue(sheet.GetName(), pos, cell.GetString())
			}
		}
	}

	buf, err := xlsxFile.WriteToBuffer()
	return buf.Bytes(), err
}
