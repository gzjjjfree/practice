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

			loadingDial := customElements.ShowCustomLoading("上传题库", "正在上传", w)

			state.ImportBanks(req, func(success bool, imported, failed int, errors []string, duplicates []string, msg string) {
				loadingDial.Hide()
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

									loadingDial2 := customElements.ShowCustomLoading("上传题库", "正在上传", w)
									state.ImportBanks(reqOverwrite, func(success2 bool, imported2, failed2 int, errors2 []string, duplicates2 []string, msg2 string) {
										loadingDial2.Hide()
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
