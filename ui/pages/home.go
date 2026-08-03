package pages

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/export"
	"github.com/gzjjjfree/practice/gz_storage"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/ui/widgets"
)

// RefreshLocalFilesList 扫描本地沙盒目录中的 .json 题库文件，更新已存储文件列表。
func RefreshLocalFilesList(state *core.AppState) {
	storageDir := gz_storage.GetStorageDir()
	files, err := os.ReadDir(storageDir)
	if err != nil {
		return
	}

	type FileInfo struct {
		Key       string
		Timestamp int64
	}

	var fileList []FileInfo

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			// 屏蔽掉错题集和收藏集，防止它们作为独立题库被展示
			if strings.Contains(f.Name(), "错题") || strings.Contains(f.Name(), "收藏") || strings.Contains(f.Name(), "答题记录") {
				continue
			}

			name := strings.TrimSuffix(f.Name(), ".json")
			// 假设 key 格式为: 类型Data_文件名_时间戳
			parts := strings.Split(name, "_")
			if len(parts) >= 3 {
				tsStr := parts[len(parts)-1]
				ts, _ := strconv.ParseInt(tsStr, 10, 64)
				fileList = append(fileList, FileInfo{Key: name, Timestamp: ts})
			}
		}
	}

	// 全局按时间戳从小到大排序，然后倒序
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Timestamp < fileList[j].Timestamp
	})

	// 反转列表，确保最新的在最上面
	var finalKeys []string
	for i := len(fileList) - 1; i >= 0; i-- {
		finalKeys = append(finalKeys, fileList[i].Key)
	}

	state.StoredFiles = finalKeys
}

// ShowHome 渲染首页，包含题库操作按钮和已存储文件列表。
func ShowHome(w fyne.Window, state *core.AppState) {
	var refreshHomeUI func()

	makeCustomGridBtn := func(icon, text string, action func()) *widgets.ClickableBox {
		iconLbl := canvas.NewText(icon, core.HexColor(core.TextBodyColor))
		iconLbl.TextSize = 24
		iconLbl.Alignment = fyne.TextAlignCenter

		textLbl := canvas.NewText(text, core.HexColor(core.TextBodyColor))
		textLbl.TextSize = 12
		textLbl.Alignment = fyne.TextAlignCenter

		btnBg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
		btnBg.CornerRadius = 6
		btnBg.StrokeColor = core.HexColor(core.BorderMediumColor)
		btnBg.StrokeWidth = 1

		boxContent := container.NewVBox(layout.NewSpacer(), iconLbl, textLbl, layout.NewSpacer())
		stack := container.NewStack(btnBg, container.NewPadded(boxContent))
		return widgets.NewClickableBox(stack, action)
	}

	grid := container.NewGridWithColumns(3)

	grid.Add(makeCustomGridBtn("📁", "读取", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			originalFileName := reader.URI().Name()

			fileBytes, readErr := io.ReadAll(reader)
			reader.Close()

			if readErr != nil {
				dialog.ShowError(fmt.Errorf("流读取失败: %v", readErr), w)
				return
			}

			storageDir := gz_storage.GetStorageDir()

			files, _ := os.ReadDir(storageDir)
			var oldFileNames []string

			searchPattern := "_" + originalFileName + "_"
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
					if strings.Contains(f.Name(), searchPattern) {
						oldFileNames = append(oldFileNames, f.Name())
					}
				}
			}

			processAndSave := func(isNewAlias bool) {
				fileName := originalFileName

				if isNewAlias {
					ext := filepath.Ext(fileName)
					baseName := strings.TrimSuffix(fileName, ext)
					counter := 1
					for {
						testFileName := fmt.Sprintf("%s(%d)%s", baseName, counter, ext)
						testPattern := "_" + testFileName + "_"
						exists := false
						for _, f := range files {
							if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
								if strings.Contains(f.Name(), testPattern) {
									exists = true
									break
								}
							}
						}
						if !exists {
							fileName = testFileName // 找到了没有被使用的名字
							break
						}
						counter++
					}
				} else {
					for _, oldName := range oldFileNames {
						_ = os.Remove(filepath.Join(storageDir, oldName))
					}
				}

				progDialog := dialog.NewCustomWithoutButtons("读取中", widget.NewProgressBarInfinite(), w)
				progDialog.Show()

				if strings.ToLower(filepath.Ext(fileName)) == ".xls" {
					newBytes, convErr := parser.ConvertXlsToXlsxBytes(fileBytes)
					if convErr == nil {
						fileBytes = newBytes
						fileName = strings.TrimSuffix(fileName, ".xls") + ".xlsx"
						fmt.Println("检测到旧版 XLS，已自动无损升级为 XLSX")
					}
				}

				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				fileExt := strings.ToLower(parser.GetFileExtension(fileName))
				storageKey := fmt.Sprintf("%sData_%s_%d", fileExt, fileName, timestamp)

				// 触发内存解析引擎
				bankData, parseErr := parser.ParseBytesToBank(fileBytes, fileName, storageKey)
				fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", storageKey)

				progDialog.Hide()

				if parseErr != nil {
					dialog.ShowError(fmt.Errorf("解析异常: %v", parseErr), w)
					return
				}

				// 转换为真正的 JSON 文本再落盘
				jsonData, marshalErr := json.Marshal(bankData)
				if marshalErr != nil {
					dialog.ShowError(fmt.Errorf("转换为JSON失败: %v", marshalErr), w)
					return
				}

				absoluteSavePath := filepath.Join(storageDir, storageKey+".json")
				err = os.WriteFile(absoluteSavePath, jsonData, 0644)
				if err != nil {
					dialog.ShowError(fmt.Errorf("存储本地失败: %v", err), w)
					return
				}

				state.CurrentStorageKey = bankData.StorageKey
				state.CurrentFileName = bankData.DisplayName

				parser.SyncQuestionsToState(state, bankData)
				RefreshLocalFilesList(state)
				refreshHomeUI()
			}

			if len(oldFileNames) > 0 {
				var confirmDialog dialog.Dialog

				cancelBtn := widget.NewButton(" 取消 ", func() {
					confirmDialog.Hide()
				})

				overwriteBtn := widget.NewButton(" 覆盖 ", func() {
					confirmDialog.Hide()
					processAndSave(false)
				})
				overwriteBtn.Importance = widget.DangerImportance

				addBtn := widget.NewButton(" 新增 ", func() {
					confirmDialog.Hide()
					processAndSave(true)
				})
				addBtn.Importance = widget.HighImportance

				buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, overwriteBtn, addBtn, layout.NewSpacer())

				msgLbl := widget.NewLabel(fmt.Sprintf("题库列表中已存在名为【%s】的文件。\n\n请选择接下来的操作：", originalFileName))
				msgLbl.Alignment = fyne.TextAlignCenter
				msgLbl.Wrapping = fyne.TextWrapWord

				content := container.NewVBox(
					msgLbl,
					container.NewGridWrap(fyne.NewSize(1, 20)),
					container.NewPadded(container.NewCenter(buttons)),
				)

				minWidthBox := container.NewGridWrap(fyne.NewSize(300, 1))

				styledContent := container.NewStack(
					minWidthBox,
					container.NewPadded(content),
				)

				confirmDialog = dialog.NewCustomWithoutButtons("文件已存在", styledContent, w)
				confirmDialog.Show()

			} else {
				processAndSave(false)
			}

		}, w)

		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx", ".xls", ".txt", ".json"}))
		fileDialog.Show()
	}))

	grid.Add(makeCustomGridBtn("📤", "导出", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		// 触发导出格式选择弹窗
		export.ShowExportDialog(w, state)
	}))

	grid.Add(makeCustomGridBtn("🗑️", "删除", func() {
		if state.CurrentStorageKey == "" {
			core.ShowCustomInformation("提示", "当前未选中任何题库", w)
			return
		}
		core.ShowCustomConfirm("提示", "确定要删除当前题库吗？", func(b bool) {
			if b {
				DeleteBank(w, state)
				refreshHomeUI()
			}
		}, w)
	}))

	grid.Add(makeCustomGridBtn("📚", "顺序练习", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		ShowTypeSelection(w, state, "顺序练习", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("🎲", "随机练习", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		ShowTypeSelection(w, state, "随机练习", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("📝", "修正题目", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}
		ShowTypeSelection(w, state, "修正题目", func() { ShowHome(w, state) })
	}))

	grid.Add(makeCustomGridBtn("❌", "错题练习", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.WrongSet) == 0 {
			core.ShowCustomInformation("提示", "本题库没有错题集", w)
			return
		}
		ShowTypeSelection(w, state, "错题练习", func() { ShowHome(w, state) })

	}))

	grid.Add(makeCustomGridBtn("⭐", "收藏练习", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.FavSet) == 0 {
			core.ShowCustomInformation("提示", "本题库没有收藏集", w)
			return
		}
		ShowTypeSelection(w, state, "收藏练习", func() { ShowHome(w, state) })

	}))

	grid.Add(makeCustomGridBtn("🔍", "搜索本题库", func() {
		if len(state.Questions) == 0 {
			core.ShowCustomInformation("提示", "请先导入题库", w)
			return
		}

		ShowSearchPage(w, state, func() {
			ShowHome(w, state)
		})
	}))

	currentFileTitleLbl := widget.NewLabelWithStyle("当前选中的题库:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	currentFileNameLbl := widget.NewLabel("未选择题库")
	currentFileNameLbl.TextStyle = fyne.TextStyle{Italic: true}
	currentFileNameLbl.Wrapping = fyne.TextWrapBreak

	listTitle := widget.NewLabelWithStyle("已存储的文件列表 (点击切换):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	topAndMiddleBox := container.NewVBox(
		widgets.GetTitle(),
		widget.NewSeparator(),
		grid,
		widget.NewSeparator(),
		currentFileTitleLbl,
		currentFileNameLbl,
	)

	fileListContainer := container.NewVBox()

	scrollList := container.NewScroll(fileListContainer)
	scrollList.SetMinSize(fyne.NewSize(0, 50))

	header := container.NewVBox(
		widget.NewSeparator(),
		listTitle,
	)

	bottomBox := container.NewBorder(header, nil, nil, nil, scrollList)

	bottomPadding := canvas.NewRectangle(color.Transparent)
	bottomPadding.SetMinSize(fyne.NewSize(0, core.BottomPaddingHeight))

	bottomWrapper := container.NewBorder(nil, bottomPadding, nil, nil, bottomBox)

	mainLayout := container.NewBorder(topAndMiddleBox, bottomWrapper, nil, nil, layout.NewSpacer())

	refreshHomeUI = func() {
		if state.CurrentFileName != "" {
			currentFileNameLbl.SetText("📂 " + state.CurrentFileName)
		} else {
			currentFileNameLbl.SetText("未选择题库")
		}

		totalCount := len(state.StoredFiles)
		listTitle.SetText(fmt.Sprintf("已存储的文件列表 (当前有 %d 个本地文件):", totalCount))

		fileListContainer.Objects = nil

		if totalCount == 0 {
			noDataText := canvas.NewText("⚠️ 暂无本地存储题库 (手机环境无缓存)", core.HexColor(core.ColorWrongBorder))
			noDataText.Alignment = fyne.TextAlignCenter
			fileListContainer.Add(noDataText)
		}

		for _, key := range state.StoredFiles {
			fileKey := key
			showName := parser.ExtractFileName(fileKey)

			bg := canvas.NewRectangle(color.Transparent)
			bg.CornerRadius = 5
			textItem := canvas.NewText(showName, color.Black)
			textItem.TextSize = 14

			if fileKey == state.CurrentStorageKey {
				bg.FillColor = core.HexColor(core.ColorSelectedBg)
				bg.StrokeColor = core.HexColor(core.ColorSelectedBorder)
				bg.StrokeWidth = 1.5
				textItem.Color = core.HexColor(core.TextPrimaryColor)
			} else {
				bg.FillColor = core.HexColor(core.PageBgColor)
				bg.StrokeColor = core.HexColor(core.BorderLightColor)
				bg.StrokeWidth = 1
				textItem.Color = core.HexColor(core.TextHintColor)
			}

			textItem.Refresh()
			bg.Refresh()

			textCenter := widget.NewLabel(" " + textItem.Text + " ")
			textCenter.Wrapping = fyne.TextWrapBreak

			rowStack := container.NewStack(bg, textCenter)

			clickableRow := widgets.NewClickableBox(rowStack, func() {
				LoadAndRenderBank(w, state, fileKey)

				// 4. 重绘首页
				refreshHomeUI()
			})

			fileListContainer.Add(clickableRow)
		}

		desiredHeight := float32(totalCount) * core.FileListItemHeight
		if totalCount == 0 {
			desiredHeight = 40.0
		}

		if desiredHeight > core.MaxFileListHeight {
			desiredHeight = core.MaxFileListHeight
		}

		scrollList.SetMinSize(fyne.NewSize(0, desiredHeight))

		fileListContainer.Refresh()
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

// LoadAndRenderBank 从本地加载指定题库并刷新界面状态。
func LoadAndRenderBank(w fyne.Window, state *core.AppState, targetKey string) {
	state.CurrentStorageKey = targetKey
	state.CurrentFileName = parser.ExtractFileName(targetKey)

	storageDir := gz_storage.GetStorageDir()
	absoluteReadPath := filepath.Join(storageDir, targetKey+".json")

	bytesData, err := os.ReadFile(absoluteReadPath)
	if err == nil {
		var bankData parser.BankData
		if err := json.Unmarshal(bytesData, &bankData); err == nil {
			parser.SyncQuestionsToState(state, &bankData)
			gz_storage.LoadRecordsForCurrentBank(state)
			gz_storage.LoadPracticeRecordsFromLocal(state)
			fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", state.CurrentStorageKey)
		} else {
			dialog.ShowInformation("错误", fmt.Sprintf("JSON反序列化失败: %v\n", err), w)
		}
	} else {
		dialog.ShowInformation("错误", fmt.Sprintf("读取沙盒题库文件失败: %v\n", err), w)
	}
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

	storageDir := gz_storage.GetStorageDir()
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

		LoadAndRenderBank(w, state, nextBankKey)

	} else {
		state.CurrentStorageKey = ""
		state.CurrentFileName = "未选择题库"
		state.Questions = nil
		state.CurrentList = nil

		// 务必同步清空偏好记录，防止下次打开 APP 时报错
		fyne.CurrentApp().Preferences().SetString("LastOpenedBankKey", "")

	}
}
