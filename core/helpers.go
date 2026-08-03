package core

import (
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"

	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Question 表示一道题目，包含题干、选项、答案和难度信息。
type Question struct {
	ID         string
	Type       string // "单选题", "多选题", "判断题", "填空题", "问答题"
	Content    string
	Options    []Option
	Answers    []string // 正确答案的 Label，例如 ["A", "C"]
	Difficulty string
}

// Option 表示题目的一个选项，包含字母标签和文本内容。
type Option struct {
	Label string // A, B, C, D
	Text  string
}

// AppState 维护全局状态和练习会话的状态。
type AppState struct {
	Questions   []Question
	CurrentList []Question
	Index       int
	UserAnswers map[string][]string
	Submitted   map[string]bool
	Title       string // 当前的练习模式名称 (如："顺序练习-全部题目", "错题练习-单选题")

	// ======= 维护本地文件状态的字段 =======
	CurrentStorageKey string   // 当前激活题库的 Key
	CurrentFileName   string   // 当前激活题库的显示名称
	StoredFiles       []string // 已导入的文件 Key 列表

	// ✨ 当前题库的错题和收藏索引集（使用 map 方便在 UI 中 O(1) 复杂度查询）
	WrongSet map[string]bool // key 为 Question.ID
	FavSet   map[string]bool // key 为 Question.ID

	// ✨ 本次练习的核心统计字段
	ModeRecords     map[string]map[string]string // ✨ 所有模式的答题记录总集
	PracticeRecords map[string]string            // 本次练习的详细答题记录。key: Question.ID, value: 已答选项(如"A") 或 "null"表示未答

	ModeIndices map[string]int // 用于独立保存每个模式的答题进度 (题号索引)

	// ✨ PracticeRecords 数据竞争保护
	PracticeRecordsMu sync.Mutex

	// ✨ 搜索页防抖 Timer，页面切换时取消以避免访问过期状态
	ActiveSearchTimer *time.Timer
	SearchGeneration  int // 每次输入递增，timer 回调检查是否匹配当前代
}

// hexColor 将十六进制颜色字符串（如 "#RRGGBB"）转换为 fyne.color.Color。
func HexColor(hex string) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	return color.Transparent
}

// Contains 检查字符串切片中是否包含指定元素。
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveElement 从字符串切片中移除指定元素，返回新切片。
func RemoveElement(slice []string, item string) []string {
	var res []string
	for _, s := range slice {
		if s != item {
			res = append(res, s)
		}
	}
	return res
}

// IsAnswerCorrect 判断用户答案是否正确，统一处理所有题型的比对逻辑。
func IsAnswerCorrect(userAnswer string, q Question) bool {
	if userAnswer == "" || userAnswer == "null" {
		return false
	}

	// 填空/问答题：只要查看了（非空）即视为正确
	if q.Type == "填空题" || q.Type == "问答题" {
		return true
	}

	// 将用户答案和标准答案都切分为排序后的字符串切片，消除顺序干扰
	userSlice := NormalizeAnswerSlice(userAnswer, q)
	correctSlice := NormalizeAnswerSlice(strings.Join(q.Answers, ","), q)

	if len(userSlice) != len(correctSlice) {
		return false
	}
	for i := range userSlice {
		if userSlice[i] != correctSlice[i] {
			return false
		}
	}
	return true
}

// NormalizeAnswerSlice 将用户答案字符串切分为排序后的标准答案切片。
func NormalizeAnswerSlice(userAnswer string, q Question) []string {
	var slice []string
	if q.Type == "多选题" && strings.Contains(userAnswer, ",") {
		for _, part := range strings.Split(userAnswer, ",") {
			slice = append(slice, strings.TrimSpace(part))
		}
	} else if len(userAnswer) == 1 {
		slice = []string{userAnswer}
	} else {
		// 填空/问答等：按顿号拆分
		for _, part := range strings.Split(userAnswer, "、") {
			if s := strings.TrimSpace(part); s != "" {
				slice = append(slice, s)
			}
		}
	}
	sort.Strings(slice)
	return slice
}

// CreateOptionLabel 创建一个带居中对齐、自动换行和标准字号的 Label。
func CreateOptionLabel(optStr string) *widget.Label {
	label := widget.NewLabel(optStr)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapBreak
	label.SizeName = theme.SizeNameSubHeadingText
	return label
}

// ShowCustomConfirm 显示一个自定义的确认对话框，替代原生 dialog.ShowConfirm。
func ShowCustomConfirm(title, message string, callback func(bool), parent fyne.Window) {
	var d dialog.Dialog

	btnConfirm := widget.NewButton("确定", func() {
		if d != nil {
			d.Hide()
		}
		if callback != nil {
			callback(true)
		}
	})
	btnConfirm.Importance = widget.HighImportance

	btnCancel := widget.NewButton("取消", func() {
		if d != nil {
			d.Hide()
		}
		if callback != nil {
			callback(false)
		}
	})

	spacer := container.NewGridWrap(fyne.NewSize(40, 1))
	btnGroup := container.NewHBox(btnCancel, spacer, btnConfirm)

	msgLabel := widget.NewLabel(message)
	msgLabel.Alignment = fyne.TextAlignCenter
	msgLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		msgLabel,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		container.NewPadded(container.NewCenter(btnGroup)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(300, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(content),
	)

	d = dialog.NewCustomWithoutButtons(title, styledContent, parent)
	d.Show()
}

// ShowCustomInformation 显示一个自定义的信息提示对话框，替代原生 dialog.ShowInformation。
func ShowCustomInformation(title, message string, parent fyne.Window) {
	var d dialog.Dialog

	btnOk := widget.NewButton("确定", func() {
		if d != nil {
			d.Hide()
		}
	})
	btnOk.Importance = widget.HighImportance

	msgLabel := widget.NewLabel(message)
	msgLabel.Alignment = fyne.TextAlignCenter
	msgLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		msgLabel,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		container.NewPadded(container.NewCenter(btnOk)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(300, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(content),
	)

	d = dialog.NewCustomWithoutButtons(title, styledContent, parent)
	d.Show()
}
