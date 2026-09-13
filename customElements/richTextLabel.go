package customElements

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// CustomLabel 包装底层 canvas.Text 的自定义控件
type CustomLabel struct {
	widget.BaseWidget
	Text     string
	Color    color.Color
	TextSize float32
	IsBold   bool
	IsCenter bool
	WordWrap bool
}

// CreateLabel 自定义文本元素
// 参数：文本、颜色、字体大小、是否加粗、是否居中、是否换行
func CreateLabel(text string, textColor color.Color, textSize float32, isBold bool, isCenter bool, wordWrap bool) *CustomLabel {
	l := &CustomLabel{
		Text:     text,
		Color:    textColor,
		TextSize: textSize,
		IsBold:   isBold,
		IsCenter: isCenter,
		WordWrap: wordWrap,
	}
	l.ExtendBaseWidget(l)
	return l
}

func (l *CustomLabel) CreateRenderer() fyne.WidgetRenderer {
	t := canvas.NewText(l.Text, l.Color)
	t.TextSize = l.TextSize
	t.TextStyle = fyne.TextStyle{Bold: l.IsBold}

	if l.IsCenter {
		t.Alignment = fyne.TextAlignCenter
	}

	return &customLabelRenderer{
		label: l,
		text:  t,
	}
}

type customLabelRenderer struct {
	label *CustomLabel
	text  *canvas.Text
}

func (r *customLabelRenderer) Layout(size fyne.Size) {
	r.text.Resize(size)
}

func (r *customLabelRenderer) MinSize() fyne.Size {
	r.text.TextSize = r.label.TextSize
	return r.text.MinSize()
}

func (r *customLabelRenderer) Refresh() {
	r.text.Text = r.label.Text
	r.text.Color = r.label.Color
	r.text.TextSize = r.label.TextSize
	r.text.TextStyle = fyne.TextStyle{Bold: r.label.IsBold}

	if r.label.IsCenter {
		r.text.Alignment = fyne.TextAlignCenter
	} else {
		r.text.Alignment = fyne.TextAlignLeading
	}
	canvas.Refresh(r.text)
}

func (r *customLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.text}
}

func (r *customLabelRenderer) Destroy() {}
