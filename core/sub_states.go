package core

import (
	"sync"
	"time"

	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
)

// UserState 维护用户认证与身份状态
type UserState struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Role       string `json:"role"`       // "superadmin"/"admin"/"user"
	IsAdmin    bool   `json:"is_admin"`
	Token      string `json:"token"`
	Username   string `json:"username"`
}

// BankState 维护本地及服务器题库状态
type BankState struct {
	CurrentStorageKey string                   `json:"current_storage_key"`
	CurrentFileName   string                   `json:"current_file_name"`
	StoredFiles       []string                 `json:"stored_files"`
	WrongSet          map[string]bool          `json:"wrong_set"` // Question.ID -> bool
	FavSet            map[string]bool          `json:"fav_set"`   // Question.ID -> bool
	ServerBanks       []network.ServerBankItem `json:"server_banks"`
	CurrentServerBank *parser.BankData         `json:"current_server_bank"`
}

// ExamState 维护考试会话与我的考试状态
type ExamState struct {
	ExamSessionID    string               `json:"exam_session_id"`
	ExamAnswers      map[string]string    `json:"exam_answers"` // question_id -> answer
	MyExams          []network.MyExamItem `json:"my_exams"`
	ExamTemplateName string               `json:"exam_template_name"`
}

// PracticeState 维护练习与答题进度状态
type PracticeState struct {
	Questions         []Question                   `json:"questions"`
	CurrentList       []Question                   `json:"current_list"`
	Index             int                          `json:"index"`
	UserAnswers       map[string][]string          `json:"user_answers"`
	Submitted         map[string]bool              `json:"submitted"`
	Title             string                       `json:"title"`
	ModeRecords       map[string]map[string]string `json:"mode_records"`
	PracticeRecords   map[string]string            `json:"practice_records"`
	ModeIndices       map[string]int               `json:"mode_indices"`
	PracticeRecordsMu sync.RWMutex                 `json:"-"`
	ActiveSearchTimer *time.Timer                  `json:"-"`
	SearchGeneration  int                          `json:"-"`
	AnswerGenCounter  int                          `json:"-"`
}

// AdminState 维护后台管理、推送考试及题目池状态
type AdminState struct {
	AvailableUsers      []network.UserItem       `json:"available_users"`
	SelectedUserIDs     map[int]bool             `json:"selected_user_ids"`
	SelectedQuestionIDs map[string]bool          `json:"selected_question_ids"`
	LocalQuestionPool   []Question               `json:"local_question_pool"`
	ServerQuestionPool  []network.ServerQuestion `json:"server_question_pool"`
}
