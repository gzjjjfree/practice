package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

type boxColor struct {
	textColor   string
	bgColor     string
	strokeColor string
	textSize    float32
}

func getDefaultColor() *boxColor {
	return &boxColor{textColor: "#333333", bgColor: "#f0f0f0", strokeColor: "#d9d9d9", textSize: 18}
}

func getBox(btnText string, width float32, hight float32, bxColor *boxColor, isDisabled bool, backfun func()) *ClickableBox {

	// ✨ 状态拦截：如果处于禁用状态，切换为置灰配色
	if isDisabled {
		bxColor.textColor = "#bbbbbb" // 文字变浅灰
		bxColor.bgColor = "#f8f8f8"   // 背景变淡
		bxColor.strokeColor = "#eeeeee"
	}

	backBtnText := canvas.NewText(btnText, hexColor(bxColor.textColor))
	backBtnText.TextSize = bxColor.textSize
	backBtnText.TextStyle = fyne.TextStyle{Bold: true}
	//backBtnText.Alignment = fyne.TextAlignCenter

	backBtnBg := canvas.NewRectangle(hexColor(bxColor.bgColor))
	backBtnBg.CornerRadius = 4
	backBtnBg.StrokeColor = hexColor(bxColor.strokeColor)
	backBtnBg.StrokeWidth = 1

	// ✨ 修正点：将“撑开尺寸的盒子”与“文字”解耦
	// sizeBox 是一个隐形的骨架，仅用于在没有网格约束时保底撑起 width 和 hight
	sizeBox := container.NewGridWrap(fyne.NewSize(width, hight))

	backBtnCustomLayout := container.NewStack(
		backBtnBg,                        // 1. 底层背景（会被网格自动完美拉伸）
		sizeBox,                          // 2. 隐形尺寸支撑架
		container.NewCenter(backBtnText), // 3. 顶层文字（摆脱尺寸束缚，永远在整个大 Stack 的正中心！）
	)

	// ✨ 逻辑拦截：如果被禁用，直接传入空函数，点击时将不会发生任何事
	finalFunc := backfun
	if isDisabled {
		finalFunc = func() {}
	}

	return NewClickableBox(backBtnCustomLayout, finalFunc)
}

func getTitle() *fyne.Container {
	// ✨ 修复问题 1：在移动端显式绘制一个 App 顶部大标题导航栏
	appTitleText := canvas.NewText("📖 题库练习系统", hexColor("#000000"))
	appTitleText.TextSize = 22
	appTitleText.Alignment = fyne.TextAlignCenter

	// 给标题增加一些上下边距，让它看起来更美观
	return container.NewPadded(container.NewVBox(layout.NewSpacer(), appTitleText, layout.NewSpacer()))
}
