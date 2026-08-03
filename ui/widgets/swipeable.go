package widgets

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
)

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
