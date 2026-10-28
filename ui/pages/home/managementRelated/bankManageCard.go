package managementRelated

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/storageRelated"
)


// buildBankCardForOwner 为所有者视图中的 BankItem 创建题库卡片。
//
// 功能说明：
// 为管理员或超级管理员查看其他用户/自己的题库时创建卡片界面，包含题库显示名称、创建者、创建时间信息，以及下载、推送（针对管理员和超级管理员）和删除按钮。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - bank: *network.BankItem 类型，题库项。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - owner: string 类型，所有者的用户名。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 返回值：
//   - fyne.CanvasObject：题库卡片容器对象。
//
// 核心业务逻辑与按钮说明：
//   - "下载题库"按钮 (core.BankManageDownloadBtnText)：点击后弹出确认对话框，确认后从服务器下载该题库并保存到本地。
//   - "删除题库"按钮 (core.BankManageDeleteBtnText)：点击后弹出确认对话框，确认后从服务器删除该题库并刷新列表。
//   - "推送题库"按钮 (core.BankManagePushBtnText)：仅对管理员和超级管理员显示，点击后打开推送题库对话框，允许将该题库推送到其他管理员或用户。
func buildBankCardForOwner(w fyne.Window, bank *network.BankItem, state *core.AppState, owner string, onBack func()) fyne.CanvasObject {
	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.CardCornerRadius
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	displayName := bank.DisplayName
	if displayName == "" {
		displayName = bank.BankName
	}
	titleText := canvas.NewText("📚 "+displayName, core.HexColor(core.TextBodyColor))
	titleText.TextSize = core.FontSizeBody
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	displayTime := bank.CreatedAt
	if t, err := parseTime(bank.CreatedAt); err == nil {
		displayTime = t.Format("2006-01-02 15:04:05")
	}

	creator := bank.CreatedByUsername
	if creator == "" {
		creator = fmt.Sprintf("用户ID %d", bank.CreatedBy)
	}
	infoText := canvas.NewText(fmt.Sprintf(core.BankManageInfoCreatorTimeFormat, creator, displayTime), core.HexColor(core.TextSecondaryColor))
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
			downloadBankForOwner(w, state, bank, onBack)
		},
	)

	// 删除题库按钮 (core.BankManageDeleteBtnText)：点击后弹出确认对话框，确认后从服务器删除该题库并刷新列表。
	deleteBtn := customElements.CreateButton(
		core.BankManageDeleteBtnText, // 按钮标识符：BankManageDeleteBtnText，按钮文本："删除题库"，功能与业务用途：向服务器发起删除请求，成功后刷新当前所有者的题库列表。
		core.BankManageCardBtnWidth,
		core.NavButtonHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.ColorDeleteBtnBg),
		core.HexColor(core.ColorDeleteBtnBg),
		core.StrokeMedium,
		core.FontSizeButton,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			customElements.ShowCustomConfirm(
				core.BankManageConfirmDeleteMsg,
				"确定",
				"取消",
				customElements.NewCenterRichText(fmt.Sprintf(core.BankManageConfirmDeleteConfirmMsg, displayName)),
				func(confirm bool) bool {
					if confirm {
						state.DeleteBank(displayName, owner, func(success bool, msg string) {
							if !success {
								customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
								return
							}
							customElements.ShowCustomInformation(core.BankManageSuccessMsgType, core.BankManageDeleteSuccessMsg, w)
							refreshBankListForOwner(w, state, owner, onBack)
						})
					}
					return true
				},
				w,
			)
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
				showPushBankDialog(w, state, displayName, owner, func() {
					refreshBankListForOwner(w, state, owner, onBack)
				})
			},
		)
		btnRow = container.NewHBox(
			downloadBtn, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)),
			pushBtn, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)), deleteBtn,
		)
	} else {
		btnRow = container.NewHBox(downloadBtn, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)), deleteBtn)
	}

	cardContent := container.NewVBox(
		container.NewBorder(titleText, nil, nil, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
		btnRow,
	)
	return container.NewStack(cardBg, container.NewPadded(cardContent))
}

// parseTime 解析时间字符串。
//
// 功能说明：
// 尝试使用多种时间格式解析给定的时间字符串，包括 RFC3339、"2006-01-02T15:04:05" 和 "2006-01-02 15:04:05"。
//
// 参数说明：
//   - s: string 类型，要解析的时间字符串。
//
// 返回值：
//   - time.Time: 解析后的时间对象。
//   - error: 如果无法解析任何格式，则返回错误。
//
// 核心业务逻辑：
//
//	按顺序尝试使用 time.RFC3339、"2006-01-02T15:04:05"、"2006-01-02 15:04:05" 三种格式进行解析，成功则返回对应的时间对象，全部失败则返回错误。
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

// ==================== 推送题库对话框 ====================

// showPushBankDialog 显示推送题库对话框。
//
// 功能说明：
// 允许管理员或超级管理员将指定的题库推送到其他管理员或用户。对话框中包含目标用户列表（按管理员分组），并支持选择是否覆盖目标用户的同名题库。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - displayName: string 类型，要推送的题库显示名称。
//   - ownerUsername: string 类型，题库归属用户的用户名。
//   - onPushed: func() 类型，推送完成后的回调函数。
//
// 核心业务逻辑与按钮说明：
//   - 从服务器获取所有用户列表，按管理员分组生成复选框列表供用户选择目标接收者。
//   - "确认推送"按钮 (core.BankManageConfirmUploadBtnText)：将选中的题库推送到用户选择的管理员或用户，支持覆盖选项。
//   - "取消"按钮 (core.BankManageCancelBtnText)：关闭推送题库对话框。
func showPushBankDialog(w fyne.Window, state *core.AppState, displayName, ownerUsername string, onPushed func()) {
	state.FetchAllUsers(func(success bool, users []network.UserItem, total int, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		var admins []network.UserItem
		for _, user := range users {
			if user.Username != ownerUsername && (user.Role == "admin" || user.Role == "superadmin") {
				admins = append(admins, user)
			}
		}

		adminUsersMap := make(map[string][]network.UserItem)
		for _, user := range users {
			if user.Username == ownerUsername || user.Role == "admin" || user.Role == "superadmin" {
				continue
			}
			adminName := user.CreatedByUsername
			if adminName == "" {
				for _, admin := range admins {
					if admin.UserID == user.CreatedBy {
						adminName = admin.Username
						break
					}
				}
			}
			if adminName != "" {
				adminUsersMap[adminName] = append(adminUsersMap[adminName], user)
			}
		}

		if len(admins) == 0 {
			customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManageNoAdminsToPushMsg, w)
			return
		}

		selectedMap := make(map[string]bool)
		var listItems []fyne.CanvasObject

		for _, admin := range admins {
			// admin := admin
			subUsers := adminUsersMap[admin.Username]

			adminCb := widget.NewCheck("👤 "+admin.Username, func(checked bool) {
				selectedMap[admin.Username] = checked
			})

			subUsersContainer := container.NewVBox()
			subUsersContainer.Hide()

			var toggleBtn *widget.Button
			if len(subUsers) > 0 {
				toggleBtn = widget.NewButton(core.BankManageToggleIconDown, nil)
				toggleBtn.Importance = widget.LowImportance
				toggleBtn.OnTapped = func() {
					if subUsersContainer.Visible() {
						subUsersContainer.Hide()
						toggleBtn.SetText(core.BankManageToggleIconDown)
					} else {
						subUsersContainer.Show()
						toggleBtn.SetText(core.BankManageToggleIconUp)
					}
				}
			} else {
				toggleBtn = widget.NewButton(core.BankManageToggleIconDisabled, nil)
				toggleBtn.Disable()
			}

			for _, user := range subUsers {
				// user := user
				userCb := widget.NewCheck(core.BankManageSubCheckboxPrefixUser+user.Username, func(checked bool) {
					selectedMap[user.Username] = checked
				})
				subUsersContainer.Add(userCb)
			}

			adminRow := container.NewBorder(nil, nil, adminCb, toggleBtn)
			listItems = append(listItems, container.NewVBox(adminRow, subUsersContainer))
		}

		overwriteChecked := false
		overwriteCb := widget.NewCheck(core.BankManageOverwriteTargetBanksMsg, func(checked bool) {
			overwriteChecked = checked
		})

		var dialogDial dialog.Dialog

		// 确认推送按钮 (core.BankManageConfirmUploadBtnText)：将选中的题库推送到用户选择的管理员或用户，支持覆盖选项。
		pushBtn := customElements.CreateButton(
			core.BankManageConfirmUploadBtnText, // 按钮标识符：BankManageConfirmUploadBtnText，按钮文本："确认推送"，功能与业务用途：向服务器发起推送请求，将题库分发给选中的目标用户。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
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
				var selectedUsernames []string
				for uname, isSelected := range selectedMap {
					if isSelected {
						selectedUsernames = append(selectedUsernames, uname)
					}
				}

				if len(selectedUsernames) == 0 {
					customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManagePleaseSelectTargetUsersMsg, w)
					return
				}

				state.PushBank(displayName, ownerUsername, selectedUsernames, overwriteChecked, func(success bool, pushed, updated int, conflict []string, msg string) {
					if !success {
						customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
						return
					}
					msgText := fmt.Sprintf(core.BankManagePushSuccessMsg, pushed, updated)
					if len(conflict) > 0 {
						msgText += fmt.Sprintf(core.BankManagePushConflictMsg, conflict)
					}
					customElements.ShowCustomInformation(core.BankManageSuccessMsgType, msgText, w)
					if dialogDial != nil {
						dialogDial.Hide()
					}
					if onPushed != nil {
						onPushed()
					}
				})
			},
		)

		// 取消按钮 (core.BankManageCancelBtnText)：关闭推送题库对话框，不执行任何推送操作。
		cancelBtn := customElements.CreateButton(
			core.BankManageCancelBtnText, // 按钮标识符：BankManageCancelBtnText，按钮文本："取消"，功能与业务用途：取消当前推送流程并关闭对话框。
			core.ActionBtnWidth,
			core.DefaultBtnHeight,
			core.HexColor(core.TextBodyColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BorderLightColor),
			core.StrokeMedium,
			core.FontSizeButton,
			true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				if dialogDial != nil {
					dialogDial.Hide()
				}
			},
		)

		btnRow := container.NewHBox(
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
			cancelBtn,
			container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
			pushBtn,
		)

		titleText := widget.NewRichTextFromMarkdown(fmt.Sprintf("**推送题库:** %s (归属: %s)", displayName, ownerUsername))
		titleText.Wrapping = fyne.TextWrapWord

		topArea := container.NewVBox(titleText, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)))
		bottomArea := container.NewVBox(container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)), container.NewCenter(btnRow))

		scrollArea := container.NewVScroll(container.NewVBox(listItems...))
		minSizeSpacer := container.NewGridWrap(fyne.NewSize(core.LocalBankDialogWidth, core.LocalBankDialogHeight))
		userListArea := container.NewStack(minSizeSpacer, scrollArea)

		checkboxArea := container.NewVBox(userListArea, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)), overwriteCb)
		content := container.NewBorder(topArea, bottomArea, nil, nil, checkboxArea)

		dialogDial = dialog.NewCustomWithoutButtons(core.BankManagePushBankDialogTitle, container.NewPadded(content), w)
		dialogDial.Show()
	})
}

// executeDownloadAndSave 统一的下载与保存处理逻辑。
//
// 功能说明：
// 从服务器下载题库详情，检查本地是否存在同名题库文件。如果存在，则提示用户选择覆盖或创建新文件；如果不存在，则直接保存并刷新本地文件列表。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - bankID: int 类型，题库的服务器 ID。
//   - displayName: string 类型，题库的显示名称。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 核心业务逻辑与按钮说明（本地文件冲突对话框）：
//   - "取消"按钮 (core.BankManageCancelBtnText)：关闭本地文件已存在确认对话框，不执行保存操作。
//   - "覆盖"按钮 (core.BankManageOverwriteBtnText)：关闭对话框并覆盖本地已存在的同名题库文件。
//   - "新建"按钮 (core.BankManageNewBtnText)：关闭对话框并以带时间戳的新名称保存题库文件，避免覆盖。
func executeDownloadAndSave(w fyne.Window, state *core.AppState, bankID int, displayName string, onBack func()) {
	loadingDial := customElements.ShowCustomLoading("下载题库", "正在下载", w)
	state.DownloadBankDetail(bankID, func(success bool, detail *parser.BankData, msg string) {
		loadingDial.Hide()
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		humanName := detail.DisplayName
		storageDir := storageRelated.GetStorageDir()

		// 使用公共函数查找冲突文件
		oldFileNames := storageRelated.FindExistingBankFiles(storageDir, humanName)

		saveBank := func(choice storageRelated.ConflictResolutionChoice) {
			var newStorageKey string
			fileName := humanName

			if choice == storageRelated.ResolveNewAlias {
				fileName = storageRelated.GenerateUniqueBankFileName(storageDir, humanName)
			} else {
				// 覆盖原有旧文件
				err := storageRelated.OverwriteExistingBankFiles(storageDir, oldFileNames)
				if err != nil {
					customElements.ShowCustomInformation(core.BankManageErrorMsgType, "保存文件失败: "+err.Error(), w)
					return
				}
			}

			timestamp := time.Now().UnixMilli()
			fileExt := strings.ToLower(parser.GetFileExtension(fileName))
			if fileExt == "" {
				fileExt = "xlsx"
			}
			newStorageKey = fmt.Sprintf("%sData_%s_%d", fileExt, fileName, timestamp)

			bankData := parser.BankData{
				DisplayName: parser.ExtractFileName(newStorageKey),
				StorageKey:  newStorageKey,
				Questions:   detail.Questions,
			}

			jsonData, err := json.Marshal(bankData)
			if err != nil {
				customElements.ShowCustomInformation(core.BankManageErrorMsgType, "转换数据失败", w)
				return
			}

			filePath := fmt.Sprintf("%s/%s.json", storageDir, newStorageKey)
			if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
				customElements.ShowCustomInformation(core.BankManageErrorMsgType, "保存文件失败", w)
				return
			}

			state.CurrentStorageKey = bankData.StorageKey
			state.CurrentFileName = bankData.DisplayName
			core.SyncQuestionsToState(state, &bankData)

			customElements.ShowCustomInformation(core.BankManageSuccessMsgType, core.BankManageDownloadSuccessMsg, w)
			if onBack != nil {
				storageRelated.RefreshLocalFilesList(state)
				onBack()
			}
		}

		// 文件重复弹窗确认
		if len(oldFileNames) > 0 {
			storageRelated.ShowFileConflictDialog(w, humanName, func(choice storageRelated.ConflictResolutionChoice) {
				saveBank(choice)
			})
		} else {
			saveBank(storageRelated.ResolveOverwrite)
		}
	})
}

// downloadMyBank 从"我的题库"列表中下载题库。
//
// 功能说明：
// 弹出确认对话框，用户确认后调用 executeDownloadAndSave 函数执行下载和保存操作。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - bank: *network.MyBankItem 类型，用户的题库项。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 核心业务逻辑：
//
//	调用 customElements.ShowCustomConfirm 弹出确认对话框，用户确认后调用 executeDownloadAndSave 执行下载与保存流程。
func downloadMyBank(w fyne.Window, state *core.AppState, bank *network.MyBankItem, onBack func()) {
	customElements.ShowCustomConfirm(
		core.BankManageConfirmDownloadMsg,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.BankManageConfirmDownloadConfirmMsg, bank.BankName)),
		func(confirm bool) bool {
			if confirm {
				// ✅ 将下载逻辑放入单独的 goroutine，并延迟执行，防止关闭动画覆盖新弹窗
				go func() {
					time.Sleep(200 * time.Millisecond) // 等待第一个确认弹窗彻底关闭
					executeDownloadAndSave(w, state, bank.BankID, bank.BankName, onBack)
				}()
			}
			return true // 立即返回 true，让 Fyne 优先关闭当前的确认弹窗
		},
		w,
	)
}
