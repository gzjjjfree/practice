package core

import (
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
)

// APIClientInterface 定义网络客户端接口，支持依赖注入与单元测试 Mock
type APIClientInterface interface {
	GetToken() string
	SetToken(token string)
	Login(username, password string) (*network.LoginData, error)
	Logout() error
	RefreshToken(refreshToken string) (*network.RefreshData, error)
	FetchBankList() ([]network.ServerBankItem, error)
	DownloadBank(bankID int) (*parser.BankData, error)
	DownloadServerBankForExam(bankID int) (*parser.BankData, error)
	UploadWrongSet(wrongQuestions []network.WrongQuestionItem) error
	CreateExamSession(examTemplateID int) (*network.ExamSessionData, error)
	SubmitExam(examSessionID string, answers map[string]string) (*network.ExamResult, error)
	GetMyExams() ([]network.MyExamItem, error)
	StartMyExam(examID int) (*network.ExamSessionData, error)
	SubmitMyExam(examSessionID string, answers map[string]string) (*network.ExamResult, error)
	FetchUserList() ([]network.UserItem, error)
	CreateExamTemplate(req network.CreateExamTemplateReq) (*network.ExamTemplateItem, error)
	PushExam(templateID int, targetUserIDs []int) error
	CreateAndPushExam(req network.CreateAndPushReq) (*network.CreateAndPushData, error)
	FetchUsers(limit, offset int) ([]network.UserItem, int, error)
	FetchAdmins(limit, offset int) ([]network.UserItem, int, error)
	CreateUser(req network.CreateUserReq) (*network.UserItem, error)
	UpdateUser(userID int, req network.UpdateUserReq) error
	DeleteUser(userID int) error
	ImportBanks(req network.BankImportReq) (*network.ImportBankData, string, error)
	FetchMyBanks() ([]network.MyBankItem, error)
	FetchAllUsers(limit, offset int) ([]network.UserItem, int, error)
	FetchBanksList(bankName, category string, limit, offset int) ([]network.BankItem, int, error)
	CreateBank(req network.CreateBankReq) (int, error)
	UpdateBank(bankID int, req network.CreateBankReq) error
	DeleteBank(displayName, ownerUsername string) error
	FetchQuestions(bankID int, content, questionType, difficulty string, limit, offset int) ([]network.QuestionItem, int, error)
	CreateQuestion(bankID int, req network.CreateQuestionReq) (int, error)
	UpdateQuestion(bankID int, questionID int, req network.CreateQuestionReq) error
	DeleteQuestion(bankID int, questionID int) error
	BatchImportQuestions(req network.BatchImportQuestionsReq) (*network.ImportBankData, error)
	FetchTemplates(limit, offset int) ([]network.TemplateItem, int, error)
	GetTemplateDetail(templateID int) (*network.TemplateDetailItem, error)
	UpdateTemplate(templateID int, req network.TemplateUpdateReq) error
	DeleteTemplate(templateID int) error
	CancelExam(req network.CancelExamReq) (*network.CancelData, error)
	ExportResults(templateID, examID int, format string) ([]network.ExportResultItem, error)
	LoadLoginLogs(limit, offset int) ([]network.LoginLogItem, int, error)
	PauseExam(req network.PauseExamReq) (*network.PauseResumeData, error)
	ResumeExam(req network.PauseResumeData) (*network.PauseResumeData, error)
	GetExamResult(sessionID string) (*network.ExamDetailResult, error)
}

// TokenStorageInterface 定义 Token 存储接口，支持依赖注入与单元测试 Mock
type TokenStorageInterface interface {
	SaveToken(token string) error
	LoadToken() string
	ClearToken() error
	SaveRefreshToken(refreshToken string) error
	LoadRefreshToken() string
	ClearRefreshToken() error
}
