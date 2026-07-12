// main.go

package main

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
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ==================== 数据模型 ====================
type Question struct {
	ID         string
	Type       string // "单选题", "多选题", "判断题"
	Content    string
	Options    []Option
	Answers    []string // 正确答案的 Label，例如 ["A", "C"]
	Difficulty string
}

type Option struct {
	Label string // A, B, C, D
	Text  string
}

// AppState 维护全局和练习会话的状态
type AppState struct {
	Questions   []Question
	CurrentList []Question
	Index       int
	UserAnswers map[string][]string
	Submitted   map[string]bool
	Title       string // 当前的练习模式名称 (如："顺序练习-全部题目", "错题练习-单选题")

	// ======= 维护本地文件状态的字段 =======
	CurrentStorageKey string   // 当前激活题库的 Key
	CurrentFileName   string   // 当前激活题库的显示名称
	StoredFiles       []string // 已导入的文件 Key 列表

	// ✨ 当前题库的错题和收藏索引集（使用 map 方便在 UI 中 O(1) 复杂度查询）
	WrongSet map[string]bool // key 为 Question.ID
	FavSet   map[string]bool // key 为 Question.ID

	// ✨ 本次练习的核心统计字段
	//PracticeCorrectCount int               // 本次练习做对的题数 (✓)
	//PracticeWrongCount   int               // 本次练习做错的题数 (✕)
	ModeRecords     map[string]map[string]string // ✨ 所有模式的答题记录总集
	PracticeRecords map[string]string            // 本次练习的详细答题记录。key: Question.ID, value: 已答选项(如"A") 或 "null"表示未答

	//CurrentMode string         // 当前的练习模式名称 (如："顺序练习", "错题练习")
	ModeIndices map[string]int // 用于独立保存每个模式的答题进度 (题号索引)
}

// ==================== 自定义可点击组件 ====================
// ClickableBox 让我们可以在 Fyne 中实现类似 HTML div 的自由点击并处理背景效果
type ClickableBox struct {
	widget.BaseWidget
	OnTapped func()
	Content  *fyne.Container
}

func NewClickableBox(content *fyne.Container, onTap func()) *ClickableBox {
	c := &ClickableBox{Content: content, OnTapped: onTap}
	c.ExtendBaseWidget(c)
	return c
}

func (c *ClickableBox) Tapped(_ *fyne.PointEvent) {
	if c.OnTapped != nil {
		c.OnTapped()
	}
}

func (c *ClickableBox) TappedSecondary(_ *fyne.PointEvent) {}

func (c *ClickableBox) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.Content)
}

// ==================== 主入口 ====================
func main() {
	a := app.NewWithID("com.gzjjjfree.practice")
	w := a.NewWindow("题库练习系统")
	w.Resize(fyne.NewSize(400, 750))

	state := &AppState{
		UserAnswers: make(map[string][]string),
		Submitted:   make(map[string]bool),
		StoredFiles: []string{}, // 初始为空
	}

	// 扫描本地已有的缓存 json 文件充当 "StoredFiles"
	refreshLocalFilesList(state)

	// ==================== 启动时自动加载上次题库 ====================
	lastBankKey := a.Preferences().String("LastOpenedBankKey")
	if lastBankKey != "" {
		// 这里调用你现有的、根据 StorageKey 从本地读取 JSON 并转为 BankData 的函数
		// 假设叫 loadBankFromLocal
		//bankData, err := loadBankFromLocal(lastBankKey)
		loadAndRenderBank(w, state, lastBankKey)
	}
	// ================================================================

	showHome(w, state)
	w.ShowAndRun()
}

// 扫描当前程序目录下所有的 .json 文件，用于模拟小程序 getAllExcelKeys
func refreshLocalFilesList(state *AppState) {
	storageDir := fyne.CurrentApp().Storage().RootURI().Path()
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

			// 🛑 【核心过滤】：屏蔽掉错题集和收藏集，防止它们作为独立题库被展示
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

	// 🔥 核心：全局按时间戳从小到大排序，然后倒序
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

// ==================== 首页页面（终极修复：标题完美展现+列表元素归位） ====================
func showHome(w fyne.Window, state *AppState) {
	fmt.Println("in showHome")
	var refreshHomeUI func()

	makeCustomGridBtn := func(icon, text string, action func()) *ClickableBox {
		iconLbl := canvas.NewText(icon, hexColor("#222222"))
		iconLbl.TextSize = 24
		iconLbl.Alignment = fyne.TextAlignCenter

		textLbl := canvas.NewText(text, hexColor("#222222"))
		textLbl.TextSize = 12
		textLbl.Alignment = fyne.TextAlignCenter

		btnBg := canvas.NewRectangle(hexColor("#ffffff"))
		btnBg.CornerRadius = 6
		btnBg.StrokeColor = hexColor("#dddddd")
		btnBg.StrokeWidth = 1

		boxContent := container.NewVBox(layout.NewSpacer(), iconLbl, textLbl, layout.NewSpacer())
		stack := container.NewStack(btnBg, container.NewPadded(boxContent))
		return NewClickableBox(stack, action)
	}

	grid := container.NewGridWithColumns(3)

	// 📁 读取
	grid.Add(makeCustomGridBtn("📁", "读取", func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if reader == nil {
				return
			}

			// 保存用户选中的原始文件名
			originalFileName := reader.URI().Name()

			// ✨ 先把字节流读取到内存，然后立刻关闭资源，防止异步弹窗导致的句柄泄漏
			fileBytes, readErr := io.ReadAll(reader)
			reader.Close()

			if readErr != nil {
				dialog.ShowError(fmt.Errorf("流读取失败: %v", readErr), w)
				return
			}

			storageDir := fyne.CurrentApp().Storage().RootURI().Path()

			// ==================== 1. 扫描是否已有同名旧文件 ====================
			files, _ := os.ReadDir(storageDir)
			var oldFileNames []string

			// 寻找命名中包含 "_原始文件名_" 的旧 JSON 文件
			searchPattern := "_" + originalFileName + "_"
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
					if strings.Contains(f.Name(), searchPattern) {
						oldFileNames = append(oldFileNames, f.Name())
					}
				}
			}

			// ==================== 2. 定义核心处理与落盘逻辑 ====================
			// 参数 isNewAlias: true表示用户选择了"新增", false表示"覆盖"或"初次读取"
			processAndSave := func(isNewAlias bool) {
				fileName := originalFileName

				if isNewAlias {
					// ====== 新增逻辑：计算并分配 (1), (2) 这样的自增别名 ======
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
					// ====== 覆盖逻辑：物理删除发现的所有旧版同名文件 ======
					for _, oldName := range oldFileNames {
						_ = os.Remove(filepath.Join(storageDir, oldName))
					}
				}

				progDialog := dialog.NewCustomWithoutButtons("读取中", widget.NewProgressBarInfinite(), w)
				progDialog.Show()

				// 🔥【关键接入点】：如果扩展名是 xls，进行无损转换
				if strings.ToLower(filepath.Ext(fileName)) == ".xls" {
					newBytes, convErr := ConvertXlsToXlsxBytes(fileBytes)
					if convErr == nil {
						fileBytes = newBytes
						fileName = strings.TrimSuffix(fileName, ".xls") + ".xlsx"
						fmt.Println("检测到旧版 XLS，已自动无损升级为 XLSX")
					}
				}

				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				fileExt := strings.ToLower(getFileExtension(fileName))
				storageKey := fmt.Sprintf("%sData_%s_%d", fileExt, fileName, timestamp)

				// 触发内存解析引擎
				bankData, parseErr := ParseBytesToBank(fileBytes, fileName, storageKey)
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

				syncQuestionsToState(state, bankData)
				refreshLocalFilesList(state)
				refreshHomeUI()
			}

			// ==================== 3. 触发三选项提示或直接读取 ====================
			if len(oldFileNames) > 0 {
				// 定义自定义弹窗的句柄（以便在按钮点击时能关掉它）
				var confirmDialog dialog.Dialog

				// 按钮 1：取消
				cancelBtn := widget.NewButton(" 取消 ", func() {
					confirmDialog.Hide() // 默默关闭，不执行任何落盘逻辑
				})

				// 按钮 2：覆盖 (标记为红色警示)
				overwriteBtn := widget.NewButton(" 覆盖 ", func() {
					confirmDialog.Hide()
					processAndSave(false) // 传入 false，执行删除旧文件逻辑
				})
				overwriteBtn.Importance = widget.DangerImportance

				// 按钮 3：新增 (标记为蓝色推荐)
				addBtn := widget.NewButton(" 新增 ", func() {
					confirmDialog.Hide()
					processAndSave(true) // 传入 true，执行自增别名逻辑
				})
				addBtn.Importance = widget.HighImportance

				// ✨ 使用 HBox 将三个按钮横向排列，并用 Spacer 挤到右侧（或居中）
				buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, overwriteBtn, addBtn, layout.NewSpacer())

				msgLbl := widget.NewLabel(fmt.Sprintf("题库列表中已存在名为【%s】的文件。\n\n请选择接下来的操作：", originalFileName))

				// 组合文本和按钮区域
				content := container.NewVBox(msgLbl, buttons)

				// 渲染并显示这个不带默认按钮的极客弹窗
				confirmDialog = dialog.NewCustomWithoutButtons("文件已存在", content, w)
				confirmDialog.Show()

			} else {
				// 如果不存在同名文件，直接走普通的初次读取逻辑
				processAndSave(false)
			}

		}, w)

		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".xlsx", ".xls", ".txt", ".json"}))
		fileDialog.Show()
	}))

	// 📤 导出
	grid.Add(makeCustomGridBtn("📤", "导出", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}

		// 触发导出格式选择弹窗
		showExportDialog(w, state)
	}))

	// 🗑️ 删除按钮
	grid.Add(makeCustomGridBtn("🗑️", "删除", func() {
		if state.CurrentStorageKey == "" {
			dialog.ShowInformation("提示", "当前未选中任何题库", w)
			return
		}
		dialog.ShowConfirm("提示", "确定要删除当前题库吗？", func(b bool) {
			if b {
				deleteBank(w, state)
				refreshHomeUI()
			}
		}, w)
	}))

	// 📚 顺序练习
	grid.Add(makeCustomGridBtn("📚", "顺序练习", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}
		// 🌟 跳转到中间选择页
		showTypeSelection(w, state, "顺序练习")
	}))

	// 🎲 随机练习
	grid.Add(makeCustomGridBtn("🎲", "随机练习", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}
		// 🌟 跳转到中间选择页
		showTypeSelection(w, state, "随机练习")
	}))

	// 📝 修正题目
	grid.Add(makeCustomGridBtn("📝", "修正题目", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}
		// 🌟 跳转到中间选择页
		showTypeSelection(w, state, "修正题目")
	}))

	// ❌ 错题练习
	grid.Add(makeCustomGridBtn("❌", "错题练习", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.WrongSet) == 0 {
			dialog.ShowInformation("提示", "本题库没有错题集", w)
			return
		}
		// 🌟 跳转到中间选择页
		showTypeSelection(w, state, "错题练习")

	}))

	// ⭐ 收藏练习
	grid.Add(makeCustomGridBtn("⭐", "收藏练习", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}

		if len(state.FavSet) == 0 {
			dialog.ShowInformation("提示", "本题库没有收藏集", w)
			return
		}
		// 🌟 跳转到中间选择页
		showTypeSelection(w, state, "收藏练习")

	}))

	// 🔍 搜索本题库
	grid.Add(makeCustomGridBtn("🔍", "搜索本题库", func() {
		if len(state.Questions) == 0 {
			dialog.ShowInformation("提示", "请先导入题库", w)
			return
		}

		// 跳转到搜索页面
		showSearchPage(w, state, func() {
			// 返回主页的回调
			showHome(w, state) // 返回主页的函数
		})
	}))

	currentFileTitleLbl := widget.NewLabelWithStyle("当前选中的题库:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	currentFileNameLbl := widget.NewLabel("未选择题库")
	currentFileNameLbl.TextStyle = fyne.TextStyle{Italic: true}
	currentFileNameLbl.Wrapping = fyne.TextWrapBreak

	listTitle := widget.NewLabelWithStyle("已存储的文件列表 (点击切换):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 将显式标题组装进顶部的核心控制区
	topAndMiddleBox := container.NewVBox(
		getTitle(), // 👈 挂载大标题
		widget.NewSeparator(),
		grid,
		widget.NewSeparator(),
		currentFileTitleLbl,
		currentFileNameLbl,
	)

	fileListContainer := container.NewVBox()

	scrollList := container.NewScroll(fileListContainer)
	scrollList.SetMinSize(fyne.NewSize(0, 50))

	// 1. 将上方的分割线和标题打包成一个 Header 头部
	header := container.NewVBox(
		widget.NewSeparator(),
		listTitle,
	)

	// 2. 🔥 核心修复：使用 NewBorder 布局
	// Top 槽位放 header，Center 槽位放 scrollList
	// 这样 scrollList 会自动吞噬掉原本 Spacer 占用的所有空间
	bottomBox := container.NewBorder(header, nil, nil, nil, scrollList)

	// ------------------ 请从 bottomBox 的定义开始向下全量替换 ------------------
	//bottomBox := container.NewVBox(
	//	widget.NewSeparator(),
	//	listTitle,
	//	layout.NewSpacer(),
	//	scrollList,
	//)

	// 1. 创建一个 15 像素高的透明色块，作为底部的安全垫片
	bottomPadding := canvas.NewRectangle(color.Transparent)
	bottomPadding.SetMinSize(fyne.NewSize(0, 15))

	// 2. 将列表和垫片组合：bottomBox 在上方，透明垫片在最底下兜底
	bottomWrapper := container.NewBorder(nil, bottomPadding, nil, nil, bottomBox)

	// 3. 终极主布局：
	// Top 卡死上部的九宫格；Bottom 卡死底部的列表；中间（Center）塞入一个大弹簧强制向两端推开空白！
	mainLayout := container.NewBorder(topAndMiddleBox, bottomWrapper, nil, nil, layout.NewSpacer())

	// 4. 重构刷新逻辑，加入【动态生长算法】
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
			noDataText := canvas.NewText("⚠️ 暂无本地存储题库 (手机环境无缓存)", hexColor("#ff4d4f"))
			noDataText.Alignment = fyne.TextAlignCenter
			fileListContainer.Add(noDataText)
		}

		for _, key := range state.StoredFiles {
			fileKey := key
			showName := extractFileName(fileKey)

			bg := canvas.NewRectangle(color.Transparent)
			bg.CornerRadius = 5
			textItem := canvas.NewText(showName, color.Black)
			textItem.TextSize = 14

			if fileKey == state.CurrentStorageKey {
				bg.FillColor = hexColor("#e6f7ff")
				bg.StrokeColor = hexColor("#1890ff")
				bg.StrokeWidth = 1.5
				textItem.Color = hexColor("#111111")
			} else {
				bg.FillColor = hexColor("#f0f2f5")
				bg.StrokeColor = hexColor("#e8e8e8")
				bg.StrokeWidth = 1
				textItem.Color = hexColor("#555555")
			}

			textItem.Refresh()
			bg.Refresh()

			// 撑高单行元素，保证触控面积
			//textCenter := container.NewCenter(textItem)
			textCenter := widget.NewLabel(" " + textItem.Text + " ")
			textCenter.Wrapping = fyne.TextWrapBreak // 🔥 允许在屏幕宽度内自动折行

			//paddedItemBox := container.NewBorder(
			//	nil, nil, nil, nil,
			//	container.NewPadded(textCenter),
			//	//textCenter,
			//)
			rowStack := container.NewStack(bg, textCenter)
			//rowStack := container.NewStack(bg, paddedItemBox)

			// ------------------ 请在 refreshHomeUI 的循环内部找到并替换此处的 clickableRow ------------------
			clickableRow := NewClickableBox(rowStack, func() {
				loadAndRenderBank(w, state, fileKey)

				// 4. 重绘首页
				refreshHomeUI()
			})
			// -----------------------------------------------------------------------------------------

			fileListContainer.Add(clickableRow)
		}

		// 🔥【核心魔法】：根据文件数量，动态计算列表容器需要撑起的高度！
		// 单个文件卡片的高度大约是 48 像素。
		desiredHeight := float32(totalCount) * 48.0
		if totalCount == 0 {
			desiredHeight = 40.0 // 空白提示语的默认高度
		}

		// 设定最大生长极限（约300像素，大概容纳6个文件不滑动），防止在小屏幕手机上把上方的九宫格顶出去
		if desiredHeight > 300 {
			desiredHeight = 300
		}

		// 强行改变滚动条区域的最小高度，促使底部的 Bottom 槽位向中央膨胀！
		scrollList.SetMinSize(fyne.NewSize(0, desiredHeight))

		fileListContainer.Refresh()
		mainLayout.Refresh()
	}

	// 执行第一次界面加载渲染
	refreshHomeUI()

	// 最终套上一层安全内边距展示
	w.SetContent(container.NewPadded(mainLayout))
}
