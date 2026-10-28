package managementRelated

import (
	"fmt"
	"time"
	"fyne.io/fyne/v2"
	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/customElements"
	"github.com/gzjjjfree/practice/network"
	"github.com/gzjjjfree/practice/parser"
)


// downloadBankForOwner 从所有者列表中下载题库。
//
// 功能说明：
// 弹出确认对话框，用户确认后调用 executeDownloadAndSave 函数执行下载和保存操作。
//
// 参数说明：
//   - w: fyne.Window 类型，表示当前的窗口对象。
//   - state: *core.AppState 类型，表示全局应用状态。
//   - bank: *network.BankItem 类型，题库项。
//   - onBack: func() 类型，操作完成后的回调函数。
//
// 核心业务逻辑：
//
//	提取题库的显示名称（若为空则使用 BankName），调用 customElements.ShowCustomConfirm 弹出确认对话框，用户确认后调用 executeDownloadAndSave 执行下载与保存流程。
func downloadBankForOwner(w fyne.Window, state *core.AppState, bank *network.BankItem, onBack func()) {
	displayName := bank.DisplayName
	if displayName == "" {
		displayName = bank.BankName
	}

	customElements.ShowCustomConfirm(
		core.BankManageConfirmDownloadMsg,
		"确定",
		"取消",
		customElements.NewCenterRichText(fmt.Sprintf(core.BankManageConfirmDownloadConfirmMsg, displayName)),
		func(confirm bool) bool {
			if confirm {
				// ✅ 同上，使用 goroutine 和延迟解耦弹窗生命周期
				go func() {
					time.Sleep(200 * time.Millisecond)
					executeDownloadAndSave(w, state, bank.BankID, displayName, onBack)
				}()
			}
			return true
		},
		w,
	)
}

// convertToImportQuestions 将本地题库题目格式转换为导入服务器的题目格式。
//
// 功能说明：
// 遍历本地题库题目列表，将其转换为网络导入所需的题目项格式。
//
// 参数说明：
//   - questions: []parser.QuestionItem 类型，本地题库题目列表。
//
// 返回值：
//   - []network.ImportQuestionItem：转换后的网络导入题目项列表。
//
// 核心业务逻辑：
//
//	遍历输入的 questions 切片，提取每道题的 ID、Type、Content、Answer、Options、Difficulty 字段，构造为 network.ImportQuestionItem 并返回。
func convertToImportQuestions(questions []parser.QuestionItem) []network.ImportQuestionItem {
	var result []network.ImportQuestionItem
	for _, q := range questions {
		result = append(result, network.ImportQuestionItem{
			ID:         q.ID,
			Type:       q.Type,
			Content:    q.Content,
			Answer:     q.Answer,
			Options:    q.Options,
			Difficulty: q.Difficulty,
		})
	}
	return result
}
