package managementRelated

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)



// throttleFunc 节流函数包装器，限制回调函数的执行频率。
//
// 功能说明：
// 使用时间戳节流（throttling）模式，确保在指定时间间隔内只执行一次回调。
// 适用于高频触发的 UI 事件（如筛选、滚动、输入等），避免过度刷新。
//
// 输入参数：
//   - fn: func()，要包装执行的函数。
//   - wait: time.Duration，节流等待时间（毫秒）。
//
// 返回值：
//   - func()：包装后的节流函数。
func throttleFunc(fn func(), wait time.Duration) func() {
	var lastCall time.Time
	var mu sync.Mutex
	return func() {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if now.Sub(lastCall) >= wait {
			lastCall = now
			fn()
		}
	}
}

// debounceFunc 防抖函数包装器，在最后一次调用后延迟执行。
//
// 功能说明：
// 使用防抖（debouncing）模式，在指定延迟时间内不再触发时再执行回调。
// 适用于用户操作结束后的事件处理（如搜索框输入完成、窗口调整等）。
//
// 输入参数：
//   - fn: func()，要包装执行的函数。
//   - wait: time.Duration，防抖等待时间。
//
// 返回值：
//   - func()：包装后的防抖函数。
func debounceFunc(fn func(), wait time.Duration) func() {
	var timer *time.Timer
	var mu sync.Mutex
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(wait, fn)
	}
}

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
	qListVBox              *fyne.Container              // 题目列表容器
	qListScroll            *container.Scroll            // 题目列表滚动容器
	titleLabel             *widget.Label                // 选择题目页标题标签（用于动态更新）
	nameEntry              *widget.Entry                // 考试名称输入框
	durationEntry          *widget.Entry                // 考试时长输入框
	startTimeLbl           *widget.Label                // 开始时间显示标签
	endTimeLbl             *widget.Label                // 结束时间显示标签
	typeEntries            map[string]*widget.Entry     // 每种题型数量输入框
	typeScoreEntries       map[string]*widget.Entry     // 每种题型分值输入框
	userVBox               *fyne.Container              // 用户列表容器
	userScroll             *container.Scroll            // 用户列表滚动容器
	createBtn              *customElements.CustomButton // 创建推送按钮
	examParamsContent      *fyne.Container              // 考试设置页面内容容器
	examParamsSummary      *fyne.Container              // 考试设置统计信息容器
	wizard                 *CustomWizardTabs            // 向导 TabContainer 引用
	templateRefreshFunc    func()                       // 模板列表刷新函数
	userPageRefreshFunc    func()                       // 目标用户页刷新函数
	confirmPageRefreshFunc func()                       // 确认推送页刷新函数
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
		sourceTab:           0,
		selectedQIDs:        make(map[string]bool),
		selectedUsers:       make(map[int]bool),
		typeScores:          make(map[string]float64),
		typeScoreEntries:    make(map[string]*widget.Entry),
		questionFilterTypes: make(map[string]bool),
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

	var wizardWidget fyne.CanvasObject
	ws.wizard, wizardWidget = NewCustomWizardTabs(
		container.NewTabItem(core.AdminTabSource, wizardPages[0]),
		container.NewTabItem(core.AdminTabExamParams, wizardPages[1]),
		container.NewTabItem(core.AdminTabSelect, wizardPages[2]),
		container.NewTabItem(core.AdminTabUser, wizardPages[3]),
		container.NewTabItem(core.AdminTabConfirm, wizardPages[4]),
		container.NewTabItem(core.AdminTabTemplates, wizardPages[5]),
	)
	ws.wizard.onSelected = func(item *container.TabItem) {
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
	mainLayout := container.NewBorder(topBar, nil, nil, nil, wizardWidget)
	wRootLayout := container.NewStack(pageBg, mainLayout)
	w.SetContent(wRootLayout)
}
