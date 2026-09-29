package home

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/export"
	"github.com/gzjjjfree/practice/gz_theme"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/storageRelated"
	"github.com/gzjjjfree/practice/ui/pages/home/managementRelated"
	"github.com/gzjjjfree/practice/ui/pages/home/practiceRelated"
)

// ShowHome 渲染首页，包含题库操作按钮和已存储文件列表。
func ShowHome(w fyne.Window, state *core.AppState) {
	var refreshHomeUI func()

	makeCustomGridBtn := func(icon, text string, action func()) fyne.CanvasObject {
		// 将 width 传入 0 或父容器宽度，去除外部 NewCenter 包裹，使其填充 Grid 单元格[cite: 1]
		btn := customElements.CreateButton(
			text+" "+icon,
			0, // 宽设置为 0，让布局自适应或直接在 CreateButton 内改用 Stack 填充[cite: 1, 2]
			core.NavButtonHeight,
			core.HexColor(core.TextBodyColor),
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BorderMediumColor),
			core.StrokeMedium,
			core.FontSizeButton,
			false,
			false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			action)
		return btn
	}

	grid := container.NewGridWithColumns(3)

	grid.Add(makeCustomGridBtn("📁", "读取", func() {
		storageRelated.ShowBankFileOpenDialog(w, state, refreshHomeUI)
	}))

	grid.Add(makeCustomGridBtn("📤", "导出", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		// 触发导出格式选择弹窗
		export.ShowExportDialog(w, state)
	}))

	grid.Add(makeCustomGridBtn("🗑️", "删除", func() {
		if state.CurrentStorageKey == "" {
			customElements.ShowCustomInformation("提示", "当前未选中任何题库", w)
			return
		}
		customElements.ShowCustomConfirm(
			"提示",
			"确定",
			"取消",
			customElements.NewCenterRichText("确定要删除当前题库吗？"),
			func(confirm bool) bool {
				if confirm {
					DeleteBank(w, state)
					refreshHomeUI()
				}
				return true
			}, w)
	}))

	grid.Add(makeCustomGridBtn("📚", "顺序练习", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		practiceRelated.ShowTypeSelection(w, state, "顺序练习", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("🎲", "随机练习", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		practiceRelated.ShowTypeSelection(w, state, "随机练习", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("📝", "修正题目", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		practiceRelated.ShowTypeSelection(w, state, "修正题目", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("❌", "错题练习", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.WrongSet) == 0 {
			customElements.ShowCustomInformation("提示", "本题库没有错题集", w)
			return
		}
		practiceRelated.ShowTypeSelection(w, state, "错题练习", func() { ShowHome(w, state) })

	}))

	grid.Add(makeCustomGridBtn("⭐", "收藏练习", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.FavSet) == 0 {
			customElements.ShowCustomInformation("提示", "本题库没有收藏集", w)
			return
		}
		practiceRelated.ShowTypeSelection(w, state, "收藏练习", func() { ShowHome(w, state) })

	}))

	grid.Add(makeCustomGridBtn("🔍", "搜索题库", func() {
		if len(state.Questions) == 0 {
			customElements.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		practiceRelated.ShowSearchPage(w, state, func() {
			ShowHome(w, state)
		})
	}))

	// New feature entry buttons — only visible for admin and above
	if state.IsAdmin {
		grid.Add(makeCustomGridBtn("👥", "用户管理", func() {
			managementRelated.ShowUserManage(w, state, func() {
				ShowHome(w, state)
			})
		}))
	}

	// Login/Logout button — 顶部全宽栏
	var topLoginBar *fyne.Container
	var updateTopLoginBar func()

	updateTopLoginBar = func() {

		// 修复重点：改用标准的 HBox 配合透明矩形作为间隔，防止网格高度压缩崩塌

		rightButtons := container.NewGridWithColumns(3)

		// All logged-in users can access "题库管理"
		bankManageBtn := customElements.CreateButton(
			"题库管理",
			130,
			core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.ColorSelectedBorder),
			core.HexColor(core.BtnSecondaryBg),
			core.StrokeMedium,
			core.FontSizeSubtitle,
			true,
			false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				managementRelated.ShowBankManage(w, state, func() {
					ShowHome(w, state)
				})
			},
		)

		rightButtons.Add(bankManageBtn)

		if state.IsAdmin {
			// Admin: show "推送考试" button
			pushExamBtn := customElements.CreateButton(
				"推送考试",
				130, core.ActionBtnHeight,
				core.HexColor(core.CardBgColor),
				core.HexColor("#8C27B0"),
				core.HexColor("#8C27B0"),
				core.StrokeMedium, core.FontSizeSubtitle,
				true, false,
				fyne.TextAlignCenter,
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {

					managementRelated.ShowAdminExamCreate(w, state, func() {
						ShowHome(w, state)
					})
				})
			rightButtons.Add(pushExamBtn)
		} else {
			// Regular user: show "我的考试" button
			myExamsBtn := customElements.CreateButton(
				"我的考试",
				130, core.ActionBtnHeight,
				core.HexColor(core.CardBgColor),
				core.HexColor("#4CAF50"),
				core.HexColor("#4CAF50"),
				core.StrokeMedium, core.FontSizeSubtitle,
				true, false,
				fyne.TextAlignCenter,
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					managementRelated.ShowMyExams(w, state, func() {
						ShowHome(w, state)
					})
				})
			rightButtons.Add(myExamsBtn)
		}

		logoutBtn := customElements.CreateButton(
			"退出 ("+core.TruncateUsername(state.Username)+")",
			120,
			core.ActionBtnHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor("#FF6B6B"),
			core.HexColor("#FF6B6B"),
			core.StrokeMedium,
			core.FontSizeSmall,
			true,
			false,
			fyne.TextAlignCenter,
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				customElements.ShowCustomConfirm(
					"提示",
					"确定",
					"取消",
					customElements.NewCenterRichText("确定要退出登录吗？"),
					func(confirm bool) bool {
						if confirm {
							state.Logout(func(success bool, msg string) {
								if success {
									// Re-render the login page so the user sees it after logout
									ShowLoginPage(w, state, func() {
										ShowHome(w, state)
									})
								} else {
									customElements.ShowCustomInformation("错误", msg, w)
								}
							})
						}
						return true
					}, w)
			},
		)

		rightButtons.Add(logoutBtn)

		topLoginBar = container.NewBorder(rightButtons, nil, nil, nil, layout.NewSpacer())
	}

	// Initialize login UI state
	updateTopLoginBar()

	currentFileTitleLbl := widget.NewLabelWithStyle("当前选中的题库:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	currentFileNameLbl := widget.NewLabel("未选择题库")
	currentFileNameLbl.TextStyle = fyne.TextStyle{Italic: true}
	currentFileNameLbl.Wrapping = fyne.TextWrapBreak

	listTitle := widget.NewLabelWithStyle("已存储的文件列表 (点击切换):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	topAndMiddleBox := container.NewVBox(
		customElements.GetTitle(),
		topLoginBar,
		widget.NewSeparator(),
		grid,
		widget.NewSeparator(),
		currentFileTitleLbl,
		currentFileNameLbl,
	)

	fileListContainer := container.NewVBox()

	// 1. 创建原生的 Scroll 容器
	scrollList := container.NewScroll(fileListContainer)

	// 2. 使用局部无阴影主题包裹
	customTheme := &gz_theme.NoShadowTheme{Theme: theme.DefaultTheme()}
	noShadowScrollWrapper := container.NewThemeOverride(scrollList, customTheme)

	header := container.NewVBox(
		widget.NewSeparator(),
		listTitle,
	)

	// 3. 将 noShadowScrollWrapper 放在 Center (第五个参数)，使其自动向上/下填充剩余全部空间
	bottomBox := container.NewBorder(header, nil, nil, nil, noShadowScrollWrapper)

	bottomPadding := canvas.NewRectangle(color.Transparent)
	bottomPadding.SetMinSize(fyne.NewSize(0, core.BottomPaddingHeight))

	bottomWrapper := container.NewBorder(nil, bottomPadding, nil, nil, bottomBox)

	mainLayout := container.NewBorder(topAndMiddleBox, bottomWrapper, nil, nil, layout.NewSpacer())

	refreshHomeUI = func() {
		fmt.Printf("[REFRESH_UI] START: state.StoredFiles=%v, len=%d\n", state.StoredFiles, len(state.StoredFiles))
		updateTopLoginBar()

		topAndMiddleBox.Objects[1] = topLoginBar
		topAndMiddleBox.Refresh()

		if state.CurrentFileName != "" {
			currentFileNameLbl.SetText("📂 " + state.CurrentFileName)
		} else {
			currentFileNameLbl.SetText("未选择题库")
		}

		totalCount := len(state.StoredFiles)
		listTitle.SetText(fmt.Sprintf("已存储的文件列表 (当前有 %d 个本地文件):", totalCount))
		fmt.Printf("[REFRESH_UI] rendering %d files from StoredFiles\n", totalCount)

		fileListContainer.Objects = nil

		if totalCount == 0 {
			noDataText := canvas.NewText("⚠️ 暂无本地存储题库 (手机环境无缓存)", core.HexColor(core.ColorWrongBorder))
			noDataText.Alignment = fyne.TextAlignCenter
			fileListContainer.Add(noDataText)
		}

		for _, key := range state.StoredFiles {
			fileKey := key
			showName := parser.ExtractFileName(fileKey)

			rowColor := &struct {
				TextColor   string
				BgColor     string
				StrokeColor string
			}{}
			if fileKey == state.CurrentStorageKey {
				rowColor = &struct {
					TextColor   string
					BgColor     string
					StrokeColor string
				}{
					TextColor:   core.TextPrimaryColor,
					BgColor:     core.ColorSelectedBg,
					StrokeColor: core.ColorSelectedBorder,
				}
			} else {
				rowColor = &struct {
					TextColor   string
					BgColor     string
					StrokeColor string
				}{
					TextColor:   core.TextHintColor,
					BgColor:     core.CardBgColor,
					StrokeColor: core.BorderLightColor,
				}
			}

			clickableRow := customElements.CreateButton(
				"📄  "+showName,
				-1,
				core.FileListItemHeight,
				core.HexColor(rowColor.TextColor),
				core.HexColor(rowColor.BgColor),
				core.HexColor(rowColor.StrokeColor),
				core.StrokeMedium,
				core.FontSizeSmall,
				false, false,
				fyne.TextAlignLeading,
				fyne.TextWrapBreak,
				fyne.TextTruncateOff,
				func() {
					storageRelated.LoadAndRenderBank(w, state, fileKey)
					refreshHomeUI()
				},
			)

			mLeft := canvas.NewRectangle(color.Transparent)
			mLeft.SetMinSize(fyne.NewSize(6, 0))

			mRight := canvas.NewRectangle(color.Transparent)
			mRight.SetMinSize(fyne.NewSize(10, 0))

			itemWithMargin := container.NewBorder(nil, nil, mLeft, mRight, clickableRow)
			fileListContainer.Add(itemWithMargin)
		}

		// =================【核心修复逻辑】=================
		// 计算列表内容的自然渲染高度（包含行间距）
		contentRealHeight := float32(totalCount) * (core.FileListItemHeight + 4)
		if totalCount == 0 {
			contentRealHeight = 40.0
		}

		if contentRealHeight > core.MaxFileListHeight {
			// 情况 A：文件太多，超过最大高度限制，限制滚动框高度并开启垂直滚动
			scrollList.SetMinSize(fyne.NewSize(0, core.MaxFileListHeight))
			scrollList.Direction = container.ScrollVerticalOnly
		} else {
			// 情况 B：空间充裕或文件较少，将 MinSize 设置为内容的真实高度（不挤压不限制），
			// 并将 Direction 设为 ScrollNone，彻底切断底层手势响应，不再有任何滑动/拖拽感！
			scrollList.SetMinSize(fyne.NewSize(0, contentRealHeight))
			scrollList.Direction = container.ScrollNone
		}
		// =================================================

		fileListContainer.Refresh()
		scrollList.Refresh()
		mainLayout.Refresh()
	}

	refreshHomeUI()

	wBackground := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	wRootLayout := container.NewStack(
		wBackground,
		container.NewPadded(mainLayout),
	)

	w.SetContent(wRootLayout)
}

// DeleteBank 执行删除题库并智能切换游标到下一个题库。
func DeleteBank(w fyne.Window, state *core.AppState) {
	targetStorageKey := state.CurrentStorageKey
	targetIndex := -1
	for i, key := range state.StoredFiles {
		if key == targetStorageKey {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return
	}

	storageDir := storageRelated.GetStorageDir()
	err := os.Remove(filepath.Join(storageDir, targetStorageKey+".json"))
	if err != nil && !os.IsNotExist(err) {
		dialog.ShowInformation("删除错误", fmt.Sprintf("删除题库错误: %v\n", err), w)
		return
	}

	if state.CurrentFileName != "" {
		files, readErr := os.ReadDir(storageDir)
		if readErr == nil {
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
					continue
				}
				fileName := f.Name()
				// 检查文件名前缀是否匹配，并且包含指定的衍生关键字
				if strings.HasPrefix(fileName, state.CurrentFileName+"_") &&
					(strings.Contains(fileName, "错题") || strings.Contains(fileName, "收藏") || strings.Contains(fileName, "记录")) {

					_ = os.Remove(filepath.Join(storageDir, fileName))
				}
			}
		}
	}

	state.StoredFiles = append(state.StoredFiles[:targetIndex], state.StoredFiles[targetIndex+1:]...)

	if len(state.StoredFiles) > 0 {
		var nextBankKey string

		if targetIndex < len(state.StoredFiles) {
			nextBankKey = state.StoredFiles[targetIndex]
		} else {
			nextBankKey = state.StoredFiles[len(state.StoredFiles)-1]
		}

		storageRelated.LoadAndRenderBank(w, state, nextBankKey)

	} else {
		state.CurrentStorageKey = ""
		state.CurrentFileName = "未选择题库"
		state.Questions = nil
		state.CurrentList = nil

		// 务必同步清空偏好记录，防止下次打开 APP 时报错
		fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", "")

	}
}
