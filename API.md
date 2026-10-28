# Fyne 智能刷题与题库管理客户端 - API 文档

本文档详细描述了客户端与后端服务交互的 RESTful API 接口规范。

- **基础 URL (Base URL)**: `https://144.34.181.64:8443/api/v1`
- **认证方式**: Bearer Token (`Authorization: Bearer <token>`)
- **通用响应结构**:
  ```json
  {
    "code": 0,          // 状态码：0 表示成功，非 0 表示错误
    "msg": "success",   // 提示或错误信息
    "data": {}          // 业务数据负载
  }
  ```

---

## 1. 认证接口 (Authentication)

### 1.1 用户登录
- **URL**: `POST /auth/login`
- **说明**: 验证用户名与密码，获取 JWT 令牌及用户角色。
- **请求体 (LoginReq)**:
  ```json
  {
    "username": "admin",
    "password": "secret_password"
  }
  ```
- **响应数据 (LoginData)**:
  ```json
  {
    "token": "jwt_token_string",
    "refresh": "refresh_token_string",
    "user_id": 1,
    "username": "admin",
    "role": "superadmin", // 可选: superadmin, admin, user
    "ip_address": "127.0.0.1"
  }
  ```

---

## 2. 用户与权限管理 (Users & Admins)

### 2.1 获取用户列表
- **URL**: `GET /users?limit=50&offset=0`
- **权限**: 管理员 (Admin / SuperAdmin)
- **响应数据 (PaginatedResp)**:
  ```json
  {
    "total": 10,
    "data": [
      {
        "user_id": 1,
        "username": "student1",
        "role": "user",
        "created_at": "2023-01-01T00:00:00Z"
      }
    ]
  }
  ```

### 2.2 创建用户
- **URL**: `POST /users`
- **请求体**:
  ```json
  {
    "username": "new_user",
    "password": "password123",
    "role": "user"
  }
  ```

### 2.3 更新用户
- **URL**: `PUT /users/{id}`
- **请求体**:
  ```json
  {
    "username": "updated_name",
    "password": "new_password",
    "role": "admin"
  }
  ```

### 2.4 删除用户
- **URL**: `DELETE /users/{id}`

### 2.5 获取管理员列表
- **URL**: `GET /admins?limit=50&offset=0`

### 2.6 获取登录日志
- **URL**: `GET /login-logs?limit=50&offset=0`

---

## 3. 考试模板与发布 (Templates & Exams)

### 3.1 获取考试模板列表
- **URL**: `GET /templates?limit=50&offset=0`

### 3.2 创建考试模板
- **URL**: `POST /templates`
- **请求体 (CreateExamTemplateReq)**:
  ```json
  {
    "local_bank_key": "local_key_uuid",
    "exam_name": "Go语言期末测验",
    "duration_min": 60,
    "start_time": "2023-10-01T08:00:00Z",
    "end_time": "2023-10-01T10:00:00Z",
    "questions": []
  }
  ```

### 3.3 推送考试
- **URL**: `POST /templates/push`
- **请求体 (PushExamReq)**:
  ```json
  {
    "template_id": 1,
    "target_user_ids": [2, 3, 4]
  }
  ```

### 3.4 创建并推送考试
- **URL**: `POST /templates/create-and-push`

### 3.5 导出考试结果
- **URL**: `GET /templates/export?format=excel&template_id=1`

---

## 4. 考生端与考试会话 (My Exams & Practice)

### 4.1 获取我的考试列表
- **URL**: `GET /my-exams?limit=50&offset=0`

### 4.2 开始考试
- **URL**: `POST /my-exams/start`
- **请求体 (StartExamReq)**:
  ```json
  {
    "exam_id": 1
  }
  ```
- **响应数据 (ExamSessionData)**: 返回会话 ID 及试题列表。

### 4.3 提交考试
- **URL**: `POST /my-exams/submit`
- **请求体 (ExamSubmitReq)**:
  ```json
  {
    "exam_session_id": "uuid-session-id",
    "answers": {
      "q1_id": "A",
      "q2_id": "True"
    }
  }
  ```

### 4.4 暂停与恢复考试
- **URL**: `POST /my-exams/pause` / `POST /my-exams/resume`

### 4.5 获取考试结果详情
- **URL**: `GET /my-exams/result/{session_id}`
