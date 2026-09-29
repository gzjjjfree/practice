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
	// 格式: "获取题库列表失败: 服务器错误信息"
	if idx := strings.Index(errStr, " 失败: "); idx != -1 {
		serverMsg = errStr[idx+4:]
	} else if idx := strings.Index(errStr, " failed: "); idx != -1 {
		serverMsg = errStr[idx+9:]
	} else if idx := strings.Index(errStr, " 出错: "); idx != -1 {
		serverMsg = errStr[idx+4:]
	}

	// 使用包级别的对话框函数（由初始化时设置）
	if serverMsg != "" {
		if showCustomConfirm != nil {
			showCustomConfirm(msgPrefix, serverMsg, func(yes bool) {}, w)
		}
	} else {
		if showCustomConfirm != nil {
			showCustomConfirm(msgPrefix, "操作失败", func(yes bool) {}, w)
		}
	}
	return true
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

// NormalizeAnswerSlice 将用户答案字符串切分为排序后的标准答案切片。
func NormalizeAnswerSlice(userAnswer string, q Question) []string {
	var slice []string
	if q.Type == "多选题" && strings.Contains(userAnswer, ",") {
		for _, part := range strings.Split(userAnswer, ",") {
			slice = append(slice, strings.TrimSpace(part))
		}
	} else if len(userAnswer) == 1 {
		slice = []string{userAnswer}
	} else {
		// 填空/问答等：按顿号拆分
		for _, part := range strings.Split(userAnswer, "、") {
			if s := strings.TrimSpace(part); s != "" {
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
		if handleServerErrorCode(err, "登录", state.Window) {
			return
		}
		callback(false, "登录失败: "+err.Error())
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
		if handleServerErrorCode(err, "退出登录", state.Window) {
			return
		}
		callback(false, "退出登录失败: "+err.Error())
		return
	}

	callback(true, "已退出登录")
}

// FetchServerBankList retrieves available question banks from server.
func (state *AppState) FetchServerBankList(callback func(success bool, banks []network.ServerBankItem, msg string)) {
	banks, err := state.APIClient.FetchBankList()
	if err != nil {
		if handleServerErrorCode(err, "获取题库列表", state.Window) {
			return
		}
		callback(false, nil, "获取题库列表失败: "+err.Error())
		return
	}

	state.ServerBanks = banks
	callback(true, banks, "获取成功")
}

// DownloadServerBank downloads a question bank from server.
func (state *AppState) DownloadServerBank(bankID int, callback func(success bool, msg string)) {
	bankData, err := state.APIClient.DownloadBank(bankID)
	if err != nil {
		if handleServerErrorCode(err, "下载题库", state.Window) {
			return
		}
		callback(false, "下载题库失败: "+err.Error())
		return
	}

	state.CurrentServerBank = bankData
	callback(true, "题库下载成功")
}

// DownloadServerBankForExam downloads a question bank from server for exam creation.
func (state *AppState) DownloadServerBankForExam(bankID int, callback func(success bool, msg string)) {
	bankData, err := state.APIClient.DownloadServerBankForExam(bankID)
	if err != nil {
		if handleServerErrorCode(err, "下载题库", state.Window) {
			return
		}
		callback(false, "下载题库失败: "+err.Error())
		return
	}

	state.CurrentServerBank = bankData
	callback(true, "题库下载成功")
}

// ==================== 考试相关方法 ====================

// CreateExamSession creates a new exam session.
func (state *AppState) CreateExamSession(examTemplateID int, callback func(success bool, sessionID string, questions []network.ServerQuestion, msg string)) {
	sessionData, err := state.APIClient.CreateExamSession(examTemplateID)
	if err != nil {
		if handleServerErrorCode(err, "创建考试", state.Window) {
			return
		}
		callback(false, "", nil, "创建考试失败: "+err.Error())
		return
	}

	state.ExamSessionID = sessionData.ExamSessionID
	state.ExamAnswers = make(map[string]string)
	callback(true, sessionData.ExamSessionID, sessionData.Questions, "考试已创建")
}

// SubmitExamSubmission submits exam answers to server.
func (state *AppState) SubmitExamSubmission(callback func(success bool, result *network.ExamResult, msg string)) {
	result, err := state.APIClient.SubmitExam(state.ExamSessionID, state.ExamAnswers)
	if err != nil {
		if handleServerErrorCode(err, "提交考试", state.Window) {
			return
		}
		callback(false, nil, "提交考试失败: "+err.Error())
		return
	}

	callback(true, result, "考试提交成功")
}

// UploadWrongSetToServer uploads wrong-set to server.
func (state *AppState) UploadWrongSetToServer(wrongQuestions []network.WrongQuestionItem, callback func(success bool, msg string)) {
	err := state.APIClient.UploadWrongSet(wrongQuestions)
	if err != nil {
		if handleServerErrorCode(err, "上传错题集", state.Window) {
			return
		}
		callback(false, "上传错题集失败: "+err.Error())
		return
	}

	callback(true, "错题集上传成功")
}

// ==================== 管理员推送考试相关方法 ====================

// FetchAvailableUsers retrieves the list of users for exam push.
func (state *AppState) FetchAvailableUsers(callback func(success bool, users []network.UserItem, msg string)) {
	users, err := state.APIClient.FetchUserList()
	if err != nil {
		if handleServerErrorCode(err, "获取用户列表", state.Window) {
			return
		}
		callback(false, nil, "获取用户列表失败: "+err.Error())
		return
	}

	state.AvailableUsers = users
	callback(true, users, "获取成功")
}

// CreateExamTemplate creates a new exam template.
func (state *AppState) CreateExamTemplate(req network.CreateExamTemplateReq, callback func(success bool, templateID int, msg string)) {
	template, err := state.APIClient.CreateExamTemplate(req)
	if err != nil {
		if handleServerErrorCode(err, "创建考试模板", state.Window) {
			return
		}
		callback(false, 0, "创建考试模板失败: "+err.Error())
		return
	}

	callback(true, template.TemplateID, "考试模板创建成功")
}

// PushExamToUsers pushes an exam to selected users.
func (state *AppState) PushExamToUsers(templateID int, targetUserIDs []int, callback func(success bool, msg string)) {
	err := state.APIClient.PushExam(templateID, targetUserIDs)
	if err != nil {
		if handleServerErrorCode(err, "推送考试", state.Window) {
			return
		}
		callback(false, "推送考试失败: "+err.Error())
		return
	}

	callback(true, "考试推送成功")
}

// CreateAndPushExam creates an exam template and pushes it to users in one call.
func (state *AppState) CreateAndPushExam(req network.CreateAndPushReq, callback func(success bool, data *network.CreateAndPushData, msg string)) {
	result, err := state.APIClient.CreateAndPushExam(req)
	if err != nil {
		if handleServerErrorCode(err, "创建并推送考试", state.Window) {
			return
		}
		callback(false, nil, "创建并推送考试失败: "+err.Error())
		return
	}

	callback(true, result, "考试创建并推送成功")
}

// ==================== 我的考试相关方法 ====================

// LoadMyExams retrieves the current user's assigned exams.
func (state *AppState) LoadMyExams(callback func(success bool, exams []network.MyExamItem, msg string)) {
	exams, err := state.APIClient.GetMyExams()
	if err != nil {
		if handleServerErrorCode(err, "获取我的考试", state.Window) {
			return
		}
		callback(false, nil, "获取我的考试失败: "+err.Error())
		return
	}

	state.MyExams = exams
	callback(true, exams, "获取成功")
}

// StartMyExam starts an exam and returns the question session.
func (state *AppState) StartMyExam(examID int, callback func(success bool, sessionID string, questions []network.ServerQuestion, msg string)) {
	sessionData, err := state.APIClient.StartMyExam(examID)
	if err != nil {
		if handleServerErrorCode(err, "开始考试", state.Window) {
			return
		}
		callback(false, "", nil, "开始考试失败: "+err.Error())
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
		if handleServerErrorCode(err, "提交考试", state.Window) {
			return
		}
		callback(false, nil, "提交考试失败: "+err.Error())
		return
	}

	callback(true, result, "考试提交成功")
}

// ==================== 刷新 Token ====================

// RefreshToken refreshes the access token using refresh token.
func (state *AppState) RefreshToken(refreshToken string, callback func(success bool, token string, msg string)) {
	data, err := state.APIClient.RefreshToken(refreshToken)
	if err != nil {
		if handleServerErrorCode(err, "刷新Token", state.Window) {
			return
		}
		callback(false, "", "刷新 Token 失败: "+err.Error())
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
	fmt.Printf("[USER_MGMT] REQUEST: GET /api/v1/users?limit=%d&offset=%d\n", limit, offset)
	users, total, err := state.APIClient.FetchUsers(limit, offset)
	if err != nil {
		fmt.Printf("[USER_MGMT] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "获取用户列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取用户列表失败: "+err.Error())
		return
	}
	fmt.Printf("[USER_MGMT] RESPONSE: total=%d, users=%+v\n", total, users)
	callback(true, users, total, "获取成功")
}

// FetchAdmins retrieves the list of admins with pagination.
func (state *AppState) FetchAdmins(limit, offset int, callback func(success bool, admins []network.UserItem, total int, msg string)) {
	fmt.Printf("[USER_MGMT] REQUEST: GET /api/v1/admins?limit=%d&offset=%d\n", limit, offset)
	admins, total, err := state.APIClient.FetchAdmins(limit, offset)
	if err != nil {
		fmt.Printf("[USER_MGMT] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "获取管理员列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取管理员列表失败: "+err.Error())
		return
	}
	fmt.Printf("[USER_MGMT] RESPONSE: total=%d, admins=%+v\n", total, admins)
	callback(true, admins, total, "获取成功")
}

// CreateUser creates a new user.
func (state *AppState) CreateUser(req network.CreateUserReq, callback func(success bool, user *network.UserItem, msg string)) {
	fmt.Printf("[USER_MGMT] REQUEST: POST /api/v1/users, body={Username:%q, Password:%q, Role:%q}\n", req.Username, req.Password, req.Role)
	user, err := state.APIClient.CreateUser(req)
	if err != nil {
		fmt.Printf("[USER_MGMT] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "创建用户", state.Window) {
			return
		}
		callback(false, nil, "创建用户失败: "+err.Error())
		return
	}
	fmt.Printf("[USER_MGMT] RESPONSE: success, user=%+v\n", user)
	callback(true, user, "用户创建成功")
}

// UpdateUser updates an existing user.
func (state *AppState) UpdateUser(userID int, req network.UpdateUserReq, callback func(success bool, msg string)) {
	var reqBody string
	if req.Username != nil {
		reqBody += fmt.Sprintf("Username:%q ", *req.Username)
	}
	if req.Password != nil {
		reqBody += fmt.Sprintf("Password:%q ", *req.Password)
	}
	if req.Role != nil {
		reqBody += fmt.Sprintf("Role:%q ", *req.Role)
	}
	fmt.Printf("[USER_MGMT] REQUEST: PUT /api/v1/users/%d, body={%s}\n", userID, reqBody)
	err := state.APIClient.UpdateUser(userID, req)
	if err != nil {
		fmt.Printf("[USER_MGMT] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "更新用户", state.Window) {
			return
		}
		callback(false, "更新用户失败: "+err.Error())
		return
	}
	fmt.Printf("[USER_MGMT] RESPONSE: success, userID=%d\n", userID)
	callback(true, "用户更新成功")
}

// DeleteUser deletes a user.
func (state *AppState) DeleteUser(userID int, callback func(success bool, msg string)) {
	fmt.Printf("[USER_MGMT] REQUEST: DELETE /api/v1/users/%d\n", userID)
	err := state.APIClient.DeleteUser(userID)
	if err != nil {
		fmt.Printf("[USER_MGMT] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "删除用户", state.Window) {
			return
		}
		callback(false, "删除用户失败: "+err.Error())
		return
	}
	fmt.Printf("[USER_MGMT] RESPONSE: success, userID=%d\n", userID)
	callback(true, "用户删除成功")
}

// ==================== 题库管理（增强） ====================

// ImportBanks batch imports question banks to server.
// The msg parameter in callback contains the server's msg field, which may include "already exist"
// to indicate duplicate banks. The duplicates parameter contains the names of duplicate banks.
func (state *AppState) ImportBanks(req network.BankImportReq, callback func(success bool, imported, failed int, errors []string, duplicates []string, msg string)) {
	result, msg, err := state.APIClient.ImportBanks(req)
	fmt.Printf("[HELPER_IMPORT] result=%+v, msg=%q, err=%v\n", result, msg, err)
	if err != nil {
		// Always pass msg, errors and duplicates to callback, even when err is not nil,
		// so the caller can check for "already exist" in msg.
		var duplicates []string
		var errors []string
		if result != nil {
			duplicates = result.Duplicates
			errors = result.Errors
		}
		fmt.Printf("[HELPER_IMPORT] err != nil, calling callback with success=false, errors=%v, duplicates=%v, msg=%q\n", errors, duplicates, msg)
		callback(false, 0, 0, errors, duplicates, msg)
		return
	}

	fmt.Printf("[HELPER_IMPORT] success, calling callback with success=true\n")
	callback(true, result.Imported, result.Failed, result.Errors, result.Duplicates, msg)
}

// FetchBanksByOwner retrieves the list of question banks by owner username.
func (state *AppState) FetchBanksByOwner(owner string, callback func(success bool, banks []network.BankItem, total int, msg string)) {
	fmt.Printf("[HELPER_FETCH_BANKS] FetchBanksByOwner called, owner=%s\n", owner)
	banks, total, err := state.APIClient.FetchBanksByOwner(owner, "", 100, 0)
	fmt.Printf("[HELPER_FETCH_BANKS] result: banks=%v, total=%d, err=%v\n", banks, total, err)
	if err != nil {
		fmt.Printf("[HELPER_FETCH_BANKS] hasServerErrorCode=%v\n", hasServerErrorCode(err))
		if handleServerErrorCode(err, "获取题库列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取题库列表失败: "+err.Error())
		return
	}

	fmt.Printf("[HELPER_FETCH_BANKS] calling callback with success=true, banks count=%d\n", len(banks))
	callback(true, banks, total, "获取成功")
}

// FetchMyBanks retrieves the current user's assigned question banks.
func (state *AppState) FetchMyBanks(callback func(success bool, banks []network.MyBankItem, msg string)) {
	banks, err := state.APIClient.FetchMyBanks()
	if err != nil {
		if handleServerErrorCode(err, "获取我的题库", state.Window) {
			return
		}
		callback(false, nil, "获取我的题库失败: "+err.Error())
		return
	}

	callback(true, banks, "获取成功")
}

// FetchAllUsers retrieves all users (admins + regular users).
func (state *AppState) FetchAllUsers(callback func(success bool, users []network.UserItem, total int, msg string)) {
	users, total, err := state.APIClient.FetchAllUsers(100, 0)
	if err != nil {
		fmt.Printf("[HELPER_FETCH_ALL_USERS] API 调用失败: %v\n", err)
		if handleServerErrorCode(err, "获取所有用户", state.Window) {
			return
		}
		callback(false, nil, 0, "获取所有用户失败: "+err.Error())
		return
	}

	fmt.Printf("[HELPER_FETCH_ALL_USERS] 获取成功: total=%d, users=%+v\n", total, users)
	callback(true, users, total, "获取成功")
}

// DownloadBankDetail retrieves the full detail of a question bank via /export endpoint.
// The server returns data structurally identical to parser.BankData.
func (state *AppState) DownloadBankDetail(bankID int, callback func(success bool, detail *parser.BankData, msg string)) {
	detail, err := state.APIClient.DownloadBankDetail(bankID)
	if err != nil {
		if handleServerErrorCode(err, "下载题库详情", state.Window) {
			return
		}
		callback(false, nil, "下载题库详情失败: "+err.Error())
		return
	}

	callback(true, detail, "获取成功")
}

// PushBank pushes a question bank from one owner to another user.
func (state *AppState) PushBank(displayName, ownerUsername string, targetUsernames []string, overwrite bool, callback func(success bool, pushed, updated int, conflict []string, msg string)) {
	fmt.Printf("[HELPER_PUSH_BANK] ========== 开始推送题库 ==========\n")
	fmt.Printf("[HELPER_PUSH_BANK] 输入参数: displayName=%q, ownerUsername=%q, targetUsernames=%v (len=%d), overwrite=%v\n",
		displayName, ownerUsername, targetUsernames, len(targetUsernames), overwrite)

	if len(targetUsernames) == 0 {
		fmt.Printf("[HELPER_PUSH_BANK] 错误: 目标用户列表为空\n")
		callback(false, 0, 0, nil, "目标用户列表不能为空")
		return
	}

	req := network.PushBankReq{
		DisplayName:     displayName,
		OwnerUsername:   ownerUsername,
		TargetUsernames: targetUsernames,
		Overwrite:       overwrite,
	}
	fmt.Printf("[HELPER_PUSH_BANK] 构造的请求: %+v\n", req)

	result, err := state.APIClient.PushBank(req)
	if err != nil {
		fmt.Printf("[HELPER_PUSH_BANK] API 调用失败: %v\n", err)
		if handleServerErrorCode(err, "推送题库", state.Window) {
			return
		}
		callback(false, 0, 0, nil, "推送题库失败: "+err.Error())
		return
	}

	fmt.Printf("[HELPER_PUSH_BANK] 推送结果: success=true, pushed=%d, updated=%d, conflict=%v\n", result.Pushed, result.Updated, result.Conflict)
	fmt.Printf("[HELPER_PUSH_BANK] ========== 推送题库结束 ==========\n")
	callback(true, result.Pushed, result.Updated, result.Conflict, "推送成功")
}

// FetchBanksList retrieves the list of question banks with pagination and filters.
func (state *AppState) FetchBanksList(bankName, category string, limit, offset int, callback func(success bool, banks []network.BankItem, total int, msg string)) {
	banks, total, err := state.APIClient.FetchBanksList(bankName, category, limit, offset)
	if err != nil {
		if handleServerErrorCode(err, "获取题库列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取题库列表失败: "+err.Error())
		return
	}

	callback(true, banks, total, "获取成功")
}

// CreateBank creates a new question bank.
func (state *AppState) CreateBank(req network.CreateBankReq, callback func(success bool, bankID int, msg string)) {
	bankID, err := state.APIClient.CreateBank(req)
	if err != nil {
		if handleServerErrorCode(err, "创建题库", state.Window) {
			return
		}
		callback(false, 0, "创建题库失败: "+err.Error())
		return
	}

	callback(true, bankID, "题库创建成功")
}

// UpdateBank updates an existing question bank.
func (state *AppState) UpdateBank(bankID int, req network.CreateBankReq, callback func(success bool, msg string)) {
	err := state.APIClient.UpdateBank(bankID, req)
	if err != nil {
		if handleServerErrorCode(err, "更新题库", state.Window) {
			return
		}
		callback(false, "更新题库失败: "+err.Error())
		return
	}

	callback(true, "题库更新成功")
}

// DeleteBank deletes a question bank by display_name and owner_username.
func (state *AppState) DeleteBank(displayName, ownerUsername string, callback func(success bool, msg string)) {
	fmt.Printf("[HELPER_DELETE_BANK] REQUEST: displayName=%q, owner_username=%q\n", displayName, ownerUsername)
	err := state.APIClient.DeleteBank(displayName, ownerUsername)
	if err != nil {
		fmt.Printf("[HELPER_DELETE_BANK] RESPONSE: error=%v\n", err)
		if handleServerErrorCode(err, "删除题库", state.Window) {
			return
		}
		callback(false, "删除题库失败: "+err.Error())
		return
	}

	callback(true, "题库删除成功")
}

// ==================== 题目管理 ====================

// FetchQuestions retrieves the list of questions with pagination and filters.
func (state *AppState) FetchQuestions(bankID int, content, questionType, difficulty string, limit, offset int, callback func(success bool, questions []network.QuestionItem, total int, msg string)) {
	questions, total, err := state.APIClient.FetchQuestions(bankID, content, questionType, difficulty, limit, offset)
	if err != nil {
		if handleServerErrorCode(err, "获取题目列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取题目列表失败: "+err.Error())
		return
	}

	callback(true, questions, total, "获取成功")
}

// CreateQuestion adds a new question to a bank.
func (state *AppState) CreateQuestion(bankID int, req network.CreateQuestionReq, callback func(success bool, questionID int, msg string)) {
	questionID, err := state.APIClient.CreateQuestion(bankID, req)
	if err != nil {
		if handleServerErrorCode(err, "创建题目", state.Window) {
			return
		}
		callback(false, 0, "添加题目失败: "+err.Error())
		return
	}

	callback(true, questionID, "题目添加成功")
}

// UpdateQuestion updates an existing question.
func (state *AppState) UpdateQuestion(bankID, questionID int, req network.CreateQuestionReq, callback func(success bool, msg string)) {
	err := state.APIClient.UpdateQuestion(bankID, questionID, req)
	if err != nil {
		if handleServerErrorCode(err, "更新题目", state.Window) {
			return
		}
		callback(false, "更新题目失败: "+err.Error())
		return
	}

	callback(true, "题目更新成功")
}

// DeleteQuestion deletes a question from a bank.
func (state *AppState) DeleteQuestion(bankID, questionID int, callback func(success bool, msg string)) {
	err := state.APIClient.DeleteQuestion(bankID, questionID)
	if err != nil {
		if handleServerErrorCode(err, "删除题目", state.Window) {
			return
		}
		callback(false, "删除题目失败: "+err.Error())
		return
	}

	callback(true, "题目删除成功")
}

// BatchImportQuestions batch imports questions to a bank.
func (state *AppState) BatchImportQuestions(req network.BatchImportQuestionsReq, callback func(success bool, imported, failed int, errors []string, msg string)) {
	result, err := state.APIClient.BatchImportQuestions(req)
	if err != nil {
		if handleServerErrorCode(err, "批量导入题目", state.Window) {
			return
		}
		callback(false, 0, 0, nil, "批量导入题目失败: "+err.Error())
		return
	}

	callback(true, result.Imported, result.Duplicates, result.Errors, "导入完成")
}

// ==================== 考试模板管理 ====================

// FetchTemplates retrieves the list of exam templates with pagination.
func (state *AppState) FetchTemplates(limit, offset int, callback func(success bool, templates []network.TemplateItem, total int, msg string)) {
	templates, total, err := state.APIClient.FetchTemplates(limit, offset)
	if err != nil {
		if handleServerErrorCode(err, "获取模板列表", state.Window) {
			return
		}
		callback(false, nil, 0, "获取考试模板列表失败: "+err.Error())
		return
	}

	callback(true, templates, total, "获取成功")
}

// GetTemplateDetail retrieves the detail of an exam template including per-type question counts.
func (state *AppState) GetTemplateDetail(templateID int, callback func(success bool, detail *network.TemplateDetailItem, msg string)) {
	detail, err := state.APIClient.GetTemplateDetail(templateID)
	if err != nil {
		if handleServerErrorCode(err, "获取模板详情", state.Window) {
			return
		}
		callback(false, nil, "获取模板详情失败: "+err.Error())
		return
	}

	callback(true, detail, "获取成功")
}

// UpdateTemplate updates an existing exam template.
func (state *AppState) UpdateTemplate(templateID int, req network.TemplateUpdateReq, callback func(success bool, msg string)) {
	fmt.Printf("[HELPER_TEMPLATE_UPDATE] templateID=%d, exam_name=%s, duration=%d, single=%d, multi=%d, judge=%d, blank=%d, essay=%d\n",
		templateID, req.ExamName, req.DurationMin,
		ptrInt(req.SingleCount), ptrInt(req.MultiCount), ptrInt(req.JudgeCount), ptrInt(req.BlankCount), ptrInt(req.EssayCount))
	err := state.APIClient.UpdateTemplate(templateID, req)
	if err != nil {
		if handleServerErrorCode(err, "更新模板", state.Window) {
			return
		}
		callback(false, "更新考试模板失败: "+err.Error())
		return
	}

	callback(true, "模板更新成功")
}

// ptrInt returns the int value of a pointer, or 0 if nil.
func ptrInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// DeleteTemplate deletes an exam template.
func (state *AppState) DeleteTemplate(templateID int, callback func(success bool, msg string)) {
	err := state.APIClient.DeleteTemplate(templateID)
	if err != nil {
		if handleServerErrorCode(err, "删除模板", state.Window) {
			return
		}
		callback(false, "删除考试模板失败: "+err.Error())
		return
	}

	callback(true, "模板删除成功")
}

// CancelExam cancels an exam.
func (state *AppState) CancelExam(templateID int, cancelReason string, callback func(success bool, affectedCount int, msg string)) {
	req := network.CancelExamReq{
		TemplateID:   templateID,
		CancelReason: cancelReason,
	}
	data, err := state.APIClient.CancelExam(req)
	if err != nil {
		if handleServerErrorCode(err, "取消考试", state.Window) {
			return
		}
		callback(false, 0, "取消考试失败: "+err.Error())
		return
	}

	callback(true, data.AffectedCount, "考试取消成功，影响 "+fmt.Sprintf("%d", data.AffectedCount)+" 个用户")
}

// ExportResults exports exam results.
func (state *AppState) ExportResults(templateID, examID int, format string, callback func(success bool, results []network.ExportResultItem, msg string)) {
	results, err := state.APIClient.ExportResults(templateID, examID, format)
	if err != nil {
		if handleServerErrorCode(err, "导出结果", state.Window) {
			return
		}
		callback(false, nil, "导出考试结果失败: "+err.Error())
		return
	}

	callback(true, results, "导出成功")
}

// ==================== 登录日志 ====================

// LoadLoginLogs retrieves the login logs with pagination.
func (state *AppState) LoadLoginLogs(limit, offset int, callback func(success bool, logs []network.LoginLogItem, total int, msg string)) {
	logs, total, err := state.APIClient.LoadLoginLogs(limit, offset)
	if err != nil {
		if handleServerErrorCode(err, "获取登录日志", state.Window) {
			return
		}
		callback(false, nil, 0, "获取登录日志失败: "+err.Error())
		return
	}

	callback(true, logs, total, "获取成功")
}

// ==================== 暂停/恢复考试 ====================

// PauseExam pauses an ongoing exam.
func (state *AppState) PauseExam(examSessionID string, callback func(success bool, data *network.PauseResumeData, msg string)) {
	req := network.PauseExamReq{
		ExamSessionID: examSessionID,
	}
	data, err := state.APIClient.PauseExam(req)
	if err != nil {
		if handleServerErrorCode(err, "暂停考试", state.Window) {
			return
		}
		callback(false, nil, "暂停考试失败: "+err.Error())
		return
	}

	callback(true, data, "考试已暂停")
}

// ResumeExam resumes a paused exam.
func (state *AppState) ResumeExam(examSessionID string, callback func(success bool, data *network.PauseResumeData, msg string)) {
	req := network.PauseExamReq{
		ExamSessionID: examSessionID,
	}
	data, err := state.APIClient.ResumeExam(req)
	if err != nil {
		if handleServerErrorCode(err, "恢复考试", state.Window) {
			return
		}
		callback(false, nil, "恢复考试失败: "+err.Error())
		return
	}

	callback(true, data, "考试已恢复")
}

// GetExamResult retrieves the detailed result of an exam.
func (state *AppState) GetExamResult(sessionID string, callback func(success bool, result *network.ExamDetailResult, msg string)) {
	result, err := state.APIClient.GetExamResult(sessionID)
	if err != nil {
		if handleServerErrorCode(err, "获取考试详情", state.Window) {
			return
		}
		callback(false, nil, "获取考试详情失败: "+err.Error())
		return
	}

	callback(true, result, "获取成功")
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
			answers = append(answers, string(char))
		}
		if len(answers) == 0 && srcQ.Answer != "" {
			answers = append(answers, srcQ.Answer)
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
