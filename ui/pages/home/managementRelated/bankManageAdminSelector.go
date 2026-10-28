package managementRelated

import (
	"fmt"
	"strings"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)


// showSuperAdminViewSelector 为超级管理员显示管理员选择器对话框。
//
// 功能说明：
// 允许超级管理员选择要查看的管理员或普通用户的题库列表。如果 skipSelector 为 true，则直接查看当前用户的题库。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - skipSelector: bool 类型，是否跳过管理员选择步骤，直接查看当前用户的题库。
//   - onSelected: func() 类型，选择完成后的回调函数。
//
// 核心业务逻辑与按钮说明：
//   - 当 skipSelector 为 true 时，直接获取并显示当前用户的题库列表。
//   - 当 skipSelector 为 false 时，从服务器获取所有管理员/超级管理员列表，生成单选框供用户选择目标所有者，然后获取并显示该所有者的题库列表。
//   - "确认"按钮 (core.BankManageConfirmBtnText)：根据选择的管理员获取并显示其题库列表。
//   - "取消"按钮 (core.BankManageCancelBtnText)：关闭管理员选择对话框。
func showSuperAdminViewSelector(w fyne.Window, state *core.AppState, skipSelector bool, onSelected func()) {
	if skipSelector {
		state.FetchBanksByOwner(state.Username, func(success bool, banks []network.BankItem, total int, msg string) {
			if !success {
				customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
				return
			}
			if len(banks) == 0 {
				customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManageNoBanksMsg, w)
				return
			}
			showBankListForOwner(w, state, state.Username, banks, onSelected)
		})
		return
	}

	state.FetchAdmins(100, 0, func(success bool, admins []network.UserItem, total int, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		var allAdmins []network.UserItem
		hasSuperAdmin := false
		for _, admin := range admins {
			if admin.Username == state.Username && admin.Role == "superadmin" {
				hasSuperAdmin = true
				break
			}
		}
		if !hasSuperAdmin {
			allAdmins = append(allAdmins, network.UserItem{
				UserID:   1,
				Username: state.Username,
				Role:     "superadmin",
			})
		}
		allAdmins = append(allAdmins, admins...)

		var radioOptions []string
		for _, admin := range allAdmins {
			radioOptions = append(radioOptions, core.BankManageCheckboxPrefixUser+admin.Username)
		}

		var selectedOwner string
		radioGroup := widget.NewRadioGroup(radioOptions, func(s string) {
			if idx := strings.Index(s, core.BankManageCheckboxPrefixUser); idx != -1 {
				selectedOwner = s[idx+len(core.BankManageCheckboxPrefixUser):]
			}
		})
		radioGroup.Horizontal = false

		var adminDialogDial dialog.Dialog

		// 确认按钮 (core.BankManageConfirmBtnText)：根据选择的管理员获取并显示其题库列表。
		confirmBtn := customElements.CreateButton(
			core.BankManageConfirmBtnText, // 按钮标识符：BankManageConfirmBtnText，按钮文本："确认"，功能与业务用途：提交用户选择的管理员/超级管理员，获取并展示其题库列表。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium,
			core.FontSizeButton,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if selectedOwner == "" {
					customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManagePleaseSelectAdminMsg, w)
					return
				}

				state.FetchBanksByOwner(selectedOwner, func(success bool, banks []network.BankItem, total int, msg string) {
					if !success {
						customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
						return
					}
					if len(banks) == 0 {
						customElements.ShowCustomInformation(core.BankManageInfoMsgType, fmt.Sprintf("%s %s", selectedOwner, core.BankManageNoBanksMsg), w)
						return
					}
					if adminDialogDial != nil {
						adminDialogDial.Hide()
					}
					showBankListForOwner(w, state, selectedOwner, banks, onSelected)
				})
			},
		)

		// 取消按钮 (core.BankManageCancelBtnText)：关闭管理员选择对话框，不执行任何查看操作。
		cancelBtn := customElements.CreateButton(
			core.BankManageCancelBtnText, // 按钮标识符：BankManageCancelBtnText，按钮文本："取消"，功能与业务用途：取消当前选择流程并关闭对话框。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
			core.HexColor(core.TextBodyColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BorderLightColor),
			core.StrokeMedium,
			core.FontSizeButton,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if adminDialogDial != nil {
					adminDialogDial.Hide()
				}
			},
		)

		btnRow := container.NewHBox(
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
			cancelBtn,
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
			confirmBtn,
		)

		topArea := container.NewVBox(
			widget.NewLabel(core.BankManageSelectAdminMsg),
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)),
		)
		bottomArea := container.NewVBox(
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)),
			container.NewCenter(btnRow),
		)
		scrollArea := container.NewVScroll(container.NewVBox(radioGroup))
		minSizeSpacer := container.NewGridWrap(fyne.NewSize(core.LocalBankDialogWidth, core.LocalBankDialogHeight))
		centerArea := container.NewStack(minSizeSpacer, scrollArea)

		content := container.NewBorder(topArea, bottomArea, nil, nil, centerArea)
		adminDialogDial = dialog.NewCustomWithoutButtons(core.BankManageSelectAdminDialogTitle, container.NewPadded(content), w)
		adminDialogDial.Show()
	})
}

// ==================== 管理员视图 ====================

// buildAdminView 为管理员创建包含上传和查看按钮的视图。
//
// 功能说明：
// 构建管理员专用的题库管理界面，提供"上传题库"和"查看题库"两个主要操作按钮。与超级管理员视图类似，但管理员只能查看自己创建的题库。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - backToHome: func() 类型，返回主页面的回调函数。
//
// 返回值：
//   - *fyne.Container：包含上传和查看按钮的垂直布局容器。
//
// 核心业务逻辑与按钮说明：
//   - "上传题库"按钮 (core.BankManageUploadBtnText)：点击后打开管理员上传题库的对话框，选择本地题库文件并上传到服务器。
//   - "查看题库"按钮 (core.BankManageViewBtnText)：点击后打开查看自己题库列表的界面。
func buildAdminView(w fyne.Window, state *core.AppState, backToHome func()) *fyne.Container {
	// 操作行，包含上传和查看按钮
	actionRow := container.NewHBox(
		// 上传题库按钮 (core.BankManageUploadBtnText)：点击后打开管理员上传题库的对话框，选择本地题库文件并上传到服务器。
		customElements.CreateButton(
			core.BankManageUploadBtnText, // 按钮标识符：BankManageUploadBtnText，按钮文本："上传题库"，功能与业务用途：提交用户选择的本地题库文件至服务器进行导入。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnPrimaryBg),
			core.HexColor(core.BtnPrimaryBg),
			core.StrokeMedium,
			core.FontSizeButton,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showAdminUpload(w, state, func() {
					// 上传完成后刷新页面
					ShowBankManage(w, state, backToHome)
				})
			},
		),
		// 查看题库按钮 (core.BankManageViewBtnText)：点击后打开查看自己题库列表的界面。
		customElements.CreateButton(
			core.BankManageViewBtnText, // 按钮标识符：BankManageViewBtnText，按钮文本："查看题库"，功能与业务用途：获取并展示当前管理员自己的题库列表。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			core.StrokeMedium,
			core.FontSizeButton,
			false, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showAdminViewBanks(w, state, func() {
					// 查看完成后刷新页面
					ShowBankManage(w, state, backToHome)
				})
			},
		),
	)
	return container.NewVBox(container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)), actionRow)
}

// showAdminUpload 为管理员显示上传题库对话框。
//
// 功能说明：
// 调用超级管理员的上传题库函数，因为管理员和超级管理员使用相同的上传逻辑。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onUploaded: func() 类型，上传完成后的回调函数，用于刷新页面。
//
// 核心业务逻辑：
//
//	直接复用 showSuperAdminUpload 函数，执行本地题库文件选择、解析与服务器导入流程。
func showAdminUpload(w fyne.Window, state *core.AppState, onUploaded func()) {
	showSuperAdminUpload(w, state, onUploaded)
}

// showAdminViewBanks 为管理员显示查看题库列表界面。
//
// 功能说明：
// 调用超级管理员的查看题库选择器函数，但跳过管理员选择步骤（skipSelector=true），直接查看当前管理员自己的题库。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onBack: func() 类型，返回后的回调函数。
//
// 核心业务逻辑：
//
//	直接复用 showSuperAdminViewSelector 函数，并传入 skipSelector=true，获取并展示当前管理员自己的题库列表。
func showAdminViewBanks(w fyne.Window, state *core.AppState, onBack func()) {
	showSuperAdminViewSelector(w, state, true, onBack)
}

// ==================== 普通用户视图 ====================

// buildUserView 为普通用户创建题库列表视图。
//
// 功能说明：
// 构建普通用户专用的题库管理界面，只显示用户自己的题库列表，不提供上传和查看其他用户题库的功能。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - backToHome: func() 类型，返回主页面的回调函数。
//
// 返回值：
//   - fyne.CanvasObject：包含用户题库列表的滚动容器。
//
// 核心业务逻辑：
//
//	从服务器获取当前用户的题库列表（通过 state.FetchMyBanks），并为每个题库生成卡片视图（buildMyBankCard），最后返回滚动容器。
func buildUserView(w fyne.Window, state *core.AppState, backToHome func()) fyne.CanvasObject {
	bankListVBox := container.NewVBox()

	loadingText := canvas.NewText(core.BankManageLoadingTextMsg, core.HexColor(core.TextMutedColor))
	loadingText.TextSize = core.FontSizeBody
	loadingText.Alignment = fyne.TextAlignCenter
	bankListVBox.Add(container.NewPadded(loadingText))

	state.FetchMyBanks(func(success bool, banks []network.MyBankItem, msg string) {
		bankListVBox.Objects = nil

		if !success {
			errText := canvas.NewText(core.BankManageErrorTextPrefixMsg+msg, core.HexColor(core.ColorWrongBorder))
			errText.TextSize = core.FontSizeBody
			errText.Alignment = fyne.TextAlignCenter
			bankListVBox.Add(container.NewPadded(errText))
			bankListVBox.Refresh()
			return
		}

		if len(banks) == 0 {
			noDataText := canvas.NewText(core.BankManageNoMyBanksMsgText, core.HexColor(core.TextMutedColor))
			noDataText.TextSize = core.FontSizeBody
			noDataText.Alignment = fyne.TextAlignCenter
			bankListVBox.Add(container.NewPadded(noDataText))
		} else {
			for _, bank := range banks {
				bankItem := bank
				bankListVBox.Add(buildMyBankCard(w, &bankItem, state, func() {
					ShowBankManage(w, state, backToHome)
				}))
				bankListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH8)))
			}
		}
		bankListVBox.Refresh()
	})

	return container.NewVScroll(bankListVBox)
}

// ==================== 题库列表显示 ====================

// showBankListForOwner 显示指定所有者的题库列表。
//
// 功能说明：
// 为指定的管理员或超级管理员显示其所有的题库卡片列表，包含返回按钮和标题栏。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - owner: string 类型，所有者的用户名。
//   - banks: []network.BankItem 类型，题库列表。
//   - onBack: func() 类型，返回后的回调函数。
//
// 核心业务逻辑与按钮说明：
//   - 遍历 banks 列表，为每个题库生成卡片视图（buildBankCardForOwner）并添加到垂直布局中。
//   - "返回"按钮 (core.BankManageBackBtnText)：从题库列表页面返回到题库管理主页面。
func showBankListForOwner(w fyne.Window, state *core.AppState, owner string, banks []network.BankItem, onBack func()) {
	bankListVBox := container.NewVBox()
	cardsSpacer := container.NewGridWrap(fyne.NewSize(core.LocalBankDialogWidth, 1))
	cardsContent := container.NewStack(cardsSpacer, bankListVBox)

	for _, bank := range banks {
		bankItem := bank
		bankListVBox.Add(buildBankCardForOwner(w, &bankItem, state, owner, onBack))
		bankListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH8)))
	}

	if len(banks) == 0 {
		noDataText := canvas.NewText(core.BankManageNoBanksMsg, core.HexColor(core.TextMutedColor))
		noDataText.TextSize = core.FontSizeBody
		noDataText.Alignment = fyne.TextAlignCenter
		bankListVBox.Add(container.NewPadded(noDataText))
	}

	backCallback := onBack
	if backCallback == nil {
		backCallback = func() { ShowBankManage(w, state, func() {}) }
	}

	// 返回按钮 (core.BankManageBackBtnText)：从题库列表页面返回到题库管理主页面。
	backBtn := customElements.CreateButton(
		core.BankManageBackBtnText, // 按钮标识符：BankManageBackBtnText，按钮文本："返回"，功能与业务用途：取消当前查看流程并返回上一级题库管理主页面。
		core.BackBtnWidth,
		core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		core.StrokeMedium,
		core.AdminHeaderFontSize,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		backCallback,
	)
	titleCenter := customElements.CreateLabel(fmt.Sprintf(core.BankManageTitleBankOfUserFormat, owner), core.HexColor(core.TextPrimaryColor), core.FontSizePageTitle, true, false, false)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	wRootLayout := container.NewStack(pageBg, container.NewBorder(topBar, nil, nil, nil, container.NewVScroll(cardsContent)))
	w.SetContent(wRootLayout)
}

// refreshBankListForOwner 刷新指定所有者的题库列表。
//
// 功能说明：
// 从服务器重新获取指定所有者的题库列表，并显示更新后的列表界面。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - owner: string 类型，所有者的用户名。
//   - onBack: func() 类型，返回后的回调函数。
//
// 核心业务逻辑：
//
//	调用 state.FetchBanksByOwner 从服务器获取最新题库列表，成功后调用 showBankListForOwner 重新渲染页面。
func refreshBankListForOwner(w fyne.Window, state *core.AppState, owner string, onBack func()) {
	state.FetchBanksByOwner(owner, func(success bool, banks []network.BankItem, total int, msg string) {
		if !success {
			customElements.ShowCustomInformation("错误", msg, w)
			return
		}
		showBankListForOwner(w, state, owner, banks, onBack)
	})
}

// buildMyBankCard 为 MyBankItem 创建题库卡片。
//
// 功能说明：
// 为普通用户自己的题库创建卡片界面，包含题库名称、归属用户、状态信息，以及下载按钮和推送按钮（针对管理员和超级管理员）。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - bank: *network.MyBankItem 类型，用户的题库项。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 返回值：
//   - fyne.CanvasObject：题库卡片容器对象。
//
// 核心业务逻辑与按钮说明：
//   - "下载题库"按钮 (core.BankManageDownloadBtnText)：点击后弹出确认对话框，确认后从服务器下载该题库并保存到本地。
//   - "推送题库"按钮 (core.BankManagePushBtnText)：仅对管理员和超级管理员显示，点击后打开推送题库对话框，允许将该题库推送到其他管理员或用户。
func buildMyBankCard(w fyne.Window, bank *network.MyBankItem, state *core.AppState, onBack func()) fyne.CanvasObject {
	bg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	bg.CornerRadius = core.CardCornerRadius
	bg.StrokeColor = core.HexColor(core.BorderLightColor)
	bg.StrokeWidth = core.StrokeThin

	nameText := canvas.NewText("📚 "+bank.BankName, core.HexColor(core.TextPrimaryColor))
	nameText.TextSize = core.FontSizeHeading
	nameText.TextStyle = fyne.TextStyle{Bold: true}

	infoText := canvas.NewText(fmt.Sprintf(core.BankManageInfoOwnerStatusFormat, bank.OwnerUsername, bank.Status), core.HexColor(core.TextSecondaryColor))
	infoText.TextSize = core.FontSizeSubtitle

	// 下载题库按钮 (core.BankManageDownloadBtnText)：点击后弹出确认对话框，确认后从服务器下载该题库并保存到本地。
	downloadBtn := customElements.CreateButton(
		core.BankManageDownloadBtnText, // 按钮标识符：BankManageDownloadBtnText，按钮文本："下载题库"，功能与业务用途：从服务器下载指定题库并保存至本地存储目录。
		core.BankManageCardBtnWidth,
		core.NavButtonHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium,
		core.FontSizeButton,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			downloadMyBank(w, state, bank, onBack)
		},
	)

	var btnRow fyne.CanvasObject
	if state.Role == "superadmin" || state.Role == "admin" {
		// 推送题库按钮 (core.BankManagePushBtnText)：点击后打开推送题库对话框，允许将该题库推送到其他管理员或用户。
		pushBtn := customElements.CreateButton(
			core.BankManagePushBtnText, // 按钮标识符：BankManagePushBtnText，按钮文本："推送题库"，功能与业务用途：将当前题库推送给选中的目标管理员或用户。
			core.BankManageCardBtnWidth,
			core.NavButtonHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.ColorPushBtnBg),
			core.HexColor(core.ColorPushBtnBg),
			core.StrokeMedium,
			core.FontSizeButton,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				showPushBankDialog(w, state, bank.BankName, bank.OwnerUsername, func() {
					ShowBankManage(w, state, onBack)
				})
			},
		)
		btnRow = container.NewHBox(downloadBtn, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)), pushBtn)
	} else {
		btnRow = container.NewHBox(downloadBtn)
	}

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, nil, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
		btnRow,
	)
	return container.NewStack(bg, container.NewPadded(cardContent))
}
