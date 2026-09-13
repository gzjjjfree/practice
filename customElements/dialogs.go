package customElements

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
)

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
