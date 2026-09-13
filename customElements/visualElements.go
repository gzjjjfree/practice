package customElements

import (
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
)

// CustomButton 自定义按钮组件
type CustomButton struct {
	widget.DisableableWidget
	OnTapped func()

	btnLabel  *widget.Label
	btnBg     *canvas.Rectangle
	container *fyne.Container

	normalBg     color.Color
	normalText   color.Color
	normalStroke color.Color

	disabledBg     color.Color
	disabledText   color.Color
	disabledStroke color.Color
	disabled       bool
}

// Tapped 点击响应
func (b *CustomButton) Tapped(_ *fyne.PointEvent) {
	// 显式校验 b.disabled 变量
	if b.disabled || b.Disabled() {
		return
	}
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

// Enable 启用按钮
func (b *CustomButton) Enable() {
	b.disabled = false
	if b.btnBg != nil {
		b.btnBg.FillColor = b.normalBg
		b.btnBg.StrokeColor = b.normalStroke
		b.btnBg.Refresh()
	}
	// 校验 b.btnLabel 防止 nil 引用
	if b.btnLabel != nil {
		// 如果需要修改文本颜色，可以通过刷新容器或重新渲染实现
		b.btnLabel.Refresh()
	}
	b.Refresh()
}

// Disable 禁用按钮
func (b *CustomButton) Disable() {
	b.disabled = true
	if b.btnBg != nil {
		b.btnBg.FillColor = b.disabledBg
		b.btnBg.StrokeColor = b.disabledStroke
		b.btnBg.Refresh()
	}
	if b.btnLabel != nil {
		b.btnLabel.Refresh()
	}
	b.Refresh()
}

// buttonTextTheme 局部自定义主题，用于覆盖 Label 的颜色和字号
type buttonTextTheme struct {
	fyne.Theme
	textColor color.Color
	textSize  float32
}

func (t *buttonTextTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 拦截前景色（文本颜色）请求，返回传入的自定义颜色
	if name == theme.ColorNameForeground {
		return t.textColor
	}
	return t.Theme.Color(name, variant)
}

func (t *buttonTextTheme) Size(name fyne.ThemeSizeName) float32 {
	// 拦截标准文本字号请求，返回传入的 float32 自定义大小
	if name == theme.SizeNameText {
		return t.textSize
	}
	return t.Theme.Size(name)
}

// CreateButton 创建带边框的自定义按钮
// 参数：文字、宽度、高度、文字颜色、背景颜色、边框颜色、边框宽度、文字大小、是否加粗、是否禁用、点击回调函数
// 参数增加：align (对齐方式, 如 fyne.TextAlignCenter / fyne.TextAlignLeading), wrap (换行模式, 如 fyne.TextTruncate / fyne.TextWrapBreak)
func CreateButton(
	text string,
	width, height float32,
	textColor, bgColor, strokeColor color.Color,
	strokeWidth float32,
	textSize float32,
	isBold, isDisabled bool,
	align fyne.TextAlign,
	wrap fyne.TextWrap,
	truncation fyne.TextTruncation,
	onTapped func(),
) *CustomButton {
	// 1. 使用支持换行的 widget.Label
	lbl := widget.NewLabel(text)
	lbl.Alignment = align
	lbl.Wrapping = wrap
	lbl.Truncation = truncation
	lbl.TextStyle = fyne.TextStyle{Bold: isBold}

	// 💡 核心魔法：使用局部主题覆盖这个 Label 的颜色和字号
	customTheme := &buttonTextTheme{
		Theme:     theme.DefaultTheme(),
		textColor: textColor,
		textSize:  textSize,
	}
	// 用 ThemeOverride 容器将 Label 包裹起来
	themedText := container.NewThemeOverride(lbl, customTheme)
	//var centeredText fyne.CanvasObject = themedText
	//if wrap == fyne.TextWrapBreak {
	//	// 如果是换行模式，使用 container.NewCenter 保证文本在按钮中垂直居中
	//	centeredText = container.NewCenter(themedText)
	//}

	// 2. 创建背景矩形
	btnBg := canvas.NewRectangle(bgColor)
	btnBg.CornerRadius = core.InputCornerRadius
	btnBg.StrokeColor = strokeColor
	btnBg.StrokeWidth = strokeWidth

	// 3. 将背景和文本进行叠加组合
	// 注意：canvas.Text 在 Stack 中会自动按设置的 Alignment 对齐（如居中）
	var btnContent fyne.CanvasObject
	if wrap == fyne.TextWrapBreak {
		// 换行模式（多行文本）：直接用 Stack 填充背景和文本，
		// 让 Label 撑满按钮宽度并正常折行，避免 NewCenter 导致单字换行崩溃
		btnContent = container.NewStack(btnBg, themedText)
	} else {
		// 单行模式：使用 NewCenter 保证文字在按钮中水平与垂直双向完美居中
		btnContent = container.NewStack(btnBg, container.NewCenter(themedText))
	}

	// 4. 处理按钮尺寸
	var sizedContainer *fyne.Container
	if width > 0 {
		sizedContainer = container.NewGridWrap(fyne.NewSize(width, height), btnContent)
	} else {
		minSizeRect := canvas.NewRectangle(color.Transparent)
		minSizeRect.SetMinSize(fyne.NewSize(10, height))
		sizedContainer = container.NewStack(minSizeRect, btnContent)
	}

	// 5. 构建 CustomButton 对象
	btn := &CustomButton{
		OnTapped:       onTapped,
		btnLabel:       lbl,
		btnBg:          btnBg,
		container:      sizedContainer,
		normalBg:       bgColor,
		normalText:     textColor,
		normalStroke:   strokeColor,
		disabledBg:     core.HexColor(core.BtnDisabledBg),
		disabledText:   core.HexColor(core.TextSecondaryColor),
		disabledStroke: core.HexColor(core.BtnDisabledStroke),
	}

	btn.ExtendBaseWidget(btn)

	if isDisabled {
		btn.Disable()
	} else {
		btn.Enable()
	}

	return btn
}

func (b *CustomButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.container)
}

// TouchInterceptor wraps content inside a scroll container to detect horizontal swipes.
type TouchInterceptor struct {
	widget.BaseWidget
	Content      fyne.CanvasObject
	ParentScroll *container.Scroll
	OnSwipeLeft  func()
	OnSwipeRight func()

	accumX float32
}

// NewTouchInterceptor creates a touch interceptor for swipe-to-navigate.
// Note: onLeft and onRight parameters are intentionally swapped
// (left swipe = next question, right swipe = previous question).
func NewTouchInterceptor(content fyne.CanvasObject, parent *container.Scroll, onLeft, onRight func()) *TouchInterceptor {
	t := &TouchInterceptor{
		Content:      content,
		ParentScroll: parent,
		OnSwipeLeft:  onRight,
		OnSwipeRight: onLeft,
	}
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer returns a simple renderer wrapping the content.
func (t *TouchInterceptor) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.Content)
}

// Dragged detects horizontal vs vertical drag direction.
func (t *TouchInterceptor) Dragged(e *fyne.DragEvent) {
	if math.Abs(float64(e.Dragged.DX)) > math.Abs(float64(e.Dragged.DY)) {
		t.accumX += e.Dragged.DX
	} else {
		t.accumX = 0
		if t.ParentScroll != nil {
			t.ParentScroll.Scrolled(&fyne.ScrollEvent{
				Scrolled: e.Dragged,
			})
		}
	}
}

// DragEnd checks if accumulated X exceeds threshold and triggers appropriate callback.
func (t *TouchInterceptor) DragEnd() {
	if t.accumX < -core.SwipeThreshold {
		if t.OnSwipeLeft != nil {
			t.OnSwipeLeft()
		}
	} else if t.accumX > core.SwipeThreshold {
		if t.OnSwipeRight != nil {
			t.OnSwipeRight()
		}
	}
	t.accumX = 0
}

// GetTitle creates the top title bar showing the app title.
func GetTitle() *fyne.Container {
	appTitleText := canvas.NewText("📖 题库练习系统", core.HexColor(core.TextPrimaryColor))
	appTitleText.TextSize = core.FontSizeHeading // Using constant instead of hardcoded 22
	appTitleText.Alignment = fyne.TextAlignCenter

	return container.NewPadded(container.NewVBox(layout.NewSpacer(), appTitleText, layout.NewSpacer()))
}

// DynamicFixedSizeLayout 根据输入框文本行数动态调整高度的布局结构
type DynamicFixedSizeLayout struct {
	Entry        *widget.Entry
	LastWidth    float32 // 用于记录容器的实际分配宽度
	InitialLines int     // 初始行数，默认为1
}

func (l *DynamicFixedSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	sampleText := canvas.NewText("测", theme.ForegroundColor())
	realLineHeight := sampleText.MinSize().Height
	paddingHeight := float32(16.0)

	text := l.Entry.Text
	if text == "" {
		initialLines := l.InitialLines
		if initialLines < 1 {
			initialLines = 1
		}
		return fyne.NewSize(l.Entry.MinSize().Width, float32(initialLines)*realLineHeight+paddingHeight)
	}

	// 👈 核心修正：将 Padding 从 16.0 改为 22.0 (或 24.0)
	// 290 - 22 = 268px，使 usableWidth 与 Entry 内部真正的文本绘制区域大小完全一致
	usableWidth := l.LastWidth - 22.0

	if usableWidth <= 0 {
		var maxChildWidth float32
		for _, o := range objects {
			if ms := o.MinSize().Width; ms > maxChildWidth {
				maxChildWidth = ms
			}
		}
		usableWidth = maxChildWidth - 22.0
		if usableWidth <= 0 {
			usableWidth = 300.0
		}
	}

	lines := strings.Split(text, "\n")
	totalLines := 0

	textSize := theme.TextSize()
	textStyle := l.Entry.TextStyle

	for _, line := range lines {
		if line == "" {
			totalLines += 1
			continue
		}

		measuredSize := fyne.MeasureText(line, textSize, textStyle)
		lineWidth := measuredSize.Width

		ratio := float64(lineWidth / usableWidth)
		lineCount := int(math.Ceil(ratio))
		if lineCount < 1 {
			lineCount = 1
		}
		totalLines += lineCount

		//fmt.Printf("[DEBUG] 段落:%d | usableWidth:%.2f | 精确lineWidth:%.2f | ratio:%.4f | 行数:%d\n",
		//	i, usableWidth, lineWidth, ratio, lineCount)
	}

	calculatedHeight := float32(totalLines)*realLineHeight + paddingHeight

	// 确保高度不小于初始行数对应的高度
	initialHeight := float32(l.InitialLines)*realLineHeight + paddingHeight
	if l.InitialLines < 1 {
		initialHeight = realLineHeight + paddingHeight
	}
	if calculatedHeight < initialHeight {
		calculatedHeight = initialHeight
	}

	return fyne.NewSize(l.Entry.MinSize().Width, calculatedHeight)
}

func (l *DynamicFixedSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// 定义外层滚动条预留的安全宽度（可根据实际视觉效果微调，通常 10~12 px 足够）
	scrollbarWidth := float32(4.0)

	// 计算输入框的实际安全宽度
	safeWidth := size.Width - scrollbarWidth
	if safeWidth < 0 {
		safeWidth = 0
	}

	// 👈 核心：记录减去滚动条之后的真实宽度，
	// 让 MinSize 里的 usableWidth 计算能与实际排版严格保持一致
	l.LastWidth = safeWidth

	for _, o := range objects {
		// 调整输入框的大小，高度保持分配的高度，宽度使用预留后的安全宽度
		o.Resize(fyne.NewSize(safeWidth, size.Height))

		// 依然放置在左上角 (0,0)，这样右侧就会自然空出 scrollbarWidth 的距离
		o.Move(fyne.NewPos(0, 0))
	}
}
