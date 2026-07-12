package main

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TouchInterceptor 作为触摸探测器，包裹在答题内容外面，放在滚动条里面
type TouchInterceptor struct {
	widget.BaseWidget
	Content      fyne.CanvasObject
	ParentScroll *container.Scroll // 💡 核心：保存对外层滚动条的引用
	OnSwipeLeft  func()
	OnSwipeRight func()

	accumX float32
}

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

func (t *TouchInterceptor) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.Content)
}

// ==================== 核心：捕获手机端真实的手指划动 (Drag) ====================

func (t *TouchInterceptor) Dragged(e *fyne.DragEvent) {
	// 判断主滑动方向：是横向(翻页) 还是 纵向(看长题)
	if math.Abs(float64(e.Dragged.DX)) > math.Abs(float64(e.Dragged.DY)) {
		// 1. 横向：累加位移，准备翻页
		t.accumX += e.Dragged.DX
	} else {
		// 2. 纵向：清空横向累加，并将手指的位移直接“投喂”给外层滚动条！
		t.accumX = 0
		if t.ParentScroll != nil {
			// 将 DragEvent 的位移完美转化为 ScrollEvent，让原生滚动条动起来
			t.ParentScroll.Scrolled(&fyne.ScrollEvent{
				Scrolled: e.Dragged,
			})
		}
	}
}

func (t *TouchInterceptor) DragEnd() {
	const threshold = float32(40.0)

	if t.accumX < -threshold {
		if t.OnSwipeLeft != nil {
			t.OnSwipeLeft()
		}
	} else if t.accumX > threshold {
		if t.OnSwipeRight != nil {
			t.OnSwipeRight()
		}
	}
	t.accumX = 0 // 判定结束后清零
}
