package managementRelated

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// checkIfQuestionsSelectedFull 检查是否已根据 ws.typeCounts 选择完所有必要题型的题目。
//
// 功能说明：
// 检查用户是否已经根据考试参数中配置的每种题型所需数量（ws.typeCounts），从本地题库和服务器题库中选择了足够数量的题目。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象，包含本地题目、服务器题目、已选题目ID集合以及各题型所需数量等信息。
//
// 返回值：
//   - bool：如果所有必要题型的题目都已选择足够数量，返回 true；否则返回 false。
//
// 核心业务逻辑：
//  1. 若 ws.typeCounts 为空，说明未配置题型数量要求，直接返回 false。
//  2. 遍历 ws.typeCounts 中的每种题型及其所需数量 required。
//  3. 对于每种题型，分别统计本地题目（ws.localQuestions）和服务器题目（ws.serverQuestions）中已被选中（ws.selectedQIDs）的题目数量。
//  4. 若某题型的已选数量小于所需数量 required，则返回 false。
//  5. 所有题型均满足数量要求时，返回 true。
func checkIfQuestionsSelectedFull(ws *AdminExamCreateState) bool {
	if ws == nil {
		fmt.Println("[DEBUG checkIfQuestionsSelectedFull] ws is nil")
		return false
	}
	// 如果 ws.typeCounts 为空，但 ws.typeEntries 存在，尝试从 typeEntries 中同步获取
	if len(ws.typeCounts) == 0 && ws.typeEntries != nil {
		ws.typeCounts = make(map[string]int)
		for t, entry := range ws.typeEntries {
			if entry != nil && entry.Text != "" {
				if val, err := strconv.Atoi(entry.Text); err == nil && val > 0 {
					ws.typeCounts[t] = val
				}
			}
		}
		fmt.Printf("[DEBUG checkIfQuestionsSelectedFull] Fallback populated typeCounts from typeEntries: %+v\n", ws.typeCounts)
	}

	if len(ws.typeCounts) == 0 {
		fmt.Println("[DEBUG checkIfQuestionsSelectedFull] ws.typeCounts is empty")
		return false
	}
	fmt.Printf("[DEBUG checkIfQuestionsSelectedFull] typeCounts: %+v, selectedQIDs count: %d\n", ws.typeCounts, len(ws.selectedQIDs))
	for qType, required := range ws.typeCounts {
		if required <= 0 {
			continue
		}
		count := 0
		// 统计本地题目
		for _, q := range ws.localQuestions {
			if q.Type == qType && ws.selectedQIDs[q.ID] {
				count++
			}
		}
		// 统计服务器题目
		for _, q := range ws.serverQuestions {
			qLocal := serverQuestionToLocal(&q)
			if qLocal.Type == qType && ws.selectedQIDs[qLocal.ID] {
				count++
			}
		}
		fmt.Printf("[DEBUG checkIfQuestionsSelectedFull] Type: %s, required: %d, actual selected count: %d\n", qType, required, count)
		if count < required {
			fmt.Printf("[DEBUG checkIfQuestionsSelectedFull] -> NOT FULL for type %s (count %d < required %d)\n", qType, count, required)
			return false
		}
	}
	fmt.Println("[DEBUG checkIfQuestionsSelectedFull] -> ALL FULL, returning true")
	return true
}

// ==================== 管理员推送考试页面 ====================

// AdminExamCreateState 跟踪多步向导的状态，包含题库来源、已选题目、考试参数、目标用户、模板相关及UI引用等信息。
//
// 字段说明：
//   - sourceTab: 题库来源标签页索引（0 = 本地题库, 1 = 服务器题库）
//   - localBankKeys: 选中的本地题库 StorageKey 列表
//   - localQuestions: 本地题库题目列表
//   - serverBankIDs: 选中的服务器题库 BankID 列表
//   - serverQuestions: 服务器题库题目列表
//   - selectedQIDs: 已选题目 ID 集合
//   - examName: 考试名称
//   - durationMin: 考试时长（分钟）
//   - startTime: 考试开始时间
//   - endTime: 考试结束时间
//   - selectedUsers: 已选目标用户 ID 集合
//   - selectedTemplateID: 选中的模板 ID（0 = 未选择）
//   - templateDetail: 模板详情（含每种题型的数量）
//   - typeCounts: 每种题型的自定义数量（从模板加载后可修改），key为题型名称
//   - examParamsConfirmed: 考试设置是否已确认
//   - qListVBox: 题目列表容器
//   - qListScroll: 题目列表滚动容器
//   - titleLabel: 选择题目页标题标签（用于动态更新）
//   - nameEntry: 考试名称输入框
//   - durationEntry: 考试时长输入框
//   - startTimeLbl: 开始时间显示标签
//   - endTimeLbl: 结束时间显示标签
//   - typeEntries: 每种题型数量输入框映射
//   - userVBox: 用户列表容器
//   - userScroll: 用户列表滚动容器
//   - createBtn: 创建推送按钮
//   - examParamsContent: 考试设置页面内容容器
//   - examParamsSummary: 考试设置统计信息容器
//   - wizard: 向导 TabContainer 引用
//   - templateRefreshFunc: 模板列表刷新函数
type AdminExamCreateState struct {
	sourceTab int // 0 = 本地题库, 1 = 服务器题库
	// 本地题库来源
	localBankKeys  []string // 选中的本地题库 StorageKey 列表
	localQuestions []core.Question
	// 服务器题库来源
	serverBankIDs   []int // 选中的服务器题库 BankID 列表
	serverQuestions []network.ServerQuestion
	// 已选题目
	selectedQIDs map[string]bool // 已选题目 ID 集合
	// 考试参数
	examName     string    // 考试名称
	durationMin  int       // 考试时长（分钟）
	startTime    time.Time // 开始时间
	endTime      time.Time // 结束时间
	startTimeStr string    // 开始时间字符串
	endTimeStr   string    // 结束时间字符串
	// 目标用户
	selectedUsers map[int]bool // 已选用户 ID 集合
	// 模板相关
	selectedTemplateID int                         // 选中的模板 ID（0 = 未选择）
	templateDetail     *network.TemplateDetailItem // 模板详情（含每种题型的数量）
	// 每种题型的自定义数量（从模板加载后可修改）
	typeCounts map[string]int // 每种题型的数量，key 为题型名称
	// 每种题型的分值
	typeScores map[string]float64 // 每种题型的分值，key 为题型名称
	// 题目类型过滤勾选状态（key为题型名称，true表示勾选显示）
	questionFilterTypes map[string]bool
	// 考试设置确认状态
	examParamsConfirmed bool // 考试设置是否已确认
	// UI 引用（用于动态更新）
	qListVBox           *fyne.Container              // 题目列表容器
	qListScroll         *container.Scroll            // 题目列表滚动容器
	titleLabel          *widget.Label                // 选择题目页标题标签（用于动态更新）
	nameEntry           *widget.Entry                // 考试名称输入框
	durationEntry       *widget.Entry                // 考试时长输入框
	startTimeLbl        *widget.Label                // 开始时间显示标签
	endTimeLbl          *widget.Label                // 结束时间显示标签
	typeEntries         map[string]*widget.Entry     // 每种题型数量输入框
	typeScoreEntries    map[string]*widget.Entry     // 每种题型分值输入框
	userVBox            *fyne.Container              // 用户列表容器
	userScroll          *container.Scroll            // 用户列表滚动容器
	createBtn           *customElements.CustomButton // 创建推送按钮
	examParamsContent   *fyne.Container              // 考试设置页面内容容器
	examParamsSummary   *fyne.Container              // 考试设置统计信息容器
	wizard              *container.AppTabs           // 向导 TabContainer 引用
	templateRefreshFunc func()                       // 模板列表刷新函数
	userPageRefreshFunc func()                       // 目标用户页刷新函数
	confirmPageRefreshFunc func()                    // 确认推送页刷新函数
}

// ShowAdminExamCreate 打开管理员考试创建向导。
//
// 功能说明：
// 初始化并展示管理员考试创建的六步向导页面，包括题库来源选择、考试参数设置、筛选条件配置、题目选择列表、目标用户选择和确认推送。
//
// 输入参数：
//   - w: fyne.Window，当前窗口对象，用于显示向导页面。
//   - state: *core.AppState，全局应用状态对象，用于管理用户数据、题库数据及API交互等。
//   - backToHome: func()，返回主页的回调函数，点击返回按钮时触发。
func ShowAdminExamCreate(w fyne.Window, state *core.AppState, backToHome func()) {
	ws := &AdminExamCreateState{
		sourceTab:        0,
		selectedQIDs:     make(map[string]bool),
		selectedUsers:    make(map[int]bool),
		typeScores:       make(map[string]float64),
		typeScoreEntries: make(map[string]*widget.Entry),
	}
	for _, t := range []string{"单选题", "多选题", "判断题", "填空题", "问答题"} {
		ws.typeScores[t] = 1.0
	}

	wizardPages := []fyne.CanvasObject{
		buildSourcePage(ws, w, state),
		buildExamParamsPage(ws, w, state),
		buildQuestionSelectPage(ws),
		buildUserSelectPage(ws, state),
		buildConfirmPage(ws, w, state),
		buildTemplateListPage(ws, w, state),
	}

	ws.wizard = container.NewAppTabs(
		container.NewTabItem(core.AdminTabSource, wizardPages[0]),
		container.NewTabItem(core.AdminTabExamParams, wizardPages[1]),
		container.NewTabItem(core.AdminTabSelect, wizardPages[2]),
		container.NewTabItem(core.AdminTabUser, wizardPages[3]),
		container.NewTabItem(core.AdminTabConfirm, wizardPages[4]),
		container.NewTabItem(core.AdminTabTemplates, wizardPages[5]),
	)
	// 移除 OnSelected 内部对 ws.wizard.Refresh() 以及动态赋值 Content 的复杂回调逻辑，避免在初始化渲染时触发 applyTheme/scroller 内部的 nil pointer dereference 异常。
	ws.wizard.OnSelected = func(item *container.TabItem) {
		switch item.Text {
		case core.AdminTabTemplates:
			if ws.templateRefreshFunc != nil {
				ws.templateRefreshFunc()
			}
		case core.AdminTabUser:
			// 当切换到目标用户页时，检查题目是否已选择完整，并刷新目标用户页
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		case core.AdminTabConfirm:
			// 当切换到确认推送页时，刷新确认推送页状态
			if ws.confirmPageRefreshFunc != nil {
				ws.confirmPageRefreshFunc()
			}
		}
	}

	// 返回按钮（标识符：backBtn，名称："返回"）：点击后返回主页，取消当前考试创建向导。
	backBtn := customElements.CreateButton(
		core.AdminBackBtnText,
		core.BackBtnWidth, core.BackBtnHeight,
		core.HexColor(core.TextPrimaryColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		1, 18, false, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			backToHome()
		},
	)

	titleCenter := customElements.CreateLabel(
		core.AdminExamCreateTitle,
		core.HexColor(core.TextPrimaryColor),
		core.FontSizePageTitle, true, true, false,
	)

	topBar := container.NewBorder(nil, nil, backBtn, nil, titleCenter)
	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))
	// 直接将 ws.wizard 放入 Border 布局，而不是用 VScroll 嵌套 AppTabs，避免 AppTabs 的 scroller 在未完全初始化时产生 nil pointer dereference
	mainLayout := container.NewBorder(topBar, nil, nil, nil, ws.wizard)
	wRootLayout := container.NewStack(pageBg, mainLayout)
	w.SetContent(wRootLayout)
	ws.wizard.SetTabLocation(container.TabLocationLeading)
}

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

	return container.NewPadded(sourceContainer)
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
		noData.TextSize = 14
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

// ==================== Step 2: 筛选条件 ====================

// ==================== Step 3: 题目选择列表 ====================

// buildQuestionSelectPage 构建题目选择列表页面，显示根据筛选条件过滤后的题目卡片。
//
// 功能说明：
// 构建考试创建向导的第四步页面，显示根据用户设置的筛选条件（题型、难度）和题库来源过滤后的题目卡片列表。
// 若考试参数尚未确认，则显示提示信息；否则渲染本地或服务器题目列表。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - fyne.CanvasObject：题目选择列表页面的UI容器对象。
func buildQuestionSelectPage(ws *AdminExamCreateState) fyne.CanvasObject {
	if !ws.examParamsConfirmed {
		return container.NewPadded(container.NewCenter(
			canvas.NewText(core.AdminParamsNotSetMsg, core.HexColor(core.TextMutedColor)),
		))
	}

	if ws.questionFilterTypes == nil {
		ws.questionFilterTypes = make(map[string]bool)
	}

	ws.qListVBox = container.NewVBox()
	ws.qListScroll = container.NewVScroll(ws.qListVBox)

	ws.titleLabel = buildQuestionSelectTitle(ws)

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	var checkObjs []fyne.CanvasObject
	for _, qType := range typeOrder {
		t := qType
		cb := widget.NewCheck(t, func(checked bool) {
			ws.questionFilterTypes[t] = checked
			refreshQuestionList(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		})
		cb.SetChecked(ws.questionFilterTypes[t])
		checkObjs = append(checkObjs, cb)
	}
	filterBox := container.NewGridWrap(fyne.NewSize(130, core.LayoutSpacingH20), checkObjs...)

	// 随机填满按钮（标识符：randomFillBtn，名称："🎲 随机填满"）：一键随机填满各题型所需数量。
	randomFillBtn := customElements.CreateButton(
		"🎲 随机填满",
		110, 32,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1, 14, true, false,
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		func() {
			randomFillQuestions(ws)
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		},
	)

	filterRow := container.NewBorder(nil, nil, filterBox, randomFillBtn)

	topContainer := container.NewVBox(
		ws.titleLabel,
		container.NewGridWrap(fyne.NewSize(1, 6)),
		filterRow,
		container.NewGridWrap(fyne.NewSize(1, 6)),
	)

	if ws.sourceTab == 0 && len(ws.localQuestions) > 0 {
		renderLocalQuestions(ws)
	} else if ws.sourceTab == 1 && len(ws.serverQuestions) > 0 {
		renderServerQuestions(ws)
	} else {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		ws.qListVBox.Add(noData)
	}

	return container.NewBorder(topContainer, nil, nil, nil, ws.qListScroll)
}

// buildQuestionSelectTitle 构建题目选择页面的标题标签，显示各题型已选/所需数量。
//
// 功能说明：
// 根据当前已选题目数量和模板配置的各题型所需数量，生成题目选择页面的标题文本，格式为"选择题目（单选题：x/y, 多选题：a/b, ...）"。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - *widget.Label：题目选择页面的标题标签对象。
func buildQuestionSelectTitle(ws *AdminExamCreateState) *widget.Label {
	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	var parts []string

	for _, qType := range typeOrder {
		count := 0
		for _, q := range ws.localQuestions {
			if q.Type == qType && ws.selectedQIDs[q.ID] {
				count++
			}
		}
		for _, q := range ws.serverQuestions {
			qLocal := serverQuestionToLocal(&q)
			if qLocal.Type == qType && ws.selectedQIDs[qLocal.ID] {
				count++
			}
		}
		required := 0
		if ws.typeCounts != nil {
			required = ws.typeCounts[qType]
		}
		if required > 0 {
			parts = append(parts, fmt.Sprintf("%s：%d/%d", qType, count, required))
		}
	}

	titleText := "选择题目（" + strings.Join(parts, ", ") + "）"
	title := widget.NewLabel(titleText)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Wrapping = fyne.TextWrapWord

	return title
}

// updateQuestionSelectTitle 更新题目选择页面的标题标签。
//
// 功能说明：
// 重新构建题目选择页面的标题标签，并更新UI中的标题文本、样式和换行属性。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func updateQuestionSelectTitle(ws *AdminExamCreateState) {
	if ws.titleLabel != nil {
		newTitle := buildQuestionSelectTitle(ws)
		ws.titleLabel.Text = newTitle.Text
		ws.titleLabel.TextStyle = newTitle.TextStyle
		ws.titleLabel.Wrapping = newTitle.Wrapping
		ws.titleLabel.Refresh()
	}
}

// renderLocalQuestions 渲染本地题库中的题目列表，按题型分组。
//
// 功能说明：
// 从本地题库中过滤题目，并按题型分组渲染题目卡片列表到UI容器中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func renderLocalQuestions(ws *AdminExamCreateState) {
	ws.qListVBox.Objects = nil

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	groupedMap := make(map[string][]core.Question)
	for _, q := range ws.localQuestions {
		groupedMap[q.Type] = append(groupedMap[q.Type], q)
	}

	for _, qType := range typeOrder {
		if !ws.questionFilterTypes[qType] {
			continue
		}
		questions, ok := groupedMap[qType]
		if !ok || len(questions) == 0 {
			continue
		}

		section := buildGroupedSection(qType, questions, ws.selectedQIDs, ws, func() {
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		})
		ws.qListVBox.Add(section)
		ws.qListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.AdminListSpacingH8)))
	}

	if len(ws.qListVBox.Objects) == 0 {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = core.AdminInfoTextSize
		ws.qListVBox.Add(noData)
	}
}

// renderServerQuestions 渲染服务器题库中的题目列表，按题型分组。
//
// 功能说明：
// 从服务器题库中过滤题目，并按题型分组渲染题目卡片列表到UI容器中。
// 仅显示模板配置中所需数量大于0的题型对应的题目。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func renderServerQuestions(ws *AdminExamCreateState) {
	ws.qListVBox.Objects = nil

	typeOrder := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	groupedMap := make(map[string][]core.Question)

	for _, q := range ws.serverQuestions {
		localQ := serverQuestionToLocal(&q)
		required := 0
		if ws.typeCounts != nil {
			required = ws.typeCounts[q.Type]
		}
		if required == 0 {
			continue
		}
		groupedMap[q.Type] = append(groupedMap[q.Type], *localQ)
	}

	for _, qType := range typeOrder {
		if !ws.questionFilterTypes[qType] {
			continue
		}
		questions, ok := groupedMap[qType]
		if !ok || len(questions) == 0 {
			continue
		}

		section := buildGroupedSection(qType, questions, ws.selectedQIDs, ws, func() {
			refreshQuestionList(ws)
			updateQuestionSelectTitle(ws)
			// 刷新目标用户页状态
			if ws.userPageRefreshFunc != nil {
				ws.userPageRefreshFunc()
			}
		})
		ws.qListVBox.Add(section)
		ws.qListVBox.Add(container.NewGridWrap(fyne.NewSize(1, core.ListWrapHeight)))
	}

	if len(ws.qListVBox.Objects) == 0 {
		noData := canvas.NewText(core.AdminNoQuestionsAvailableMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = core.AdminInfoTextSize
		ws.qListVBox.Add(noData)
	}
}

// ==================== Step 4: 考试参数设置 ====================

// buildTemplateSelectBtn 创建模板选择按钮：点击后弹出模板选择对话框，从服务器获取模板列表。
//
// 功能说明：
// 构建并返回"选择模板"按钮，点击该按钮会弹出模板选择对话框，从服务器获取可用模板列表及详情供用户选择。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - state: *core.AppState，全局应用状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//
// 返回值：
//   - *customElements.CustomButton：模板选择按钮对象。
func buildTemplateSelectBtn(ws *AdminExamCreateState, state *core.AppState, w fyne.Window) *customElements.CustomButton {
	// 模板选择按钮（标识符：返回的自定义按钮对象，名称："选择模板"）：点击后弹出模板选择对话框，从服务器获取可用模板列表及详情供用户选择。

	return customElements.CreateButton(
		core.AdminTemplateBtnText,
		core.BtnMediumWideWidth, core.BtnMediumWideHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnSuccessBg),
		core.HexColor(core.BtnSuccessStroke),
		1, core.AdminConfirmBtnFontSize, true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			showTemplateSelectDialog(ws, state, w)
		},
	)
}

type tplInfo struct {
	Item      network.TemplateItem
	Detail    *network.TemplateDetailItem
	HasDetail bool
}

// formatTemplateInfo 格式化模板信息字符串，包含考试名称、总题数及各题型题数分布。
//
// 功能说明：
// 将模板信息格式化为可读的字符串，格式为"考试名称（共 X 题）[单选题X/多选题Y/判断题Z/填空题W/问答题V]"。
//
// 输入参数：
//   - ti: tplInfo，包含模板基本信息、详情信息及是否有详情的结构体。
//
// 返回值：
//   - string：格式化后的模板信息字符串。
func formatTemplateInfo(ti tplInfo) string {
	info := fmt.Sprintf("%s（共 %d 题）", ti.Item.ExamName, ti.Item.QuestionCount)
	if ti.HasDetail && ti.Detail != nil {
		var parts []string
		if ti.Detail.SingleCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeSingleChoice, ti.Detail.SingleCount))
		}
		if ti.Detail.MultiCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeMultiChoice, ti.Detail.MultiCount))
		}
		if ti.Detail.JudgeCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeJudge, ti.Detail.JudgeCount))
		}
		if ti.Detail.BlankCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeFillIn, ti.Detail.BlankCount))
		}
		if ti.Detail.EssayCount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d", core.QTypeEssay, ti.Detail.EssayCount))
		}
		if len(parts) > 0 {
			info += " [" + strings.Join(parts, "/") + "]"
		}
	}
	return info
}

// showTemplateSelectDialog 显示模板选择对话框，从服务器获取模板列表及详情供用户选择。
//
// 功能说明：
// 从服务器获取可用模板列表，并并发获取每个模板的详细信息，然后显示模板详情选择对话框供用户选择并应用模板参数。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - state: *core.AppState，全局应用状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
func showTemplateSelectDialog(ws *AdminExamCreateState, state *core.AppState, w fyne.Window) {
	state.FetchTemplates(100, 0, func(success bool, templates []network.TemplateItem, total int, msg string) {
		if !success {
			customElements.ShowCustomInformation(core.AdminErrorText, msg, w)
			return
		}

		if len(templates) == 0 {
			customElements.ShowCustomInformation(core.AdminErrorText, core.AdminNoQuestionsAvailableMsg, w)
			return
		}

		tplInfos := make([]tplInfo, len(templates))
		for i, tmpl := range templates {
			tplInfos[i] = tplInfo{Item: tmpl, HasDetail: false}
		}

		var fetchedCount int
		var mu sync.Mutex

		for i := range tplInfos {
			tmplID := tplInfos[i].Item.TemplateID
			go func(idx int, id int) {
				state.GetTemplateDetail(id, func(success bool, detail *network.TemplateDetailItem, detailMsg string) {
					mu.Lock()
					defer mu.Unlock()
					if success && detail != nil {
						tplInfos[idx].Detail = detail
						tplInfos[idx].HasDetail = true
					}
					fetchedCount++
					if fetchedCount == len(tplInfos) {
						showTemplateDetailDialog(ws, tplInfos, w)
					}
				})
			}(i, tmplID)
		}
	})
}

// showTemplateDetailDialog 显示模板详情选择对话框，允许用户选择并应用模板参数。
//
// 功能说明：
// 显示模板详情选择对话框，包含模板列表的单选按钮组，以及"应用模板"和"取消"按钮。用户选择模板后点击"应用模板"按钮可将模板参数应用到考试设置中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - templateInfos: []tplInfo，模板信息及详情列表。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
func showTemplateDetailDialog(ws *AdminExamCreateState, templateInfos []tplInfo, w fyne.Window) {
	var displayNames []string
	for _, ti := range templateInfos {
		displayNames = append(displayNames, formatTemplateInfo(ti))
	}

	radioGroup := widget.NewRadioGroup(displayNames, func(s string) {
		for _, ti := range templateInfos {
			if formatTemplateInfo(ti) == s {
				ws.selectedTemplateID = ti.Item.TemplateID
				if ti.Detail != nil {
					ws.templateDetail = ti.Detail
					ws.typeCounts = map[string]int{
						"单选题": ti.Detail.SingleCount,
						"多选题": ti.Detail.MultiCount,
						"判断题": ti.Detail.JudgeCount,
						"填空题": ti.Detail.BlankCount,
						"问答题": ti.Detail.EssayCount,
					}
					if ws.nameEntry != nil {
						ws.nameEntry.SetText(ti.Item.ExamName)
					}
					if ws.durationEntry != nil {
						ws.durationEntry.SetText(fmt.Sprintf("%d", ti.Item.DurationMin))
					}
					if ws.typeEntries != nil {
						ws.typeEntries["单选题"].SetText(fmt.Sprintf("%d", ti.Detail.SingleCount))
						ws.typeEntries["多选题"].SetText(fmt.Sprintf("%d", ti.Detail.MultiCount))
						ws.typeEntries["判断题"].SetText(fmt.Sprintf("%d", ti.Detail.JudgeCount))
						ws.typeEntries["填空题"].SetText(fmt.Sprintf("%d", ti.Detail.BlankCount))
						ws.typeEntries["问答题"].SetText(fmt.Sprintf("%d", ti.Detail.EssayCount))
					}
				}
				break
			}
		}
	})

	if len(templateInfos) > 0 {
		radioGroup.SetSelected(displayNames[0])
	}

	selectLabel := widget.NewLabel(core.AdminTemplateSelectPrompt)
	selectLabel.Wrapping = fyne.TextWrapWord
	selectLabel.TextStyle = fyne.TextStyle{Bold: true}
	radioGroup.Horizontal = false

	var dlg dialog.Dialog
	// 应用模板按钮（标识符：applyBtn，名称："应用模板"）：确认并应用选中的模板参数到考试设置中，包括考试名称、时长及各题型数量。
	applyBtn := customElements.CreateButton(
		core.AdminApplyTemplateBtn,
		core.DialogActionBtnWidth, core.DialogActionBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		1, 16, true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if ws.selectedTemplateID > 0 && ws.templateDetail != nil {
				applyTemplate(ws)
				customElements.ShowCustomInformation(core.AdminSuccessText, fmt.Sprintf("已从模板【%s】快速设置考试参数", ws.templateDetail.ExamName), w)
			}
			if dlg != nil {
				dlg.Hide()
			}
		},
	)
	// 取消按钮（标识符：cancelBtn，名称："取消"）：关闭模板选择对话框，不应用任何模板参数。
	cancelBtn := customElements.CreateButton(
		core.AdminCancelText,
		core.DialogActionBtnWidth, core.DialogActionBtnHeight,
		core.HexColor(core.TextBodyColor),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BorderLightColor),
		1.5, core.AdminDialogBtnFontSize, false, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if dlg != nil {
				dlg.Hide()
			}
		},
	)

	btnRow := container.NewHBox(cancelBtn, layout.NewSpacer(), applyBtn)
	content := container.NewVBox(
		selectLabel,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		radioGroup,
		container.NewGridWrap(fyne.NewSize(1, 15)),
		container.NewCenter(btnRow),
	)

	dlg = dialog.NewCustomWithoutButtons(core.AdminTemplateDialogTitle, content, w)
	dlg.Show()
}

// applyTemplate 将选中的模板参数应用到考试设置页面（名称、时长、各题型数量）。
//
// 功能说明：
// 将选中的模板参数（考试名称、考试时长、各题型数量）应用到考试设置页面的输入框中。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func applyTemplate(ws *AdminExamCreateState) {
	if ws.templateDetail == nil {
		return
	}
	ws.examName = ws.templateDetail.ExamName
	ws.durationMin = ws.templateDetail.DurationMin

	if ws.nameEntry != nil {
		ws.nameEntry.SetText(ws.templateDetail.ExamName)
	}
	if ws.durationEntry != nil {
		ws.durationEntry.SetText(fmt.Sprintf("%d", ws.templateDetail.DurationMin))
	}
	if ws.typeEntries != nil {
		ws.typeEntries["单选题"].SetText(fmt.Sprintf("%d", ws.templateDetail.SingleCount))
		ws.typeEntries["多选题"].SetText(fmt.Sprintf("%d", ws.templateDetail.MultiCount))
		ws.typeEntries["判断题"].SetText(fmt.Sprintf("%d", ws.templateDetail.JudgeCount))
		ws.typeEntries["填空题"].SetText(fmt.Sprintf("%d", ws.templateDetail.BlankCount))
		ws.typeEntries["问答题"].SetText(fmt.Sprintf("%d", ws.templateDetail.EssayCount))
	}
	if ws.typeScoreEntries != nil {
		for _, t := range []string{"单选题", "多选题", "判断题", "填空题", "问答题"} {
			if ws.typeScoreEntries[t] != nil {
				ws.typeScoreEntries[t].SetText("1")
			}
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			ws.typeScores[t] = 1.0
		}
	}
}

// buildExamParamsPage 构建考试参数设置页面，包含考试名称、时长、时间及各题型数量输入框。
//
// 功能说明：
// 构建考试创建向导的第二步页面，允许用户设置考试名称、考试时长、开始/结束时间，以及各题型的数量。
// 页面包含模板选择按钮和确认考试参数按钮，确认后生成统计信息并更新页面。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：考试参数设置页面的UI容器对象。
func buildExamParamsPage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	if len(ws.localBankKeys) == 0 && len(ws.serverBankIDs) == 0 {
		return container.NewPadded(container.NewCenter(
			canvas.NewText(core.AdminSourceNotSetMsg, core.HexColor(core.TextMutedColor)),
		))
	}

	makeFormLabel := func(text string) *widget.Label {
		lbl := widget.NewLabel(text)
		lbl.TextStyle = fyne.TextStyle{Bold: true}
		return lbl
	}

	ws.nameEntry = widget.NewEntry()
	ws.nameEntry.SetPlaceHolder(core.AdminExamParamsNamePlaceholder)
	if ws.examName != "" {
		ws.nameEntry.SetText(ws.examName)
	}
	ws.nameEntry.OnChanged = func(s string) {
		ws.examName = s
	}
	nameRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminExamParamsNameLabel), nil, ws.nameEntry)

	ws.durationEntry = widget.NewEntry()
	ws.durationEntry.SetPlaceHolder(core.AdminExamDurationPlaceholder)
	if ws.durationMin > 0 {
		ws.durationEntry.SetText(fmt.Sprintf("%d", ws.durationMin))
	}
	ws.durationEntry.OnChanged = func(s string) {
		val, _ := strconv.Atoi(s)
		ws.durationMin = val
	}
	durationRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminExamDurationLabel), nil, ws.durationEntry)

	startTimeDefault := time.Now().Format("2006-01-02 15:04")
	if ws.startTimeStr != "" {
		startTimeDefault = ws.startTimeStr
	}
	endTimeDefault := time.Now().Add(2 * time.Hour).Format("2006-01-02 15:04")
	if ws.endTimeStr != "" {
		endTimeDefault = ws.endTimeStr
	}

	ws.startTimeLbl = widget.NewLabel(startTimeDefault)
	ws.endTimeLbl = widget.NewLabel(endTimeDefault)
	ws.startTimeStr = startTimeDefault
	ws.endTimeStr = endTimeDefault

	// 设置时间按钮（标识符：setTimeBtn返回的自定义按钮对象，名称："设置时间"）：点击后快速将对应的时间标签设置为当前时间或当前时间加偏移量。
	setTimeBtn := func(lbl *widget.Label, defaultOffset time.Duration) *customElements.CustomButton {

		return customElements.CreateButton(
			core.AdminTimeEditBtnText,
			core.BtnSmallWidth, core.BtnSmallHeight,
			core.HexColor(core.CardBgColor),
			core.HexColor(core.BtnSecondaryBg),
			core.HexColor(core.BtnSecondaryBg),
			1, 12, true, false,
			fyne.TextAlignCenter, // 👈 居中对齐
			fyne.TextWrapOff,     // 👈 不换行
			fyne.TextTruncateOff, // 👈 不换行（截断）
			func() {
				now := time.Now().Add(defaultOffset)
				formatted := now.Format("2006-01-02 15:04")
				lbl.SetText(formatted)
				if lbl == ws.startTimeLbl {
					ws.startTimeStr = formatted
				} else if lbl == ws.endTimeLbl {
					ws.endTimeStr = formatted
				}
			},
		)
	}

	startRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminStartTimeLabel), nil, container.NewHBox(ws.startTimeLbl, setTimeBtn(ws.startTimeLbl, 0)))
	endRow := container.NewBorder(nil, nil, makeFormLabel(core.AdminEndTimeLabel), nil, container.NewHBox(ws.endTimeLbl, setTimeBtn(ws.endTimeLbl, 2*time.Hour)))

	typeLabels := []string{"单选题", "多选题", "判断题", "填空题", "问答题"}
	typeEntries := make(map[string]*widget.Entry)
	typeScoreEntries := make(map[string]*widget.Entry)
	var typeCards []fyne.CanvasObject

	for _, t := range typeLabels {
		tName := t
		label := widget.NewLabel(t)
		label.TextStyle = fyne.TextStyle{Bold: true}

		countEntry := widget.NewEntry()
		countEntry.SetPlaceHolder(core.AdminTypeEntryPlaceholder)
		if ws.typeCounts != nil {
			if count, ok := ws.typeCounts[tName]; ok && count > 0 {
				countEntry.SetText(fmt.Sprintf("%d", count))
			}
		}
		countEntry.OnChanged = func(s string) {
			val, _ := strconv.Atoi(s)
			if ws.typeCounts == nil {
				ws.typeCounts = make(map[string]int)
			}
			ws.typeCounts[tName] = val
		}
		typeEntries[tName] = countEntry

		scoreEntry := widget.NewEntry()
		scoreEntry.SetPlaceHolder("分值")
		defaultScore := 1.0
		if ws.typeScores != nil {
			if sc, ok := ws.typeScores[tName]; ok {
				defaultScore = sc
			}
		}
		scoreEntry.SetText(fmt.Sprintf("%g", defaultScore))
		scoreEntry.OnChanged = func(s string) {
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				val = 1.0
			}
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			ws.typeScores[tName] = val
		}
		typeScoreEntries[tName] = scoreEntry

		compactCount := container.NewGridWrap(fyne.NewSize(55, core.TypeEntryBoxSizeH), countEntry)
		compactScore := container.NewGridWrap(fyne.NewSize(55, core.TypeEntryBoxSizeH), scoreEntry)

		itemBox := container.NewHBox(
			label,
			layout.NewSpacer(),
			widget.NewLabel("题数:"),
			compactCount,
			widget.NewLabel("分值:"),
			compactScore,
		)

		typeCard := container.NewGridWrap(fyne.NewSize(280, core.TypeCardGridWrapHeight), itemBox)
		typeCards = append(typeCards, typeCard)
	}

	ws.typeEntries = typeEntries
	ws.typeScoreEntries = typeScoreEntries

	typeGridFlow := container.NewGridWrap(fyne.NewSize(280, core.TypeCardGridWrapHeight), typeCards...)
	typeCountsSection := container.NewVBox(
		makeFormLabel(core.AdminTypeSettingsTitle),
		typeGridFlow,
	)

	templateBtn := buildTemplateSelectBtn(ws, state, w)

	var mainContainer *fyne.Container

	// 确认考试参数按钮（标识符：confirmParamsBtn，名称："确认考试参数"）：验证并确认考试名称、时长及各题型数量设置，生成统计信息并更新页面。
	var confirmParamsBtn *customElements.CustomButton
	confirmParamsBtn = customElements.CreateButton(
		core.AdminConfirmParamsBtnText,
		core.ExamParamsConfirmBtnWidth, core.ExamParamsConfirmBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.BtnPrimaryBg),
		core.ConfirmParamsBtnStrokeWidth,
		core.ConfirmParamsBtnFontSize,
		true, false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if ws.nameEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminNameRequiredErr, w)
				return
			}
			if ws.durationEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminDurationInvalidErr, w)
				return
			}
			_, err := fmt.Sscanf(ws.durationEntry.Text, "%d", &ws.durationMin)
			if err != nil || ws.durationMin <= 0 {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminDurationInvalidErr, w)
				return
			}

			ws.examName = ws.nameEntry.Text
			ws.startTimeStr = ws.startTimeLbl.Text
			ws.endTimeStr = ws.endTimeLbl.Text

			totalQuestions := 0
			totalScore := 0.0
			ws.typeCounts = make(map[string]int)
			if ws.typeScores == nil {
				ws.typeScores = make(map[string]float64)
			}
			for _, t := range typeLabels {
				val := typeEntries[t].Text
				count := 0
				if val != "" {
					count, _ = strconv.Atoi(val)
				}
				ws.typeCounts[t] = count

				scoreVal := typeScoreEntries[t].Text
				sc := 1.0
				if scoreVal != "" {
					if parsed, err := strconv.ParseFloat(scoreVal, 64); err == nil && parsed >= 0 {
						sc = parsed
					}
				}
				ws.typeScores[t] = sc

				totalQuestions += count
				totalScore += float64(count) * sc
			}

			if totalQuestions == 0 {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminTypeCountRequiredErr, w)
				return
			}

			ws.examParamsConfirmed = true

			if ws.wizard != nil && len(ws.wizard.Items) > 2 {
				qSelectPage := buildQuestionSelectPage(ws)
				ws.wizard.Items[2] = container.NewTabItem(ws.wizard.Items[2].Text, qSelectPage)
				ws.wizard.Refresh()
			}

			typeDetails := ""
			for _, t := range typeLabels {
				if ws.typeCounts[t] > 0 {
					typeDetails += fmt.Sprintf("%s: %d题(%.1f分/题)  |  ", t, ws.typeCounts[t], ws.typeScores[t])
				}
			}
			typeDetails = strings.TrimSuffix(typeDetails, "  |  ")

			typeDetailsLbl := widget.NewLabel(fmt.Sprintf(core.AdminTypeDistributionLabel, typeDetails))
			typeDetailsLbl.Wrapping = fyne.TextWrapWord

			summaryContent := container.NewVBox(
				canvas.NewText(core.AdminSuccessSetConfirmed, core.HexColor(core.ColorSelectedBorder)),
				container.NewGridWrap(fyne.NewSize(1, 4)),
				canvas.NewText(fmt.Sprintf("📋 考试名称：%s", ws.nameEntry.Text), core.HexColor(core.TextPrimaryColor)),
				canvas.NewText(fmt.Sprintf("⏱️ 考试时长：%d 分钟", ws.durationMin), core.HexColor(core.TextBodyColor)),
				canvas.NewText(fmt.Sprintf("🕒 考试时间：%s 至 %s", ws.startTimeLbl.Text, ws.endTimeLbl.Text), core.HexColor(core.TextBodyColor)),
				typeDetailsLbl,
				canvas.NewText(fmt.Sprintf("🎯 试卷总题数：%d 题，总分：%.1f 分", totalQuestions, totalScore), core.HexColor(core.ColorSelectedBorder)),
			)

			summaryCard := container.NewPadded(summaryContent)

			bottomBtnRow := container.NewHBox(
				layout.NewSpacer(),
				templateBtn,
				container.NewGridWrap(fyne.NewSize(20, 1)),
				confirmParamsBtn,
				layout.NewSpacer(),
			)

			mainContainer.Objects = []fyne.CanvasObject{
				nameRow,
				durationRow,
				startRow,
				endRow,
				typeCountsSection,
				summaryCard,
				bottomBtnRow,
			}
			mainContainer.Refresh()
		})

	bottomBtnRow := container.NewHBox(
		layout.NewSpacer(),
		templateBtn,
		container.NewGridWrap(fyne.NewSize(20, 1)),
		confirmParamsBtn,
		layout.NewSpacer(),
	)

	var initialObjects []fyne.CanvasObject
	initialObjects = append(initialObjects, nameRow, durationRow, startRow, endRow, typeCountsSection)

	if ws.examParamsConfirmed {
		totalQuestions := 0
		totalScore := 0.0
		typeDetails := ""
		if ws.typeCounts != nil {
			for _, t := range typeLabels {
				count := ws.typeCounts[t]
				sc := 1.0
				if ws.typeScores != nil {
					if s, ok := ws.typeScores[t]; ok {
						sc = s
					}
				}
				totalQuestions += count
				totalScore += float64(count) * sc
				if count > 0 {
					typeDetails += fmt.Sprintf("%s: %d题(%.1f分/题)  |  ", t, count, sc)
				}
			}
		}
		typeDetails = strings.TrimSuffix(typeDetails, "  |  ")
		typeDetailsLbl := widget.NewLabel(fmt.Sprintf(core.AdminTypeDistributionLabel, typeDetails))
		typeDetailsLbl.Wrapping = fyne.TextWrapWord

		summaryContent := container.NewVBox(
			canvas.NewText(core.AdminSuccessSetConfirmed, core.HexColor(core.ColorSelectedBorder)),
			container.NewGridWrap(fyne.NewSize(1, 4)),
			canvas.NewText(fmt.Sprintf("📋 考试名称：%s", ws.examName), core.HexColor(core.TextPrimaryColor)),
			canvas.NewText(fmt.Sprintf("⏱️ 考试时长：%d 分钟", ws.durationMin), core.HexColor(core.TextBodyColor)),
			canvas.NewText(fmt.Sprintf("🕒 考试时间：%s 至 %s", ws.startTimeStr, ws.endTimeStr), core.HexColor(core.TextBodyColor)),
			typeDetailsLbl,
			canvas.NewText(fmt.Sprintf("🎯 试卷总题数：%d 题，总分：%.1f 分", totalQuestions, totalScore), core.HexColor(core.ColorSelectedBorder)),
		)
		summaryCard := container.NewPadded(summaryContent)
		initialObjects = append(initialObjects, summaryCard)
	} else {
		initialObjects = append(initialObjects, container.NewGridWrap(fyne.NewSize(1, 8)))
	}

	initialObjects = append(initialObjects, bottomBtnRow)
	mainContainer = container.NewVBox(initialObjects...)

	ws.examParamsContent = mainContainer
	return container.NewPadded(mainContainer)
}

// ==================== Step 5: 目标用户选择 ====================

// buildUserSelectPage 构建目标用户选择页面，按管理员及其下属用户分组显示，并支持单选/多选。
//
// 功能说明：
// 构建考试创建向导的第五步页面，从服务器获取所有用户列表，按管理员及其下属用户分组显示，并支持单选/多选目标用户。
// 若题目尚未选择完整，则显示提示信息。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：目标用户选择页面的UI容器对象。
func buildUserSelectPage(ws *AdminExamCreateState, state *core.AppState) fyne.CanvasObject {
	ws.userVBox = container.NewVBox()
	ws.userScroll = container.NewVScroll(ws.userVBox)
	ws.userScroll.SetMinSize(fyne.NewSize(0, core.AdminUserListHeight))

	titleLabel := widget.NewLabel(fmt.Sprintf(core.AdminUserSelectionTitleTemplate, len(ws.selectedUsers)))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	updateTitle := func() {
		titleLabel.SetText(fmt.Sprintf("选择目标用户（已选 %d 人）", len(ws.selectedUsers)))
	}

	// 定义刷新函数
	refreshUserPage := func() {
		isFull := checkIfQuestionsSelectedFull(ws)
		fmt.Printf("[DEBUG refreshUserPage] checkIfQuestionsSelectedFull result: %v\n", isFull)
		
		// 清空用户列表容器
		ws.userVBox.Objects = nil
		
		if !isFull {
			noData := canvas.NewText(core.AdminNoQuestionsSelectedMsg, core.HexColor(core.TextMutedColor))
			noData.Alignment = fyne.TextAlignCenter
			noData.TextSize = 14
			ws.userVBox.Add(noData)
			ws.userVBox.Refresh()
			return
		}

		loadingText := canvas.NewText(core.AdminLoadingText, core.HexColor(core.TextHintColor))
		loadingText.Alignment = fyne.TextAlignCenter
		loadingText.TextSize = 14
		ws.userVBox.Add(loadingText)

		state.FetchAllUsers(func(success bool, users []network.UserItem, total int, msg string) {
			ws.userVBox.Objects = nil

			if !success {
				customElements.ShowCustomInformation(core.AdminErrorText, msg, nil)
				ws.userVBox.Refresh()
				return
			}

			var admins []network.UserItem
			for _, user := range users {
				if user.Role == "admin" || user.Role == "superadmin" {
					admins = append(admins, user)
				}
			}

			adminUsersMap := make(map[string][]network.UserItem)
			var orphanUsers []network.UserItem

			for _, user := range users {
				if user.Role == "admin" || user.Role == "superadmin" {
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
				} else {
					orphanUsers = append(orphanUsers, user)
				}
			}

			if len(admins) == 0 && len(orphanUsers) == 0 {
				noData := canvas.NewText(core.AdminNoUsersAvailableMsg, core.HexColor(core.TextMutedColor))
				noData.Alignment = fyne.TextAlignCenter
				ws.userVBox.Add(noData)
				ws.userVBox.Refresh()
				return
			}

			for _, admin := range admins {
				adminItem := admin
				subUsers := adminUsersMap[adminItem.Username]

				var subUserCbs []*widget.Check

				adminCb := widget.NewCheck(core.IconUserPrefix+adminItem.Username, nil)
				adminCb.SetChecked(ws.selectedUsers[adminItem.UserID])

				subUsersContainer := container.NewVBox()
				subUsersContainer.Hide()

				for _, user := range subUsers {
					uItem := user
					userCb := widget.NewCheck(core.IconSubUserPrefix+uItem.Username, func(checked bool) {
						if checked {
							ws.selectedUsers[uItem.UserID] = true
						} else {
							delete(ws.selectedUsers, uItem.UserID)
						}
						updateTitle()
						// 刷新确认推送页状态
						if ws.confirmPageRefreshFunc != nil {
							ws.confirmPageRefreshFunc()
						}
					})
					userCb.SetChecked(ws.selectedUsers[uItem.UserID])
					subUserCbs = append(subUserCbs, userCb)
					subUsersContainer.Add(userCb)
				}

				adminCb.OnChanged = func(checked bool) {
					if checked {
						ws.selectedUsers[adminItem.UserID] = true
					} else {
						delete(ws.selectedUsers, adminItem.UserID)
					}
					for i, u := range subUsers {
						subUserCbs[i].SetChecked(checked)
						if checked {
							ws.selectedUsers[u.UserID] = true
						} else {
							delete(ws.selectedUsers, u.UserID)
						}
					}
					updateTitle()
					// 刷新确认推送页状态
					if ws.confirmPageRefreshFunc != nil {
						ws.confirmPageRefreshFunc()
					}
				}

				var toggleBtn *widget.Button
				if len(subUsers) > 0 {
					toggleBtn = widget.NewButton(core.AdminToggleIconDown, nil)
					toggleBtn.Importance = widget.LowImportance
					toggleBtn.OnTapped = func() {
						if subUsersContainer.Visible() {
							subUsersContainer.Hide()
							toggleBtn.SetText(core.AdminToggleIconDown)
						} else {
							subUsersContainer.Show()
							toggleBtn.SetText(core.AdminToggleIconUp)
						}
					}
				} else {
					toggleBtn = widget.NewButton(core.BankManageToggleIconDisabled, nil)
					toggleBtn.Disable()
				}

				adminRow := container.NewBorder(nil, nil, adminCb, toggleBtn)
				ws.userVBox.Add(container.NewVBox(adminRow, subUsersContainer))
				ws.userVBox.Add(container.NewGridWrap(fyne.NewSize(1, 4)))
			}

			if len(orphanUsers) > 0 {
				orphanTitle := widget.NewLabel(core.AdminOtherUsersLabel)
				orphanTitle.TextStyle = fyne.TextStyle{Bold: true}
				ws.userVBox.Add(orphanTitle)

				for _, user := range orphanUsers {
					uItem := user
					userCb := widget.NewCheck(core.IconUserPrefix+uItem.Username, func(checked bool) {
						if checked {
							ws.selectedUsers[uItem.UserID] = true
						} else {
							delete(ws.selectedUsers, uItem.UserID)
						}
						updateTitle()
						// 刷新确认推送页状态
						if ws.confirmPageRefreshFunc != nil {
							ws.confirmPageRefreshFunc()
						}
					})
					userCb.SetChecked(ws.selectedUsers[uItem.UserID])
					ws.userVBox.Add(userCb)
				}
			}

			ws.userVBox.Refresh()
		})
	}

	// 保存刷新函数到状态中
	ws.userPageRefreshFunc = refreshUserPage

	// 初始检查并加载用户列表
	refreshUserPage()

	content := container.NewVBox(titleLabel, ws.userScroll)
	return container.NewPadded(content)
}

// ==================== Step 6: 确认推送 ====================

// buildConfirmPage 构建确认推送页面，包含考试参数摘要和创建推送按钮。
//
// 功能说明：
// 构建考试创建向导的第六步（最后一步）页面，显示考试参数摘要信息，并提供"创建推送"按钮用于提交考试创建请求并推送到目标用户。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：确认推送页面的UI容器对象。
func buildConfirmPage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	// 定义刷新函数
	refreshConfirmPage := func() {
		if ws.wizard != nil && len(ws.wizard.Items) > 4 {
			confirmPage := buildConfirmPage(ws, w, state)
			ws.wizard.Items[4] = container.NewTabItem(ws.wizard.Items[4].Text, confirmPage)
			ws.wizard.Refresh()
		}
	}

	// 保存刷新函数到状态中
	ws.confirmPageRefreshFunc = refreshConfirmPage

	// 创建推送按钮（标识符：ws.createBtn，名称："创建推送"）：验证所有考试参数和选题状态，调用接口创建考试并推送到目标用户。
	ws.createBtn = customElements.CreateButton(
		core.AdminCreatePushBtnText,
		core.CreatePushBtnWidth, core.CreatePushBtnHeight,
		core.HexColor(core.BtnPrimaryBg),
		core.HexColor(core.CardBgColor),
		core.HexColor(core.BtnPrimaryBg),
		core.CreatePushBtnStrokeWidth,
		core.CreatePushBtnFontSize,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		func() {
			if ws.nameEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminNameRequiredErr, nil)
				return
			}
			if ws.durationEntry.Text == "" {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminTimeFormatErrMsg, nil)
				return
			}

			duration, err := fmt.Sscanf(ws.durationEntry.Text, "%d", &ws.durationMin)
			if err != nil || duration == 0 || ws.durationMin <= 0 {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminDurationInvalidErr, nil)
				return
			}

			if !checkIfQuestionsSelectedFull(ws) {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminParamsNotSetMsg, w)
				return
			}

			if len(ws.selectedUsers) == 0 {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminNoUsersSelectedMsg, nil)
				return
			}

			// 开始提交，禁用按钮防重击
			ws.createBtn.Disable()

			var targetUserIDs []int
			for uid := range ws.selectedUsers {
				targetUserIDs = append(targetUserIDs, uid)
			}
			sort.Ints(targetUserIDs)

			var questions []network.ServerQuestion
			if ws.sourceTab == 0 {
				for _, lq := range ws.localQuestions {
					if ws.selectedQIDs[lq.ID] {
						opts := make(map[string]string)
						for _, opt := range lq.Options {
							opts[opt.Label] = opt.Text
						}
						score := 1.0
						if ws.typeScores != nil {
							if sc, ok := ws.typeScores[lq.Type]; ok {
								score = sc
							}
						}
						questions = append(questions, network.ServerQuestion{
							ID:         lq.ID,
							Type:       lq.Type,
							Content:    lq.Content,
							Options:    opts,
							Answer:     strings.Join(lq.Answers, ""),
							Difficulty: lq.Difficulty,
							Score:      score,
						})
					}
				}
			} else {
				for _, sq := range ws.serverQuestions {
					sqID := formatQuestionID(sq.ID)
					if ws.selectedQIDs[sqID] {
						score := sq.Score
						if score <= 0 {
							score = 1.0
						}
						if ws.typeScores != nil {
							if sc, ok := ws.typeScores[sq.Type]; ok && sc > 0 {
								score = sc
							}
						}
						questions = append(questions, network.ServerQuestion{
							ID:         sqID,
							Type:       sq.Type,
							Content:    sq.Content,
							Options:    sq.Options,
							Answer:     sq.Answer,
							Difficulty: sq.Difficulty,
							Score:      score,
						})
					}
				}
			}

			req := network.CreateAndPushReq{
				ExamName:      ws.nameEntry.Text,
				DurationMin:   ws.durationMin,
				StartTime:     ws.startTimeLbl.Text,
				EndTime:       ws.endTimeLbl.Text,
				Questions:     questions,
				TargetUserIDs: targetUserIDs,
			}

			if ws.sourceTab == 0 {
				if len(ws.localBankKeys) > 0 {
					req.LocalBankKey = ws.localBankKeys[0]
				}
			} else {
				if len(ws.serverBankIDs) > 0 {
					req.ServerBankID = ws.serverBankIDs[0]
				}
			}

			state.CreateAndPushExam(req, func(success bool, data *network.CreateAndPushData, msg string) {
				ws.createBtn.Enable()
				if !success {
					customElements.ShowCustomInformation(core.AdminErrorText, msg, w)
					return
				}

				if data != nil {
					customElements.ShowCustomInformation(core.AdminSuccessText, fmt.Sprintf("考试【%s】已创建（%d 题），并推送给 %d 位用户！", req.ExamName, data.QuestionCount, data.PushedCount), w)
				} else {
					customElements.ShowCustomInformation(core.AdminSuccessText, fmt.Sprintf("考试【%s】已创建并推送！", req.ExamName), w)
				}
			})
		},
	)

	title := canvas.NewText(core.AdminConfirmPushTitle, core.HexColor(core.TextPrimaryColor))
	title.TextSize = core.AdminConfirmTitleFontSize
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	if len(ws.selectedUsers) < 1 {
		return container.NewPadded(container.NewCenter(
			canvas.NewText(core.AdminNoUsersSelectedForConfirmMsg, core.HexColor(core.TextMutedColor)),
		))
	}

	summaryItems := []string{
		"考试名称: " + ws.nameEntry.Text,
		"题目数量: " + fmt.Sprintf("%d", len(ws.selectedQIDs)),
		"考试时间: " + fmt.Sprintf("%d 分钟", ws.durationMin),
		"目标用户: " + fmt.Sprintf("%d 人", len(ws.selectedUsers)),
	}

	summaryStr := strings.Join(summaryItems, "\n\n")
	fmt.Printf("[CONFIRM_PAGE] summaryItems: %v\n", summaryItems)
	fmt.Printf("[CONFIRM_PAGE] summaryStr: %q\n", summaryStr)

	summaryLbl := widget.NewLabel(summaryStr)
	summaryLbl.Alignment = fyne.TextAlignCenter

	return container.NewPadded(container.NewVBox(
		title,
		container.NewGridWrap(fyne.NewSize(1, 20)),
		summaryLbl,
		container.NewGridWrap(fyne.NewSize(1, 30)),
		container.NewCenter(ws.createBtn),
	))
}

// ==================== Helper functions ====================

// formatAnswer 根据题型和选项格式化题目答案文本。
//
// 功能说明：
// 根据题目的类型（单选题、多选题、判断题、填空题、问答题）和提供的选项列表，格式化并返回题目的答案文本。
// 对于选择/判断题，返回有效的选项标签组合；对于填空/问答题，返回答案内容或默认选项文本。
//
// 输入参数：
//   - q: *core.Question，题目对象。
//   - options: []core.Option，题目选项列表。
//
// 返回值：
//   - string：格式化后的答案文本。
func formatAnswer(q *core.Question, options []core.Option) string {
	if len(q.Answers) == 0 {
		return core.AdminNoAnswerText
	}

	switch q.Type {
	case core.QTypeSingleChoice, core.QTypeMultiChoice, core.QTypeJudge:
		validLabels := make(map[string]bool)
		for _, opt := range options {
			if strings.TrimSpace(opt.Text) != "" {
				validLabels[opt.Label] = true
			}
		}

		var parts []string
		for _, ans := range q.Answers {
			if validLabels[ans] {
				parts = append(parts, ans)
			} else {
				parts = append(parts, "？")
			}
		}
		if len(parts) == 0 {
			return core.AdminNoAnswerText
		}
		return strings.Join(parts, "、")
	case core.QTypeFillIn, core.QTypeEssay:
		result := strings.Join(q.Answers, "")
		if result == "" {
			for _, opt := range q.Options {
				if strings.TrimSpace(opt.Text) != "" {
					result = strings.TrimSpace(opt.Text)
					break
				}
			}
			if result == "" {
				return core.AdminNoAnswerText
			}
		}
		return result
	default:
		return strings.Join(q.Answers, "、")
	}
}

// buildGroupedSection 构建按题型分组的题目卡片区域，包含标题和所有题目卡片。
//
// 功能说明：
// 为指定题型的题目列表构建UI分组区域，包含题型统计标题（总题数、已选题数）和所有题目的详细卡片。
//
// 输入参数：
//   - qType: string，题型名称（如"单选题"、"多选题"等）。
//   - questions: []core.Question，该题型的题目列表。
//   - selected: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - onRefresh: func()，题目选择状态改变时的刷新回调函数。
//
// 返回值：
//   - fyne.CanvasObject：按题型分组的题目卡片区域UI容器对象。
func buildGroupedSection(qType string, questions []core.Question, selected map[string]bool, ws *AdminExamCreateState, onRefresh func()) fyne.CanvasObject {
	totalCount := len(questions)
	selectedCount := 0
	for _, q := range questions {
		if selected[q.ID] {
			selectedCount++
		}
	}

	headerText := fmt.Sprintf("【%s】共 %d 题，已选 %d 题", qType, totalCount, selectedCount)
	headerTextCanvas := canvas.NewText(headerText, core.HexColor(core.TextPrimaryColor))
	headerTextCanvas.TextSize = 15
	headerTextCanvas.TextStyle = fyne.TextStyle{Bold: true}
	headerTextCanvas.Alignment = fyne.TextAlignLeading

	var cardObjs []fyne.CanvasObject
	for idx, q := range questions {
		qCopy := q
		card := buildDetailedQuestionCard(&qCopy, selected, ws, onRefresh, idx+1, qType)
		cardObjs = append(cardObjs, card)
	}

	questionsVBox := container.NewVBox(cardObjs...)

	return container.NewVBox(
		headerTextCanvas,
		container.NewGridWrap(fyne.NewSize(1, 4)),
		questionsVBox,
	)
}

// enforceTypeLimit 检查并限制特定题型的选题数量是否已达到模板设定的上限。
//
// 功能说明：
// 检查指定题目的题型选择数量是否已达到模板配置的上限要求。若已达到上限且题目未被选中，则阻止继续选择。
//
// 输入参数：
//   - q: *core.Question，题目对象。
//   - selectedMap: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//
// 返回值：
//   - bool：若未达到上限或题目已被选中，返回 true；若已达到上限且题目未被选中，返回 false。
func enforceTypeLimit(q *core.Question, selectedMap map[string]bool, ws *AdminExamCreateState) bool {
	if ws == nil || len(ws.typeCounts) == 0 {
		return true
	}

	required := ws.typeCounts[q.Type]
	if required <= 0 || selectedMap[q.ID] {
		return true
	}

	currentCount := 0
	for _, lq := range ws.localQuestions {
		if lq.Type == q.Type && selectedMap[lq.ID] {
			currentCount++
		}
	}
	for _, sq := range ws.serverQuestions {
		sqID := formatQuestionID(sq.ID)
		if sq.Type == q.Type && selectedMap[sqID] {
			currentCount++
		}
	}

	if currentCount >= required {
		fmt.Printf("[TYPE_LIMIT] %s 选择数量已达上限 (%d/%d)，阻止继续选择\n", q.Type, currentCount, required)
		return false
	}

	return true
}

// buildDetailedQuestionCard 构建单个题目的详细卡片，包含题目内容、选项、答案及选择状态。
//
// 功能说明：
// 为单个题目构建详细的UI卡片，包含题目序号与内容、选项列表（选择/判断题）、答案区域，以及支持点击切换选中状态的背景按钮。
//
// 输入参数：
//   - q: *core.Question，题目对象。
//   - selected: map[string]bool，已选题目ID集合。
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - onRefresh: func()，题目选择状态改变时的刷新回调函数。
//   - serialNum: int，题目在当前题型中的序号。
//   - qType: string，题型名称。
//
// 返回值：
//   - fyne.CanvasObject：单个题目的详细卡片UI容器对象。
func buildDetailedQuestionCard(q *core.Question, selected map[string]bool, ws *AdminExamCreateState, onRefresh func(), serialNum int, qType string) fyne.CanvasObject {
	isSelected := selected[q.ID]

	bgColorStr := core.CardBgColor
	strokeColorStr := core.BorderLightColor
	checkIcon := "○"

	if isSelected {
		bgColorStr = core.ColorSelectedBg
		strokeColorStr = core.ColorSelectedBorder
		checkIcon = "●"
	}

	bgColor := core.HexColor(bgColorStr)
	strokeColor := core.HexColor(strokeColorStr)

	contentLines := []fyne.CanvasObject{}

	// 1. 构建标题
	titleStr := fmt.Sprintf("%s %d、%s", checkIcon, serialNum, q.Content)
	titleLabel := widget.NewLabel(titleStr)
	titleLabel.Wrapping = fyne.TextWrapWord
	titleLabel.TextStyle = fyne.TextStyle{Bold: false}
	contentLines = append(contentLines, titleLabel)

	// 2. 构建选项列表（选择/判断题）
	if len(q.Options) > 0 && q.Type != core.QTypeFillIn && q.Type != core.QTypeEssay {
		contentLines = append(contentLines, layout.NewSpacer())

		type optItem struct {
			Label string
			Text  string
		}
		var items []optItem
		for _, opt := range q.Options {
			if strings.TrimSpace(opt.Text) == "" {
				continue
			}
			items = append(items, optItem{Label: opt.Label, Text: opt.Text})
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].Label < items[j].Label
		})

		var optionObjs []fyne.CanvasObject
		for _, item := range items {
			optText := item.Label + ". " + item.Text
			optLabel := widget.NewLabel(optText)
			optLabel.TextStyle = fyne.TextStyle{Bold: false}
			optLabel.Alignment = fyne.TextAlignLeading
			optionObjs = append(optionObjs, optLabel)
		}

		var rowObjs []fyne.CanvasObject
		for i, opt := range optionObjs {
			rowObjs = append(rowObjs, opt)
			if (i+1)%2 == 0 || i == len(optionObjs)-1 {
				contentLines = append(contentLines, container.NewHBox(rowObjs...))
				rowObjs = nil
			}
		}
	}

	contentLines = append(contentLines, layout.NewSpacer())

	// 3. 构建答案区域
	if q.Type == core.QTypeFillIn || q.Type == core.QTypeEssay {
		var fullAnswer strings.Builder
		for _, ans := range q.Answers {
			fullAnswer.WriteString(ans)
		}
		candidate := fullAnswer.String()
		if candidate == "" {
			for _, opt := range q.Options {
				if strings.TrimSpace(opt.Text) != "" {
					candidate = strings.TrimSpace(opt.Text)
					break
				}
			}
		}
		if candidate == "" {
			candidate = core.AdminNoAnswerText
		}

		ansTextLabel := widget.NewLabel(candidate)
		ansTextLabel.Wrapping = fyne.TextWrapWord
		ansTextLabel.TextStyle = fyne.TextStyle{Bold: true}

		ansHintText := canvas.NewText(core.AdminAnswerHintText, core.HexColor(core.ColorSelectedBorder))
		ansHintText.TextSize = 13
		ansHintText.TextStyle = fyne.TextStyle{Bold: true}

		contentLines = append(contentLines, ansTextLabel, ansHintText)
	} else {
		answerText := canvas.NewText(core.AdminAnswerPrefix+formatAnswer(q, q.Options), core.HexColor(core.ColorSelectedBorder))
		answerText.TextSize = 13
		answerText.TextStyle = fyne.TextStyle{Bold: true}
		contentLines = append(contentLines, answerText)
	}

	// 4. 卡片整体内容
	cardContent := container.NewVBox(contentLines...)
	cardLayout := container.NewPadded(cardContent)

	// 点击回调
	newOnToggle := func() {
		if !enforceTypeLimit(q, selected, ws) {
			return
		}
		selected[q.ID] = !selected[q.ID]
		onRefresh()
	}

	// 5. 使用 CreateButton 创建底层的背景卡片与响应对象（标识符：btnBg，名称："题目卡片背景按钮"）：提供可点击切换题目选中状态的背景按钮，内容由上层覆盖。
	btnBg := customElements.CreateButton(
		"",    // 空文本，因为内容由上层覆盖
		-1, 0, // 宽度 -1 代表宽度自适应填充，高度 0 由容器内部撑开
		core.HexColor(core.TextBodyColor),
		bgColor,
		strokeColor,
		1.0,                 // 边框宽度
		core.FontSizeButton, // 字号参数
		false, false,        // 不加粗，不禁用
		fyne.TextAlignCenter,
		fyne.TextWrapOff,
		fyne.TextTruncateOff,
		newOnToggle,
	)

	// 6. 将自定义按钮背景与多行卡片内容通过 Stack 叠加组合
	return container.NewStack(btnBg, cardLayout)
}

func formatQuestionID(id interface{}) string {
	switch v := id.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%d", int(v))
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// serverQuestionToLocal 将服务器题库题目转换为本地题目结构体。
//
// 功能说明：
// 将服务器题库的题目数据（network.ServerQuestion）转换为本地题目结构体（core.Question），以便在UI中统一处理。
//
// 输入参数：
//   - q: *network.ServerQuestion，服务器题库题目对象。
//
// 返回值：
//   - *core.Question：转换后的本地题目对象。
func serverQuestionToLocal(q *network.ServerQuestion) *core.Question {
	var opts []core.Option
	for label, text := range q.Options {
		opts = append(opts, core.Option{Label: label, Text: text})
	}
	var answers []string
	for _, ch := range q.Answer {
		answers = append(answers, string(ch))
	}

	return &core.Question{
		ID:         formatQuestionID(q.ID),
		Type:       q.Type,
		Content:    q.Content,
		Options:    opts,
		Answers:    answers,
		Difficulty: q.Difficulty,
	}
}

// convertBankDataToQuestions 将本地题库数据转换为本地题目结构体列表。
//
// 功能说明：
// 将本地题库数据（parser.BankData）中的题目信息转换为本地题目结构体（core.Question）列表，以便在UI中统一处理。
//
// 输入参数：
//   - bankData: *parser.BankData，本地题库数据对象。
//
// 返回值：
//   - []core.Question：转换后的本地题目列表。
func convertBankDataToQuestions(bankData *parser.BankData) []core.Question {
	var questions []core.Question
	for _, q := range bankData.Questions {
		var opts []core.Option
		for label, text := range q.Options {
			opts = append(opts, core.Option{Label: label, Text: text})
		}

		var answers []string
		if q.Answer != "" {
			for _, ch := range q.Answer {
				answers = append(answers, string(ch))
			}
		}

		questions = append(questions, core.Question{
			ID:         fmt.Sprintf("%d", q.ID),
			Type:       q.Type,
			Content:    q.Content,
			Options:    opts,
			Answers:    answers,
			Difficulty: q.Difficulty,
		})
	}
	return questions
}

// randomFillQuestions 随机填满各题型缺少的题目
//
// 功能说明：
// 根据考试参数中配置的每种题型所需数量（ws.typeCounts），从未被选中的可用题目中随机挑选并标记选中，直到达到所需数量。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
func randomFillQuestions(ws *AdminExamCreateState) {
	if ws == nil || len(ws.typeCounts) == 0 {
		return
	}

	var allQuestions []core.Question
	if ws.sourceTab == 0 {
		allQuestions = ws.localQuestions
	} else {
		for _, sq := range ws.serverQuestions {
			allQuestions = append(allQuestions, *serverQuestionToLocal(&sq))
		}
	}

	typeMap := make(map[string][]core.Question)
	for _, q := range allQuestions {
		typeMap[q.Type] = append(typeMap[q.Type], q)
	}

	rand.Seed(time.Now().UnixNano())

	for qType, required := range ws.typeCounts {
		if required <= 0 {
			continue
		}
		currentCount := 0
		var unselected []core.Question
		for _, q := range typeMap[qType] {
			if ws.selectedQIDs[q.ID] {
				currentCount++
			} else {
				unselected = append(unselected, q)
			}
		}

		needed := required - currentCount
		if needed <= 0 {
			continue
		}

		if len(unselected) <= needed {
			for _, q := range unselected {
				ws.selectedQIDs[q.ID] = true
			}
		} else {
			rand.Shuffle(len(unselected), func(i, j int) {
				unselected[i], unselected[j] = unselected[j], unselected[i]
			})
			for i := 0; i < needed; i++ {
				ws.selectedQIDs[unselected[i].ID] = true
			}
		}
	}
}

// buildTemplateListPage 构建考试模板列表页面，显示所有考试模板并支持取消考试操作。
//
// 功能说明：
// 构建考试创建向导的第一步页面，显示所有考试模板的列表，并提供刷新、编辑、删除、取消考试和导出结果等功能。
//
// 输入参数：
//   - ws: *AdminExamCreateState，管理员考试创建状态对象。
//   - w: fyne.Window，当前窗口对象，用于显示对话框。
//   - state: *core.AppState，全局应用状态对象。
//
// 返回值：
//   - fyne.CanvasObject：考试模板列表页面的UI容器对象。
func buildTemplateListPage(ws *AdminExamCreateState, w fyne.Window, state *core.AppState) fyne.CanvasObject {
	// Template list
	templateListVBox := container.NewVBox()
	var refreshTemplateList func()

	refreshTemplateList = func() {
		if !state.IsLoggedIn {
			return
		}

		state.FetchTemplates(100, 0, func(success bool, templates []network.TemplateItem, total int, msg string) {
			if !success {
				return
			}

			if templateListVBox == nil {
				templateListVBox = container.NewVBox()
			}

			if len(templates) == 0 {
				noDataText := customElements.CreateLabel("暂无考试模板", core.HexColor(core.TextMutedColor), core.FontSizeSubtitle, false, true, false)
				templateListVBox.Objects = []fyne.CanvasObject{container.NewPadded(noDataText)}
				templateListVBox.Refresh()
				return
			}

			// Render cards directly from templates since the list API now includes question_type_distribution
			var objects []fyne.CanvasObject
			for _, tmpl := range templates {
				tmplCard := buildTemplateCardForPushExam(&tmpl, state, func() {
					showDeleteTemplateConfirmForPushExam(w, state, &tmpl, func() {
						refreshTemplateList()
					})
				})
				objects = append(objects, tmplCard, container.NewGridWrap(fyne.NewSize(1, 8)))
			}

			// Safely update the container objects on completion
			templateListVBox.Objects = objects
			fyne.Do(func() { templateListVBox.Refresh() })
		})
	}

	// 保存刷新函数到状态中，供点击时调用
	ws.templateRefreshFunc = refreshTemplateList

	scrollContent := container.NewVScroll(templateListVBox)
	scrollContent.SetMinSize(fyne.NewSize(0, 400))

	pageBg := canvas.NewRectangle(core.HexColor(core.PageBgColor))

	mainLayout := container.NewBorder(nil, nil, nil, nil, scrollContent)
	wRootLayout := container.NewStack(pageBg, mainLayout)

	return wRootLayout
}

// buildTemplateCardForPushExam 为模板项创建一个卡片，只保留删除按钮。
//
// 参数:
//   - tmpl: *network.TemplateItem 类型，表示模板项的基本信息。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - onDelete: func() 类型，删除按钮的回调函数。
//
// 返回值:
//   - fyne.CanvasObject 类型，返回构建好的模板卡片对象。
func buildTemplateCardForPushExam(tmpl *network.TemplateItem, state *core.AppState, onDelete func()) fyne.CanvasObject {
	// Card background
	bg := canvas.NewRectangle(core.HexColor(core.CardBgColor))
	bg.CornerRadius = core.TemplateCardCornerRadius

	// Template name
	nameText := customElements.CreateLabel("📋 "+tmpl.ExamName, core.HexColor(core.TextPrimaryColor), core.FontSizeHeading, true, true, false)

	// Status badge
	statusColor := core.TextMutedColor
	statusText := tmpl.Status
	switch tmpl.Status {
	case "draft":
		statusColor = core.TextHintColor
		statusText = core.TemplateStatusDraftText
	case "active":
		statusColor = core.BtnPrimaryBg
		statusText = core.TemplateStatusActiveText
	case "expired":
		statusColor = core.ColorWrongBorder
		statusText = core.TemplateStatusExpiredText
	}
	statusBadge := canvas.NewText("● "+statusText, core.HexColor(statusColor))
	statusBadge.TextSize = core.TemplateStatusBadgeFontSize

	// Info row
	// Convert bank source to Chinese display
	bankSourceText := "未知来源"
	switch tmpl.BankSource {
	case "local":
		bankSourceText = "本地题库"
	case "server":
		bankSourceText = "服务器题库"
	case "client":
		bankSourceText = "客户端"
	}

	infoParts := []string{
		fmt.Sprintf(core.TemplateInfoQuestionCountLabel, tmpl.QuestionCount),
		fmt.Sprintf(core.TemplateInfoDurationLabel, tmpl.DurationMin),
		fmt.Sprintf(core.TemplateInfoSourceLabel, bankSourceText),
	}

	// Add per-type counts from QuestionTypeDistribution
	if tmpl.QuestionTypeDistribution.SingleChoice > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeSingleChoice, tmpl.QuestionTypeDistribution.SingleChoice))
	}
	if tmpl.QuestionTypeDistribution.MultipleChoice > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeMultiChoice, tmpl.QuestionTypeDistribution.MultipleChoice))
	}
	if tmpl.QuestionTypeDistribution.TrueFalse > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeJudge, tmpl.QuestionTypeDistribution.TrueFalse))
	}
	if tmpl.QuestionTypeDistribution.FillBlank > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeFillIn, tmpl.QuestionTypeDistribution.FillBlank))
	}
	if tmpl.QuestionTypeDistribution.Essay > 0 {
		infoParts = append(infoParts, fmt.Sprintf(core.TemplateTypeEssay, tmpl.QuestionTypeDistribution.Essay))
	}

	infoText := canvas.NewText(strings.Join(infoParts, "  |  "), core.HexColor(core.TextSecondaryColor))
	infoText.TextSize = core.TemplateInfoTextFontSize

	// Time info - use VBox for proper line breaks
	startTimeText := canvas.NewText(fmt.Sprintf("开始: %s", tmpl.StartTime), core.HexColor(core.TextMutedColor))
	startTimeText.TextSize = core.TemplateTimeTextFontSize

	endTimeText := canvas.NewText(fmt.Sprintf("结束: %s", tmpl.EndTime), core.HexColor(core.TextMutedColor))
	endTimeText.TextSize = core.TemplateTimeTextFontSize

	timeInfoContainer := container.NewVBox(startTimeText, endTimeText)

	// 删除按钮：标识符为"deleteBtn"，功能为点击后弹出确认对话框以删除当前模板（调用 onDelete 回调）。
	deleteBtn := customElements.CreateButton(
		core.TemplateDeleteBtnText,
		core.TemplateDeleteBtnWidth,
		core.TemplateBtnHeight,
		core.HexColor(core.CardBgColor),
		core.HexColor(core.ColorErrorBg),
		core.HexColor(core.ColorErrorBorder),
		core.StrokeMedium,
		core.FontSizeButton,
		true,
		false,
		fyne.TextAlignCenter, // 👈 居中对齐
		fyne.TextWrapOff,     // 👈 不换行
		fyne.TextTruncateOff, // 👈 不换行（截断）
		onDelete,
	)

	btnRow := container.NewHBox(deleteBtn)

	cardContent := container.NewVBox(
		container.NewBorder(nameText, nil, statusBadge, nil, nil),
		container.NewGridWrap(fyne.NewSize(1, 8)),
		infoText,
		container.NewGridWrap(fyne.NewSize(1, 5)),
		timeInfoContainer,
		container.NewGridWrap(fyne.NewSize(1, 10)),
		btnRow,
	)

	return container.NewStack(
		bg,
		container.NewPadded(cardContent),
	)
}

// showDeleteTemplateConfirmForPushExam 显示用于删除模板的确认对话框（用于推送考试模板列表页面）。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行删除模板操作。
//   - tmpl: *network.TemplateItem 类型，表示要删除的模板项的基本信息。
//   - onDeleted: func() 类型，删除成功后的回调函数，用于刷新模板列表。
func showDeleteTemplateConfirmForPushExam(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onDeleted func()) {
	customElements.ShowCustomConfirm(
		core.TemplateDeleteConfirmTitle,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.TemplateDeleteConfirmMsg, tmpl.ExamName)),
		func(confirm bool) bool {
			if confirm {
				state.DeleteTemplate(tmpl.TemplateID, func(success bool, msg string) {
					if success {
						customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, core.TemplateDeleteSuccessMsg, w)
						if onDeleted != nil {
							onDeleted()
						}
					} else {
						customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
					}
				})
			}
			return true // 返回 true 表示对话框可以关闭
		}, w)
}

// showCancelExamConfirmForTemplate 显示用于取消考试的确认对话框（用于模板列表页面）。
//
// 参数:
//   - w: fyne.Window 类型，表示当前窗口对象，用于显示对话框。
//   - state: *core.AppState 类型，表示全局应用状态，用于执行取消考试操作。
//   - tmpl: *network.TemplateItem 类型，表示要取消考试的模板项的基本信息。
//   - onCanceled: func() 类型，取消成功后的回调函数，用于刷新模板列表。
func showCancelExamConfirmForTemplate(w fyne.Window, state *core.AppState, tmpl *network.TemplateItem, onCanceled func()) {
	customElements.ShowCustomConfirm(
		core.TemplateCancelExamConfirmTitle,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.TemplateCancelExamConfirmMsg, tmpl.ExamName, 0)),
		func(confirm bool) bool {
			if confirm {
				state.CancelExam(tmpl.TemplateID, "", func(success bool, affectedCount int, msg string) {
					if success {
						customElements.ShowCustomInformation(core.TemplateManageSuccessMsgType, fmt.Sprintf(core.TemplateCancelExamSuccessMsg, affectedCount), w)
						if onCanceled != nil {
							onCanceled()
						}
					} else {
						customElements.ShowCustomInformation(core.TemplateManageErrorMsgType, msg, w)
					}
				})
			}
			return true // 返回 true 表示对话框可以关闭
		}, w)
}
