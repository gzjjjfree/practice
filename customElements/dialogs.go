package customElements

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
)

// emptyCell creates a transparent placeholder for grid cells that should be empty.
// Fyne containers cannot accept nil as child elements, so we use an invisible canvas.Rectangle.
func emptyCell() *canvas.Rectangle {
	r := canvas.NewRectangle(color.Transparent) // Start with transparent fill
	r.FillColor = color.Transparent             // Keep it transparent
	r.StrokeColor = color.Transparent           // No border
	r.Refresh()                                 // Ensure it renders correctly
	return r
}

// ShowCustomConfirm 显示一个自定义的确认对话框，替代原生 dialog.ShowConfirm。
// 接收自定义的 content fyne.CanvasObject、确定/取消按钮文案，并集成了自定义按钮样式与空窗口安全兜底。
// callback 返回值表示是否关闭对话框：true 表示关闭，false 表示保持打开。
func ShowCustomConfirm(title string, confirm string, dismiss string, content fyne.CanvasObject, callback func(bool) bool, parent fyne.Window) {
	var d dialog.Dialog

	btnConfirm := CreateButton(
		confirm, 120, 42,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1.5, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
		func() {
			if callback != nil {
				closeDialog := callback(true)
				if closeDialog && d != nil {
					d.Hide()
				}
			}
		},
	)

	btnCancel := CreateButton(
		dismiss, 120, 42,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.SectionBgColor),
		core.HexColor(core.SectionBgColor),
		1.5, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
		func() {
			if callback != nil {
				callback(false)
			}
			if d != nil {
				d.Hide()
			}
		},
	)

	spacer := container.NewGridWrap(fyne.NewSize(core.DialogButtonGap, 1))
	btnGroup := container.NewHBox(btnCancel, spacer, btnConfirm)

	// 计算屏幕/窗口允许的最大可用内容高度（保留 75% 给内容，留出空间给标题栏和底部按钮）
	maxContentHeight := float32(600) // 默认安全上限高度
	if parent != nil {
		canvasHeight := parent.Canvas().Size().Height
		if canvasHeight > 0 {
			maxContentHeight = canvasHeight * 0.75
		}
	}

	// 获取内容自身需要的真实完整高度
	contentRealHeight := content.MinSize().Height

	var displayContent fyne.CanvasObject

	if contentRealHeight > maxContentHeight {
		// 【超长情况】：内容超过允许的最大高度，启用 VScroll 滚动条并限制最大高度
		scrollableContent := container.NewVScroll(content)
		scrollableContent.SetMinSize(fyne.NewSize(core.DialogMinWidth, maxContentHeight))
		displayContent = scrollableContent
	} else {
		// 【正常情况】：内容较少，直接展示原始容器，使其自然撑开（解决压缩、显示异常问题）
		displayContent = content
	}

	// 组合布局：上方自适应内容区 + 间距 + 下方按钮区
	dialogContent := container.NewVBox(
		container.NewPadded(displayContent),
		container.NewGridWrap(fyne.NewSize(1, core.DialogVerticalGap)),
		container.NewPadded(container.NewCenter(btnGroup)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(core.DialogMinWidth, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(dialogContent),
	)

	if parent != nil {
		d = dialog.NewCustomWithoutButtons(title, styledContent, parent)
		d.Show()
	} else {
		app := fyne.CurrentApp()
		if app != nil {
			win := app.NewWindow("")
			win.Hide()
			d = dialog.NewCustomWithoutButtons(title, styledContent, win)
			d.Show()
			return
		}
		d = dialog.NewCustomWithoutButtons(title, styledContent, nil)
		d.Show()
	}
}

// ShowCustomInformation 显示一个自定义的信息提示对话框，替代原生 dialog.ShowInformation。
// 包含一个"确定"按钮，消息文本居中显示，最小宽度为 300px。
func ShowCustomInformation(title, message string, parent fyne.Window) {
	var d dialog.Dialog

	btnOk := CreateButton(
		"确定", 120, 42,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1.5, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
		func() {
			if d != nil {
				d.Hide()
			}
		},
	)

	msgText := widget.NewRichText(&widget.TextSegment{
		Text: message,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			Alignment: fyne.TextAlignCenter,
		},
	})
	msgText.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		msgText,
		container.NewGridWrap(fyne.NewSize(1, core.DialogVerticalGap)),
		container.NewPadded(container.NewCenter(btnOk)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(core.DialogMinWidth, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(content),
	)

	if parent != nil {
		d = dialog.NewCustomWithoutButtons(title, styledContent, parent)
		d.Show()
	} else {
		// When parent is nil, create a dialog without parent
		// Use a hidden top-level window as parent to avoid nil pointer in createBeforeShowHook
		app := fyne.CurrentApp()
		if app != nil {
			win := app.NewWindow("")
			win.Hide()
			d = dialog.NewCustomWithoutButtons(title, styledContent, win)
			d.Show()
			return
		}
		// Fallback: show without any window context
		d = dialog.NewCustomWithoutButtons(title, styledContent, nil)
		d.Show()
	}
}

// NewCenterRichText 创建一个居中对齐、字号较大且支持自动换行的富文本消息组件
func NewCenterRichText(text string) *widget.RichText {
	msgText := widget.NewRichText(&widget.TextSegment{
		Text: text,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			Alignment: fyne.TextAlignCenter,
		},
	})
	msgText.Wrapping = fyne.TextWrapWord
	return msgText
}

// TimePickerResult 时间选择器结果
type TimePickerResult struct {
	Time time.Time
}

// chineseMonths 中文月份名称
var chineseMonths = []string{"一月", "二月", "三月", "四月", "五月", "六月",
	"七月", "八月", "九月", "十月", "十一月", "十二月"}

// getCalendarDayColor 获取日历日期按钮背景色
func getCalendarDayColor(isSelected bool) string {
	if isSelected {
		return core.BtnPrimaryBg // 选中时使用主题色背景
	}
	return core.CardBgColor // 未选中使用卡片背景
}

// getCalendarDayBorderColor 获取日历日期按钮边框色
func getCalendarDayBorderColor(isSelected bool) string {
	if isSelected {
		return core.BtnPrimaryBg // 选中时使用主题色边框
	}
	return core.BorderLightColor // 未选中使用浅色边框
}

// daysInMonth 获取某年某月的天数
func daysInMonth(year int, month int) int {
	// 获取下个月第一天，减去一天得到当月最后一天
	nextMonthFirst := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.Local)
	lastDay := nextMonthFirst.AddDate(0, 0, -1)
	return lastDay.Day()
}

// getFirstWeekday 获取某年某月第一天是星期几（0=周日）
func getFirstWeekday(year int, month int) int {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return int(firstDay.Weekday())
}

// ShowTimePickerDialog 显示时间选择对话框，包含日期选择（自定义日历）和时间选择（小时/分钟下拉框）
// parent: 父窗口
// defaultTime: 默认时间
// onConfirm: 确认回调，传入选中的时间
func ShowTimePickerDialog(parent fyne.Window, defaultTime time.Time, onConfirm func(TimePickerResult)) {
	var d dialog.Dialog

	// 初始化默认值
	selectedYear := defaultTime.Year()
	if selectedYear < 2026 {
		selectedYear = 2026
	} else if selectedYear > 2050 {
		selectedYear = 2050
	}
	selectedMonth := int(defaultTime.Month()) // 1-12
	selectedDay := defaultTime.Day()
	selectedHour := defaultTime.Hour()
	selectedMin := defaultTime.Minute()

	var refreshCalendar func()

	// 创建月份选择（中文）
	monthSelect := widget.NewSelect(chineseMonths, func(s string) {
		for i, m := range chineseMonths {
			if m == s {
				selectedMonth = i + 1
				break
			}
		}
	})
	monthSelect.Selected = chineseMonths[selectedMonth-1]

	// 日期输入
	dayEntry := widget.NewEntry()
	dayEntry.SetText(fmt.Sprintf("%d 日", selectedDay))
	dayEntry.SetPlaceHolder("日")
	dayEntry.OnChanged = func(s string) {
		var val int
		_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &val)
		if err == nil && val > 0 && val <= daysInMonth(selectedYear, selectedMonth) {
			selectedDay = val
		}
	}

	// 年份选择（2026-2050）
	years := make([]string, 0, 25)
	for y := 2026; y <= 2050; y++ {
		years = append(years, fmt.Sprintf("%d", y))
	}
	yearSelect := widget.NewSelect(years, func(s string) {
		val, err := strconv.Atoi(s)
		if err == nil && val != selectedYear {
			newYear := val
			// 调整日期：如果新年份的当前月份没有这一天，调整为最后一天
			maxDay := daysInMonth(newYear, selectedMonth)
			if selectedDay > maxDay {
				selectedDay = maxDay
				dayEntry.SetText(fmt.Sprintf("%d 日", selectedDay))
			}
			selectedYear = newYear
			if refreshCalendar != nil {
				refreshCalendar()
			}
		}
	})
	yearSelect.Selected = fmt.Sprintf("%d", selectedYear)

	// 月份选择行（月份下拉内容本身已含"月"字，如"十月"，故其后不再重复显示"月"标签）
	monthRow := container.NewHBox(
		widget.NewLabel("选择日期:"),
		container.NewGridWrap(fyne.NewSize(85, 38), yearSelect),
		widget.NewLabel("年"),
		container.NewGridWrap(fyne.NewSize(80, 38), monthSelect),
		container.NewGridWrap(fyne.NewSize(65, 38), dayEntry),
	)

	// 星期标题
	weekdays := []string{"日", "一", "二", "三", "四", "五", "六"}
	var weekdayLabels []*widget.Label
	for _, w := range weekdays {
		lbl := widget.NewLabel(w)
		lbl.Alignment = fyne.TextAlignCenter
		lbl.TextStyle.Bold = true
		weekdayLabels = append(weekdayLabels, lbl)
	}
	weekdayRow := container.NewHBox()
	for _, lbl := range weekdayLabels {
		weekdayRow.Add(container.NewGridWrap(fyne.NewSize(40, 25), lbl))
	}

	// 创建日期按钮的辅助函数
	createDayButton := func(text string, isSelected bool, onClick func(int), textColor, bgColor, borderColor string) *CustomButton {
		return CreateButton(
			text, 38, 35,
			core.HexColor(textColor), core.HexColor(bgColor), core.HexColor(borderColor),
			2, core.FontSizeBody, true, false,
			fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
			func() {
				dayNum, _ := strconv.Atoi(text)
				onClick(dayNum)
			},
		)
	}

	var calendarContainer *fyne.Container

	// 渲染日历网格（返回容器）
	type renderCalendarFunc func() fyne.CanvasObject
	var renderCalendar renderCalendarFunc
	renderCalendar = func() fyne.CanvasObject {
		startWeekday := getFirstWeekday(selectedYear, selectedMonth)
		daysInCurrentMonth := daysInMonth(selectedYear, selectedMonth)

		// 创建日期网格容器
		gridContainer := container.NewVBox()

		// 第一行：空白 + 前7天
		row1 := container.NewHBox()
		for cell := 0; cell < 7; cell++ {
			if cell < startWeekday {
				row1.Add(container.NewGridWrap(fyne.NewSize(40, 35), emptyCell()))
			} else {
				dayNum := cell - startWeekday + 1
				isSelected := dayNum == selectedDay
				btn := createDayButton(fmt.Sprintf("%d", dayNum), isSelected, func(d int) {
					selectedDay = d
					dayEntry.SetText(fmt.Sprintf("%d 日", d))
					if refreshCalendar != nil {
						refreshCalendar()
					}
				}, core.TextPrimaryColor, getCalendarDayColor(isSelected), getCalendarDayBorderColor(isSelected))
				row1.Add(container.NewGridWrap(fyne.NewSize(40, 35), btn))
			}
		}
		gridContainer.Add(row1)

		// 第2-6行：完整周或剩余日期
		for row := 1; row < 6; row++ {
			currentRow := container.NewHBox()
			hasDays := false
			for col := 0; col < 7; col++ {
				dayNum := row*7 + col
				if dayNum >= startWeekday && dayNum < startWeekday+daysInCurrentMonth {
					hasDays = true
					actualDay := dayNum - startWeekday + 1
					isSelected := actualDay == selectedDay
					btn := createDayButton(fmt.Sprintf("%d", actualDay), isSelected, func(d int) {
						selectedDay = d
						dayEntry.SetText(fmt.Sprintf("%d 日", d))
						if refreshCalendar != nil {
							refreshCalendar()
						}
					}, core.TextPrimaryColor, getCalendarDayColor(isSelected), getCalendarDayBorderColor(isSelected))
					currentRow.Add(container.NewGridWrap(fyne.NewSize(40, 35), btn))
				} else if hasDays {
					// 已有日期但超出当月天数，显示空白
					currentRow.Add(container.NewGridWrap(fyne.NewSize(40, 35), emptyCell()))
				} else {
					// 尚未开始，显示空白
					currentRow.Add(container.NewGridWrap(fyne.NewSize(40, 35), emptyCell()))
				}
			}
			if hasDays || row < 5 {
				gridContainer.Add(currentRow)
			}
		}

		return gridContainer
	}

	// 初始化日历容器与刷新函数
	calendarContainer = container.NewVBox(renderCalendar())
	refreshCalendar = func() {
		calendarContainer.Objects = []fyne.CanvasObject{renderCalendar()}
		calendarContainer.Refresh()
	}

	// 月份变化时重新渲染
	oldYear := selectedYear
	monthSelect.OnChanged = func(s string) {
		for i, m := range chineseMonths {
			if m == s {
				newMonth := i + 1
				if newMonth != selectedMonth || selectedYear != oldYear {
					selectedMonth = newMonth
					// 调整日期：如果新月份没有这一天，调整为最后一天
					maxDay := daysInMonth(selectedYear, selectedMonth)
					if selectedDay > maxDay {
						selectedDay = maxDay
						dayEntry.SetText(fmt.Sprintf("%d 日", selectedDay))
					}
					refreshCalendar()
					oldYear = selectedYear
				}
				break
			}
		}
	}



	// 日期输入变化时更新按钮状态
	dayEntry.OnChanged = func(s string) {
		var val int
		_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &val)
		if err == nil && val > 0 {
			maxDay := daysInMonth(selectedYear, selectedMonth)
			if val <= maxDay {
				selectedDay = val
				refreshCalendar()
			}
		}
	}

	// 小时选择
	hours := make([]string, 0, 24)
	for h := 0; h < 24; h++ {
		hours = append(hours, fmt.Sprintf("%02d", h))
	}
	hourCombo := widget.NewSelect(hours, func(s string) {
		val, _ := strconv.Atoi(s)
		selectedHour = val
	})
	// 通过直接设置 Selected 字段来设置默认值
	hourCombo.Selected = fmt.Sprintf("%02d", selectedHour)

	// 分钟选择
	minutes := make([]string, 0, 60)
	for m := 0; m < 60; m++ {
		minutes = append(minutes, fmt.Sprintf("%02d", m))
	}
	minuteCombo := widget.NewSelect(minutes, func(s string) {
		val, _ := strconv.Atoi(s)
		selectedMin = val
	})
	// 通过直接设置 Selected 字段来设置默认值
	minuteCombo.Selected = fmt.Sprintf("%02d", selectedMin)

	// 时间行布局
	timeRow := container.NewHBox(
		widget.NewLabel("  "),
		widget.NewLabel("时"),
		container.NewGridWrap(fyne.NewSize(70, 38), hourCombo),
		widget.NewLabel("分"),
		container.NewGridWrap(fyne.NewSize(70, 38), minuteCombo),
	)

	content := container.NewVBox(
		container.NewPadded(monthRow),
		container.NewGridWrap(fyne.NewSize(1, 5)),
		weekdayRow,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		calendarContainer,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		container.NewPadded(timeRow),
	)

	// 确认按钮
	btnConfirm := CreateButton(
		"确定", 80, 36,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1.5, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
		func() {
			if onConfirm != nil {
				// 构建选中的完整时间
				resultTime := time.Date(
					selectedYear,
					time.Month(selectedMonth),
					selectedDay,
					selectedHour,
					selectedMin,
					0, 0,
					defaultTime.Location(),
				)
				onConfirm(TimePickerResult{Time: resultTime})
			}
			if d != nil {
				d.Hide()
			}
		},
	)

	// 取消按钮
	btnCancel := CreateButton(
		"取消", 80, 36,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.SectionBgColor),
		core.HexColor(core.SectionBgColor),
		1.5, core.FontSizeBody,
		true, false,
		fyne.TextAlignCenter, fyne.TextWrapOff, fyne.TextTruncateOff,
		func() {
			if d != nil {
				d.Hide()
			}
		},
	)

	spacer := container.NewGridWrap(fyne.NewSize(core.DialogButtonGap, 1))
	btnGroup := container.NewHBox(btnCancel, spacer, btnConfirm)

	dialogContent := container.NewVBox(
		container.NewPadded(content),
		container.NewGridWrap(fyne.NewSize(1, core.DialogVerticalGap)),
		container.NewPadded(container.NewCenter(btnGroup)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(core.DialogMinWidth, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(dialogContent),
	)

	if parent != nil {
		d = dialog.NewCustomWithoutButtons("选择时间", styledContent, parent)
		d.Show()
	} else {
		app := fyne.CurrentApp()
		if app != nil {
			win := app.NewWindow("")
			win.Hide()
			d = dialog.NewCustomWithoutButtons("选择时间", styledContent, win)
			d.Show()
			return
		}
		d = dialog.NewCustomWithoutButtons("选择时间", styledContent, nil)
		d.Show()
	}
}

// TimeResultToString 将 TimePickerResult 转换为 "2006-01-02 15:04" 格式的时间字符串
func TimeResultToString(result TimePickerResult) string {
	return result.Time.Format("2006-01-02 15:04")
}

// ShowCustomLoading 显示一个无按钮的加载/等待提示对话框（如"正在上传"、"正在下载"），防止用户重复操作。
func ShowCustomLoading(title, message string, parent fyne.Window) dialog.Dialog {
	var d dialog.Dialog

	msgText := widget.NewRichText(&widget.TextSegment{
		Text: message,
		Style: widget.RichTextStyle{
			SizeName:  theme.SizeNameSubHeadingText,
			Alignment: fyne.TextAlignCenter,
		},
	})
	msgText.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		container.NewGridWrap(fyne.NewSize(1, core.DialogVerticalGap)),
		container.NewCenter(msgText),
		container.NewGridWrap(fyne.NewSize(1, core.DialogVerticalGap)),
	)

	minWidthBox := container.NewGridWrap(fyne.NewSize(core.DialogMinWidth, 1))

	styledContent := container.NewStack(
		minWidthBox,
		container.NewPadded(content),
	)

	if parent != nil {
		d = dialog.NewCustomWithoutButtons(title, styledContent, parent)
		d.Show()
	} else {
		app := fyne.CurrentApp()
		if app != nil {
			win := app.NewWindow("")
			win.Hide()
			d = dialog.NewCustomWithoutButtons(title, styledContent, win)
			d.Show()
			return d
		}
		d = dialog.NewCustomWithoutButtons(title, styledContent, nil)
		d.Show()
	}
	return d
}

