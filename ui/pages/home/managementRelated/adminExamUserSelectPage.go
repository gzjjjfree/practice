package managementRelated

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
)


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
	if !checkIfQuestionsSelectedFull(ws) {
		noData := canvas.NewText(core.AdminNoQuestionsSelectedMsg, core.HexColor(core.TextMutedColor))
		noData.Alignment = fyne.TextAlignCenter
		noData.TextSize = 30
		return container.NewPadded(container.NewCenter(
			noData,
		))
	}

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
		if ws.wizard != nil && len(ws.wizard.Items) > 3 {
			userPage := buildUserSelectPage(ws, state)
			ws.wizard.Items[3] = container.NewTabItem(ws.wizard.Items[3].Text, userPage)
			ws.wizard.Refresh()
		}
	}

	// 保存刷新函数到状态中
	ws.userPageRefreshFunc = refreshUserPage

	ws.userVBox.Objects = nil

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

			startTime, err1 := time.Parse("2006-01-02 15:04", ws.startTimeLbl.Text)
			endTime, err2 := time.Parse("2006-01-02 15:04", ws.endTimeLbl.Text)
			if err1 != nil || err2 != nil || endTime.Sub(startTime) < 30*time.Minute {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminEndTimeInvalidErr, w)
				return
			}

			if !checkIfQuestionsSelectedFull(ws) {
				customElements.ShowCustomInformation(core.AdminErrorText, core.AdminParamsNotSetMsg, w)
				return
			}

			if len(ws.selectedUsers) == 0 {
				customElements.ShowCustomInformation(core.AdminInfoText, core.AdminUsersNotSetdMsg, nil)
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
		noUsersData := canvas.NewText(core.AdminUsersNotSetdMsg, core.HexColor(core.TextMutedColor))
		noUsersData.Alignment = fyne.TextAlignCenter
		noUsersData.TextSize = 30
		return container.NewPadded(container.NewCenter(
			noUsersData,
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
