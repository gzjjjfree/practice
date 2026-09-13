package managementRelated

import (
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)

// ShowUserManage 渲染用户管理页面。
//
// 功能说明：
// 该函数负责展示用户管理界面，包括用户列表的显示、创建用户按钮和（针对管理员）查看管理员按钮。
// 支持动态刷新用户列表，并根据用户角色（普通用户或管理员）显示不同的操作按钮。
// 对于管理员，还可以进入管理员管理页面。
//
// 输入参数：
//   - w: fyne.Window 类型，表示当前应用程序的窗口对象，用于渲染UI和显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，包含用户登录状态、角色信息以及用户数据获取等方法。
//   - backToHome: func() 类型，返回主页的回调函数。
//
// 核心业务逻辑：
//  1. 创建顶部导航栏，包含返回按钮和页面标题。
//  2. 根据当前用户是否为管理员，创建操作按钮行（创建用户按钮，以及可选的管理员管理按钮）。
//  3. 定义 refreshUserList 函数用于获取并渲染用户列表：
//     - 检查用户权限，非管理员或未登录则提示并返回主页。
//     - 显示加载指示器。
//     - 调用 state.FetchUsers 获取用户数据。
//     - 过滤掉 superadmin 和 createdBy 为 0 的用户。
//     - 根据特定规则对用户进行排序：superadmin 的直接非管理员用户优先，然后是每个管理员及其管理的用户。
//     - 批量构建用户卡片并渲染到 UI。
func ShowUserManage(w fyne.Window, state *core.AppState, backToHome func()) {
	// 返回按钮：用于返回到主页。标识符为 backBtn，功能为触发 backToHome 回调函数以退出当前用户管理页面并返回主界面。
	backBtn := customElements.CreateButton(
		core.AdminBackBtnText,
		core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			backToHome()
		},
	)

	titleCenter := customElements.CreateLabel(core.UserManageTitleText, core.HexColor(core.TextPrimaryColor), core.FontSizePageTitle, true, false, false)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// User list
	var userListVBox *fyne.Container
	var refreshUserList func()

	// 操作按钮行
	// 创建用户按钮：用于打开创建新用户对话框。标识符为 createUserBtn，功能为调用 showCreateUserDialog 创建新用户，并在创建成功后刷新用户列表。
	createUserBtn := customElements.CreateButton(
		core.UserManageCreateUserBtnText,
		core.ActionBtnWidth, core.ActionBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			showCreateUserDialog(w, state, func() {
				refreshUserList()
			})
		},
	)

	var actionRow *fyne.Container
	if state.IsAdmin {
		// 查看管理员按钮：用于进入管理员管理页面。标识符为 viewAdminsBtn，功能为调用 ShowAdminManage 展示管理员管理界面，并在返回时重新渲染用户管理页面以刷新状态。
		viewAdminsBtn := customElements.CreateButton(
			core.UserManageViewAdminsBtnText,
			core.ActionBtnWidth, core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.RoleAdminColor),
			core.HexColor(core.RoleAdminColor),
			core.StrokeMedium,
			core.AdminDialogBtnFontSize,
			true,
			false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				ShowAdminManage(w, state, func() {
					// Re-render user management page to refresh state
					ShowUserManage(w, state, backToHome)
				})
			},
		)

		actionRow = container.NewHBox(createUserBtn, viewAdminsBtn)
	} else {
		actionRow = container.NewHBox(createUserBtn)
	}

	refreshUserList = func() {
		if !state.IsAdmin && !state.IsLoggedIn {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.UserManageNoAdminPermissionMsg, w)
			backToHome()
			return
		}

		if userListVBox == nil {
			userListVBox = container.NewVBox()
		}

		// Show loading indicator
		loadingText := canvas.NewText(core.AdminLoadingText, core.HexColor(core.TextMutedColor))
		loadingText.TextSize = core.FontSizeBody
		loadingText.Alignment = fyne.TextAlignCenter
		userListVBox.Objects = []fyne.CanvasObject{container.NewPadded(loadingText)}
		userListVBox.Refresh()

		state.FetchUsers(100, 0, func(success bool, users []network.UserItem, total int, msg string) {
			if !success {
				errText := canvas.NewText(core.AdminErrorText+": "+msg, core.HexColor(core.ColorWrongBorder))
				errText.TextSize = core.FontSizeBody
				errText.Alignment = fyne.TextAlignCenter
				userListVBox.Objects = []fyne.CanvasObject{container.NewPadded(errText)}
				userListVBox.Refresh()
				return
			}

			if len(users) == 0 {
				noDataText := canvas.NewText(core.UserManageNoUsersMsg, core.HexColor(core.TextMutedColor))
				noDataText.TextSize = core.FontSizeBody
				noDataText.Alignment = fyne.TextAlignCenter
				userListVBox.Objects = []fyne.CanvasObject{container.NewPadded(noDataText)}
				userListVBox.Refresh()
				return
			}

			// Filter superadmin users and create quick lookup map for creator names
			filteredUsers := make([]network.UserItem, 0, len(users))
			userMap := make(map[int]string, len(users))

			for _, u := range users {
				userMap[u.UserID] = u.Username
				if u.CreatedBy == 0 || u.Role == "superadmin" {
					continue
				}
				filteredUsers = append(filteredUsers, u)
			}
			users = filteredUsers

			// Sort users: superadmin's direct non-admin users first, then each admin with their managed users
			sort.Slice(users, func(i, j int) bool {
				u1, u2 := users[i], users[j]

				getGroupID := func(u network.UserItem) int {
					if u.Role == "admin" {
						return u.UserID
					}
					return u.CreatedBy
				}

				g1 := getGroupID(u1)
				g2 := getGroupID(u2)

				if g1 == 1 && g2 == 1 {
					return u1.UserID < u2.UserID
				}
				if g1 == 1 {
					return true
				}
				if g2 == 1 {
					return false
				}
				if g1 != g2 {
					return g1 < g2
				}
				if u1.Role == "admin" {
					return true
				}
				if u2.Role == "admin" {
					return false
				}
				return u1.UserID < u2.UserID
			})

			// Batch build UI objects to prevent multiple layout recalculations
			objects := make([]fyne.CanvasObject, 0, len(users)*2)
			for _, user := range users {
				userItem := user
				userCard := buildUserCard(&userItem, state, userMap, func() {
					showUpdateUserDialog(w, state, &userItem, func() {
						refreshUserList()
					})
				}, func() {
					customElements.ShowCustomConfirm(
						core.UserManageDeleteConfirmTitle,
						"确定",
						"取消",
						customElements.NewCenterRichText(fmt.Sprintf("确定要删除用户 %s 吗？", userItem.Username)),
						func(confirm bool) bool {
							if confirm {
								state.DeleteUser(userItem.UserID, func(success bool, msg string) {
									if success {
										refreshUserList()
									} else {
										customElements.ShowCustomInformation(core.AdminErrorText, msg, w)
									}
								})
							}
							return true
						}, w)
				})
				objects = append(objects, userCard, container.NewGridWrap(fyne.NewSize(1, 8)))
			}

			userListVBox.Objects = objects
			userListVBox.Refresh()
		})
	}

	// Initial render
	refreshUserList()

	scrollContent := container.NewVScroll(userListVBox)
	scrollContent.SetMinSize(fyne.NewSize(0, core.UserManageScrollMinHeight))

	contentArea := container.NewBorder(actionRow, nil, nil, nil, scrollContent)
	mainLayout := container.NewBorder(topBar, nil, nil, nil, contentArea)
	wRootLayout := container.NewStack(pageBg, mainLayout)
	w.SetContent(wRootLayout)
}

// buildUserCard 为用户项创建卡片UI组件。
//
// 功能说明：
// 该函数用于为单个用户生成一个包含用户名、角色徽章、操作按钮（编辑和删除）以及归属信息的用户卡片。
//
// 输入参数：
//   - user: *network.UserItem 类型，表示要展示的用户信息项。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - userMap: map[int]string 类型，用于快速查找创建者的用户名（通过用户ID映射到用户名）。
//   - onEdit: func() 类型，点击编辑按钮时的回调函数。
//   - onDelete: func() 类型，点击删除按钮时的回调函数。
//
// 返回值：
//   - fyne.CanvasObject 类型，返回构建好的用户卡片UI组件。
//
// 核心业务逻辑：
//  1. 创建卡片背景并设置圆角和边框样式。
//  2. 显示用户名（带用户图标）和角色徽章（根据角色设置不同颜色）。
//  3. 创建编辑按钮和删除按钮，分别绑定 onEdit 和 onDelete 回调。
//  4. 对于非 superadmin 用户，显示归属信息（通过 createdBy 查找创建者用户名）。
//  5. 组合所有UI元素并返回卡片组件。
func buildUserCard(user *network.UserItem, state *core.AppState, userMap map[int]string, onEdit func(), onDelete func()) fyne.CanvasObject {
	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.CardCornerRadius
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	// Username
	nameText := canvas.NewText(core.IconUserPrefix+user.Username, core.HexColor(core.TextPrimaryColor))
	nameText.TextSize = core.CardNameTextSize
	nameText.TextStyle = fyne.TextStyle{Bold: true}

	// Role badge
	roleColor := core.TextMutedColor
	roleText := "user"
	if user.Role == "admin" {
		roleColor = core.RoleAdminColor
		roleText = "admin"
	} else if user.Role == "superadmin" {
		roleColor = core.RoleSuperAdminColor
		roleText = "superadmin"
	}
	roleBadge := canvas.NewText(core.IconSelectedCircle+" "+roleText, core.HexColor(roleColor))
	roleBadge.TextSize = core.FontSizeSubtitle

	// 操作按钮
	// 编辑用户按钮：用于打开更新用户对话框。标识符为 editBtn，功能为触发 onEdit 回调以编辑当前用户的详细信息（如用户名、密码或角色）。
	editBtn := customElements.CreateButton(
		core.UserManageEditBtnText,
		core.CardActionBtnWidth, core.ActionBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.StrokeMedium,
		core.CardActionBtnFontSize,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onEdit,
	)

	// 删除用户按钮：用于删除当前用户。标识符为 deleteBtn，功能为触发 onDelete 回调以弹出确认对话框并执行用户删除操作。
	deleteBtn := customElements.CreateButton(
		core.UserManageDeleteBtnText,
		core.CardActionBtnWidth, core.ActionBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.ColorErrorBg),
		core.HexColor(core.ColorErrorBg),
		core.StrokeMedium,
		core.CardActionBtnFontSize,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onDelete,
	)

	btnRow := container.NewHBox(editBtn, deleteBtn)

	// Build info line for non-superadmin users: O(1) map lookup
	var infoText *canvas.Text
	if user.Role != "superadmin" && user.CreatedBy != 0 {
		creatorName := user.CreatedByUsername
		if creatorName == "" {
			if name, ok := userMap[user.CreatedBy]; ok {
				creatorName = name
			} else {
				creatorName = fmt.Sprintf("用户ID %d", user.CreatedBy)
			}
		}
		infoText = canvas.NewText(fmt.Sprintf("归属: %s", creatorName), core.HexColor(core.TextSecondaryColor))
		infoText.TextSize = core.FontSizeSubtitle
	}

	var cardContent fyne.CanvasObject
	if infoText != nil {
		cardContent = container.NewVBox(
			container.NewBorder(nameText, nil, roleBadge, nil, nil),
			container.NewGridWrap(fyne.NewSize(1, 5)),
			infoText,
			container.NewGridWrap(fyne.NewSize(1, 5)),
			btnRow,
		)
	} else {
		cardContent = container.NewVBox(
			container.NewBorder(nameText, nil, roleBadge, nil, nil),
			container.NewGridWrap(fyne.NewSize(1, 10)),
			btnRow,
		)
	}

	return container.NewStack(
		cardBg,
		container.NewPadded(cardContent),
	)
}

// showCreateUserDialog 显示用于创建新用户的对话框，使用标准化的 Form 组件。
//
// 功能说明：
// 该函数弹出一个表单对话框，允许管理员或超级管理员创建新用户。表单包含用户名、密码输入框，以及针对超级管理员的角色选择下拉框。
//
// 输入参数：
//   - w: fyne.Window 类型，表示当前应用程序的窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于判断当前用户角色（是否为 superadmin）以及调用创建用户API。
//   - onCreated: func() 类型，创建成功后的回调函数，通常用于刷新用户列表。
//
// 核心业务逻辑：
//  1. 创建用户名和密码输入框。
//  2. 如果当前用户角色为 superadmin，则添加角色选择下拉框（可选值为 admin 或 user）。
//  3. 构建表单并设置取消和提交按钮的回调函数。
//  4. 在提交时验证用户名和密码是否为空，以及密码长度是否至少为6位。
//  5. 调用 state.CreateUser 创建用户，并根据结果提示成功或失败信息，成功后触发 onCreated 回调。
func showCreateUserDialog(w fyne.Window, state *core.AppState, onCreated func()) {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder(core.UsernameEntryPlaceholder)
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder(core.PasswordEntryCreatePlaceholder)

	items := []*widget.FormItem{
		widget.NewFormItem(core.UsernameFormItemLabel, usernameEntry),
		widget.NewFormItem(core.PasswordFormItemLabel, passwordEntry),
	}

	var roleSelect *widget.Select
	if state.Role == "superadmin" {
		roleSelect = widget.NewSelect([]string{"admin", "user"}, nil)
		roleSelect.SetSelected("user")
		items = append(items, widget.NewFormItem(core.RoleFormItemLabel, roleSelect))
	}

	var dialogDial dialog.Dialog

	form := widget.NewForm(items...)
	form.CancelText = core.AdminCancelText
	form.SubmitText = core.CreateUserSubmitText

	form.OnCancel = func() {
		if dialogDial != nil {
			dialogDial.Hide()
		}
	}

	form.OnSubmit = func() {
		username := usernameEntry.Text
		password := passwordEntry.Text

		if username == "" || password == "" {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.UserManageEmptyFieldsMsg, w)
			return
		}
		if len(password) < 6 {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.UserManagePasswordLengthMsg, w)
			return
		}

		req := network.CreateUserReq{
			Username: username,
			Password: password,
			Role:     "user",
		}
		if roleSelect != nil {
			req.Role = roleSelect.Selected
		}

		state.CreateUser(req, func(success bool, user *network.UserItem, msg string) {
			if success {
				if dialogDial != nil {
					dialogDial.Hide()
				}
				customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, fmt.Sprintf(core.UserManageCreateSuccessMsg, username), w)
				if onCreated != nil {
					onCreated()
				}
			} else {
				customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
			}
		})
	}

	content := container.NewPadded(form)
	dialogDial = dialog.NewCustomWithoutButtons(core.CreateUserDialogTitle, content, w)
	dialogDial.Resize(fyne.NewSize(core.UserManageDialogWidth, core.UserManageDialogHeight))
	dialogDial.Show()
}

// showUpdateUserDialog 显示用于更新现有用户的对话框。
//
// 功能说明：
// 该函数弹出一个表单对话框，允许管理员或超级管理员更新现有用户的用户名、密码和角色（针对 superadmin）。
//
// 输入参数：
//   - w: fyne.Window 类型，表示当前应用程序的窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于判断当前用户角色（是否为 superadmin）以及调用更新用户API。
//   - user: *network.UserItem 类型，表示要更新的现有用户信息项。
//   - onUpdated: func() 类型，更新成功后的回调函数，通常用于刷新用户列表。
//
// 核心业务逻辑：
//  1. 创建用户名和密码输入框，并初始化为用户的当前用户名和密码占位符。
//  2. 如果当前用户角色为 superadmin，则添加角色选择下拉框，并根据用户的当前角色设置默认选中项。
//  3. 构建表单并设置取消和提交按钮的回调函数。
//  4. 在提交时，检查用户名和密码是否有更新，以及角色是否有变化，构建更新请求对象。
//  5. 调用 state.UpdateUser 更新用户，并根据结果提示成功或失败信息，成功后触发 onUpdated 回调。
func showUpdateUserDialog(w fyne.Window, state *core.AppState, user *network.UserItem, onUpdated func()) {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(user.Username)

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder(core.PasswordEntryUpdatePlaceholder)

	items := []*widget.FormItem{
		widget.NewFormItem(core.UsernameFormItemLabel, usernameEntry),
		widget.NewFormItem(core.PasswordFormItemLabel, passwordEntry),
	}

	var roleSelect *widget.Select
	if state.Role == "superadmin" {
		roleSelect = widget.NewSelect([]string{"admin", "user"}, nil)
		if user.Role == "admin" || user.Role == "user" {
			roleSelect.SetSelected(user.Role)
		} else {
			roleSelect.SetSelected("user")
		}
		items = append(items, widget.NewFormItem(core.RoleFormItemLabel, roleSelect))
	}

	var dialogDial dialog.Dialog

	form := widget.NewForm(items...)
	form.CancelText = core.AdminCancelText
	form.SubmitText = core.UpdateUserSubmitText

	form.OnCancel = func() {
		if dialogDial != nil {
			dialogDial.Hide()
		}
	}
	form.OnSubmit = func() {
		newUsername := usernameEntry.Text
		newPassword := passwordEntry.Text

		req := network.UpdateUserReq{}
		if newUsername != "" {
			req.Username = &newUsername
		}
		if newPassword != "" {
			req.Password = &newPassword
		}
		if roleSelect != nil && roleSelect.Selected != user.Role {
			req.Role = &roleSelect.Selected
		}

		state.UpdateUser(user.UserID, req, func(success bool, msg string) {
			if dialogDial != nil {
				dialogDial.Hide()
			}
			if success {
				customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, core.UserManageUpdateSuccessMsg, w)
				if onUpdated != nil {
					onUpdated()
				}
			} else {
				customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
			}
		})
	}

	content := container.NewPadded(form)
	dialogDial = dialog.NewCustomWithoutButtons(core.UpdateUserDialogTitle, content, w)
	dialogDial.Resize(fyne.NewSize(core.UserManageDialogWidth, core.UserManageDialogHeight))
	dialogDial.Show()
}

// ShowAdminManage 渲染管理员管理页面。
//
// 功能说明：
// 该函数负责展示管理员管理界面，包括管理员列表的显示。仅允许管理员或登录用户访问。
//
// 输入参数：
//   - w: fyne.Window 类型，表示当前应用程序的窗口对象，用于渲染UI和显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，包含用户登录状态、角色信息以及管理员数据获取等方法。
//   - backToHome: func() 类型，返回主页的回调函数。
//
// 核心业务逻辑：
//  1. 创建顶部导航栏，包含返回按钮和页面标题。
//  2. 定义 refreshAdminList 函数用于获取并渲染管理员列表：
//     - 检查用户权限，非管理员或未登录则提示并返回主页。
//     - 显示加载指示器。
//     - 调用 state.FetchAdmins 获取管理员数据。
//     - 批量构建管理员卡片并渲染到 UI。
func ShowAdminManage(w fyne.Window, state *core.AppState, backToHome func()) {
	// 返回按钮：用于返回到主页。标识符为 backBtn，功能为触发 backToHome 回调函数以退出当前管理员管理页面并返回主界面。
	backBtn := customElements.CreateButton(
		core.AdminBackBtnText,
		core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			backToHome()
		},
	)

	titleCenter := customElements.CreateLabel(core.AdminManageTitleText, core.HexColor(core.TextPrimaryColor), core.FontSizePageTitle, true, false, false)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// Admin list
	var adminListVBox *fyne.Container
	var refreshAdminList func()

	refreshAdminList = func() {
		if !state.IsAdmin && !state.IsLoggedIn {
			customElements.ShowCustomInformation(core.TemplateManageInfoMsgType, core.AdminManageNoAdminPermissionMsg, w)
			backToHome()
			return
		}

		if adminListVBox == nil {
			adminListVBox = container.NewVBox()
		}

		// Show loading indicator
		loadingText := canvas.NewText(core.AdminLoadingText, core.HexColor(core.TextMutedColor))
		loadingText.TextSize = core.FontSizeBody
		loadingText.Alignment = fyne.TextAlignCenter
		adminListVBox.Objects = []fyne.CanvasObject{container.NewPadded(loadingText)}
		adminListVBox.Refresh()

		state.FetchAdmins(100, 0, func(success bool, admins []network.UserItem, total int, msg string) {
			if !success {
				errText := canvas.NewText(core.AdminErrorText+": "+msg, core.HexColor(core.ColorWrongBorder))
				errText.TextSize = core.FontSizeBody
				errText.Alignment = fyne.TextAlignCenter
				adminListVBox.Objects = []fyne.CanvasObject{container.NewPadded(errText)}
				adminListVBox.Refresh()
				return
			}

			if len(admins) == 0 {
				noDataText := canvas.NewText(core.AdminManageNoAdminsMsg, core.HexColor(core.TextMutedColor))
				noDataText.TextSize = core.FontSizeBody
				noDataText.Alignment = fyne.TextAlignCenter
				adminListVBox.Objects = []fyne.CanvasObject{container.NewPadded(noDataText)}
				adminListVBox.Refresh()
				return
			}

			// Batch build admin cards
			objects := make([]fyne.CanvasObject, 0, len(admins)*2)
			for _, admin := range admins {
				adminItem := admin
				adminCard := buildAdminCard(&adminItem, state)
				objects = append(objects, adminCard, container.NewGridWrap(fyne.NewSize(1, 8)))
			}

			adminListVBox.Objects = objects
			adminListVBox.Refresh()
		})
	}

	// Initial render
	refreshAdminList()

	scrollContent := container.NewVScroll(adminListVBox)
	scrollContent.SetMinSize(fyne.NewSize(0, core.UserManageScrollMinHeight))

	mainLayout := container.NewBorder(topBar, nil, nil, nil, scrollContent)
	wRootLayout := container.NewStack(pageBg, mainLayout)
	w.SetContent(wRootLayout)
}

// buildAdminCard 为管理员项创建卡片UI组件。
//
// 功能说明：
// 该函数用于为单个管理员生成一个包含用户名、角色徽章的管理员卡片。与用户卡片不同，管理员卡片不包含操作按钮和归属信息。
//
// 输入参数：
//   - admin: *network.UserItem 类型，表示要展示的管理员信息项。
//   - state: *core.AppState 类型，表示全局应用状态（当前未直接使用，但作为参数保留以保持一致性）。
//
// 返回值：
//   - fyne.CanvasObject 类型，返回构建好的管理员卡片UI组件。
//
// 核心业务逻辑：
//  1. 创建卡片背景并设置圆角和边框样式。
//  2. 显示用户名（带管理员图标）和角色徽章（根据角色设置不同颜色）。
//  3. 组合所有UI元素并返回卡片组件。
func buildAdminCard(admin *network.UserItem, state *core.AppState) fyne.CanvasObject {
	cardBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	cardBg.CornerRadius = core.CardCornerRadius
	cardBg.StrokeColor = core.HexColor(core.BorderLightColor)
	cardBg.StrokeWidth = core.StrokeThin

	// Username
	nameText := canvas.NewText(core.IconAdminPrefix+admin.Username, core.HexColor(core.TextPrimaryColor))
	nameText.TextSize = core.CardNameTextSize
	nameText.TextStyle = fyne.TextStyle{Bold: true}

	// Role badge
	roleColor := core.RoleAdminColor
	roleText := "admin"
	if admin.Role == "superadmin" {
		roleColor = core.RoleSuperAdminColor
		roleText = "superadmin"
	}
	roleBadge := canvas.NewText(core.IconSelectedCircle+" "+roleText, core.HexColor(roleColor))
	roleBadge.TextSize = core.FontSizeSubtitle

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, roleBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 10)),
	)

	return container.NewStack(
		cardBg,
		container.NewPadded(cardContent),
	)
}
