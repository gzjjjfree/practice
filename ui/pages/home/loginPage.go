package home

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
)

// ShowLoginPage 渲染用户登录页面。
func ShowLoginPage(w fyne.Window, state *core.AppState, backToHome func()) {
	// Page background
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// Card background
	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.LargeCardCorner
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	// 密码框与登录按钮之间增加占位隔离 (24px)
	btnGap := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	btnGap.SetMinSize(fyne.NewSize(1, 24))

	// Title with icon
	titleText := customElements.CreateLabel("🔐 用户登录", core.HexColor(core.TextPrimaryColor), core.FontSizeTitle, true, true, false)
	subtitleText := customElements.CreateLabel("请登录以使用全部功能", core.HexColor(core.TextSecondaryColor), core.FontSizeSubtitle, false, true, false)

	// Username input group
	// 用户名标签：左侧增加内边距（例如 10 像素）
	usernameLabel := customElements.CreateLabel("用户名", core.HexColor(core.TextSecondaryColor), core.FontSizeSmall, false, false, false)
	paddedUsernameLabel := container.New(
		layout.NewCustomPaddedLayout(0, 0, 10, 0), // 参数依次为：上, 下, 左, 右
		usernameLabel,
	)

	usernameInput := widget.NewEntry()
	usernameInput.SetPlaceHolder("请输入用户名")

	// 用户名输入框：右侧增加内边距（例如 10 像素）
	paddedUsernameInput := container.New(
		layout.NewCustomPaddedLayout(0, 0, 0, 10), // 右侧设为 10 像素
		usernameInput,
	)

	// Password input group
	passwordLabel := customElements.CreateLabel("密码", core.HexColor(core.TextSecondaryColor), core.FontSizeSmall, false, false, false)
	paddedPasswordLabel := container.New(
		layout.NewCustomPaddedLayout(0, 0, 10, 0), // 参数依次为：上, 下, 左, 右
		passwordLabel,
	)

	passwordInput := widget.NewPasswordEntry()
	passwordInput.SetPlaceHolder("请输入密码")

	paddedPasswordInput := container.New(
		layout.NewCustomPaddedLayout(0, 0, 0, 10), // 右侧设为 10 像素
		passwordInput,
	)

	// Unified login action to satisfy DRY principle
	onLoginSubmit := func() {
		handleLogin(
			state,
			usernameInput,
			passwordInput,
			func() {},
			func(msg string) { customElements.ShowCustomInformation("登录失败", msg, w) },
			backToHome,
			w,
		)
	}

	// Pressing Enter on Username shifts focus to Password; Pressing Enter on Password triggers Login
	usernameInput.OnSubmitted = func(_ string) {
		if canvasRef := w.Canvas(); canvasRef != nil {
			canvasRef.Focus(passwordInput)
		}
	}
	passwordInput.OnSubmitted = func(_ string) {
		onLoginSubmit()
	}

	// 登录按钮
	loginBtn := customElements.CreateButton(
		"登    录",
		core.DefaultBtnWidth,
		core.DefaultBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium,
		core.FontSizeHeading,
		true,
		false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		onLoginSubmit,
	)
	fmt.Printf("DEBUG ShowLoginPage: loginBtn created, type=%T, disabled=%v\n", loginBtn, loginBtn.Disabled())

	// 1. 在 FormLayout 中加入透明占位块，拉开“用户名”与“密码”的上下行距 (18px)
	spacerLeft := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	spacerLeft.SetMinSize(fyne.NewSize(1, 18))
	spacerRight := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	spacerRight.SetMinSize(fyne.NewSize(1, 18))

	formInputs := container.New(layout.NewFormLayout(),
		paddedUsernameLabel, paddedUsernameInput,
		spacerLeft, spacerRight, // 👈 占位拉开两行输入框间距
		paddedPasswordLabel, paddedPasswordInput,
	)

	btnContainer := container.NewCenter(
		container.NewGridWrap(fyne.NewSize(core.DefaultBtnWidth, core.DefaultBtnHeight), loginBtn),
	)

	formVBox := container.NewVBox(
		formInputs,
		btnGap,
		btnContainer,
		btnGap,
	)

	// 标题与副标题之间间隔 (8px)
	titleGap := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	titleGap.SetMinSize(fyne.NewSize(1, 8))

	headerVBox := container.NewVBox(
		btnGap,
		titleText,
		titleGap,
		subtitleText,
	)

	// 头部与表单部分之间间隔 (20px)
	headerFormGap := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	headerFormGap.SetMinSize(fyne.NewSize(1, 20))

	cardInner := container.NewVBox(
		headerVBox,
		headerFormGap,
		formVBox,
	)

	// Card content — 内边距包裹
	cardContent := container.NewPadded(cardInner)
	cardStack := container.NewStack(cardBg, cardContent)

	// 固定卡片宽度为 380px
	sizedCard := container.NewGridWrap(fyne.NewSize(380, cardStack.MinSize().Height), cardStack)

	// =================【重点：使用 1:2 空间比例锁定 1/3 位置】=================
	topOneThirdContainer := container.New(
		layout.NewVBoxLayout(),
		layout.NewSpacer(),             // 上方占 1 份空间
		container.NewCenter(sizedCard), // 登录卡片（置于上方 1/3 处）
		layout.NewSpacer(),             // 下方占 2 份空间（2 个 Spacer 叠加）
		layout.NewSpacer(),
	)

	title := customElements.CreateLabel("📖 题库练习系统", core.HexColor(core.TextPrimaryColor), core.FontSizeHeading, true, true, false)

	// Main layout
	mainContent := container.NewBorder(
		title,
		nil,
		nil,
		nil,
		topOneThirdContainer, // 👈 插入 1/3 位置容器
	)

	mainLayout := container.NewStack(pageBg, mainContent)

	w.SetContent(mainLayout)
}

// handleLogin 执行用户登录操作。
func handleLogin(state *core.AppState, usernameInput *widget.Entry, passwordInput *widget.Entry, onSuccess func(), onFailed func(string), backToHome func(), w fyne.Window) {
	username := strings.TrimSpace(usernameInput.Text)
	password := passwordInput.Text

	if username == "" || password == "" {
		customElements.ShowCustomInformation("提示", "请输入用户名和密码", w)
		return
	}

	state.Login(username, password, func(success bool, msg string) {
		if success {
			onSuccess()
			backToHome()
		} else {
			onFailed(msg)
		}
	})
}
