package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget" // 添加 widget 包的引用

	"github.com/gzjjjfree/practice/core"
)

// BoxColor defines button color and font configuration.
type BoxColor struct {
	TextColor   string
	BgColor     string
	StrokeColor string
	TextSize    float32
}

// ClickableBox is a customizable clickable box component.
type ClickableBox struct {
	widget.BaseWidget // 【修改1】使用 BaseWidget 替代 fyne.CanvasObject

	bgText      *canvas.Text
	bgRect      *canvas.Rectangle
	normalColor BoxColor
	isDisabled  bool
	OnTapped    func()

	Content *fyne.Container
}

// GetDefaultColor returns the default secondary button color scheme.
func GetDefaultColor() *BoxColor {
	return &BoxColor{TextColor: core.TextSecondaryColor, BgColor: core.PageBgColor, StrokeColor: core.BorderLightColor, TextSize: 18}
}

// NewClickableBox creates a clickable box component.
func NewClickableBox(content *fyne.Container, onTap func()) *ClickableBox {
	c := &ClickableBox{Content: content, OnTapped: onTap}
	c.ExtendBaseWidget(c) // 【修改2】必须调用 ExtendBaseWidget 初始化
	return c
}

// SetDisabled dynamically sets whether the button is disabled.
func (b *ClickableBox) SetDisabled(disabled bool) {
	b.isDisabled = disabled

	if disabled {
		if b.bgText != nil {
			b.bgText.Color = core.HexColor(core.TextDisabledColor)
		}
		if b.bgRect != nil {
			b.bgRect.FillColor = core.HexColor(core.BtnDisabledBg)
			b.bgRect.StrokeColor = core.HexColor(core.BtnDisabledStroke)
		}
	} else {
		if b.bgText != nil {
			b.bgText.Color = core.HexColor(b.normalColor.TextColor)
		}
		if b.bgRect != nil {
			b.bgRect.FillColor = core.HexColor(b.normalColor.BgColor)
			b.bgRect.StrokeColor = core.HexColor(b.normalColor.StrokeColor)
		}
	}

	if b.bgText != nil {
		b.bgText.Refresh()
	}
	if b.bgRect != nil {
		b.bgRect.Refresh()
	}
	if b.Content != nil {
		b.Content.Refresh() // 使用 Content.Refresh 替代 b.Refresh 防止死循环
	}
}

// Tapped implements fyne.Tappable.
func (b *ClickableBox) Tapped(_ *fyne.PointEvent) {
	if b.isDisabled || b.OnTapped == nil {
		return
	}
	b.OnTapped()
}

// TappedSecondary implements right-click (no-op).
func (b *ClickableBox) TappedSecondary(_ *fyne.PointEvent) {}

// 【修改3】删除了 MinSize, Resize, Position, Move, Scale 等方法，因为 BaseWidget 会自动接管并交给 Renderer 处理。

// CreateRenderer implements fyne.WidgetRenderer.
func (b *ClickableBox) CreateRenderer() fyne.WidgetRenderer {
	var objs []fyne.CanvasObject

	// 如果 Content 存在，直接渲染 Content (在 GetBox 中 bgRect 已经被包裹在 Content 里了)
	if b.Content != nil {
		objs = append(objs, b.Content)
	} else if b.bgRect != nil {
		// 容错处理：如果只有 bgRect 没有 Content
		objs = append(objs, b.bgRect)
	}

	return &clickableBoxRenderer{
		objects: objs,
		obj:     b,
	}
}

type clickableBoxRenderer struct {
	objects []fyne.CanvasObject
	obj     *ClickableBox
}

func (r *clickableBoxRenderer) Layout(size fyne.Size) {
	for _, o := range r.objects {
		if o != nil { // 增加安全判断
			o.Resize(size)
		}
	}
}

func (r *clickableBoxRenderer) MinSize() fyne.Size {
	var min fyne.Size
	for _, o := range r.objects {
		if o != nil { // 增加安全判断
			m := o.MinSize()
			if m.Width > min.Width {
				min.Width = m.Width
			}
			if m.Height > min.Height {
				min.Height = m.Height
			}
		}
	}
	return min
}

func (r *clickableBoxRenderer) Refresh() {
	for _, obj := range r.objects {
		if obj != nil { // 增加安全判断
			obj.Refresh()
		}
	}
}

func (r *clickableBoxRenderer) Append(o fyne.CanvasObject) {}
func (r *clickableBoxRenderer) Remove(target fyne.CanvasObject) {
	for i, o := range r.objects {
		if o == target {
			r.objects = append(r.objects[:i], r.objects[i+1:]...)
			return
		}
	}
}
func (r *clickableBoxRenderer) Replace(old, new fyne.CanvasObject) {
	for i, o := range r.objects {
		if o == old {
			r.objects[i] = new
			return
		}
	}
}
func (r *clickableBoxRenderer) Count() int                   { return len(r.objects) }
func (r *clickableBoxRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *clickableBoxRenderer) Destroy()                     {}

// GetBox creates a styled clickable box button with custom colors, size, and tap callback.
func GetBox(btnText string, width float32, height float32, inColor *BoxColor, bold bool, isDisabled bool, backfun func()) *ClickableBox {
	bxColor := *inColor

	backBtnText := canvas.NewText(btnText, core.HexColor(bxColor.TextColor))
	backBtnText.TextSize = bxColor.TextSize
	backBtnText.TextStyle = fyne.TextStyle{Bold: bold}

	backBtnBg := canvas.NewRectangle(core.HexColor(bxColor.BgColor))
	backBtnBg.CornerRadius = 4
	backBtnBg.StrokeColor = core.HexColor(bxColor.StrokeColor)
	backBtnBg.StrokeWidth = 1

	sizeBox := container.NewGridWrap(fyne.NewSize(width, height))

	backBtnCustomLayout := container.NewStack(
		backBtnBg,
		sizeBox,
		container.NewCenter(backBtnText),
	)

	box := &ClickableBox{
		bgText:      backBtnText,
		bgRect:      backBtnBg,
		Content:     backBtnCustomLayout,
		normalColor: *inColor,
		OnTapped:    backfun,
	}

	box.ExtendBaseWidget(box) // 【修改4】这里也必须调用 ExtendBaseWidget
	box.SetDisabled(isDisabled)

	return box
}

// GetTitle creates the top title bar showing the app title.
func GetTitle() *fyne.Container {
	appTitleText := canvas.NewText("📖 题库练习系统", core.HexColor(core.TextPrimaryColor))
	appTitleText.TextSize = 22
	appTitleText.Alignment = fyne.TextAlignCenter

	return container.NewPadded(container.NewVBox(layout.NewSpacer(), appTitleText, layout.NewSpacer()))
}
