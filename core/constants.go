package core

// ==================== 全局配置常量 ====================
// 将所有硬编码的阈值、延迟时间、尺寸抽为常量，方便统一调优。

// --- 手势滑动 ---
const (
	SwipeThreshold = float32(40.0) // 横向滑动触发翻页的最小累积位移阈值
)

// --- 自动跳转延迟 ---
const (
	SingleCorrectDelay = 350 // 单选/判断题答对后自动跳下一题的延迟（毫秒）
	MultiCorrectDelay  = 500 // 多选题答对后自动跳下一题的延迟（毫秒）
)

// --- 搜索防抖 ---
const (
	SearchDebounceMs = 300 // 搜索输入框防抖延迟（毫秒）
)

// --- UI 尺寸与间距 (Dimensions & Spacing) ---
const (
	OptionSpacing     = float32(2)  // 选项之间的垂直间距
	CardCornerRadius  = float32(8)  // 标准卡片圆角半径
	LargeCardCorner   = float32(12) // 大尺寸容器（如登录页）圆角半径
	InputCornerRadius = float32(6)  // 输入框圆角半径
	SmallButtonCorner = float32(4)  // 小按钮/基础组件圆角半径

	ActionBtnHeight     = float32(35)  // 底部操作按钮高度
	FileListItemHeight  = float32(35)  // 首页文件列表单项高度
	MaxFileListHeight   = float32(300) // 首页文件列表最大生长高度
	BottomPaddingHeight = float32(15)  // 底部安全垫片高度
	PageMarginHeight    = float32(12)  // 页面边距高度
	IconFixedBoxSize    = float32(24)  // 选项右侧图标固定盒子尺寸

	DialogMinWidth        = float32(300) // 对话框最小宽度
	DialogButtonGap       = float32(40)  // 对话框按钮间的水平间距
	DialogVerticalGap     = float32(20)  // 对对话框内部垂直间距
	DialogActionBtnWidth  = float32(100) // 对话框操作按钮宽度
	DialogActionBtnHeight = float32(35)  // 对对话框操作按钮高度

	AdminExamWizardHeight     = float32(500) // 向导滚动区域最小高度
	FilterItemHeight          = float32(15)  // 筛选页面列表项高度
	ListWrapHeight            = float32(8)   // 通用列表包装高度
	AdminHeaderFontSize       = float32(15)  // 管理员界面标题字体大小 (如分组标题)
	AdminAnsHintFontSize      = float32(13)  // 答案提示文字文字大小
	AdminConfirmTitleFontSize = float32(18)  // 确认页面标题字体大小
	AdminSummaryTextSize      = float32(15)  // 确认页面摘要文字大小
	CreatePushBtnWidth        = float32(200) // 创建并推送按钮宽度
	CreatePushBtnHeight       = float32(45)  // 创建并推送按钮高度
	AdminUserListHeight       = float32(300) // 用户选择页滚动区域最小高度
	AdminSummarySpacingH20    = float32(20)  // 确认页面摘要间距高度
	AdminInfoTextSize         = float32(14)  // 信息提示文字大小 (如暂无数据时)
	AdminListSpacingH8        = float32(8)   // 列表项间距高度
	AdminListSpacingH20       = float32(20)  // 分割线/容器间距高度
	AdminListSpacingH30       = float32(30)  // 大间距高度

	LocalBankDialogWidth       = float32(300) // 本地题库选择对话框宽度
	LocalBankDialogHeight      = float32(200) // 本地题库选择对话框高度
	ExamParamsConfirmBtnWidth  = float32(110) // 考试参数确认按钮宽度
	ExamParamsConfirmBtnHeight = float32(32)  // 考试参数确认按钮高度
	TypeEntryBoxSizeW          = float32(60)  // 题型数量输入框宽度
	TypeEntryBoxSizeH          = float32(36)  // 题型数量输入框高度
	TypeCardGridWrapWidth      = float32(135) // 题型卡片网格包装宽度
	TypeCardGridWrapHeight     = float32(40)  // 题型卡片网格包装高度

	DefaultBtnWidth   = float32(220) // 默认按钮宽度
	DefaultBtnHeight  = float32(48)  // 默认按钮高度
	BackBtnWidth      = float32(90)  // 返回按钮宽度
	BtnAutoSizeWidth  = float32(0)   // 自适应按钮宽度（为0表示自适应）
	BtnAutoSizeHeight = float32(35)  // 自适应按钮高度（为0表示自适应）
	BackBtnHeight     = float32(35)  // 返回按钮高度
	NavButtonWidth    = float32(100) // 导航/操作类按钮宽度 (如上一题、下一题、提交)
	NavButtonHeight   = float32(35)  // 导航/操作类按钮高度
	ActionBtnWidth    = float32(120) // 操作类按钮宽度 (如开始考试, 查看详情)

	ModalGridItemSize   = float32(50)  // 题目导航弹窗网格项尺寸
	ModalCloseBtnWidth  = float32(120) // 弹窗关闭按钮宽度
	ModalCloseBtnHeight = float32(36)  // 弹窗关闭按钮高度

	SpacingSmall      = float32(8)  // 小间距 (如列表项内边距)
	SpacingMedium     = float32(18) // 中等间距 (如输入框上下间距)
	SpacingLarge      = float32(30) // 大间距
	SpacingExtraSmall = float32(10) // 极小间距

	LayoutSpacingH4  = float32(4)  // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH5  = float32(5)  // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH8  = float32(8)  // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH10 = float32(10) // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH15 = float32(15) // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH20 = float32(20) // 布局间隔高度 (GridWrap使用)
	LayoutSpacingH30 = float32(30) // 布局间隔高度 (GridWrap使用)

	StrokeThin   = float32(1.0) // 薄边框
	StrokeMedium = float32(1.5) // 中等粗细边框
)

// --- Admin Exam Create Constants ---
const (
	// Question Types
	QTypeSingleChoice = "单选题"
	QTypeMultiChoice  = "多选题"
	QTypeJudge        = "判断题"
	QTypeFillIn       = "填空题"
	QTypeEssay        = "问答题"

	// Difficulty Options
	DiffAll    = "全部"
	DiffEasy   = "简单"
	DiffMedium = "中等"
	DiffHard   = "困难"

	// Local Bank File Filtering
	BankFilePrefixXlsx    = "xlsxData_"
	BankFilePrefixXls     = "xlsData_"
	BankFilePrefixTxt     = "txtData_"
	BankFilePrefixTxts    = "txtsData_"
	BankFilePrefixJson    = "jsonData_"
	BankExcludeSubstring1 = "错题"
	BankExcludeSubstring2 = "收藏"
	BankExcludeSubstring3 = "答题记录"

	// UI Strings - Admin Exam Create
	AdminBackBtnText     = "← 返回"
	AdminExamCreateTitle = "📋 推送考试"
	AdminTabSource       = "题库来源"
	AdminTabExamParams   = "考试设置"
	AdminTabSelect       = "选择题目"
	AdminTabUser         = "目标用户"
	AdminTabConfirm      = "确认推送"
	AdminTabTemplates    = "考试列表"

	AdminSourcePageTitle           = "选择题库来源"
	AdminSourceHint                = "从本地题库或服务器题库中选题"
	AdminLocalBankBtnLabel         = "📁 本地题库"
	AdminServerBankBtnLabel        = "🌐 服务器题库"
	AdminSelectLocalBankTitle      = "选择本地题库"
	AdminSelectLocalBankMsg        = "请选择本地题库（可多选）："
	AdminNoLocalBankMsg            = "暂无本地题库"
	AdminConfirmText               = "确定"
	AdminCancelText                = "取消"
	AdminFilterPageTitle           = "筛选条件"
	AdminTypeLabel                 = "题型:"
	AdminDiffLabel                 = "难度:"
	AdminTemplateBtnText           = "📄 从模板快速设置"
	AdminTypeEntryPlaceholder      = "0" // 题型数量输入框占位符
	AdminTemplateDialogTitle       = "选择考试模板"
	AdminTemplateSelectMsg         = "请选择模板（点击选择后自动填充考试参数）："
	AdminApplyTemplateBtn          = "应用模板"
	AdminExamParamsNamePlaceholder = "请输入考试名称"
	AdminExamParamsNameLabel       = "考试名称:"
	AdminExamDurationPlaceholder   = "分钟（如：120）"
	AdminExamDurationLabel         = "考试时长:"
	AdminStartTimeLabel            = "开始时间:"
	AdminEndTimeLabel              = "结束时间:"
	AdminTimeEditBtnText           = "修改"
	AdminConfirmParamsBtnText      = "确认设置"
	AdminNoQuestionsSelectedMsg    = "请先完成选择题目"
	AdminNoFilteredQuestionsMsg    = "没有符合筛选条件的题目"
	AdminParamsNotSetMsg           = "请先完成考试设置"
	AdminSourceNotSetMsg           = "请先完成题库来源设置"
	AdminTemplateSelectPrompt      = "请选择模板（点击选择后自动填充考试参数）："
	AdminSuccessSetConfirmed       = "✅ 考试设置已确认成功"

	// Admin Summary Templates (Additional)
	AdminTypeSettingsTitle          = "各题型数量设置:"
	AdminNoUsersAvailableMsg        = "暂无可选用户"
	AdminCreatePushBtnText          = "✅ 创建并推送考试"
	AdminNoQuestionsAvailableMsg    = "暂无题目可选题"
	AdminConfirmPushTitle           = "确认推送"
	AdminNoUsersSelectedMsg         = "请先完成用户选择"
	AdminUserSelectionTitleTemplate = "选择目标用户（已选 %d 人）"
	AdminLoadingText                = "加载中..."
	AdminExamNameSummaryTemplate    = "考试名称: %s"
	AdminTimeRangeSummaryTemplate   = "🕒 考试时间：%s 至 %s"

	// Admin Summary Templates
	AdminSummaryNameTemplate          = "考试名称: %s"
	AdminSummaryQuestionCountTemplate = "题目数量: %d"
	AdminSummaryTimeTemplate          = "考试时间: %d 分钟"
	AdminSummaryUserCountTemplate     = "目标用户: %d 人"

	// UI Icons/Symbols
	IconSelectedCircle   = "●"
	IconUnselectedCircle = "○"
	IconUserPrefix       = "👤 "
	IconSubUserPrefix    = "    └ 👤 "
	IconAdminPrefix      = "👑 "
	AdminToggleIconDown  = "▶"
	AdminToggleIconUp    = "▼"

	// Error Messages
	AdminErrorText                    = "错误"
	AdminSuccessText                  = "成功"
	AdminInfoText                     = "提示"
	AdminReadFileErrMsg               = "读取文件失败: "
	AdminParseJSONErrMsg              = "解析数据失败"
	AdminNameRequiredErr              = "请输入考试名称"
	AdminDurationInvalidErr           = "请输入有效的考试时长"
	AdminTimeFormatErrorMsg           = "请输入有效的考试时间（分钟）"
	AdminTimeFormatErrMsg             = "请输入考试时间（分钟）"
	AdminTypeCountRequiredErr         = "请至少设置一种题型的数量"
	AdminNoUsersSelectedErr           = "请先完成用户选择"
	AdminNoUsersSelectedForConfirmMsg = "请先完成选择用户"
	AdminOtherUsersLabel              = "其他用户:"

	// Answer Display Texts
	AdminNoAnswerText   = "未提供答案"
	AdminAnswerHintText = "答案：如上"
	AdminAnswerPrefix   = "答案："
)

const (
	AdminDialogBtnFontSize      = float32(16)  // 对话框按钮字体大小
	AdminConfirmBtnFontSize     = float32(14)  // 参数确认按钮字体大小
	AdminLocalBtnFontSize       = float32(18)  // 本地题库选择对话框按钮字体大小
	AdminButtonStrokeWidth      = float32(1.0) // 标准操作按钮边框宽度
	AdminCancelBtnStrokeWidth   = float32(1.5) // 取消/次要操作按钮边框宽度
	ConfirmParamsBtnStrokeWidth = float32(1.5) // 考试参数确认按钮边框宽度
	ConfirmParamsBtnFontSize    = float32(14)  // 考试参数确认按钮字体大小
	CreatePushBtnStrokeWidth    = float32(1.5) // 创建并推送按钮边框宽度
	CreatePushBtnFontSize       = float32(18)  // 创建并推送按钮字体大小
	BtnLargeWidth               = float32(200) // 大型操作按钮宽度
	BtnLargeHeight              = float32(50)  // 大型操作按钮高度
	BtnMediumWideWidth          = float32(180) // 中等宽按钮宽度
	BtnMediumWideHeight         = float32(35)  // 中等宽按钮高度
	BtnSmallWidth               = float32(45)  // 小型辅助按钮宽度
	BtnSmallHeight              = float32(24)  // 小型辅助按钮高度
)

// --- 字体大小 (Typography) ---
const (
	FontSizeTitle     = float32(26) // 大标题 (如登录页标题)
	FontSizePageTitle = float32(20) // 页面主标题
	FontSizeHeading   = float32(22) // 页面头部/主要标题
	FontSizeBody      = float32(18) // 正文文字 / 列表项文字
	FontSizeSmallest  = float32(13) // 极小文字
	FontSizeSmall     = float32(15) // 小字 (如输入框标签)
	FontSizeMedium    = float32(18) // 中等文字
	FontSizeSubtitle  = float32(14) // 副标题/辅助信息
	FontSizeButton    = float32(18) // 按钮文字大小
	FontSizeDialogMsg = float32(16) // 对话框消息文字大小
	CardNameTextSize  = float32(16) // 卡片名称文字大小
)

// --- Admin Exam Create UI Templates ---
const (
	AdminQuestionSelectTitleTemplate = "选择题目（%s）"
	AdminTypeDistributionLabel       = "📊 题型分配：%s"
	AdminExamDurationSummaryTemplate = "⏱️ 考试时长：%d 分钟"
)

// --- 亮色主题颜色常量 ---
const PageBgColor = "#f0f2f5"     // 页面整体背景
const SectionBgColor = "#f7f7f7"  // 分区/底部栏背景
const CardBgColor = "#ffffff"     // 卡片、列表项背景
const InputBgColor = "#ffffff"    // 输入框背景
const OptionDefaultBg = "#f9f9f9" // 选项默认背景

const BorderLightColor = "#e8e8e8"  // 轻边框
const BorderMediumColor = "#dddddd" // 中等边框
const SeparatorColor = "#e8e8e8"    // 分割线

const TextPrimaryColor = "#111111"   // 主要文字（标题）
const TextBodyColor = "#222222"      // 正文文字
const TextSecondaryColor = "#333333" // 次要文字
const TextHintColor = "#555555"      // 提示文字
const TextMutedColor = "#666666"     // 弱化文字
const TextDisabledColor = "#bbbbbb"  // 禁用文字

const BtnPrimaryBg = "#418BEC"      // 主按钮背景（蓝色）
const BtnSecondaryBg = "#5D5D5D"    // 次要按钮背景（灰色）
const BtnDisabledBg = "#979595"     // 禁用按钮背景
const BtnDisabledStroke = "#eeeeee" // 禁用按钮边框

const ColorCorrectBg = "#E8F5E8"      // 正确背景（绿）
const ColorCorrectBorder = "#06a050"  // 正确边框（深绿）
const ColorWrongBg = "#FFEBEE"        // 错误背景（红）
const ColorWrongBorder = "#ff4d4f"    // 错误边框（红）
const ColorWarningBg = "#FF9800"      // 警告/操作颜色（橙）
const ColorWarningBorder = "#FF9800"  // 警告/操作颜色（橙）
const ColorErrorBg = "#FF6B6B"        // 错误背景（红）
const ColorErrorBorder = "#FF6B6B"    // 错误边框（红）
const ColorSelectedBg = "#e6f7ff"     // 选中背景（浅蓝）
const ColorSelectedBorder = "#1890ff" // 选中边框（深蓝）

const StarColor = "#999999"         // 收藏星标颜色
const AnswerRevealBg = "#e6f7ff"    // 答案展开背景
const SearchHighlightBg = "#5aeb55" // 搜索结果高亮绿

const ColorBg = "#2B2D30" // 深色模式背景
const ColorFg = "#0D437D" // 深色模式前景

const BtnSuccessBg = "#4CAF50"     // 成功/应用按钮背景（绿）
const BtnSuccessStroke = "#4CAF50" // 成功/应用按钮边框

const ColorPushBtnBg = "#FF8F00"   // 推送操作颜色（橙）
const ColorDeleteBtnBg = "#BB4D4C" // 删除/覆盖操作颜色（红）

// ==================== 题库管理相关常量 ====================

// --- 按钮文本 ---
const BankManageBackBtnText = "← 返回"
const BankManageTitleText = "📚 题库管理"
const BankManageUploadBtnText = "📤 上传"
const BankManageViewBtnText = "📋 查看"
const BankManageConfirmUploadBtnText = "确认上传"
const BankManageCancelBtnText = "取消"
const BankManageConfirmBtnText = "确认"
const BankManageDownloadBtnText = "下载"
const BankManagePushBtnText = "推送"
const BankManageDeleteBtnText = "删除"
const BankManageOverwriteBtnText = "覆盖"
const BankManageNewBtnText = "新增"

// --- 对话框标题 ---
const BankManageUploadDialogTitle = "上传题库"
const BankManageSelectAdminDialogTitle = "选择管理员"
const BankManagePushBankDialogTitle = "推送题库"
const BankManageFileExistsDialogTitle = "文件已存在"

// --- 提示信息 ---
const BankManageNoLocalBankFilesMsg = "暂无本地题库文件"
const BankManagePleaseSelectBanksMsg = "请至少选择一个题库"
const BankManageDuplicateBanksFoundMsg = "发现重复题库"
const BankManageDuplicateBanksConfirmMsg = "以下题库已存在，是否覆盖？\n\n%s"
const BankManageUploadCompleteMsg = "上传完成：成功 %d 个，失败 %d 个"
const BankManageReadLocalFilesErrMsg = "读取本地文件失败"
const BankManageReadFileErrMsg = "读取文件失败: %v"
const BankManageParseFileErrMsg = "解析文件失败: %v"
const BankManagePleaseSelectAdminMsg = "请选择一个管理员账号"
const BankManageNoBanksMsg = "暂无题库"
const BankManageNoMyBanksMsg = "暂无我的题库"
const BankManageConfirmDeleteMsg = "确认删除"
const BankManageConfirmDeleteConfirmMsg = "确定要删除题库【%s】吗？"
const BankManageDeleteSuccessMsg = "题库已删除"
const BankManagePleaseSelectTargetUsersMsg = "请至少选择一个目标用户"
const BankManagePushSuccessMsg = "推送成功！推送: %d 个，更新: %d 个"
const BankManagePushConflictMsg = "冲突: %v"
const BankManageOverwriteTargetBanksMsg = "如果目标用户已有同名题库，是否覆盖"
const BankManageLocalFileExistsMsg = "本地已存在名为【%s】的题库。\n\n请选择接下来的操作："
const BankManageConfirmDownloadMsg = "确认下载"
const BankManageConfirmDownloadConfirmMsg = "确定要下载题库【%s】吗？"
const BankManageDownloadSuccessMsg = "题库下载成功"

// --- 选择提示 ---
const BankManageSelectUploadBanksMsg = "选择要上传的题库："
const BankManageSelectAdminMsg = "选择要查看的管理员："

// --- 状态文本 ---
const BankManageLoadingText = "加载中..."
const BankManageErrorTextPrefix = "错误: "

// --- 消息类型 ---
const BankManageSuccessMsgType = "成功"
const BankManageErrorMsgType = "错误"
const BankManageInfoMsgType = "提示"

// ==================== 我的考试相关常量 ====================

// --- 页面标题 ---
const MyExamsTitleText = "📝 我的考试"

// --- 提示信息 ---
const MyExamsNoDataText = "暂无考试"

// --- 按钮文本 ---
const MyExamsStartExamBtnText = "开始考试"
const MyExamsContinueExamBtnText = "继续考试"
const MyExamsViewDetailsBtnText = "查看详情"
const MyExamsNotOpenBtnText = "未开放"
const MyExamsConfirmSubmitTitle = "确认交卷"
const MyExamsConfirmSubmitMsg = "确定要交卷吗？交卷后将显示得分。"
const MyExamsResultDialogTitle = "考试结果"
const MyExamsShowAnswerBtnText = "点击显示答案"
const MyExamsHideAnswerBtnText = "点击隐藏答案"
const MyExamsPrevBtnText = "上一题"
const MyExamsNextBtnText = "下一题"
const MyExamsDetailDialogTitle = "考试详情"
const MyExamsPauseExamBtnText = "暂停考试"
const MyExamsPauseSuccessMsg = "考试已暂停"

// ==================== 考试模板管理相关常量 ====================

// --- 页面标题 ---
const TemplateManageTitleText = "📋 考试模板管理"

// --- 提示信息 ---
const TemplateManageNoDataText = "暂无考试模板"
const TemplateManagePleaseLoginMsg = "请先登录"

// --- 消息类型 ---
const TemplateManageSuccessMsgType = "成功"
const TemplateManageErrorMsgType = "错误"
const TemplateManageInfoMsgType = "提示"

// --- 状态文本 ---
const TemplateStatusDraftText = "草稿"
const TemplateStatusActiveText = "进行中"
const TemplateStatusExpiredText = "已过期"

// --- 信息标签 ---
const TemplateInfoQuestionCountLabel = "题目数: %d"
const TemplateInfoDurationLabel = "时长: %d分钟"
const TemplateInfoSourceLabel = "来源: %s"
const TemplateInfoTypePrefix = "题型:"
const TemplateInfoStartTimeLabel = "开始: %s"
const TemplateInfoEndTimeLabel = "结束: %s"

// --- 题型标签 ---
const TemplateTypeSingleChoice = "单选题%d"
const TemplateTypeMultiChoice = "多选题%d"
const TemplateTypeJudge = "判断题%d"
const TemplateTypeFillIn = "填空题%d"
const TemplateTypeEssay = "问答题%d"

// --- 按钮文本 ---
const TemplateEditBtnText = "编辑"
const TemplateCancelExamBtnText = "取消考试"
const TemplateExportResultsBtnText = "导出结果"
const TemplateDeleteBtnText = "删除"
const TemplateUpdateBtnText = "更新"
const TemplateCancelBtnText = "取消"

// --- 对话框标题 ---
const TemplateEditDialogTitle = "编辑考试模板"
const TemplateExportResultsDialogTitle = "导出考试结果"
const TemplateDeleteConfirmTitle = "确认删除"
const TemplateCancelExamConfirmTitle = "确认取消"

// --- 对话框提示信息 ---
const TemplateEditTypeCountsTitle = "各题型数量设置"
const TemplateEditExamNameLabel = "考试名称："
const TemplateEditExamDurationLabel = "考试时长（分钟）："
const TemplateDurationInvalidMsg = "时长必须为正整数"
const TemplateUpdateSuccessMsg = "模板更新成功"
const TemplateDeleteConfirmMsg = "确定要删除考试模板 %s 吗？"
const TemplateDeleteSuccessMsg = "模板删除成功"
const TemplateCancelExamConfirmMsg = "确定要取消考试模板 %s 吗？取消后将影响 %d 个用户。"
const TemplateCancelExamSuccessMsg = "考试取消成功，影响 %d 个用户"
const TemplateExportResultsTemplateLabel = "考试模板: %s"
const TemplateExportFormatLabel = "导出格式："

// ==================== 用户管理相关常量 ====================

// --- 页面标题 ---
const UserManageTitleText = "👥 用户管理"
const AdminManageTitleText = "👑 管理员管理"

// --- 按钮文本 ---
const UserManageCreateUserBtnText = "➕ 创建用户"
const UserManageViewAdminsBtnText = "👑 查看管理员"
const UserManageEditBtnText = "编辑"
const UserManageDeleteBtnText = "删除"

// --- 对话框标题 ---
const CreateUserDialogTitle = "创建用户"
const UpdateUserDialogTitle = "编辑用户"

// --- 输入框占位符 ---
const UsernameEntryPlaceholder = "用户名"
const PasswordEntryCreatePlaceholder = "密码（至少6位）"
const PasswordEntryUpdatePlaceholder = "新密码（留空则不修改）"

// --- 表单标签 ---
const UsernameFormItemLabel = "用户名："
const PasswordFormItemLabel = "密码："
const RoleFormItemLabel = "角色："

// --- 对话框按钮文本 ---
const CreateUserSubmitText = "创建"
const UpdateUserSubmitText = "更新"

// --- 提示信息 ---
const UserManageNoAdminPermissionMsg = "只有管理员可以查看用户列表"
const AdminManageNoAdminPermissionMsg = "只有管理员可以查看管理员列表"
const UserManageNoUsersMsg = "暂无用户"
const AdminManageNoAdminsMsg = "暂无管理员"
const UserManageDeleteConfirmMsg = "确定要删除用户 %s 吗？"
const UserManageEmptyFieldsMsg = "用户名和密码不能为空"
const UserManagePasswordLengthMsg = "密码长度至少 6 位"
const UserManageCreateSuccessMsg = "用户 %s 创建成功"
const UserManageUpdateSuccessMsg = "用户更新成功"

// --- 角色颜色 ---
const RoleAdminColor = "#9C27B0"      // Admin角色颜色（紫色）
const RoleSuperAdminColor = "#FF6B6B" // Superadmin角色颜色（红色/粉色）

// --- 卡片操作按钮尺寸 ---
const CardActionBtnWidth = float32(80)    // 卡片操作按钮（编辑/删除）宽度
const CardActionBtnFontSize = float32(14) // 卡片操作按钮字体大小

// --- 用户管理对话框尺寸 ---
const UserManageDialogWidth = float32(380)  // 用户管理对话框宽度
const UserManageDialogHeight = float32(240) // 用户管理对话框高度

// --- 用户/管理员列表滚动区域最小高度 ---
const UserManageScrollMinHeight = float32(400) // 用户/管理员列表滚动区域最小高度

// --- 用户管理确认对话框标题 ---
const UserManageDeleteConfirmTitle = "确认删除" // 用户删除确认对话框标题

// ==================== 考试模板管理相关 UI 常量 ====================

// --- 卡片和按钮尺寸 ---
const TemplateCardCornerRadius = float32(8)        // 模板卡片圆角半径
const TemplateEditBtnWidth = float32(80)           // 编辑按钮宽度
const TemplateCancelExamBtnWidth = float32(100)    // 取消考试按钮宽度
const TemplateExportResultsBtnWidth = float32(100) // 导出结果按钮宽度
const TemplateDeleteBtnWidth = float32(80)         // 删除按钮宽度
const TemplateBtnHeight = float32(35)              // 模板操作按钮高度

// --- 字体大小 ---
const TemplateStatusBadgeFontSize = float32(13) // 状态徽章字体大小
const TemplateInfoTextFontSize = float32(13)    // 信息文本字体大小
const TemplateTimeTextFontSize = float32(12)    // 时间信息字体大小
const TemplateTypeLabelFontSize = float32(14)   // 类型标签字体大小
const TemplateEditTitleFontSize = float32(15)   // 编辑对话框标题字体大小

// --- 占位符和提示信息 ---
const TemplateCountEntryPlaceholder = "0" // 数量输入框占位符

// --- 颜色常量 ---
const TemplateStatusDraftColor = TextHintColor      // 草稿状态颜色
const TemplateStatusActiveColor = BtnPrimaryBg      // 进行中状态颜色
const TemplateStatusExpiredColor = ColorWrongBorder // 已过期状态颜色

// ==================== 我的考试相关 UI 尺寸与常量 ====================

// --- 页面滚动区域 ---
const MyExamsScrollMinHeight = float32(400) // 我的考试列表滚动区域最小高度

// --- 按钮文本 ---
const MyExamsSubmitBtnText = "← 交卷" // 交卷按钮文本

// --- 结果格式 ---
const MyExamsScoreResultFormat = "得分: %.1f / 100\n正确: %d / %d" // 考试结果得分格式

// --- 答案展开卡片样式 ---
const AnswerRevealStrokeWidth = float32(2)  // 答案展开卡片边框宽度
const AnswerRevealCornerRadius = float32(6) // 答案展开卡片圆角半径

// --- 字体大小 ---
const RightIconTextFontSize = float32(16)  // 选项右侧图标文字大小
const ExamTitleLabelFontSize = float32(16) // 考试题目页面标题字体大小

// ==================== 题库管理卡片按钮尺寸 ====================
const BankManageCardBtnWidth = float32(80) // 题库管理卡片操作按钮（下载/推送/删除）宽度

// ==================== 题库管理文本前缀与图标 ====================
const BankManageCheckboxPrefixBank = "📚 "          // 题库复选框前缀
const BankManageCheckboxPrefixUser = "👤 "          // 用户单选/复选框前缀
const BankManageSubCheckboxPrefixUser = "    └ 👤 " // 子用户复选框前缀
const BankManageToggleIconDown = "▶"               // 展开图标
const BankManageToggleIconUp = "▼"                 // 折叠图标
const BankManageToggleIconDisabled = "  "          // 禁用按钮文本

// ==================== 题库管理状态与提示信息 ====================
const BankManageLoadingTextMsg = "加载中..."                    // 加载文本
const BankManageErrorTextPrefixMsg = "错误: "                  // 错误文本前缀
const BankManageNoMyBanksMsgText = "暂无我的题库"                  // 无我的题库提示
const BankManageTitleBankOfUserFormat = "📚 %s 的题库"           // 题库标题格式
const BankManageInfoOwnerStatusFormat = "归属: %s | 状态: %s"    // 信息文本格式（归属/状态）
const BankManageInfoCreatorTimeFormat = "创建者: %s | 创建时间: %s" // 信息文本格式（创建者/时间）
const BankManageNoAdminsToPushMsg = "没有其他管理员可推送"             // 无管理员可推送提示

// ==================== 练习页面相关常量 ====================

// --- 按钮文本 ---
const PracticePrevBtnText = "上一题"           // 上一题按钮文本
const PracticeNextBtnText = "下一题"           // 下一题按钮文本
const PracticeSubmitBtnText = "提交"          // 提交按钮文本
const PracticeReturnTypeBtnText = "返回题型"    // 返回题型按钮文本
const PracticeDeleteCurrentBtnText = "删除本题" // 删除本题按钮文本
const PracticeDeleteRecordBtnText = "删除记录"  // 删除记录按钮文本
const PracticeShowAnswerBtnText = "点击显示答案"  // 显示答案按钮文本
const PracticeHideAnswerBtnText = "点击隐藏答案"  // 隐藏答案按钮文本
const PracticeDifficultyLabel = "难度: 普通"    // 难度标签文本

// --- 对话框标题和提示信息 ---
const PracticeConfirmDialogTitle = "提示"                          // 确认对话框标题
const PracticeDeleteCurrentConfirmMsg = "确定要将本题从当前练习集中移除吗？"      // 删除本题确认信息
const PracticeDeleteRecordConfirmMsg = "确定要清空本次练习的所有答题记录和对错统计吗？" // 删除记录确认信息

// --- 题目导航弹窗 ---
const PracticeQuestionModalTitle = "题目导航" // 题目导航弹窗标题
const PracticeModalCloseBtnText = "关闭"    // 弹窗关闭按钮文本

// --- 尺寸常量 ---
const StarBtnSize = float32(24)      // 收藏星标按钮尺寸
const StatsRowGapWidth = float32(35) // 底部统计信息 gap 宽度
const StatsRowHeight = float32(35)   // 底部统计行高度

// ==================== 搜索页面相关常量 ====================

// --- 按钮文本 ---
const SearchBtnText = "搜索" // 搜索按钮文本

// --- 页面标题和提示信息 ---
const SearchPageTitleText = "搜索本题库"            // 搜索页面标题
const SearchEmptyResultMsg = "未找到相关题目"         // 搜索无结果提示
const SearchResultCountMsgFormat = "找到 %d 个结果" // 搜索结果数量提示格式
const QuestionIDHeaderFormat = "第%s题"          // 题目ID头部格式

// --- 输入框占位符和默认文本 ---
const SearchInputPlaceholderText = "请输入题目关键词" // 搜索输入框占位符
const DefaultBankNameText = "未命名题库"           // 默认题库名称
const BankTitlePrefix = "📖 "                  // 题库标题前缀

// --- 题型过滤选项 ---
const FilterTypeAllText = "全部题型"         // 全部题型选项
const FilterTypeSingleChoiceText = "单选题" // 单选题选项
const FilterTypeMultiChoiceText = "多选题"  // 多选题选项
const FilterTypeJudgeText = "判断题"        // 判断题选项
const FilterTypeFillInText = "填空题"       // 填空题选项
const FilterTypeEssayText = "问答题"        // 问答题选项

// --- 选中状态后缀 ---
const FilterSelectedSuffixText = " ✓" // 选中题型后缀符号
