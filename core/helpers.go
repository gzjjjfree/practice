package core

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"

	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
)

// Question 表示一道题目，包含题干、选项、答案和难度信息。
type Question struct {
	ID         string
	Type       string // "单选题", "多选题", "判断题", "填空题", "问答题"
	Content    string
	Options    []Option
	Answers    []string // 正确答案的 Label，例如 ["A", "C"]
	Difficulty string
	Score      float64
}

// Option 表示题目的一个选项，包含字母标签和文本内容。
type Option struct {
	Label string // A, B, C, D
	Text  string
}

// AppState 维护全局状态和练习会话的状态。
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
	ModeRecords     map[string]map[string]string // ✨ 所有模式的答题记录总集
	PracticeRecords map[string]string            // 本次练习的详细答题记录。key: Question.ID, value: 已答选项(如"A") 或 "null"表示未答

	ModeIndices map[string]int // 用于独立保存每个模式的答题进度 (题号索引)

	// ✨ PracticeRecords 数据竞争保护
	PracticeRecordsMu sync.Mutex

	// ✨ 搜索页防抖 Timer，页面切换时取消以避免访问过期状态
	ActiveSearchTimer *time.Timer
	SearchGeneration  int // 每次输入递增，timer 回调检查是否匹配当前代

	// ✨ 自动推进生成计数器，防止快速答题产生多个并发自动跳转
	AnswerGenCounter int // 每次用户手动答题递增；goroutine 回调比对当前 generation

	// ======= 用户认证相关字段 =======
	IsLoggedIn   bool                  // 是否已登录
	Role         string                // 用户角色 ("superadmin"/"admin"/"user")
	IsAdmin      bool                  // 是否为管理员（admin 或 superadmin）
	Token        string                // JWT token
	Username     string                // 用户名
	APIClient    *network.APIClient    // API客户端
	TokenStorage *network.TokenStorage // Token存储

	// ======= 服务器题库相关字段 =======
	ServerBanks       []network.ServerBankItem // 服务器题库列表
	CurrentServerBank *network.ServerBankData  // 当前下载的服务器题库

	// ======= 考试模式相关字段 =======
	ExamSessionID string            // 当前考试会话ID
	ExamAnswers   map[string]string // 考试答案记录 (question_id -> answer)

	// ======= 管理员推送考试相关字段 =======
	AvailableUsers      []network.UserItem       // 可选推送的用户列表
	SelectedUserIDs     map[int]bool             // 已选中的目标用户 ID
	SelectedQuestionIDs map[string]bool          // 已勾选的题目 ID（用于推送考试）
	ExamTemplateName    string                   // 当前正在创建的考试名称
	LocalQuestionPool   []Question               // 本地题库题目池（筛选用）
	ServerQuestionPool  []network.ServerQuestion // 服务器题库题目池（筛选用）

	// ======= 我的考试相关字段 =======
	MyExams []network.MyExamItem // 当前用户的考试列表

	// ======= UI 相关字段 =======
	Window fyne.Window // 当前窗口，用于显示自定义对话框
}

// NewAppState creates a fully-initialized AppState with all maps/slices ready to use.
func NewAppState() *AppState {
	return &AppState{
		UserAnswers:         make(map[string][]string),
		Submitted:           make(map[string]bool),
		ModeRecords:         make(map[string]map[string]string),
		PracticeRecords:     make(map[string]string),
		WrongSet:            make(map[string]bool),
		FavSet:              make(map[string]bool),
		ModeIndices:         make(map[string]int),
		ExamAnswers:         make(map[string]string),
		SelectedUserIDs:     make(map[int]bool),
		SelectedQuestionIDs: make(map[string]bool),
	}
}

// SetWindow 设置当前窗口，用于显示自定义对话框。
func (state *AppState) SetWindow(w fyne.Window) {
	state.Window = w
}

// hasServerErrorCode 检查错误信息是否包含服务器返回的错误码（code != 0），格式如 "xxx failed: 服务器错误信息"。
func hasServerErrorCode(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// 检查是否包含 "failed: " 或 "失败: " 模式
	return strings.Contains(errStr, " failed: ") || strings.Contains(errStr, " 失败: ") || strings.Contains(errStr, " 出错: ")
}

// handleServerErrorCode 当服务器返回错误码（code != 0）时，显示错误信息。
// 返回 true 表示已处理（已显示对话框），调用方应返回。
// 返回 false 表示不是服务器错误码，调用方应继续处理。
func handleServerErrorCode(err error, msgPrefix string, w fyne.Window) bool {
	if !hasServerErrorCode(err) || w == nil {
		return false
	}
	// 提取服务器返回的错误信息
	errStr := err.Error()
	serverMsg := ""
	for _, prefix := range []string{" 失败: ", " failed: ", " 出错: "} {
		if idx := strings.Index(errStr, prefix); idx != -1 {
			serverMsg = errStr[idx+len(prefix):]
			break
		}
	}

	if showCustomConfirm != nil {
		msg := serverMsg
		if msg == "" {
			msg = "操作失败"
		}
		showCustomConfirm(msgPrefix, msg, func(yes bool) {}, w)
	}
	return true
}

// HandleAPIError 统一处理 API 调用错误（全局错误提示规范）。
// 规范：所有 API 调用失败时均调用此方法，内部优先检查是否为服务器返回的错误码（通过 handleServerErrorCode 弹窗提示），
// 若不是则执行 failCallback 传入格式化的错误信息。
// 返回 true 表示已由服务器错误弹窗处理完毕；返回 false 表示已执行 failCallback。
func (state *AppState) HandleAPIError(err error, actionName string, failCallback func(errMsg string)) bool {
	if err == nil {
		return false
	}
	if handleServerErrorCode(err, actionName, state.Window) {
		return true
	}
	if failCallback != nil {
		failCallback(actionName + "失败: " + err.Error())
	}
	return false
}

// doAPIRequest 统一封装"调用 API + 错误处理 + 回调"的重复模式。
// fn 执行真正的 API 调用并返回 (result, err)；err 为 nil 视为成功。
// failCallback 在发生非服务器错误码时被触发（由 HandleAPIError 内部调用），用于向 UI 报告失败。
// successCallback 在成功时被调用，传入 API 返回值 result。
func doAPIRequest[T any](
	state *AppState,
	actionName string,
	fn func() (T, error),
	failCallback func(errMsg string),
	successCallback func(result T),
) {
	result, err := fn()
	if err != nil {
		state.HandleAPIError(err, actionName, failCallback)
		return
	}
	successCallback(result)
}

// doAPIRequestVoid 专用于只返回 error 的 API 调用（如创建/更新/删除类接口）。
// 语义与 doAPIRequest 一致，仅省略返回值处理。
func doAPIRequestVoid(
	state *AppState,
	actionName string,
	fn func() error,
	failCallback func(errMsg string),
	successCallback func(),
) {
	err := fn()
	if err != nil {
		state.HandleAPIError(err, actionName, failCallback)
		return
	}
	successCallback()
}

// showCustomConfirm 用于显示确认对话框的函数（由初始化时设置）
var showCustomConfirm func(title, message string, callback func(bool), parent fyne.Window)

// SetShowCustomConfirm 设置用于显示确认对话框的函数。
func SetShowCustomConfirm(fn func(title, message string, callback func(bool), parent fyne.Window)) {
	showCustomConfirm = fn
}

// hexColor 将十六进制颜色字符串（如 "#RRGGBB"）转换为 fyne.color.Color。
func HexColor(hex string) color.Color {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 6 {
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	return color.Transparent
}

// Contains 检查字符串切片中是否包含指定元素。
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveElement 从字符串切片中移除指定元素，返回新切片。
func RemoveElement(slice []string, item string) []string {
	var res []string
	for _, s := range slice {
		if s != item {
			res = append(res, s)
		}
	}
	return res
}

// IsAnswerCorrect 判断用户答案是否正确，统一处理所有题型的比对逻辑。
func IsAnswerCorrect(userAnswer string, q Question) bool {
	if userAnswer == "" || userAnswer == "null" {
		return false
	}

	// 填空/问答题：只要查看了（非空）即视为正确
	if q.Type == "填空题" || q.Type == "问答题" {
		return true
	}

	// 将用户答案和标准答案都切分为排序后的字符串切片，消除顺序干扰
	userSlice := NormalizeAnswerSlice(userAnswer, q)
	correctSlice := NormalizeAnswerSlice(strings.Join(q.Answers, ","), q)

	if len(userSlice) != len(correctSlice) {
		return false
	}
	for i := range userSlice {
		if userSlice[i] != correctSlice[i] {
			return false
		}
	}
	return true
}

// NormalizeAnswerLabel 将全角/半角字母或答案进行规范化（如全角 "Ｃ" 转为半角 "C"）
func NormalizeAnswerLabel(s string) string {
	s = strings.TrimSpace(s)
	var sb strings.Builder
	for _, r := range s {
		// 全角大写字母 A-Z (U+FF21 - U+FF3A) 转半角
		if r >= '\uff21' && r <= '\uff3a' {
			sb.WriteRune(r - '\uff21' + 'A')
		} else if r >= '\uff41' && r <= '\uff5a' {
			// 全角小写字母 a-z (U+FF41 - U+FF5A) 转半角大写
			sb.WriteRune(r - '\uff41' + 'A')
		} else if r >= 'a' && r <= 'z' {
			// 小写转大写
			sb.WriteRune(r - 'a' + 'A')
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// NormalizeAnswerSlice 将用户答案字符串切分为排序后的标准答案切片。
func NormalizeAnswerSlice(userAnswer string, q Question) []string {
	var slice []string
	userAnswer = NormalizeAnswerLabel(userAnswer)
	if q.Type == "多选题" && strings.Contains(userAnswer, ",") {
		for _, part := range strings.Split(userAnswer, ",") {
			if s := NormalizeAnswerLabel(part); s != "" {
				slice = append(slice, s)
			}
		}
	} else if len(userAnswer) == 1 || (len(userAnswer) > 0 && len(userAnswer) <= 3 && !strings.Contains(userAnswer, "、")) {
		slice = []string{userAnswer}
	} else {
		// 填空/问答等：按顿号拆分
		for _, part := range strings.Split(userAnswer, "、") {
			if s := NormalizeAnswerLabel(part); s != "" {
				slice = append(slice, s)
			}
		}
	}
	sort.Strings(slice)
	return slice
}

// CreateOptionLabel 创建一个带居中对齐、自动换行和标准字号的 Label。
func CreateOptionLabel(optStr string) *widget.Label {
	label := widget.NewLabel(optStr)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapBreak
	label.SizeName = theme.SizeNameSubHeadingText
	return label
}

// ==================== 用户认证相关方法 ====================

// InitNetwork initializes the API client and token storage.
func (state *AppState) InitNetwork() {
	state.APIClient = network.NewAPIClient()
	state.TokenStorage = network.NewTokenStorage()

	// Try to load saved token
	savedToken := state.TokenStorage.LoadToken()
	if savedToken != "" {
		state.APIClient.SetToken(savedToken)
		state.Token = savedToken
		state.IsLoggedIn = true
	}
}

// Login performs user login.
func (state *AppState) Login(username, password string, callback func(success bool, msg string)) {
	data, err := state.APIClient.Login(username, password)
	if err != nil {
		if state.HandleAPIError(err, "登录", func(msg string) { callback(false, msg) }) {
			return
		}
		return
	}

	state.Token = data.Token
	state.Username = data.Username
	state.Role = data.Role
	state.IsLoggedIn = true
	state.IsAdmin = (data.Role == "admin" || data.Role == "superadmin")
	state.ExamAnswers = make(map[string]string)

	// Save token and refresh token to local storage
	if err := state.TokenStorage.SaveToken(data.Token); err != nil {
		callback(false, "Token保存失败")
		return
	}
	if data.Refresh != "" {
		if err := state.TokenStorage.SaveRefreshToken(data.Refresh); err != nil {
			// refresh token save failure is non-critical
			_ = err
		}
	}

	callback(true, "登录成功")
}

// Logout performs user logout.
func (state *AppState) Logout(callback func(success bool, msg string)) {
	// Clear local state
	state.Token = ""
	state.Username = ""
	state.Role = ""
	state.IsLoggedIn = false
	state.IsAdmin = false
	state.ExamAnswers = make(map[string]string)
	state.CurrentServerBank = nil
	state.ServerBanks = nil

	// Clear saved token and refresh token
	if err := state.TokenStorage.ClearToken(); err != nil {
		if state.HandleAPIError(err, "退出登录", func(msg string) { callback(false, msg) }) {
			return
		}
		return
	}

	callback(true, "已退出登录")
}

// FetchServerBankList retrieves available question banks from server.
func (state *AppState) FetchServerBankList(callback func(success bool, banks []network.ServerBankItem, msg string)) {
	doAPIRequest(state, "获取题库列表",
		func() ([]network.ServerBankItem, error) { return state.APIClient.FetchBankList() },
		func(msg string) { callback(false, nil, msg) },
		func(banks []network.ServerBankItem) {
			state.ServerBanks = banks
			callback(true, banks, "获取成功")
		},
	)
}

// DownloadServerBank downloads a question bank from server.
func (state *AppState) DownloadServerBank(bankID int, callback func(success bool, msg string)) {
	doAPIRequest(state, "下载题库",
		func() (*network.ServerBankData, error) { return state.APIClient.DownloadBank(bankID) },
		func(msg string) { callback(false, msg) },
		func(bankData *network.ServerBankData) {
			state.CurrentServerBank = bankData
			callback(true, "题库下载成功")
		},
	)
}

// DownloadServerBankForExam downloads a question bank from server for exam creation.
func (state *AppState) DownloadServerBankForExam(bankID int, callback func(success bool, msg string)) {
	doAPIRequest(state, "下载题库",
		func() (*network.ServerBankData, error) { return state.APIClient.DownloadServerBankForExam(bankID) },
		func(msg string) { callback(false, msg) },
		func(bankData *network.ServerBankData) {
			state.CurrentServerBank = bankData
			callback(true, "题库下载成功")
		},
	)
}

// ==================== 考试相关方法 ====================

// CreateExamSession creates a new exam session.
func (state *AppState) CreateExamSession(examTemplateID int, callback func(success bool, sessionID string, questions []network.ServerQuestion, msg string)) {
	doAPIRequest(state, "创建考试",
		func() (*network.ExamSessionData, error) { return state.APIClient.CreateExamSession(examTemplateID) },
		func(msg string) { callback(false, "", nil, msg) },
		func(sessionData *network.ExamSessionData) {
			state.ExamSessionID = sessionData.ExamSessionID
			state.ExamAnswers = make(map[string]string)
			callback(true, sessionData.ExamSessionID, sessionData.Questions, "考试已创建")
		},
	)
}

// SubmitExamSubmission submits exam answers to server.
func (state *AppState) SubmitExamSubmission(callback func(success bool, result *network.ExamResult, msg string)) {
	doAPIRequest(state, "提交考试",
		func() (*network.ExamResult, error) { return state.APIClient.SubmitExam(state.ExamSessionID, state.ExamAnswers) },
		func(msg string) { callback(false, nil, msg) },
		func(result *network.ExamResult) { callback(true, result, "考试提交成功") },
	)
}

// UploadWrongSetToServer uploads wrong-set to server.
func (state *AppState) UploadWrongSetToServer(wrongQuestions []network.WrongQuestionItem, callback func(success bool, msg string)) {
	doAPIRequest(state, "上传错题集",
		func() (error, error) { return nil, state.APIClient.UploadWrongSet(wrongQuestions) },
		func(msg string) { callback(false, msg) },
		func(_ error) { callback(true, "错题集上传成功") },
	)
}

// ==================== 管理员推送考试相关方法 ====================

// FetchAvailableUsers retrieves the list of users for exam push.
func (state *AppState) FetchAvailableUsers(callback func(success bool, users []network.UserItem, msg string)) {
	doAPIRequest(state, "获取用户列表",
		func() ([]network.UserItem, error) { return state.APIClient.FetchUserList() },
		func(msg string) { callback(false, nil, msg) },
		func(users []network.UserItem) {
			state.AvailableUsers = users
			callback(true, users, "获取成功")
		},
	)
}

// CreateExamTemplate creates a new exam template.
func (state *AppState) CreateExamTemplate(req network.CreateExamTemplateReq, callback func(success bool, templateID int, msg string)) {
	doAPIRequest(state, "创建考试模板",
		func() (*network.ExamTemplateItem, error) { return state.APIClient.CreateExamTemplate(req) },
		func(msg string) { callback(false, 0, msg) },
		func(template *network.ExamTemplateItem) { callback(true, template.TemplateID, "考试模板创建成功") },
	)
}

// PushExamToUsers pushes an exam to selected users.
func (state *AppState) PushExamToUsers(templateID int, targetUserIDs []int, callback func(success bool, msg string)) {
	doAPIRequest(state, "推送考试",
		func() (error, error) { return nil, state.APIClient.PushExam(templateID, targetUserIDs) },
		func(msg string) { callback(false, msg) },
		func(_ error) { callback(true, "考试推送成功") },
	)
}

// CreateAndPushExam creates an exam template and pushes it to users in one call.
func (state *AppState) CreateAndPushExam(req network.CreateAndPushReq, callback func(success bool, data *network.CreateAndPushData, msg string)) {
	doAPIRequest(state, "创建并推送考试",
		func() (*network.CreateAndPushData, error) { return state.APIClient.CreateAndPushExam(req) },
		func(msg string) { callback(false, nil, msg) },
		func(result *network.CreateAndPushData) { callback(true, result, "考试创建并推送成功") },
	)
}

// ==================== 我的考试相关方法 ====================

// LoadMyExams retrieves the current user's assigned exams.
func (state *AppState) LoadMyExams(callback func(success bool, exams []network.MyExamItem, msg string)) {
	exams, err := state.APIClient.GetMyExams()
	if err != nil {
		if state.HandleAPIError(err, "获取我的考试", func(msg string) { callback(false, nil, msg) }) {
			return
		}
		return
	}

	state.MyExams = exams
	callback(true, exams, "获取成功")
}

// StartMyExam starts an exam and returns the question session.
func (state *AppState) StartMyExam(examID int, callback func(success bool, sessionID string, questions []network.ServerQuestion, msg string)) {
	sessionData, err := state.APIClient.StartMyExam(examID)
	if err != nil {
		if strings.Contains(err.Error(), "exam has timeout") || strings.Contains(err.Error(), "考试已超时") {
			callback(false, "", nil, "考试已超时")
			return
		}
		if state.HandleAPIError(err, "开始考试", func(msg string) { callback(false, "", nil, msg) }) {
			return
		}
		return
	}

	state.ExamSessionID = sessionData.ExamSessionID
	state.ExamAnswers = make(map[string]string)
	callback(true, sessionData.ExamSessionID, sessionData.Questions, "考试已开始")
}

// SubmitMyExamSubmission submits my exam answers and gets results.
func (state *AppState) SubmitMyExamSubmission(callback func(success bool, result *network.ExamResult, msg string)) {
	result, err := state.APIClient.SubmitMyExam(state.ExamSessionID, state.ExamAnswers)
	if err != nil {
		if state.HandleAPIError(err, "提交考试", func(msg string) { callback(false, nil, msg) }) {
			return
		}
		return
	}

	callback(true, result, "考试提交成功")
}

// ==================== 刷新 Token ====================

// RefreshToken refreshes the access token using refresh token.
func (state *AppState) RefreshToken(refreshToken string, callback func(success bool, token string, msg string)) {
	data, err := state.APIClient.RefreshToken(refreshToken)
	if err != nil {
		if state.HandleAPIError(err, "刷新Token", func(msg string) { callback(false, "", msg) }) {
			return
		}
		return
	}

	state.Token = data.Token
	// Save the new refresh token
	if data.Token != "" {
		if err := state.TokenStorage.SaveToken(data.Token); err != nil {
			// non-critical
			_ = err
		}
	}
	callback(true, data.Token, "Token 刷新成功")
}

// RefreshTokenAuto refreshes the access token using the stored refresh token.
func (state *AppState) RefreshTokenAuto(callback func(success bool, token string, msg string)) {
	refreshToken := state.TokenStorage.LoadRefreshToken()
	if refreshToken == "" {
		callback(false, "", "没有可用的 refresh token，请重新登录")
		return
	}
	state.RefreshToken(refreshToken, callback)
}

// ==================== 用户管理 ====================

// FetchUsers retrieves the list of users with pagination.
func (state *AppState) FetchUsers(limit, offset int, callback func(success bool, users []network.UserItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取用户列表",
		func() ([]network.UserItem, error) {
			users, t, err := state.APIClient.FetchUsers(limit, offset)
			total = t
			return users, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(users []network.UserItem) { callback(true, users, total, "获取成功") },
	)
}

// FetchAdmins retrieves the list of admins with pagination.
func (state *AppState) FetchAdmins(limit, offset int, callback func(success bool, admins []network.UserItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取管理员列表",
		func() ([]network.UserItem, error) {
			admins, t, err := state.APIClient.FetchAdmins(limit, offset)
			total = t
			return admins, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(admins []network.UserItem) { callback(true, admins, total, "获取成功") },
	)
}

// CreateUser creates a new user.
func (state *AppState) CreateUser(req network.CreateUserReq, callback func(success bool, user *network.UserItem, msg string)) {
	doAPIRequest(state, "创建用户",
		func() (*network.UserItem, error) { return state.APIClient.CreateUser(req) },
		func(msg string) { callback(false, nil, msg) },
		func(user *network.UserItem) { callback(true, user, "用户创建成功") },
	)
}

// UpdateUser updates an existing user.
func (state *AppState) UpdateUser(userID int, req network.UpdateUserReq, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "更新用户",
		func() error { return state.APIClient.UpdateUser(userID, req) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "用户更新成功") },
	)
}

// DeleteUser deletes a user.
func (state *AppState) DeleteUser(userID int, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "删除用户",
		func() error { return state.APIClient.DeleteUser(userID) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "用户删除成功") },
	)
}

// ==================== 题库管理（增强） ====================

// ImportBanks batch imports question banks to server.
// The msg parameter in callback contains the server's msg field, which may include "already exist"
// to indicate duplicate banks. The duplicates parameter contains the names of duplicate banks.
func (state *AppState) ImportBanks(req network.BankImportReq, callback func(success bool, imported, failed int, errors []string, duplicates []string, msg string)) {
	result, msg, err := state.APIClient.ImportBanks(req)
	if err != nil {
		var duplicates []string
		var errors []string
		if result != nil {
			duplicates = result.Duplicates
			errors = result.Errors
		}
		callback(false, 0, 0, errors, duplicates, msg)
		return
	}

	callback(true, result.Imported, result.Failed, result.Errors, result.Duplicates, msg)
}

// FetchBanksByOwner retrieves the list of question banks by owner username.
func (state *AppState) FetchBanksByOwner(owner string, callback func(success bool, banks []network.BankItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取题库列表",
		func() ([]network.BankItem, error) {
			banks, t, err := state.APIClient.FetchBanksByOwner(owner, "", 100, 0)
			total = t
			return banks, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(banks []network.BankItem) { callback(true, banks, total, "获取成功") },
	)
}

// FetchMyBanks retrieves the current user's assigned question banks.
func (state *AppState) FetchMyBanks(callback func(success bool, banks []network.MyBankItem, msg string)) {
	doAPIRequest(state, "获取我的题库",
		func() ([]network.MyBankItem, error) { return state.APIClient.FetchMyBanks() },
		func(msg string) { callback(false, nil, msg) },
		func(banks []network.MyBankItem) { callback(true, banks, "获取成功") },
	)
}

// FetchAllUsers retrieves all users (admins + regular users).
func (state *AppState) FetchAllUsers(callback func(success bool, users []network.UserItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取所有用户",
		func() ([]network.UserItem, error) {
			users, t, err := state.APIClient.FetchAllUsers(100, 0)
			total = t
			return users, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(users []network.UserItem) { callback(true, users, total, "获取成功") },
	)
}

// DownloadBankDetail retrieves the full detail of a question bank via /export endpoint.
// The server returns data structurally identical to parser.BankData.
func (state *AppState) DownloadBankDetail(bankID int, callback func(success bool, detail *parser.BankData, msg string)) {
	doAPIRequest(state, "下载题库详情",
		func() (*parser.BankData, error) { return state.APIClient.DownloadBankDetail(bankID) },
		func(msg string) { callback(false, nil, msg) },
		func(detail *parser.BankData) { callback(true, detail, "获取成功") },
	)
}

// PushBank pushes a question bank from one owner to another user.
func (state *AppState) PushBank(displayName, ownerUsername string, targetUsernames []string, overwrite bool, callback func(success bool, pushed, updated int, conflict []string, msg string)) {
	if len(targetUsernames) == 0 {
		callback(false, 0, 0, nil, "目标用户列表不能为空")
		return
	}

	req := network.PushBankReq{
		DisplayName:     displayName,
		OwnerUsername:   ownerUsername,
		TargetUsernames: targetUsernames,
		Overwrite:       overwrite,
	}

	result, err := state.APIClient.PushBank(req)
	if err != nil {
		if state.HandleAPIError(err, "推送题库", func(msg string) { callback(false, 0, 0, nil, msg) }) {
			return
		}
		return
	}

	callback(true, result.Pushed, result.Updated, result.Conflict, "推送成功")
}

// FetchBanksList retrieves the list of question banks with pagination and filters.
func (state *AppState) FetchBanksList(bankName, category string, limit, offset int, callback func(success bool, banks []network.BankItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取题库列表",
		func() ([]network.BankItem, error) {
			banks, t, err := state.APIClient.FetchBanksList(bankName, category, limit, offset)
			total = t
			return banks, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(banks []network.BankItem) { callback(true, banks, total, "获取成功") },
	)
}

// CreateBank creates a new question bank.
func (state *AppState) CreateBank(req network.CreateBankReq, callback func(success bool, bankID int, msg string)) {
	doAPIRequest(state, "创建题库",
		func() (int, error) { return state.APIClient.CreateBank(req) },
		func(msg string) { callback(false, 0, msg) },
		func(bankID int) { callback(true, bankID, "题库创建成功") },
	)
}

// UpdateBank updates an existing question bank.
func (state *AppState) UpdateBank(bankID int, req network.CreateBankReq, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "更新题库",
		func() error { return state.APIClient.UpdateBank(bankID, req) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "题库更新成功") },
	)
}

// DeleteBank deletes a question bank by display_name and owner_username.
func (state *AppState) DeleteBank(displayName, ownerUsername string, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "删除题库",
		func() error { return state.APIClient.DeleteBank(displayName, ownerUsername) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "题库删除成功") },
	)
}

// ==================== 题目管理 ====================

// FetchQuestions retrieves the list of questions with pagination and filters.
func (state *AppState) FetchQuestions(bankID int, content, questionType, difficulty string, limit, offset int, callback func(success bool, questions []network.QuestionItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取题目列表",
		func() ([]network.QuestionItem, error) {
			questions, t, err := state.APIClient.FetchQuestions(bankID, content, questionType, difficulty, limit, offset)
			total = t
			return questions, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(questions []network.QuestionItem) { callback(true, questions, total, "获取成功") },
	)
}

// CreateQuestion adds a new question to a bank.
func (state *AppState) CreateQuestion(bankID int, req network.CreateQuestionReq, callback func(success bool, questionID int, msg string)) {
	doAPIRequest(state, "创建题目",
		func() (int, error) { return state.APIClient.CreateQuestion(bankID, req) },
		func(msg string) { callback(false, 0, msg) },
		func(questionID int) { callback(true, questionID, "题目添加成功") },
	)
}

// UpdateQuestion updates an existing question.
func (state *AppState) UpdateQuestion(bankID, questionID int, req network.CreateQuestionReq, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "更新题目",
		func() error { return state.APIClient.UpdateQuestion(bankID, questionID, req) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "题目更新成功") },
	)
}

// DeleteQuestion deletes a question from a bank.
func (state *AppState) DeleteQuestion(bankID, questionID int, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "删除题目",
		func() error { return state.APIClient.DeleteQuestion(bankID, questionID) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "题目删除成功") },
	)
}

// BatchImportQuestions batch imports questions to a bank.
func (state *AppState) BatchImportQuestions(req network.BatchImportQuestionsReq, callback func(success bool, imported, failed int, errors []string, msg string)) {
	doAPIRequest(state, "批量导入题目",
		func() (*network.ImportResult, error) { return state.APIClient.BatchImportQuestions(req) },
		func(msg string) { callback(false, 0, 0, nil, msg) },
		func(result *network.ImportResult) { callback(true, result.Imported, result.Duplicates, result.Errors, "导入完成") },
	)
}

// ==================== 考试模板管理 ====================

// FetchTemplates retrieves the list of exam templates with pagination.
func (state *AppState) FetchTemplates(limit, offset int, callback func(success bool, templates []network.TemplateItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取模板列表",
		func() ([]network.TemplateItem, error) {
			templates, t, err := state.APIClient.FetchTemplates(limit, offset)
			total = t
			return templates, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(templates []network.TemplateItem) { callback(true, templates, total, "获取成功") },
	)
}

// GetTemplateDetail retrieves the detail of an exam template including per-type question counts.
func (state *AppState) GetTemplateDetail(templateID int, callback func(success bool, detail *network.TemplateDetailItem, msg string)) {
	doAPIRequest(state, "获取模板详情",
		func() (*network.TemplateDetailItem, error) { return state.APIClient.GetTemplateDetail(templateID) },
		func(msg string) { callback(false, nil, msg) },
		func(detail *network.TemplateDetailItem) { callback(true, detail, "获取成功") },
	)
}

// UpdateTemplate updates an existing exam template.
func (state *AppState) UpdateTemplate(templateID int, req network.TemplateUpdateReq, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "更新模板",
		func() error { return state.APIClient.UpdateTemplate(templateID, req) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "模板更新成功") },
	)
}

// DeleteTemplate deletes an exam template.
func (state *AppState) DeleteTemplate(templateID int, callback func(success bool, msg string)) {
	doAPIRequestVoid(state, "删除模板",
		func() error { return state.APIClient.DeleteTemplate(templateID) },
		func(msg string) { callback(false, msg) },
		func() { callback(true, "模板删除成功") },
	)
}

// CancelExam cancels an exam.
func (state *AppState) CancelExam(templateID int, cancelReason string, callback func(success bool, affectedCount int, msg string)) {
	req := network.CancelExamReq{
		TemplateID:   templateID,
		CancelReason: cancelReason,
	}
	doAPIRequest(state, "取消考试",
		func() (*network.CancelData, error) { return state.APIClient.CancelExam(req) },
		func(msg string) { callback(false, 0, msg) },
		func(data *network.CancelData) { callback(true, data.AffectedCount, "考试取消成功，影响 "+fmt.Sprintf("%d", data.AffectedCount)+" 个用户") },
	)
}

// ExportResults exports exam results.
func (state *AppState) ExportResults(templateID, examID int, format string, callback func(success bool, results []network.ExportResultItem, msg string)) {
	doAPIRequest(state, "导出结果",
		func() ([]network.ExportResultItem, error) { return state.APIClient.ExportResults(templateID, examID, format) },
		func(msg string) { callback(false, nil, msg) },
		func(results []network.ExportResultItem) { callback(true, results, "导出成功") },
	)
}

// ==================== 登录日志 ====================

// LoadLoginLogs retrieves the login logs with pagination.
func (state *AppState) LoadLoginLogs(limit, offset int, callback func(success bool, logs []network.LoginLogItem, total int, msg string)) {
	var total int
	doAPIRequest(state, "获取登录日志",
		func() ([]network.LoginLogItem, error) {
			logs, t, err := state.APIClient.LoadLoginLogs(limit, offset)
			total = t
			return logs, err
		},
		func(msg string) { callback(false, nil, 0, msg) },
		func(logs []network.LoginLogItem) { callback(true, logs, total, "获取成功") },
	)
}

// ==================== 暂停/恢复考试 ====================

// PauseExam pauses an ongoing exam.
func (state *AppState) PauseExam(examSessionID string, callback func(success bool, data *network.PauseResumeData, msg string)) {
	req := network.PauseExamReq{
		ExamSessionID: examSessionID,
	}
	doAPIRequest(state, "暂停考试",
		func() (*network.PauseResumeData, error) { return state.APIClient.PauseExam(req) },
		func(msg string) { callback(false, nil, msg) },
		func(data *network.PauseResumeData) { callback(true, data, "考试已暂停") },
	)
}

// ResumeExam resumes a paused exam.
func (state *AppState) ResumeExam(examSessionID string, callback func(success bool, data *network.PauseResumeData, msg string)) {
	req := network.PauseExamReq{
		ExamSessionID: examSessionID,
	}
	doAPIRequest(state, "恢复考试",
		func() (*network.PauseResumeData, error) { return state.APIClient.ResumeExam(req) },
		func(msg string) { callback(false, nil, msg) },
		func(data *network.PauseResumeData) { callback(true, data, "考试已恢复") },
	)
}

// GetExamResult retrieves the detailed result of an exam.
func (state *AppState) GetExamResult(sessionID string, callback func(success bool, result *network.ExamDetailResult, msg string)) {
	doAPIRequest(state, "获取考试详情",
		func() (*network.ExamDetailResult, error) { return state.APIClient.GetExamResult(sessionID) },
		func(msg string) { callback(false, nil, msg) },
		func(result *network.ExamDetailResult) { callback(true, result, "获取成功") },
	)
}

// SyncQuestionsToState converts parsed parser.BankData into the in-memory Question slice inside AppState.
func SyncQuestionsToState(state *AppState, bankData *parser.BankData) {
	state.Questions = make([]Question, len(bankData.Questions))
	for idx, srcQ := range bankData.Questions {
		var convertedOpts []Option
		for _, k := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"} {
			if srcQ.Options[k] != "" {
				convertedOpts = append(convertedOpts, Option{Label: k, Text: srcQ.Options[k]})
			}
		}
		var answers []string
		cleanAns := strings.NewReplacer("、", "", ",", "", " ", "").Replace(srcQ.Answer)
		for _, char := range cleanAns {
			norm := NormalizeAnswerLabel(string(char))
			if norm != "" {
				answers = append(answers, norm)
			}
		}
		if len(answers) == 0 && srcQ.Answer != "" {
			answers = append(answers, NormalizeAnswerLabel(srcQ.Answer))
		}

		state.Questions[idx] = Question{
			ID:         fmt.Sprintf("%d", srcQ.ID),
			Type:       srcQ.Type,
			Content:    srcQ.Content,
			Options:    convertedOpts,
			Answers:    answers,
			Difficulty: srcQ.Difficulty,
		}
	}

	if state.ModeIndices == nil {
		state.ModeIndices = make(map[string]int)
	}
}

// CalculateCurrentModeStats counts correct/wrong answers from PracticeRecords for the current mode.
func CalculateCurrentModeStats(state *AppState) (correctCount int, wrongCount int) {
	state.PracticeRecordsMu.Lock()
	defer state.PracticeRecordsMu.Unlock()

	for _, q := range state.CurrentList {
		historyAns, exists := state.PracticeRecords[q.ID]
		if !exists || historyAns == "null" || historyAns == "" {
			continue
		}

		if IsAnswerCorrect(historyAns, q) {
			correctCount++
		} else {
			wrongCount++
		}
	}
	return correctCount, wrongCount
}

// TruncateUsername 截断用户名用于显示：长度超过 8 个字符时，显示前 5 个字符 + "..."；否则返回原用户名。
func TruncateUsername(username string) string {
	if len(username) > 8 {
		return username[:5] + "..."
	}
	return username
}

// ==================== 12. 响应式布局与屏幕尺寸检测 ====================

// GetCanvasSize 获取窗口的当前画布尺寸，如果窗口为空或画布为空则返回默认尺寸 (400, 750)。
func GetCanvasSize(w fyne.Window) fyne.Size {
	if w != nil && w.Canvas() != nil {
		size := w.Canvas().Size()
		if size.Width > 0 && size.Height > 0 {
			return size
		}
	}
	return fyne.NewSize(400, 750)
}

// IsSmallScreen 检测当前窗口/屏幕是否为小屏幕（例如手机端或小分辨率窗口宽度 < 550 或高度 < 700）。
func IsSmallScreen(w fyne.Window) bool {
	size := GetCanvasSize(w)
	return size.Width < 550 || size.Height < 700
}

// GetResponsiveWidth 根据窗口宽度计算响应式内容宽度（例如最大限制或边距适配）。
func GetResponsiveWidth(w fyne.Window, maxAllowedWidth float32, margin float32) float32 {
	canvasSize := GetCanvasSize(w)
	width := canvasSize.Width - (margin * 2)
	if width > maxAllowedWidth {
		width = maxAllowedWidth
	}
	if width < 100 {
		width = 100
	}
	return width
}

// SetupResponsiveWindow 设置窗口的自适应大小及响应式调整策略。
func SetupResponsiveWindow(w fyne.Window, minWidth, minHeight float32) {
	if w != nil {
		w.Resize(fyne.NewSize(minWidth, minHeight))
	}
}
