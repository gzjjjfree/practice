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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
	"github.com/gzjjjfree/practice/storageRelated"
)


// ==================== Step 1: 题库来源选择 ====================

// buildSourcePage 构建题库来源选择页面，包含本地题库和服务器题库两个按钮。
//
// 功能说明：
// 构建考试创建向导的第一步页面，允许用户选择题库来源：本地题库或服务器题库。
// 点击“本地题库”按钮会弹出本地题库选择对话框，从本地文件加载题库数据；
// 点击“服务器题库”按钮会切换到服务器题库来源模式。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：题库来源选择页面的UI容器对象。
func buildSourcePage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	var localBtn, serverBtn *customElements.CustomButton

	// 本地题库按钮（标识符：localBtn，名称："本地题库"）：点击后弹出本地题库选择对话框，从本地文件加载题库数据供用户选择题目。
	localBtn = customElements.CreateButton(
		core.AdminLocalBankBtnLabel,
		core.BtnLargeWidth, core.BtnLargeHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.AdminButtonStrokeWidth, core.AdminLocalBtnFontSize, true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			storageDir := storageRelated.GetStorageDir()
			files, err := os.ReadDir(storageDir)
			if err != nil {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminReadFileErrMsg+err.Error(), w)
				return
			}

			type LocalBankItem struct {
				Key       string
				Timestamp int64
			}

			var fileList []LocalBankItem
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
					continue
				}
				if strings.Contains(f.Name(), core.BankExcludeSubstring1) ||
					strings.Contains(f.Name(), core.BankExcludeSubstring2) ||
					strings.Contains(f.Name(), core.BankExcludeSubstring3) {
					continue
				}
				name := strings.TrimSuffix(f.Name(), ".json")

				if !strings.HasPrefix(name, core.BankFilePrefixXlsx) &&
					!strings.HasPrefix(name, core.BankFilePrefixXls) &&
					!strings.HasPrefix(name, core.BankFilePrefixTxt) &&
					!strings.HasPrefix(name, core.BankFilePrefixTxts) &&
					!strings.HasPrefix(name, core.BankFilePrefixJson) {
					continue
				}

				parts := strings.Split(name, "_")
				if len(parts) >= 3 {
					tsStr := parts[len(parts)-1]
					ts, _ := strconv.ParseInt(tsStr, 10, 64)
					fileList = append(fileList, LocalBankItem{Key: name, Timestamp: ts})
				} else {
					fileList = append(fileList, LocalBankItem{Key: name, Timestamp: time.Now().UnixNano() / int64(time.Millisecond)})
				}
			}

			if len(fileList) == 0 {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminNoLocalBankMsg, w)
				return
			}

			sort.Slice(fileList, func(i, j int) bool {
				return fileList[i].Timestamp > fileList[j].Timestamp
			})

			selectLabel := widget.NewLabel(core.AdminSelectLocalBankMsg)
			selectLabel.Wrapping = fyne.TextWrapWord
			selectLabel.TextStyle = fyne.TextStyle{Bold: true}

			var checkObjs []fyne.CanvasObject
			var checkedKeys []string
			for _, item := range fileList {
				showName := parser.ExtractFileName(item.Key)
				cb := widget.NewCheck(showName, func(checked bool) {
					if checked {
						checkedKeys = append(checkedKeys, item.Key)
					} else {
						for j, k := range checkedKeys {
							if k == item.Key {
								checkedKeys = append(checkedKeys[:j], checkedKeys[j+1:]...)
								break
							}
						}
					}
				})
				checkObjs = append(checkObjs, cb)
			}

			checkVBox := container.NewVBox(checkObjs...)
			checkScroll := container.NewVScroll(checkVBox)
			checkScroll.SetMinSize(fyne.NewSize(core.LocalBankDialogWidth, core.LocalBankDialogHeight))

			var selDialog dialog.Dialog
			// 确认选择按钮（标识符：confirmBtn，名称："确认"）：确认选中的本地题库并加载题目数据到本地题库列表中。
			confirmBtn := customElements.CreateButton(
				core.AdminConfirmText,
				core.DialogActionBtnWidth,
				core.DialogActionBtnHeight,
				core.HexColor(core.CardBgColor),
				core.HexColor(core.BtnPrimaryBg),
				core.HexColor(core.BtnPrimaryBg),
				core.AdminButtonStrokeWidth,
				core.AdminDialogBtnFontSize,
				true, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					if selDialog != nil {
						selDialog.Hide()
					}
					ws.sourceTab = 0
					ws.localBankKeys = checkedKeys
					ws.localQuestions = nil

					for _, key := range checkedKeys {
						filePath := fmt.Sprintf("%s/%s.json", storageDir, key)
						data, err := os.ReadFile(filePath)
						if err != nil {
							customElements.ShowCustomInformation(core.AdminErrorText, core.AdminReadFileErrMsg+err.Error(), w)
							return
						}
						var bankData parser.BankData
						if err := json.Unmarshal(data, &bankData); err != nil {
							customElements.ShowCustomInformation(core.AdminErrorText, core.AdminParseJSONErrMsg, w)
							return
						}
						bankQuestions := convertBankDataToQuestions(&bankData)
						ws.localQuestions = append(ws.localQuestions, bankQuestions...)
					}
					updateSourceSelection(ws, localBtn, serverBtn, w, state)
				})

			// 取消按钮（标识符：cancelBtn，名称："取消"）：关闭本地题库选择对话框，不执行任何加载操作。
			cancelBtn := customElements.CreateButton(
				core.AdminCancelText,
				core.DialogActionBtnWidth,
				core.DialogActionBtnHeight,
				core.HexColor(core.TextBodyColor),
				core.HexColor(core.CardBgColor),
				core.HexColor(core.BorderLightColor),
				core.AdminCancelBtnStrokeWidth,
				core.AdminDialogBtnFontSize,
				false, false,
				fyne.TextAlignCenter, // 👈 居中对齐
				fyne.TextWrapOff,     // 👈 不换行
				fyne.TextTruncateOff, // 👈 不换行（截断）
				func() {
					if selDialog != nil {
						selDialog.Hide()
					}
				})

			btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), confirmBtn)
			selContent := container.NewVBox(
				selectLabel,
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
				container.NewCenter(checkScroll),
				container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)),
				container.NewCenter(btnRow),
			)

			selDialog = dialog.NewCustomWithoutButtons(core.AdminSelectLocalBankTitle, selContent, w)
			selDialog.Show()
		},
	)

	// 服务器题库按钮（标识符：serverBtn，名称："服务器题库"）：点击后弹出服务器题库选择对话框，从服务器加载题库数据。
	serverBtn = customElements.CreateButton(
		core.AdminServerBankBtnLabel,
		core.BtnLargeWidth, core.BtnLargeHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnSecondaryBg),
		core.HexColor(core.BtnSecondaryBg),
		core.AdminButtonStrokeWidth, core.AdminLocalBtnFontSize, false, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			// 从服务器获取题库列表
			state.FetchServerBankList(func(success bool, banks []network.ServerBankItem, msg string) {
				if !success {
					customElements.ShowCustomInformation(core.AdminErrorText, msg, w)
					return
				}

				if len(banks) == 0 {
					customElements.ShowCustomInformation(core.AdminInfoText, "暂无服务器题库", w)
					return
				}

				selectLabel := widget.NewLabel("请选择服务器题库")
				selectLabel.Wrapping = fyne.TextWrapWord
				selectLabel.TextStyle = fyne.TextStyle{Bold: true}

				var checkObjs []fyne.CanvasObject
				var checkedBankIDs []int
				for _, bank := range banks {
					showName := bank.DisplayName
					cb := widget.NewCheck(showName, func(checked bool) {
						if checked {
							checkedBankIDs = append(checkedBankIDs, bank.BankID)
						} else {
							for j, id := range checkedBankIDs {
								if id == bank.BankID {
									checkedBankIDs = append(checkedBankIDs[:j], checkedBankIDs[j+1:]...)
									break
								}
							}
						}
					})
					checkObjs = append(checkObjs, cb)
				}

				checkVBox := container.NewVBox(checkObjs...)
				checkScroll := container.NewVScroll(checkVBox)
				checkScroll.SetMinSize(fyne.NewSize(core.LocalBankDialogWidth, core.LocalBankDialogHeight))

				var selDialog dialog.Dialog
				// 确认选择按钮（标识符：confirmBtn，名称："确认"）：确认选中的服务器题库并下载题目数据。
				confirmBtn := customElements.CreateButton(
					core.AdminConfirmText,
					core.DialogActionBtnWidth,
					core.DialogActionBtnHeight,
					core.HexColor(core.CardBgColor),
					core.HexColor(core.BtnPrimaryBg),
					core.HexColor(core.BtnPrimaryBg),
					core.AdminButtonStrokeWidth,
					core.AdminDialogBtnFontSize,
					true, false,
					fyne.TextAlignCenter, // 👈 居中对齐
					fyne.TextWrapOff,     // 👈 不换行
					fyne.TextTruncateOff, // 👈 不换行（截断）
					func() {
						if selDialog != nil {
							selDialog.Hide()
						}
						ws.sourceTab = 1
						ws.serverBankIDs = checkedBankIDs
						ws.serverQuestions = nil

						// 下载选中的服务器题库数据
						downloadCount := 0
						totalBanks := len(checkedBankIDs)

						for _, bankID := range checkedBankIDs {
							state.DownloadServerBankForExam(bankID, func(success bool, downloadMsg string) {
								if success && state.CurrentServerBank != nil {
									// 将服务器题目数据追加到 ws.serverQuestions
									for _, sq := range state.CurrentServerBank.Questions {
										ws.serverQuestions = append(ws.serverQuestions, sq)
									}
								} else {
									customElements.ShowCustomInformation(core.AdminErrorText, "下载题库失败: "+downloadMsg, w)
								}

								downloadCount++
								if downloadCount == totalBanks {
									updateSourceSelection(ws, localBtn, serverBtn, w, state)
								}
							})
						}
					},
				)

				// 取消按钮（标识符：cancelBtn，名称："取消"）：关闭服务器题库选择对话框，不执行任何加载操作。
				cancelBtn := customElements.CreateButton(
					core.AdminCancelText,
					core.DialogActionBtnWidth,
					core.DialogActionBtnHeight,
					core.HexColor(core.TextBodyColor),
					core.HexColor(core.CardBgColor),
					core.HexColor(core.BorderLightColor),
					core.AdminCancelBtnStrokeWidth,
					core.AdminDialogBtnFontSize,
					false, false,
					fyne.TextAlignCenter, // 👈 居中对齐
					fyne.TextWrapOff,     // 👈 不换行
					fyne.TextTruncateOff, // 👈 不换行（截断）
					func() {
						if selDialog != nil {
							selDialog.Hide()
						}
					},
				)

				btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), confirmBtn)
				selContent := container.NewVBox(
					selectLabel,
					container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH10)),
					container.NewCenter(checkScroll),
					container.NewGridWrap(fyne.NewSize(1, core.LayoutSpacingH15)),
					container.NewCenter(btnRow),
				)

				selDialog = dialog.NewCustomWithoutButtons("选择服务器题库", selContent, w)
				selDialog.Show()
			})
		},
	)

	sourceTitle := customElements.CreateLabel(core.AdminSourcePageTitle, core.HexColor(core.TextPrimaryColor), core.FontSizeMedium, true, true, false)
	sourceHint := canvas.NewText(core.AdminSourceHint, core.HexColor(core.TextSecondaryColor))
	sourceHint.TextSize = core.FontSizeSmall
	sourceHint.Alignment = fyne.TextAlignCenter

	sourceContainer := container.NewVBox(
		sourceTitle,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		sourceHint,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		container.NewHBox(layout.NewSpacer(), localBtn, serverBtn, layout.NewSpacer()),
	)

	return container.NewPadded(container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(sourceContainer),
		layout.NewSpacer(),
		layout.NewSpacer(),
	))
}

// updateSourceSelection 更新题库来源选择状态并刷新题目列表。
//
// 功能说明：
// 刷新本地题库和服务器题库按钮的视觉状态，并根据当前选中的题库来源刷新题目列表显示。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - localBtn: *customElements.CustomButton，本地题库按钮引用。
//   - serverBtn: *customElements.CustomButton，服务器题库按钮引用。
func updateSourceSelection(ws *AdminExamCreateState, localBtn, serverBtn *customElements.CustomButton, w fyne.Window, state *core.AppState) {
	localBtn.Refresh()
	serverBtn.Refresh()
	refreshQuestionList(ws)
	if ws.wizard != nil && len(ws.wizard.Items) > 1 {
		ws.wizard.Items[1].Content = buildExamParamsPage(ws, w, state)
		ws.wizard.Refresh()
	}
}

// refreshQuestionList 根据题库来源刷新题目列表显示。
//
// 功能说明：
// 根据当前考试参数确认状态、题库来源（本地或服务器），重新渲染并刷新题目列表的UI显示。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func refreshQuestionList(ws *AdminExamCreateState) {
	if ws.qListVBox == nil {
		return
	}
	ws.qListVBox.Objects = nil
	if !ws.examParamsConfirmed {
		noData := canvas.NewText(core.AdminParamsNotSetMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = 30
		ws.qListVBox.Add(noData)
		ws.qListVBox.Refresh()
		return
	}
	if ws.sourceTab == 0 && len(ws.localQuestions) > 0 {
		renderLocalQuestions(ws)
	} else if ws.sourceTab == 1 && len(ws.serverQuestions) > 0 {
		renderServerQuestions(ws)
	} else {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = 14
		ws.qListVBox.Add(noData)
	}
	ws.qListVBox.Refresh()
}
