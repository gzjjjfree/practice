package network

// ==================== 登录相关 ====================

// LoginReq 登录请求体，包含用户名和密码。
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResp 登录 API 的响应体。
// 服务器返回包装格式：{code:0, msg:"success", data: LoginData}
type LoginResp struct {
	Code int       `json:"code"`
	Msg  string    `json:"msg"`
	Data LoginData `json:"data"`
}

// LoginData 登录成功后返回的用户信息。
// 注意：/auth/login 端点返回扁平化 JSON（无 code/msg/data 包裹层），
// 而其他所有 API 端点返回 {code, msg, data} 格式。
type LoginData struct {
	Token     string `json:"token"`      // JWT 认证令牌
	Refresh   string `json:"refresh"`    // 刷新令牌
	UserID    int    `json:"user_id"`    // 用户 ID
	Username  string `json:"username"`   // 用户名
	Role      string `json:"role"`       // 角色："superadmin"、"admin" 或 "user"
	IPAddress string `json:"ip_address"` // 登录 IP 地址
}

// ==================== 题库列表相关 ====================

// BankListResp 题库列表 API 的响应体。
type BankListResp struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data []ServerBankItem `json:"data"`
}

// ServerBankItem 服务器上可用的题库项。
type ServerBankItem struct {
	BankID        int    `json:"bank_id"`        // 题库 ID
	DisplayName   string `json:"display_name"`   // 显示名称
	QuestionCount int    `json:"question_count"` // 题目数量
	Description   string `json:"description"`    // 描述
}

// ==================== 下载题库相关 ====================

// BankDownloadReq 下载题库的请求体。
type BankDownloadReq struct {
	BankID int `json:"bank_id"` // 题库 ID
}

// BankDownloadResp 下载题库 API 的响应体。
type BankDownloadResp struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data ServerBankData `json:"data"`
}

// ServerBankData 来自服务器的完整题库数据。
type ServerBankData struct {
	BankID      int              `json:"bank_id"`      // 题库 ID
	DisplayName string           `json:"display_name"` // 显示名称
	Questions   []ServerQuestion `json:"questions"`    // 题目列表
}

// ServerQuestion 来自服务器的题目。
type ServerQuestion struct {
	ID         interface{}       `json:"id"`         // 题目 ID (can be int or string)
	Type       string            `json:"type"`       // 题型
	Content    string            `json:"content"`    // 题干内容
	Options    map[string]string `json:"options"`    // 选项（A→文本）
	Answer     string            `json:"answer"`     // 答案
	Difficulty string            `json:"difficulty"` // 难度
	Score      float64           `json:"score"`      // 分值
}

// ==================== 考试相关 ====================

// ExamCreateReq 创建考试会话的请求体。
type ExamCreateReq struct {
	ExamTemplateID int `json:"exam_template_id"` // 考试模板 ID
}

// ExamCreateResp 创建考试会话 API 的响应体。
type ExamCreateResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data ExamSessionData `json:"data"`
}

// ExamSessionData 考试会话信息。
type ExamSessionData struct {
	ExamSessionID string           `json:"exam_session_id"` // 会话 ID
	Questions     []ServerQuestion `json:"questions"`       // 题目列表
}

// ExamSubmitReq 提交考试答案的请求体。
type ExamSubmitReq struct {
	ExamSessionID string            `json:"exam_session_id"` // 会话 ID
	Answers       map[string]string `json:"answers"`         // 答案（题目ID→用户答案）
}

// ExamSubmitResp 提交考试答案 API 的响应体。
type ExamSubmitResp struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data ExamResult `json:"data"`
}

// ExamResult 考试评分信息。
type ExamResult struct {
	Score        float64          `json:"score"`         // 得分
	CorrectCount int              `json:"correct_count"` // 正确题数
	TotalCount   int              `json:"total_count"`   // 总题数
	TimeUsedSec  int              `json:"time_used_sec"` // 用时（秒）
	Items        []ExamResultItem `json:"items"`         // 每题详情
}

// ==================== 错题集上传相关 ====================

// WrongSetUploadReq 上传错题集的请求体。
type WrongSetUploadReq struct {
	WrongQuestions []WrongQuestionItem `json:"wrong_questions"` // 错题列表
}

// WrongQuestionItem 用于上传的错题项。
type WrongQuestionItem struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	Answer     string `json:"answer"`
	UserAnswer string `json:"user_answer"`
}

// WrongSetUploadResp is the response from wrong-set upload API.
type WrongSetUploadResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ==================== 用户列表相关（管理员推送考试用） ====================

// UserItem represents a user available for exam push.
type UserItem struct {
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	Role              string `json:"role"` // "admin", "superadmin", or "user"
	CreatedBy         int    `json:"created_by"`
	CreatedAt         string `json:"created_at"`
	CreatedByUsername string `json:"created_by_username"`
}

// UserListResp is the response from user list API.
type UserListResp struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data []UserItem `json:"data"`
}

// ==================== 考试模板创建相关（管理员） ====================

// CreateExamTemplateReq is the request body for creating an exam template.
type CreateExamTemplateReq struct {
	LocalBankKey string           `json:"local_bank_key,omitempty"` // local题库 storage key (one of)
	ServerBankID int              `json:"server_bank_id,omitempty"` // server bank ID (one of)
	QuestionIDs  []string         `json:"question_ids,omitempty"`   // selected question IDs (optional)
	Questions    []ServerQuestion `json:"questions"`                // complete questions
	ExamName     string           `json:"exam_name"`                // exam name
	DurationMin  int              `json:"duration_min"`             // duration in minutes
	StartTime    string           `json:"start_time"`               // ISO 8601
	EndTime      string           `json:"end_time"`                 // ISO 8601
}

// CreateExamTemplateResp is the response from exam template create API.
type CreateExamTemplateResp struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data ExamTemplateItem `json:"data"`
}

// CreateAndPushReq is the request body for creating an exam template and pushing it to users.
// This merges the two-step create+push into a single API call.
type CreateAndPushReq struct {
	// Template parameters
	LocalBankKey  string           `json:"local_bank_key,omitempty"` // local题库 storage key
	ServerBankID  int              `json:"server_bank_id,omitempty"` // server bank ID
	QuestionIDs   []string         `json:"question_ids,omitempty"`   // manually selected question IDs
	TypeCounts    map[string]int   `json:"type_counts,omitempty"`    // per-type question counts: {"单选题": 10, "多选题": 5, ...}
	Questions     []ServerQuestion `json:"questions"`                // complete questions
	ExamName      string           `json:"exam_name"`                // exam name
	DurationMin   int              `json:"duration_min"`             // duration in minutes
	StartTime     string           `json:"start_time"`               // ISO 8601
	EndTime       string           `json:"end_time"`                 // ISO 8601
	// Push targets
	TargetUserIDs []int            `json:"target_user_ids"`          // target user IDs to push to
}

// CreateAndPushResp is the response from create-and-push API.
type CreateAndPushResp struct {
	Code int               `json:"code"`
	Msg  string            `json:"msg"`
	Data CreateAndPushData `json:"data"`
}

// CreateAndPushData contains the result of the create-and-push operation.
type CreateAndPushData struct {
	TemplateID    int `json:"template_id"`
	QuestionCount int `json:"question_count"`
	PushedCount   int `json:"pushed_count"`
}

// ExamTemplateItem represents an exam template created by admin.
type ExamTemplateItem struct {
	TemplateID    int    `json:"template_id"`
	BankName      string `json:"bank_name"`
	QuestionCount int    `json:"question_count"`
	DurationMin   int    `json:"duration_min"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Status        string `json:"status"` // "draft" / "active" / "expired"
}

// ==================== 推送考试相关 ====================

// PushExamReq is the request body for pushing exam to users.
type PushExamReq struct {
	TemplateID    int   `json:"template_id"`
	TargetUserIDs []int `json:"target_user_ids"`
}

// PushExamResp is the response from push exam API.
type PushExamResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ==================== 我的考试相关（普通用户） ====================

// MyExamItem represents an exam assigned to a user.
type MyExamItem struct {
	ExamID        int      `json:"exam_id"`
	TemplateID    int      `json:"template_id"`
	BankName      string   `json:"bank_name"`
	QuestionCount int      `json:"question_count"`
	DurationMin   int      `json:"duration_min"`
	StartTime     string   `json:"start_time"`
	EndTime       string   `json:"end_time"`
	Status        string   `json:"status"`            // "upcoming" / "active" / "in_progress" / "timeout" / "completed"
	Score         *float64 `json:"score,omitempty"`   // nil = not taken
	ExamSessionID string   `json:"exam_session_id,omitempty"` // session ID
}

// MyExamResp is the response from my exams list API.
type MyExamResp struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data []MyExamItem `json:"data"`
}

// ==================== 开始考试 / 提交考试 ====================

// StartExamReq is the request body for starting an exam.
type StartExamReq struct {
	ExamID int `json:"exam_id"`
}

// StartExamResp is the response from start exam API.
type StartExamResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data ExamSessionData `json:"data"`
}

// SubmitMyExamReq is the request body for submitting my exam answers.
type SubmitMyExamReq struct {
	ExamSessionID string            `json:"exam_session_id"`
	Answers       map[string]string `json:"answers"`
}

// SubmitMyExamResp is the response from submit my exam API.
type SubmitMyExamResp struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data ExamResult `json:"data"`
}

// ==================== 分页响应 ====================

// PaginatedResp is the standard paginated response wrapper.
type PaginatedResp struct {
	Code   int         `json:"code"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// ==================== 刷新 Token 相关 ====================

// RefreshReq is the request body for refreshing access token.
type RefreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResp is the response from refresh token API.
type RefreshResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data RefreshData `json:"data"`
}

// RefreshData contains the new access token.
type RefreshData struct {
	Token string `json:"token"`
}

// ==================== 用户管理相关 ====================

// CreateUserReq is the request body for creating a user.
type CreateUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // "user" or "admin", default "user"
}

// UpdateUserReq is the request body for updating a user.
type UpdateUserReq struct {
	Username *string `json:"username,omitempty"` // pointer to allow nil
	Password *string `json:"password,omitempty"` // pointer to allow nil
	Role     *string `json:"role,omitempty"`     // pointer to allow nil
}

// UserCreateResp is the response from create user API.
type UserCreateResp struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data UserItem `json:"data"`
}

// UserUpdateResp is the response from update user API.
type UserUpdateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// UserDeleteResp is the response from delete user API.
type UserDeleteResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ImportResult contains import statistics.
type ImportResult struct {
	Imported   int      `json:"imported"`
	Duplicates int      `json:"duplicates"`
	Errors     []string `json:"errors"`
}

// ==================== 题库管理相关 ====================

// CreateBankReq is the request body for creating a question bank.
type CreateBankReq struct {
	BankName string `json:"bank_name"`
	Category string `json:"category"`
	LocalKey string `json:"local_key"`
	IsActive bool   `json:"is_active"`
}

// BankItem represents a question bank item in the list.
type BankItem struct {
	BankID            int    `json:"bank_id"`
	BankName          string `json:"bank_name"`
	DisplayName       string `json:"display_name"`
	Category          string `json:"category"`
	LocalKey          string `json:"local_key"`
	CreatedBy         int    `json:"created_by"`
	CreatedByUsername string `json:"created_by_username"`
	IsActive          bool   `json:"is_active"`
	CreatedAt         string `json:"created_at"`
}

// BankCreateResp is the response from create bank API.
type BankCreateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		BankID int `json:"bank_id"`
	} `json:"data"`
}

// BankUpdateResp is the response from update bank API.
type BankUpdateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// BankDeleteResp is the response from delete bank API.
type BankDeleteResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ==================== 题目管理相关 ====================

// CreateQuestionReq is the request body for adding a question to a bank.
type CreateQuestionReq struct {
	Content       string            `json:"content"`
	QuestionType  string            `json:"question_type"`
	Options       map[string]string `json:"options"`
	CorrectAnswer string            `json:"correct_answer"`
	Explanation   string            `json:"explanation"`
	Difficulty    string            `json:"difficulty"`
	Score         float64           `json:"score"`
}

// QuestionItem represents a question item in the list.
type QuestionItem struct {
	QuestionID    int               `json:"question_id"`
	BankID        int               `json:"bank_id"`
	Content       string            `json:"content"`
	QuestionType  string            `json:"question_type"`
	Options       map[string]string `json:"options"`
	CorrectAnswer string            `json:"correct_answer"`
	Explanation   string            `json:"explanation"`
	Difficulty    string            `json:"difficulty"`
	Score         float64           `json:"score"`
	CreatedAt     string            `json:"created_at"`
}

// QuestionCreateResp is the response from create question API.
type QuestionCreateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		QuestionID int `json:"question_id"`
	} `json:"data"`
}

// QuestionUpdateResp is the response from update question API.
type QuestionUpdateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// QuestionDeleteResp is the response from delete question API.
type QuestionDeleteResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// BatchImportQuestionsReq is the request body for batch importing questions.
type BatchImportQuestionsReq struct {
	BankID    int                 `json:"bank_id"`
	Questions []CreateQuestionReq `json:"questions"`
}

// BatchImportQuestionsResp is the response from batch import questions API.
type BatchImportQuestionsResp struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data ImportResult `json:"data"`
}

// ==================== 考试模板管理相关 ====================

// TemplateItem is the response from list templates API.
type TemplateItem struct {
	TemplateID    int                            `json:"template_id"`
	ExamName      string                         `json:"exam_name"`
	DurationMin   int                            `json:"duration_min"`
	QuestionCount int                            `json:"question_count"`
	StartTime     string                         `json:"start_time"`
	EndTime       string                         `json:"end_time"`
	BankSource    string                         `json:"bank_source"` // "local" or "server"
	CreatedBy     int                            `json:"created_by"`
	CreatedAt     string                         `json:"created_at"`
	Status        string                         `json:"status"` // "draft" / "active" / "expired"
	QuestionTypeDistribution QuestionTypeDistribution `json:"question_type_distribution"`
}

// QuestionTypeDistribution represents the per-type question counts in a template.
type QuestionTypeDistribution struct {
	SingleChoice int `json:"single_choice"` // 单选题数量
	MultipleChoice int `json:"multiple_choice"` // 多选题数量
	TrueFalse    int `json:"true_false"`    // 判断题数量
	FillBlank    int `json:"fill_blank"`    // 填空题数量
	Essay        int `json:"essay"`         // 问答题数量
}

// TemplateUpdateReq is the request body for updating a template.
type TemplateUpdateReq struct {
	ExamName    string `json:"exam_name,omitempty"`
	DurationMin int    `json:"duration_min,omitempty"`
	// Per-type question counts
	SingleCount *int `json:"single_count,omitempty"`
	MultiCount  *int `json:"multi_count,omitempty"`
	JudgeCount  *int `json:"judge_count,omitempty"`
	BlankCount  *int `json:"blank_count,omitempty"`
	EssayCount  *int `json:"essay_count,omitempty"`
}

// TemplateUpdateResp is the response from update template API.
type TemplateUpdateResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TemplateDeleteResp is the response from delete template API.
type TemplateDeleteResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TemplateDetailItem represents the detail of an exam template with per-type question counts.
type TemplateDetailItem struct {
	TemplateID    int    `json:"template_id"`
	ExamName      string `json:"exam_name"`
	DurationMin   int    `json:"duration_min"`
	QuestionCount int    `json:"question_count"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	BankSource    string `json:"bank_source"` // "local" or "server"
	CreatedBy     int    `json:"created_by"`
	CreatedAt     string `json:"created_at"`
	Status        string `json:"status"` // "draft" / "active" / "expired"
	// Per-type question counts
	SingleCount int `json:"single_count"` // 单选题数量
	MultiCount  int `json:"multi_count"`  // 多选题数量
	JudgeCount  int `json:"judge_count"`  // 判断题数量
	BlankCount  int `json:"blank_count"`  // 填空题数量
	EssayCount  int `json:"essay_count"`  // 问答题数量
}

// GetTemplateDetailResp is the response from template detail API.
type GetTemplateDetailResp struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data TemplateDetailItem `json:"data"`
}

// CancelExamReq is the request body for canceling an exam.
type CancelExamReq struct {
	TemplateID   int    `json:"template_id"`
	CancelReason string `json:"cancel_reason,omitempty"`
}

// CancelExamResp is the response from cancel exam API.
type CancelExamResp struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data CancelData `json:"data"`
}

// CancelData contains the number of affected users.
type CancelData struct {
	AffectedCount int `json:"affected_count"`
}

// ExportResultsResp is the response from export results API.
type ExportResultsResp struct {
	Code int                `json:"code"`
	Msg  string             `json:"msg"`
	Data []ExportResultItem `json:"data"`
}

// ExportResultItem represents an exam result for export.
type ExportResultItem struct {
	Username     string  `json:"username"`
	UserID       int     `json:"user_id"`
	ExamName     string  `json:"exam_name"`
	StartTime    string  `json:"start_time"`
	SubmitTime   string  `json:"submit_time"`
	Score        float64 `json:"score"`
	TotalScore   float64 `json:"total_score"`
	CorrectCount int     `json:"correct_count"`
	TotalCount   int     `json:"total_count"`
	Status       string  `json:"status"`
}

// ==================== 登录日志相关 ====================

// LoginLogItem represents a login log entry.
type LoginLogItem struct {
	LogID      int    `json:"log_id"`
	Username   string `json:"username"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	Success    bool   `json:"success"`
	FailReason string `json:"fail_reason"`
	CreatedAt  string `json:"created_at"`
}

// ==================== 考试详情/结果相关 ====================

// ExamResultItem represents a single question result in exam.
type ExamResultItem struct {
	QuestionID    string            `json:"question_id"`
	IsCorrect     bool              `json:"is_correct"`
	UserAnswer    string            `json:"user_answer"`
	CorrectAnswer string            `json:"correct_answer"`
	QuestionType  string            `json:"question_type"`
	Content       string            `json:"content"`
	Options       map[string]string `json:"options"`
}

// ExamDetailResult contains full exam result details.
type ExamDetailResult struct {
	Score        float64          `json:"score"`
	CorrectCount int              `json:"correct_count"`
	TotalCount   int              `json:"total_count"`
	TimeUsedSec  int              `json:"time_used_sec"`
	Items        []ExamResultItem `json:"items"`
}

// ==================== 题库批量导入相关（新增） ====================

// ImportQuestionItem represents a single question in a bank import request.
type ImportQuestionItem struct {
	ID         int               `json:"id"`
	Type       string            `json:"type"`
	Content    string            `json:"content"`
	Answer     string            `json:"answer"`
	Options    map[string]string `json:"options,omitempty"`
	Difficulty string            `json:"difficulty,omitempty"`
	Score      float64           `json:"score,omitempty"`
}

// BankImportItem represents a single bank in a batch import request.
type BankImportItem struct {
	DisplayName string               `json:"displayName"`
	StorageKey  string               `json:"storageKey"`
	Questions   []ImportQuestionItem `json:"questions"`
	Overwrite   bool                 `json:"overwrite,omitempty"` // whether to overwrite existing bank
}

// BankImportReq is the request body for batch importing question banks.
type BankImportReq struct {
	OwnerUsername string           `json:"owner_username"`
	Banks         []BankImportItem `json:"banks"`
}

// BankImportResp is the response from batch import banks API.
type BankImportResp struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data ImportBankData `json:"data"`
}

// ImportBankData contains import statistics.
type ImportBankData struct {
	Imported   int      `json:"imported"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors"`
	Duplicates []string `json:"duplicates"`
}

// ==================== 按归属获取题库列表相关 ====================

// FetchBanksByOwnerResp is the response from /api/v1/banks/owner.
// Reuses BankItem which already has the correct fields.

// ==================== 我的题库相关 ====================

// MyBankItem represents a question bank in the user's my-banks list.
type MyBankItem struct {
	MyBankID      int    `json:"my_bank_id"`
	BankID        int    `json:"bank_id"`
	BankName      string `json:"bank_name"`
	OwnerUsername string `json:"owner_username"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// MyBankResp is the response from my-banks API.
type MyBankResp struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data []MyBankItem `json:"data"`
}

// ==================== 题库详情相关 ====================

// BankDetailResp is the response from /api/v1/banks/{bank_id}/detail.
type BankDetailResp struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data BankDetailData `json:"data"`
}

// BankDetailData contains the full bank detail including questions (nested format).
type BankDetailData struct {
	BankID    int            `json:"bank_id"`
	BankName  string         `json:"bank_name"`
	LocalKey  string         `json:"local_key"`
	Data      BankImportItem `json:"data"`
	CreatedBy int            `json:"created_by"`
	CreatedAt string         `json:"created_at"`
}

// BankDetailExportData is used for /export endpoint which returns a flat BankData-like structure: displayName/storageKey/questions directly under data.
type BankDetailExportData struct {
	DisplayName string               `json:"displayName"`
	StorageKey  string               `json:"storageKey"`
	Questions   []ExportQuestionItem `json:"questions"`
}

// ExportQuestionItem represents a question from the /export endpoint.
type ExportQuestionItem struct {
	ID         int               `json:"id"`
	Type       string            `json:"type"`
	Content    string            `json:"content"`
	Answer     string            `json:"answer"`
	Options    map[string]string `json:"options,omitempty"`
	Difficulty string            `json:"difficulty,omitempty"`
	Score      float64           `json:"score,omitempty"`
}

// ==================== 题库推送相关 ====================

// PushBankReq is the request body for pushing a question bank to another user.
// POST /api/v1/banks/push-by-names
// 服务器接口规范：target_usernames 为数组类型（支持多用户推送）
type PushBankReq struct {
	DisplayName     string   `json:"display_name"`        // 原归属下的题库名称
	OwnerUsername   string   `json:"owner_username"`      // 题库原归属的管理员账号
	TargetUsernames []string `json:"target_usernames"`    // 目标归属（目标管理员账号列表，支持多用户推送）
	Overwrite       bool     `json:"overwrite,omitempty"` // 目标已有同名题库时是否覆盖，默认 false
}

// PushBankResp is the response from push bank API.
type PushBankResp struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data PushBankData `json:"data"`
}

// PushBankData contains push statistics.
type PushBankData struct {
	Pushed   int      `json:"pushed"`
	Updated  int      `json:"updated"`
	Conflict []string `json:"conflict"`
}

// ==================== 暂停/恢复考试相关 ====================

// PauseExamReq is the request body for pausing an exam.
type PauseExamReq struct {
	ExamSessionID string `json:"exam_session_id"`
}

// PauseExamResp is the response from pause exam API.
type PauseExamResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data PauseResumeData `json:"data"`
}

// ResumeExamResp is the response from resume exam API.
type ResumeExamResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data PauseResumeData `json:"msg"`
}

// PauseResumeData contains paused exam state.
type PauseResumeData struct {
	ExamSessionID string            `json:"exam_session_id"`
	Questions     []ServerQuestion  `json:"questions"`
	Answers       map[string]string `json:"answers"`
	RemainingSec  int               `json:"remaining_sec"`
}
