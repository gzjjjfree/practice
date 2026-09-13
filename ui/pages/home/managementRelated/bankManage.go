package managementRelated

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
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

// ==================== 题库管理主页面 ====================

// ShowBankManage 渲染题库管理主页面。
//
// 功能说明：
// 根据当前用户的角色（superadmin、admin 或普通用户），显示相应的题库管理视图。超级管理员和管理员可以上传和查看题库，普通用户只能查看自己的题库列表。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象，用于 UI 渲染和对话框展示。
//   - state: *core.AppState 类型，表示全局应用状态，包含用户信息、角色、用户名等。
//   - backToHome: func() 类型，返回主页面的回调函数。
//
// 核心业务逻辑：
// 1. 创建返回按钮（backBtn），绑定 backToHome 回调函数。
// 2. 根据 state.Role 判断用户角色：superadmin 调用 buildSuperAdminView，admin 调用 buildAdminView，普通用户调用 buildUserView。
// 3. 构建页面根布局并设置到窗口 w。
func ShowBankManage(w fyne.Window, state *core.AppState, backToHome func()) {
	// 返回按钮：用于从题库管理页面返回到主页面。
	backBtn := customElements.CreateButton(
		core.BankManageBackBtnText, // 按钮文本："返回"
		core.BackBtnWidth,
		core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		core.StrokeMedium,
		core.AdminHeaderFontSize,
		false, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			println(">>> 点击了返回按钮！") // 👈 添加调试打印
			backToHome()
		},
	)

	titleText := customElements.CreateLabel(core.BankManageTitleText, core.HexColor(core.TextPrimaryColor), core.FontSizePageTitle, true, false, false)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleText)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	// Determine user role based on server-returned role field
	isSuperAdmin := state.Role == "superadmin"

	var mainContent fyne.CanvasObject

	if isSuperAdmin {
		// 超级管理员：显示上传/查看按钮
		mainContent = buildSuperAdminView(w, state, backToHome)
	} else if state.Role == "admin" {
		// 管理员：显示上传/查看按钮
		mainContent = buildAdminView(w, state, backToHome)
	} else {
		// 普通用户：直接显示我的题库列表
		mainContent = buildUserView(w, state, backToHome)
	}

	wRootLayout := container.NewStack(pageBg, container.NewBorder(topBar, nil, nil, nil, mainContent))
	w.SetContent(wRootLayout)
}

// ==================== 超级管理员视图 ====================

// buildSuperAdminView 为超级管理员创建包含上传和查看按钮的视图。
//
// 功能说明：
// 构建超级管理员专用的题库管理界面，提供"上传题库"和"查看题库"两个主要操作按钮。
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
//   - "上传题库"按钮 (core.BankManageUploadBtnText)：点击后打开超级管理员上传题库的对话框，选择本地题库文件并上传到服务器。
//   - "查看题库"按钮 (core.BankManageViewBtnText)：点击后打开管理员选择对话框，用于查看指定管理员或超级管理员的题库列表。
func buildSuperAdminView(w fyne.Window, state *core.AppState, backToHome func()) *fyne.Container {
	// 初始操作行，包含上传和查看按钮
	actionRow := container.NewHBox(
		// 上传题库按钮：点击后打开超级管理员上传题库的对话框，选择本地题库文件并上传到服务器。
		customElements.CreateButton(
			core.BankManageUploadBtnText, // 按钮文本："上传题库"
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
				showSuperAdminUpload(w, state, func() {
					// 上传完成后刷新页面
					ShowBankManage(w, state, backToHome)
				})
			},
		),
		// 查看题库按钮：点击后打开管理员选择对话框，用于查看指定管理员或超级管理员的题库列表。
		customElements.CreateButton(
			core.BankManageViewBtnText, // 按钮文本："查看题库"
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
				showSuperAdminViewSelector(w, state, false, func() {
					ShowBankManage(w, state, backToHome)
				})
			},
		),
	)

	return container.NewVBox(
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)),
		actionRow,
	)
}

// showSuperAdminUpload 为超级管理员显示上传题库对话框。
//
// 功能说明：
// 读取本地存储的题库文件（以 xlsxData_、xlsData_、txtData_、txtsData_、jsonData_ 开头的 JSON 文件），
// 让用户选择要上传的题库，并将选中的题库通过 API 上传到服务器。如果存在同名题库冲突，会提示用户是否覆盖。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onUploaded: func() 类型，上传完成后的回调函数，用于刷新页面。
//
// 核心业务逻辑与按钮说明：
//   - 本地题库文件列表展示：读取并过滤出有效的题库 JSON 文件（排除错题、收藏、答题记录等），按时间戳降序排序，生成复选框供用户选择。
//   - "确认上传"按钮 (core.BankManageConfirmUploadBtnText)：将用户选中的本地题库文件解析并打包为 network.BankImportReq，调用 state.ImportBanks 接口上传到服务器；若存在同名题库冲突，会弹出确认对话框询问用户是否覆盖已有题库。
//   - "取消"按钮 (core.BankManageCancelBtnText)：关闭上传题库对话框，不执行任何上传操作。
func showSuperAdminUpload(w fyne.Window, state *core.AppState, onUploaded func()) {
	// Get local stored files
	storageDir := storageRelated.GetStorageDir()
	files, err := os.ReadDir(storageDir)
	if err != nil {
		customElements.ShowCustomInformation(core.BankManageErrorMsgType, core.BankManageReadLocalFilesErrMsg, w)
		return
	}

	type FileInfo struct {
		Key       string
		Timestamp int64
	}

	var fileList []FileInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			if strings.Contains(f.Name(), "错题") || strings.Contains(f.Name(), "收藏") || strings.Contains(f.Name(), "答题记录") {
				continue
			}
			name := strings.TrimSuffix(f.Name(), ".json")

			// Only process bank files (xxxData_ prefix)
			if !strings.HasPrefix(name, "xlsxData_") && !strings.HasPrefix(name, "xlsData_") &&
				!strings.HasPrefix(name, "txtData_") && !strings.HasPrefix(name, "txtsData_") &&
				!strings.HasPrefix(name, "jsonData_") {
				continue
			}

			parts := strings.Split(name, "_")
			if len(parts) >= 3 {
				tsStr := parts[len(parts)-1]
				ts, _ := strconv.ParseInt(tsStr, 10, 64)
				fileList = append(fileList, FileInfo{Key: name, Timestamp: ts})
			} else {
				// No timestamp in filename, use current time
				fileList = append(fileList, FileInfo{Key: name, Timestamp: time.Now().UnixMilli()})
			}
		}
	}

	// Sort by timestamp descending
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Timestamp > fileList[j].Timestamp
	})

	if len(fileList) == 0 {
		customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManageNoLocalBankFilesMsg, w)
		return
	}

	var selectedKeys []string
	var checkboxObjs []fyne.CanvasObject

	for _, fi := range fileList {
		ckKey := fi.Key
		showName := parser.ExtractFileName(ckKey)
		cb := widget.NewCheck(core.BankManageCheckboxPrefixBank+showName, func(checked bool) {
			if checked {
				selectedKeys = append(selectedKeys, ckKey)
			} else {
				for i, sk := range selectedKeys {
					if sk == ckKey {
						selectedKeys = append(selectedKeys[:i], selectedKeys[i+1:]...)
						break
					}
				}
			}
		})
		checkboxObjs = append(checkboxObjs, cb)
	}

	// 确认上传按钮 (core.BankManageConfirmUploadBtnText)：将用户选中的本地题库文件解析并打包为 network.BankImportReq，调用 state.ImportBanks 接口上传到服务器。若存在同名题库冲突，会弹出确认对话框询问用户是否覆盖已有题库。
	uploadBtn := customElements.CreateButton(
		core.BankManageConfirmUploadBtnText, // 按钮标识符：BankManageConfirmUploadBtnText，按钮文本："确认上传"，功能与业务用途：提交用户选择的本地题库文件至服务器进行导入。
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
			if len(selectedKeys) == 0 {
				customElements.ShowCustomInformation(core.BankManageInfoMsgType, core.BankManagePleaseSelectBanksMsg, w)
				return
			}

			var banks []network.BankImportItem
			for _, key := range selectedKeys {
				filePath := fmt.Sprintf("%s/%s.json", storageRelated.GetStorageDir(), key)
				bytesData, err := os.ReadFile(filePath)
				if err != nil {
					customElements.ShowCustomInformation(core.BankManageErrorMsgType, fmt.Sprintf(core.BankManageReadFileErrMsg, err), w)
					return
				}

				var bankData parser.BankData
				if err := json.Unmarshal(bytesData, &bankData); err != nil {
					customElements.ShowCustomInformation(core.BankManageErrorMsgType, fmt.Sprintf(core.BankManageParseFileErrMsg, err), w)
					return
				}

				banks = append(banks, network.BankImportItem{
					DisplayName: bankData.DisplayName,
					StorageKey:  bankData.StorageKey,
					Questions:   convertToImportQuestions(bankData.Questions),
				})
			}

			req := network.BankImportReq{
				OwnerUsername: state.Username,
				Banks:         banks,
			}

			state.ImportBanks(req, func(success bool, imported, failed int, errors []string, duplicates []string, msg string) {
				hasDuplicate := strings.Contains(strings.ToLower(msg), "already exist")

				if hasDuplicate {
					var duplicateNames []string
					if len(duplicates) > 0 {
						duplicateNames = duplicates
					} else {
						for _, e := range errors {
							if strings.Contains(e, "already exists") {
								if start := strings.Index(e, "\""); start != -1 {
									if end := strings.Index(e[start+1:], "\""); end != -1 {
										duplicateNames = append(duplicateNames, e[start+1:start+1+end])
									}
								}
							}
						}
					}

					if len(duplicateNames) > 0 {
						nameList := strings.Join(duplicateNames, "\n")

						customElements.ShowCustomConfirm(
							core.BankManageDuplicateBanksFoundMsg,
							"确定",
							"取消",
							customElements.NewCenterRichText(fmt.Sprintf(core.BankManageDuplicateBanksConfirmMsg, nameList)),
							func(confirm bool) bool {
								if confirm {
									var banksOverwrite []network.BankImportItem
									for _, item := range banks {
										item.Overwrite = true
										banksOverwrite = append(banksOverwrite, item)
									}

									reqOverwrite := network.BankImportReq{
										OwnerUsername: state.Username,
										Banks:         banksOverwrite,
									}

									state.ImportBanks(reqOverwrite, func(success2 bool, imported2, failed2 int, errors2 []string, duplicates2 []string, msg2 string) {
										if success2 {
											customElements.ShowCustomInformation(core.BankManageSuccessMsgType, fmt.Sprintf(core.BankManageUploadCompleteMsg, imported2, failed2), w)
											if onUploaded != nil {
												onUploaded()
											}
										} else {
											customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg2, w)
										}
									})
								}
								return true
							},
							w,
						)
					} else {
						customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
					}
					return
				}

				if success {
					customElements.ShowCustomInformation(core.BankManageSuccessMsgType, fmt.Sprintf(core.BankManageUploadCompleteMsg, imported, failed), w)
					if onUploaded != nil {
						onUploaded()
					}
				} else {
					customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
				}
			})
		},
	)

	var dialogDial dialog.Dialog

	// 取消按钮 (core.BankManageCancelBtnText)：关闭上传题库对话框，不执行任何上传操作。
	cancelBtn := customElements.CreateButton(
		core.BankManageCancelBtnText, // 按钮标识符：BankManageCancelBtnText，按钮文本："取消"，功能与业务用途：取消当前上传流程并关闭对话框。
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
			if dialogDial != nil {
				dialogDial.Hide()
			}
		},
	)

	btnRow := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
		cancelBtn,
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
		uploadBtn,
	)

	topArea := container.NewVBox(
		widget.NewLabel(core.BankManageSelectUploadBanksMsg),
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)),
	)
	bottomArea := container.NewVBox(
		container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)),
		container.NewCenter(btnRow),
	)

	scrollArea := container.NewVScroll(container.NewVBox(checkboxObjs...))
	minSizeSpacer := container.NewGridWrap(fyne.NewSize(core.LocalBankDialogWidth, core.LocalBankDialogHeight))
	centerArea := container.NewStack(minSizeSpacer, scrollArea)

	content := container.NewBorder(topArea, bottomArea, nil, nil, centerArea)
	styledContent := container.NewPadded(content)

	dialogDial = dialog.NewCustomWithoutButtons(core.BankManageUploadDialogTitle, styledContent, w)
	dialogDial.Show()
}

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
	state.DownloadBankDetail(bankID, func(success bool, detail *parser.BankData, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.BankManageErrorMsgType, msg, w)
			return
		}

		humanName := detail.DisplayName
		storageDir := storageRelated.GetStorageDir()
		files, _ := os.ReadDir(storageDir)

		var dupFileNames []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
				filePath := fmt.Sprintf("%s/%s", storageDir, f.Name())
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}
				var localBank parser.BankData
				if err := json.Unmarshal(data, &localBank); err == nil && localBank.DisplayName == humanName {
					dupFileNames = append(dupFileNames, localBank.DisplayName)
				}
			}
		}

		saveBank := func(overwrite bool) {
			var newStorageKey string
			cleanName := strings.ReplaceAll(humanName, " ", "_")

			if overwrite {
				newStorageKey = fmt.Sprintf("xlsxData_%s", cleanName)
				for _, f := range files {
					if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
						lfPath := fmt.Sprintf("%s/%s", storageDir, f.Name())
						lfData, _ := os.ReadFile(lfPath)
						var lfBank parser.BankData
						if err := json.Unmarshal(lfData, &lfBank); err == nil && lfBank.DisplayName == humanName {
							_ = os.Remove(lfPath)
						}
					}
				}
			} else {
				newStorageKey = fmt.Sprintf("xlsxData_%s_%d", cleanName, time.Now().UnixMilli())
			}

			bankData := parser.BankData{
				DisplayName: humanName,
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

		if len(dupFileNames) > 0 {
			var confirmDialog dialog.Dialog
			// 取消按钮 (core.BankManageCancelBtnText)：关闭本地文件已存在确认对话框，不执行保存操作。
			cancelBtn := customElements.CreateButton(
				core.BankManageCancelBtnText,
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
					if confirmDialog != nil {
						confirmDialog.Hide()
					}
				})
			// 覆盖按钮 (core.BankManageOverwriteBtnText)：关闭对话框并覆盖本地已存在的同名题库文件。
			overwriteBtn := customElements.CreateButton(
				core.BankManageOverwriteBtnText,
				core.ActionBtnWidth,
				core.DefaultBtnHeight,
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
					if confirmDialog != nil {
						confirmDialog.Hide()
					}
					saveBank(true)
				})
			// 新建按钮 (core.BankManageNewBtnText)：关闭对话框并以带时间戳的新名称保存题库文件，避免覆盖。
			newBtn := customElements.CreateButton(
				core.BankManageNewBtnText,
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
					if confirmDialog != nil {
						confirmDialog.Hide()
					}
					saveBank(false)
				})

			btnRow := container.NewHBox(
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)), cancelBtn,
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)), overwriteBtn,
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH5)), newBtn,
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
			)

			msgText := widget.NewRichTextFromMarkdown(fmt.Sprintf(core.BankManageLocalFileExistsMsg, dupFileNames[0]))
			msgText.Wrapping = fyne.TextWrapWord
			content := container.NewVBox(msgText, container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH20)), container.NewCenter(btnRow))

			styledContent := container.NewStack(container.NewGridWrap(fyne.NewSize(core.DialogMinWidth, 1)), container.NewPadded(content))
			confirmDialog = dialog.NewCustomWithoutButtons(core.BankManageFileExistsDialogTitle, styledContent, w)
			confirmDialog.Show()
		} else {
			saveBank(false)
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
				executeDownloadAndSave(w, state, bank.BankID, bank.BankName, onBack)
			}
			return true // 返回 true 表示对话框可以关闭
		},
		w,
	)
}

// downloadBankForOwner 从所有者列表中下载题库。
//
// 功能说明：
// 弹出确认对话框，用户确认后调用 executeDownloadAndSave 函数执行下载和保存操作。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - bank: *network.BankItem 类型，题库项。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 核心业务逻辑：
//
//	提取题库的显示名称（若为空则使用 BankName），调用 customElements.ShowCustomConfirm 弹出确认对话框，用户确认后调用 executeDownloadAndSave 执行下载与保存流程。
func downloadBankForOwner(w fyne.Window, state *core.AppState, bank *network.BankItem, onBack func()) {
	displayName := bank.DisplayName
	if displayName == "" {
		displayName = bank.BankName
	}

	// 2. 按照新的参数顺序进行调用：标题, 确定按钮文案, 取消按钮文案, 内容对象, 回调函数, 父窗口
	customElements.ShowCustomConfirm(
		core.BankManageConfirmDownloadMsg,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.BankManageConfirmDownloadConfirmMsg, displayName)),
		func(confirm bool) bool {
			if confirm {
				executeDownloadAndSave(w, state, bank.BankID, displayName, onBack)
			}
			return true // 返回 true 表示对话框可以关闭
		},
		w,
	)
}

// convertToImportQuestions 将本地题库题目格式转换为导入服务器的题目格式。
//
// 功能说明：
// 遍历本地题库题目列表，将其转换为网络导入所需的题目项格式。
//
// 参数说明：
//   - questions: []parser.QuestionItem 类型，本地题库题目列表。
//
// 返回值：
//   - []network.ImportQuestionItem：转换后的网络导入题目项列表。
//
// 核心业务逻辑：
//
//	遍历输入的 questions 切片，提取每道题的 ID、Type、Content、Answer、Options、Difficulty 字段，构造为 network.ImportQuestionItem 并返回。
func convertToImportQuestions(questions []parser.QuestionItem) []network.ImportQuestionItem {
	var result []network.ImportQuestionItem
	for _, q := range questions {
		result = append(result, network.ImportQuestionItem{
			ID:         q.ID,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     q.Answer,
			Options:    q.Options,
			Difficulty: q.Difficulty,
		})
	}
	return result
}
