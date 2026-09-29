package network

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gzjjjfree/practice/parser"
)

// BaseURL is the server address — must use IP 144.34.181.64 with HTTPS and port 8443.
// All API paths use the /api/v1/ prefix per FRONTEND_API_GUIDE.md.
const BaseURL = "https://144.34.181.64:8443/api/v1"

// APIClient holds the HTTP client and auth state.
type APIClient struct {
	client *http.Client
	token  string
}

// NewAPIClient creates a new API client with explicit timeout and safe transport.
// TLS verification is skipped because the server uses a CloudFlare origin certificate
// that does not include the IP address 144.34.181.64 in its SANs.
func NewAPIClient() *APIClient {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // IP cert workaround
	}
	return &APIClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // 30s request timeout to avoid infinite hangs
		},
		token: "",
	}
}

// SetToken sets the authentication token.
func (c *APIClient) SetToken(token string) {
	c.token = token
}

// GetToken returns the current token.
func (c *APIClient) GetToken() string {
	return c.token
}

// doRequest performs an HTTP request with JSON body and returns parsed response.
// Accepts all 2xx success status codes (200 OK, 201 Created, etc.).
func (c *APIClient) doRequest(method, endpoint string, reqBody, respBody interface{}) error {
	var req *http.Request
	var err error

	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequest(method, BaseURL+endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, BaseURL+endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	// Always send Authorization header — server requires it for all requests (including login).
	// When token is empty, send empty Bearer to satisfy server's header check.
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	bodyBytes = parser.FixEncodingBytes(bodyBytes)

	// Accept all 2xx success codes (200 OK, 201 Created, etc.).
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if respBody != nil {
			if err := json.Unmarshal(bodyBytes, respBody); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}
		return nil
	}

	// For non-2xx status codes, try to parse JSON response body for {code, msg}.
	bodyStr := string(bodyBytes)
	fmt.Printf("[API_DOREQUEST] non-2xx status=%d, body=%q\n", resp.StatusCode, bodyStr)

	var apiErr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(bodyBytes, &apiErr); err == nil && apiErr.Msg != "" {
		// Also try to parse into respBody if it's a wrapped response
		if respBody != nil {
			var wrappedResp struct {
				Code int         `json:"code"`
				Msg  string      `json:"msg"`
				Data interface{} `json:"data"`
			}
			if err2 := json.Unmarshal(bodyBytes, &wrappedResp); err2 == nil {
				wrappedResp.Code = apiErr.Code
				wrappedResp.Msg = apiErr.Msg
				// Use jsonx to deep merge or re-marshal
				dataBytes, _ := json.Marshal(wrappedResp)
				json.Unmarshal(dataBytes, respBody)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Msg)
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}

// Login performs user login.
func (c *APIClient) Login(username, password string) (*LoginData, error) {
	req := LoginReq{
		Username: username,
		Password: password,
	}

	// Parse wrapped response: {code:0, msg:"success", data:{...}}
	var resp LoginResp
	if err := c.doRequest("POST", "/auth/login", req, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("login failed: %s", resp.Msg)
	}
	data := &resp.Data
	c.token = data.Token
	return data, nil
}

// Logout performs user logout.
func (c *APIClient) Logout() error {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := c.doRequest("POST", "/auth/logout", nil, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("logout failed: %s", resp.Msg)
	}

	c.token = ""
	return nil
}

// FetchBankList retrieves the list of available question banks.
func (c *APIClient) FetchBankList() ([]ServerBankItem, error) {
	var resp BankListResp

	if err := c.doRequest("GET", "/banks", nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("fetch bank list failed: %s", resp.Msg)
	}

	return resp.Data, nil
}

// DownloadBank downloads a complete question bank from server.
func (c *APIClient) DownloadBank(bankID int) (*ServerBankData, error) {
	req := BankDownloadReq{
		BankID: bankID,
	}
	var resp BankDownloadResp

	if err := c.doRequest("GET", fmt.Sprintf("/banks/%d", req.BankID), nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("download bank failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// DownloadServerBankForExam downloads a complete question bank from server for exam creation.
// This uses the /banks/{bank_id}/export endpoint which returns ServerBankData format.
func (c *APIClient) DownloadServerBankForExam(bankID int) (*ServerBankData, error) {
	fmt.Printf("[API_DOWNLOAD_SERVER_BANK] REQUEST: GET /banks/%d/export\n", bankID)

	type RawResp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	var raw RawResp

	endpoint := fmt.Sprintf("/banks/%d/export", bankID)
	req, _ := http.NewRequest("GET", BaseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Printf("[API_DOWNLOAD_SERVER_BANK] HTTP request error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("[API_DOWNLOAD_SERVER_BANK] HTTP status=%d, body=%s\n", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		json.Unmarshal(bodyBytes, &apiErr)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Msg)
	}

	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		fmt.Printf("[API_DOWNLOAD_SERVER_BANK] JSON unmarshal error: %v\n", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("download bank failed: %s", raw.Msg)
	}

	fmt.Printf("[API_DOWNLOAD_SERVER_BANK] raw.Data = %s\n", string(raw.Data))

	// The /export endpoint returns data that is structurally identical to ServerBankData (bank_id/display_name/questions).
	var sbd ServerBankData
	if err := json.Unmarshal(raw.Data, &sbd); err != nil {
		fmt.Printf("[API_DOWNLOAD_SERVER_BANK] ServerBankData unmarshal error: %v\n", err)
		return nil, fmt.Errorf("failed to parse bank data: %w", err)
	}

	fmt.Printf("[API_DOWNLOAD_SERVER_BANK] BankID=%d, DisplayName=%q, len(Questions)=%d\n", sbd.BankID, sbd.DisplayName, len(sbd.Questions))

	return &sbd, nil
}

// CreateExamSession creates a new exam session.
func (c *APIClient) CreateExamSession(examTemplateID int) (*ExamSessionData, error) {
	req := ExamCreateReq{
		ExamTemplateID: examTemplateID,
	}
	var resp ExamCreateResp

	if err := c.doRequest("POST", "/templates", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("create exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// SubmitExam submits exam answers and gets results.
func (c *APIClient) SubmitExam(examSessionID string, answers map[string]string) (*ExamResult, error) {
	req := ExamSubmitReq{
		ExamSessionID: examSessionID,
		Answers:       answers,
	}
	var resp ExamSubmitResp

	if err := c.doRequest("POST", "/my-exams/submit", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("submit exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// UploadWrongSet uploads the user's wrong-set to server.
func (c *APIClient) UploadWrongSet(wrongQuestions []WrongQuestionItem) error {
	req := WrongSetUploadReq{
		WrongQuestions: wrongQuestions,
	}
	var resp WrongSetUploadResp

	if err := c.doRequest("POST", "/wrongset/upload", req, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("upload wrong-set failed: %s", resp.Msg)
	}

	return nil
}

// ==================== 用户列表（管理员推送考试用） ====================

// FetchUserList retrieves the list of users for exam push.
func (c *APIClient) FetchUserList() ([]UserItem, error) {
	var resp UserListResp

	if err := c.doRequest("GET", "/users", nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("fetch user list failed: %s", resp.Msg)
	}

	return resp.Data, nil
}

// ==================== 考试模板创建（管理员） ====================

// CreateExamTemplate creates a new exam template.
func (c *APIClient) CreateExamTemplate(req CreateExamTemplateReq) (*ExamTemplateItem, error) {
	var resp CreateExamTemplateResp

	if err := c.doRequest("POST", "/templates", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("create exam template failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// CreateAndPushExam creates an exam template and pushes it to users in one call.
func (c *APIClient) CreateAndPushExam(req CreateAndPushReq) (*CreateAndPushData, error) {
	var resp CreateAndPushResp

	fmt.Printf("[API_CREATE_AND_PUSH_EXAM] ========== 创建并推送考试 ==========\n")
	fmt.Printf("[API_CREATE_AND_PUSH_EXAM] REQUEST: POST /templates/create-and-push\n")
	if reqBytes, err := json.MarshalIndent(req, "", "  "); err == nil {
		fmt.Printf("[API_CREATE_AND_PUSH_EXAM] Request Body:\n%s\n", string(reqBytes))
	} else {
		fmt.Printf("[API_CREATE_AND_PUSH_EXAM] Request Body (raw): %+v\n", req)
	}

	if err := c.doRequest("POST", "/templates/create-and-push", req, &resp); err != nil {
		fmt.Printf("[API_CREATE_AND_PUSH_EXAM] doRequest error: %v\n", err)
		return nil, err
	}

	fmt.Printf("[API_CREATE_AND_PUSH_EXAM] 服务器响应:\n")
	fmt.Printf("[API_CREATE_AND_PUSH_EXAM]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_CREATE_AND_PUSH_EXAM]   Msg    = %q\n", resp.Msg)
	fmt.Printf("[API_CREATE_AND_PUSH_EXAM]   Data   = %+v\n", resp.Data)
	fmt.Printf("[API_CREATE_AND_PUSH_EXAM] ========== 创建并推送考试结束 ==========\n")

	if resp.Code != 0 {
		return nil, fmt.Errorf("create and push exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ==================== 推送考试 ====================

// PushExam pushes an exam to specified users.
func (c *APIClient) PushExam(templateID int, targetUserIDs []int) error {
	req := PushExamReq{
		TemplateID:    templateID,
		TargetUserIDs: targetUserIDs,
	}
	var resp PushExamResp

	fmt.Printf("[API_PUSH_EXAM] ========== 推送考试 ==========\n")
	fmt.Printf("[API_PUSH_EXAM] REQUEST: POST /templates/push\n")
	if reqBytes, err := json.MarshalIndent(req, "", "  "); err == nil {
		fmt.Printf("[API_PUSH_EXAM] Request Body:\n%s\n", string(reqBytes))
	} else {
		fmt.Printf("[API_PUSH_EXAM] Request Body (raw): %+v\n", req)
	}

	if err := c.doRequest("POST", "/templates/push", req, &resp); err != nil {
		fmt.Printf("[API_PUSH_EXAM] doRequest error: %v\n", err)
		return err
	}

	fmt.Printf("[API_PUSH_EXAM] 服务器响应:\n")
	fmt.Printf("[API_PUSH_EXAM]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_PUSH_EXAM]   Msg    = %q\n", resp.Msg)
	fmt.Printf("[API_PUSH_EXAM] ========== 推送考试结束 ==========\n")

	if resp.Code != 0 {
		return fmt.Errorf("push exam failed: %s", resp.Msg)
	}

	return nil
}

// ==================== 我的考试（普通用户） ====================

// GetMyExams retrieves the current user's assigned exams.
func (c *APIClient) GetMyExams() ([]MyExamItem, error) {
	var resp MyExamResp

	fmt.Printf("[API_GET_MY_EXAMS] ========== 获取我的考试列表 ==========\n")
	endpoint := "/my-exams?limit=50&offset=0"
	fmt.Printf("[API_GET_MY_EXAMS] REQUEST: GET %s\n", BaseURL+endpoint)

	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		fmt.Printf("[API_GET_MY_EXAMS] doRequest error: %v\n", err)
		return nil, err
	}

	fmt.Printf("[API_GET_MY_EXAMS] 服务器响应:\n")
	fmt.Printf("[API_GET_MY_EXAMS]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_GET_MY_EXAMS]   Msg    = %q\n", resp.Msg)
	if dataBytes, err := json.MarshalIndent(resp.Data, "", "  "); err == nil {
		fmt.Printf("[API_GET_MY_EXAMS]   Data   = %s\n", string(dataBytes))
	} else {
		fmt.Printf("[API_GET_MY_EXAMS]   Data   = %+v\n", resp.Data)
	}
	fmt.Printf("[API_GET_MY_EXAMS] ========== 获取我的考试列表结束 ==========\n")

	if resp.Code != 0 {
		return nil, fmt.Errorf("fetch my exams failed: %s", resp.Msg)
	}

	return resp.Data, nil
}

// StartMyExam starts an exam and returns the question session.
func (c *APIClient) StartMyExam(examID int) (*ExamSessionData, error) {
	req := StartExamReq{
		ExamID: examID,
	}
	var resp StartExamResp

	fmt.Printf("[API_START_MY_EXAM] ========== 开始考试 ==========\n")
	fmt.Printf("[API_START_MY_EXAM] REQUEST: POST /my-exams/start\n")
	if reqBytes, err := json.MarshalIndent(req, "", "  "); err == nil {
		fmt.Printf("[API_START_MY_EXAM] Request Body:\n%s\n", string(reqBytes))
	} else {
		fmt.Printf("[API_START_MY_EXAM] Request Body (raw): %+v\n", req)
	}

	if err := c.doRequest("POST", "/my-exams/start", req, &resp); err != nil {
		fmt.Printf("[API_START_MY_EXAM] doRequest error: %v\n", err)
		return nil, err
	}

	fmt.Printf("[API_START_MY_EXAM] 服务器响应:\n")
	fmt.Printf("[API_START_MY_EXAM]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_START_MY_EXAM]   Msg    = %q\n", resp.Msg)
	fmt.Printf("[API_START_MY_EXAM]   Data   = %+v\n", resp.Data)
	fmt.Printf("[API_START_MY_EXAM] ========== 开始考试结束 ==========\n")

	if resp.Code != 0 {
		return nil, fmt.Errorf("start exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// SubmitMyExam submits my exam answers and gets results.
func (c *APIClient) SubmitMyExam(examSessionID string, answers map[string]string) (*ExamResult, error) {
	req := SubmitMyExamReq{
		ExamSessionID: examSessionID,
		Answers:       answers,
	}
	var resp SubmitMyExamResp

	if err := c.doRequest("POST", "/my-exams/submit", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("submit exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ==================== 刷新 Token ====================

// RefreshToken refreshes the access token using refresh token.
func (c *APIClient) RefreshToken(refreshToken string) (*RefreshData, error) {
	req := RefreshReq{
		RefreshToken: refreshToken,
	}
	var resp RefreshResp

	if err := c.doRequest("POST", "/auth/refresh", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("refresh token failed: %s", resp.Msg)
	}

	c.token = resp.Data.Token
	return &resp.Data, nil
}

// ==================== 用户管理 ====================

// FetchUsers retrieves the list of users with pagination.
func (c *APIClient) FetchUsers(limit, offset int) ([]UserItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	endpoint := fmt.Sprintf("/users?limit=%d&offset=%d", limit, offset)
	fmt.Printf("[API_USERS] REQUEST: GET %s\n", BaseURL+endpoint)
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, 0, err
	}
	fmt.Printf("[API_USERS] RESPONSE: Code=%d, Msg=%q, Total=%d, Data=%+v\n", resp.Code, resp.Msg, resp.Total, resp.Data)
	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("fetch users failed: %s", resp.Msg)
	}

	// Parse the data as []UserItem
	dataBytes, _ := json.Marshal(resp.Data)
	var users []UserItem
	json.Unmarshal(dataBytes, &users)

	return users, resp.Total, nil
}

// FetchAllUsers retrieves all users (including admins and regular users) with pagination.
func (c *APIClient) FetchAllUsers(limit, offset int) ([]UserItem, int, error) {
	return c.FetchUsers(limit, offset)
}

// FetchAdmins retrieves the list of admins with pagination.
func (c *APIClient) FetchAdmins(limit, offset int) ([]UserItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	endpoint := fmt.Sprintf("/admins?limit=%d&offset=%d", limit, offset)
	fmt.Printf("[API_ADMINS] REQUEST: GET %s\n", BaseURL+endpoint)
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, 0, err
	}
	fmt.Printf("[API_ADMINS] RESPONSE: Code=%d, Msg=%q, Total=%d, Data=%+v\n", resp.Code, resp.Msg, resp.Total, resp.Data)
	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("fetch admins failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var admins []UserItem
	json.Unmarshal(dataBytes, &admins)

	return admins, resp.Total, nil
}

// CreateUser creates a single user.
func (c *APIClient) CreateUser(req CreateUserReq) (*UserItem, error) {
	var resp UserCreateResp
	fmt.Printf("[API_USERS] REQUEST: POST /users, body={Username:%q, Password:%q, Role:%q}\n", req.Username, req.Password, req.Role)

	if err := c.doRequest("POST", "/users", req, &resp); err != nil {
		return nil, err
	}
	fmt.Printf("[API_USERS] RESPONSE: Code=%d, Msg=%q, Data={UserID:%d, Username:%q, Role:%q}\n", resp.Code, resp.Msg, resp.Data.UserID, resp.Data.Username, resp.Data.Role)

	if resp.Code != 0 {
		return nil, fmt.Errorf("create user failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// UpdateUser updates an existing user.
func (c *APIClient) UpdateUser(userID int, req UpdateUserReq) error {
	var resp UserUpdateResp
	endpoint := fmt.Sprintf("/users/%d", userID)
	fmt.Printf("[API_USERS] REQUEST: PUT %s, body={Username:%v, Password:%v, Role:%v}\n", endpoint, req.Username, req.Password, req.Role)

	if err := c.doRequest("PUT", endpoint, req, &resp); err != nil {
		return err
	}
	fmt.Printf("[API_USERS] RESPONSE: Code=%d, Msg=%q\n", resp.Code, resp.Msg)

	if resp.Code != 0 {
		return fmt.Errorf("update user failed: %s", resp.Msg)
	}

	return nil
}

// DeleteUser deletes a user.
func (c *APIClient) DeleteUser(userID int) error {
	var resp UserDeleteResp
	endpoint := fmt.Sprintf("/users/%d", userID)
	fmt.Printf("[API_USERS] REQUEST: DELETE %s\n", endpoint)

	if err := c.doRequest("DELETE", endpoint, nil, &resp); err != nil {
		return err
	}
	fmt.Printf("[API_USERS] RESPONSE: Code=%d, Msg=%q\n", resp.Code, resp.Msg)

	if resp.Code != 0 {
		return fmt.Errorf("delete user failed: %s", resp.Msg)
	}

	return nil
}

// ==================== 题库管理 ====================

// FetchBanksList retrieves the list of question banks with pagination and filters.
func (c *APIClient) FetchBanksList(bankName, category string, limit, offset int) ([]BankItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)
	if bankName != "" {
		query += "&bank_name=" + bankName
	}
	if category != "" {
		query += "&category=" + category
	}

	if err := c.doRequest("GET", "/banks"+query, nil, &resp); err != nil {
		return nil, 0, err
	}

	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("fetch banks list failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var banks []BankItem
	json.Unmarshal(dataBytes, &banks)

	return banks, resp.Total, nil
}

// CreateBank creates a new question bank.
func (c *APIClient) CreateBank(req CreateBankReq) (int, error) {
	var resp BankCreateResp

	if err := c.doRequest("POST", "/banks", req, &resp); err != nil {
		return 0, err
	}

	if resp.Code != 0 {
		return 0, fmt.Errorf("create bank failed: %s", resp.Msg)
	}

	return resp.Data.BankID, nil
}

// UpdateBank updates an existing question bank.
func (c *APIClient) UpdateBank(bankID int, req CreateBankReq) error {
	var resp BankUpdateResp
	endpoint := fmt.Sprintf("/banks/%d", bankID)

	if err := c.doRequest("PUT", endpoint, req, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("update bank failed: %s", resp.Msg)
	}

	return nil
}

// DeleteBank deletes a question bank by display_name and owner_username.
func (c *APIClient) DeleteBank(displayName, ownerUsername string) error {
	var resp BankDeleteResp
	endpoint := "/banks"
	reqBody := map[string]string{
		"display_name":   displayName,
		"owner_username": ownerUsername,
	}
	fmt.Printf("[API_DELETE_BANK] REQUEST: DELETE %s, body={display_name:%q, owner_username:%q}\n", endpoint, displayName, ownerUsername)

	if err := c.doRequest("DELETE", endpoint, reqBody, &resp); err != nil {
		fmt.Printf("[API_DELETE_BANK] RESPONSE: error=%v\n", err)
		return err
	}
	fmt.Printf("[API_DELETE_BANK] RESPONSE: Code=%d, Msg=%q\n", resp.Code, resp.Msg)

	if resp.Code != 0 {
		return fmt.Errorf("delete bank failed: %s", resp.Msg)
	}

	return nil
}

// ==================== 题目管理 ====================

// FetchQuestions retrieves the list of questions with pagination and filters.
func (c *APIClient) FetchQuestions(bankID int, content, questionType, difficulty string, limit, offset int) ([]QuestionItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	query := fmt.Sprintf("?limit=%d&offset=%d", limit, offset)
	if content != "" {
		query += "&content=" + content
	}
	if questionType != "" {
		query += "&question_type=" + questionType
	}
	if difficulty != "" {
		query += "&difficulty=" + difficulty
	}

	endpoint := fmt.Sprintf("/banks/%d/questions%s", bankID, query)

	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, 0, err
	}

	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("fetch questions failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var questions []QuestionItem
	json.Unmarshal(dataBytes, &questions)

	return questions, resp.Total, nil
}

// CreateQuestion adds a new question to a bank.
func (c *APIClient) CreateQuestion(bankID int, req CreateQuestionReq) (int, error) {
	var resp QuestionCreateResp
	endpoint := fmt.Sprintf("/banks/%d/questions", bankID)

	if err := c.doRequest("POST", endpoint, req, &resp); err != nil {
		return 0, err
	}

	if resp.Code != 0 {
		return 0, fmt.Errorf("create question failed: %s", resp.Msg)
	}

	return resp.Data.QuestionID, nil
}

// UpdateQuestion updates an existing question.
func (c *APIClient) UpdateQuestion(bankID, questionID int, req CreateQuestionReq) error {
	var resp QuestionUpdateResp
	endpoint := fmt.Sprintf("/banks/%d/questions/%d", bankID, questionID)

	if err := c.doRequest("PUT", endpoint, req, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("update question failed: %s", resp.Msg)
	}

	return nil
}

// DeleteQuestion deletes a question from a bank.
func (c *APIClient) DeleteQuestion(bankID, questionID int) error {
	var resp QuestionDeleteResp
	endpoint := fmt.Sprintf("/banks/%d/questions/%d", bankID, questionID)

	if err := c.doRequest("DELETE", endpoint, nil, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("delete question failed: %s", resp.Msg)
	}

	return nil
}

// BatchImportQuestions batch imports questions to a bank.
func (c *APIClient) BatchImportQuestions(req BatchImportQuestionsReq) (*ImportResult, error) {
	var resp BatchImportQuestionsResp

	if err := c.doRequest("POST", "/banks/import", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("batch import questions failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ==================== 考试模板管理 ====================

// FetchTemplates retrieves the list of exam templates with pagination.
func (c *APIClient) FetchTemplates(limit, offset int) ([]TemplateItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	fmt.Printf("[API_FETCH_TEMPLATES] ========== 获取考试模板列表 ==========\n")
	fmt.Printf("[API_FETCH_TEMPLATES] REQUEST: GET /templates?limit=%d&offset=%d\n", limit, offset)

	endpoint := fmt.Sprintf("/templates?limit=%d&offset=%d", limit, offset)
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		fmt.Printf("[API_FETCH_TEMPLATES] doRequest error: %v\n", err)
		return nil, 0, err
	}

	fmt.Printf("[API_FETCH_TEMPLATES] 服务器响应:\n")
	fmt.Printf("[API_FETCH_TEMPLATES]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_FETCH_TEMPLATES]   Msg    = %q\n", resp.Msg)
	fmt.Printf("[API_FETCH_TEMPLATES]   Total  = %d\n", resp.Total)
	fmt.Printf("[API_FETCH_TEMPLATES]   Limit  = %d\n", resp.Limit)
	fmt.Printf("[API_FETCH_TEMPLATES]   Offset = %d\n", resp.Offset)

	if resp.Code != 0 {
		fmt.Printf("[API_FETCH_TEMPLATES] ========== 获取考试模板列表失败 ==========\n")
		return nil, 0, fmt.Errorf("fetch templates failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var templates []TemplateItem
	json.Unmarshal(dataBytes, &templates)

	fmt.Printf("[API_FETCH_TEMPLATES] 模板数量: %d\n", len(templates))
	for i, tmpl := range templates {
		fmt.Printf("[API_FETCH_TEMPLATES]   [%d] ID=%d, 名称=%s, 状态=%s, 题目数=%d, 时长=%d分钟\n",
			i+1, tmpl.TemplateID, tmpl.ExamName, tmpl.Status, tmpl.QuestionCount, tmpl.DurationMin)
	}
	fmt.Printf("[API_FETCH_TEMPLATES] ========== 获取考试模板列表结束 ==========\n")

	return templates, resp.Total, nil
}

// UpdateTemplate updates an existing exam template.
func (c *APIClient) UpdateTemplate(templateID int, req TemplateUpdateReq) error {
	var resp TemplateUpdateResp
	endpoint := fmt.Sprintf("/templates/%d", templateID)

	fmt.Printf("[API_TEMPLATE_UPDATE] PUT %s, exam_name=%s, duration=%d, single=%d, multi=%d, judge=%d, blank=%d, essay=%d\n",
		endpoint, req.ExamName, req.DurationMin,
		ptrInt(req.SingleCount), ptrInt(req.MultiCount), ptrInt(req.JudgeCount), ptrInt(req.BlankCount), ptrInt(req.EssayCount))

	if err := c.doRequest("PUT", endpoint, req, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("update template failed: %s", resp.Msg)
	}

	return nil
}

// ptrInt returns the int value of a pointer, or 0 if nil.
func ptrInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// GetTemplateDetail retrieves the detail of an exam template including per-type question counts.
func (c *APIClient) GetTemplateDetail(templateID int) (*TemplateDetailItem, error) {
	var resp GetTemplateDetailResp
	endpoint := fmt.Sprintf("/templates/%d/detail", templateID)

	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("get template detail failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// DeleteTemplate deletes an exam template.
func (c *APIClient) DeleteTemplate(templateID int) error {
	var resp TemplateDeleteResp
	endpoint := fmt.Sprintf("/templates/%d", templateID)

	if err := c.doRequest("DELETE", endpoint, nil, &resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("delete template failed: %s", resp.Msg)
	}

	return nil
}

// CancelExam cancels an exam.
func (c *APIClient) CancelExam(req CancelExamReq) (*CancelData, error) {
	var resp CancelExamResp

	if err := c.doRequest("POST", "/templates/cancel", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("cancel exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ExportResults exports exam results.
func (c *APIClient) ExportResults(templateID, examID int, format string) ([]ExportResultItem, error) {
	var resp ExportResultsResp

	query := fmt.Sprintf("?format=%s", format)
	if templateID != 0 {
		query += "&template_id=" + fmt.Sprintf("%d", templateID)
	} else if examID != 0 {
		query += "&exam_id=" + fmt.Sprintf("%d", examID)
	}

	endpoint := "/templates/export" + query
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("export results failed: %s", resp.Msg)
	}

	return resp.Data, nil
}

// ==================== 题库批量导入（新增） ====================

// ImportBanks batch imports question banks to server.
func (c *APIClient) ImportBanks(req BankImportReq) (*ImportBankData, string, error) {
	var resp BankImportResp
	fmt.Printf("[API_IMPORT] REQUEST: POST /banks/import, req={OwnerUsername:%q, Banks count:%d}\n", req.OwnerUsername, len(req.Banks))

	if err := c.doRequest("POST", "/banks/import", req, &resp); err != nil {
		fmt.Printf("[API_IMPORT] doRequest error: %v\n", err)
		return nil, "", err
	}

	fmt.Printf("[API_IMPORT] resp.Code=%d, resp.Msg=%q, resp.Data.Imported=%d, resp.Data.Failed=%d, resp.Data.Errors=%v, resp.Data.Duplicates=%v\n", resp.Code, resp.Msg, resp.Data.Imported, resp.Data.Failed, resp.Data.Errors, resp.Data.Duplicates)

	if resp.Code != 0 {
		err := fmt.Errorf("import banks failed: %s", resp.Msg)
		fmt.Printf("[API_IMPORT] code != 0, returning error: %v\n", err)
		// Still return resp.Data so caller can access Duplicates
		return &resp.Data, resp.Msg, err
	}

	fmt.Printf("[API_IMPORT] success, returning data\n")
	return &resp.Data, resp.Msg, nil
}

// ==================== 按归属获取题库列表（新增） ====================

// FetchBanksByOwner retrieves the list of question banks by owner username.
func (c *APIClient) FetchBanksByOwner(owner, bankName string, limit, offset int) ([]BankItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	query := fmt.Sprintf("?owner=%s&limit=%d&offset=%d", owner, limit, offset)
	if bankName != "" {
		query += "&bank_name=" + bankName
	}

	fmt.Printf("[API_FETCH_BANKS] endpoint=/banks/owner%s, owner=%s\n", query, owner)

	if err := c.doRequest("GET", "/banks/owner"+query, nil, &resp); err != nil {
		fmt.Printf("[API_FETCH_BANKS] doRequest error: %v\n", err)
		return nil, 0, err
	}

	fmt.Printf("[API_FETCH_BANKS] resp.Code=%d, resp.Msg=%q, resp.Total=%d, resp.Data=%v\n", resp.Code, resp.Msg, resp.Total, resp.Data)

	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("fetch banks by owner failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var banks []BankItem
	if err := json.Unmarshal(dataBytes, &banks); err != nil {
		fmt.Printf("[API_FETCH_BANKS] json.Unmarshal error: %v, dataBytes=%s\n", err, string(dataBytes))
		return nil, 0, fmt.Errorf("failed to parse banks: %v", err)
	}

	fmt.Printf("[API_FETCH_BANKS] parsed %d banks\n", len(banks))
	return banks, resp.Total, nil
}

// ==================== 我的题库（新增） ====================

// FetchMyBanks retrieves the current user's assigned question banks.
func (c *APIClient) FetchMyBanks() ([]MyBankItem, error) {
	var resp MyBankResp

	if err := c.doRequest("GET", "/my-banks", nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("fetch my banks failed: %s", resp.Msg)
	}

	return resp.Data, nil
}

// ==================== 题库详情（新增） ====================

// DownloadBankDetail retrieves the full detail of a question bank by bank_id via /export endpoint.
// The /export endpoint returns a data field that is structurally identical to parser.BankData
// (displayName/storageKey/questions), so we parse it directly as *parser.BankData.
func (c *APIClient) DownloadBankDetail(bankID int) (*parser.BankData, error) {
	fmt.Printf("[API_DOWNLOAD_DETAIL] REQUEST: GET /banks/%d/export\n", bankID)

	type RawResp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	var raw RawResp

	endpoint := fmt.Sprintf("/banks/%d/export", bankID)
	req, _ := http.NewRequest("GET", BaseURL+endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Printf("[API_DOWNLOAD_DETAIL] HTTP request error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("[API_DOWNLOAD_DETAIL] HTTP status=%d, body=%s\n", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		json.Unmarshal(bodyBytes, &apiErr)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Msg)
	}

	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		fmt.Printf("[API_DOWNLOAD_DETAIL] JSON unmarshal error: %v\n", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("download bank detail failed: %s", raw.Msg)
	}

	fmt.Printf("[API_DOWNLOAD_DETAIL] raw.Data = %s\n", string(raw.Data))

	// The /export endpoint returns data that is directly a parser.BankData (displayName/storageKey/questions).
	var bd parser.BankData
	if err := json.Unmarshal(raw.Data, &bd); err != nil {
		fmt.Printf("[API_DOWNLOAD_DETAIL] BankData unmarshal error: %v\n", err)
		return nil, fmt.Errorf("failed to parse bank data: %w", err)
	}

	fmt.Printf("[API_DOWNLOAD_DETAIL] DisplayName=%q, StorageKey=%q, len(Questions)=%d\n", bd.DisplayName, bd.StorageKey, len(bd.Questions))

	return &bd, nil
}

// ==================== 登录日志 ====================

// LoadLoginLogs retrieves the login logs with pagination.
func (c *APIClient) LoadLoginLogs(limit, offset int) ([]LoginLogItem, int, error) {
	var resp PaginatedResp
	if limit == 0 {
		limit = 50
	}
	if offset == 0 {
		offset = 0
	}

	endpoint := fmt.Sprintf("/login-logs?limit=%d&offset=%d", limit, offset)
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, 0, err
	}

	if resp.Code != 0 {
		return nil, 0, fmt.Errorf("load login logs failed: %s", resp.Msg)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var logs []LoginLogItem
	json.Unmarshal(dataBytes, &logs)

	return logs, resp.Total, nil
}

// ==================== 暂停/恢复考试 ====================

// PauseExam pauses an ongoing exam.
func (c *APIClient) PauseExam(req PauseExamReq) (*PauseResumeData, error) {
	var resp PauseExamResp

	if err := c.doRequest("POST", "/my-exams/pause", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("pause exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ResumeExam resumes a paused exam.
func (c *APIClient) ResumeExam(req PauseExamReq) (*PauseResumeData, error) {
	var resp ResumeExamResp

	if err := c.doRequest("POST", "/my-exams/resume", req, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("resume exam failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// GetExamResult retrieves the detailed result of an exam.
func (c *APIClient) GetExamResult(sessionID string) (*ExamDetailResult, error) {
	var resp struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data ExamDetailResult `json:"data"`
	}

	endpoint := fmt.Sprintf("/my-exams/result/%s", sessionID)
	if err := c.doRequest("GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("get exam result failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}

// ==================== 题库推送 ====================

// PushBank pushes a question bank from one owner to another user.
// POST /api/v1/banks/push-by-names
func (c *APIClient) PushBank(req PushBankReq) (*PushBankData, error) {
	var resp PushBankResp
	fmt.Printf("[API_PUSH_BANK] ========== 开始推送题库 ==========\n")
	fmt.Printf("[API_PUSH_BANK] REQUEST: POST /banks/push-by-names\n")
	fmt.Printf("[API_PUSH_BANK]   display_name    = %q\n", req.DisplayName)
	fmt.Printf("[API_PUSH_BANK]   owner_username  = %q\n", req.OwnerUsername)
	fmt.Printf("[API_PUSH_BANK]   target_usernames = %v (len=%d)\n", req.TargetUsernames, len(req.TargetUsernames))
	fmt.Printf("[API_PUSH_BANK]   overwrite       = %v\n", req.Overwrite)

	if err := c.doRequest("POST", "/banks/push-by-names", req, &resp); err != nil {
		fmt.Printf("[API_PUSH_BANK] doRequest error: %v\n", err)
		return nil, err
	}

	fmt.Printf("[API_PUSH_BANK] 服务器响应:\n")
	fmt.Printf("[API_PUSH_BANK]   Code   = %d\n", resp.Code)
	fmt.Printf("[API_PUSH_BANK]   Msg    = %q\n", resp.Msg)
	fmt.Printf("[API_PUSH_BANK]   Data.Pushed   = %d\n", resp.Data.Pushed)
	fmt.Printf("[API_PUSH_BANK]   Data.Updated  = %d\n", resp.Data.Updated)
	fmt.Printf("[API_PUSH_BANK]   Data.Conflict = %v\n", resp.Data.Conflict)
	fmt.Printf("[API_PUSH_BANK] ========== 推送题库结束 ==========\n")

	if resp.Code != 0 {
		return nil, fmt.Errorf("push bank failed: %s", resp.Msg)
	}

	return &resp.Data, nil
}
