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
	//StatusBarHeight    = 30  // 状态栏高度（像素）
)

// --- 搜索防抖 ---
const (
	SearchDebounceMs = 300 // 搜索输入框防抖延迟（毫秒）
)

// --- UI 尺寸 ---
const (
	OptionSpacing       = float32(2)   // 选项之间的垂直间距
	CardCornerRadius    = float32(8)   // 答题卡片圆角半径
	ActionBtnHeight     = float32(35)  // 底部操作按钮高度
	FileListItemHeight  = float32(48)  // 首页文件列表单项高度
	MaxFileListHeight   = float32(300) // 首页文件列表最大生长高度
	BottomPaddingHeight = float32(15)  // 底部安全垫片高度
	PageMarginHeight    = float32(12)  // 页面边距高度
	IconFixedBoxSize    = float32(24)  // 选项右侧图标固定盒子尺寸
)

// ==================== 亮色主题颜色常量 ====================
// --- 页面背景 ---
const PageBgColor = "#f0f2f5"    // 页面整体背景
const SectionBgColor = "#f7f7f7" // 分区/底部栏背景

// --- 卡片/容器背景 ---
const CardBgColor = "#ffffff"     // 卡片、列表项背景
const InputBgColor = "#ffffff"    // 输入框背景
const OptionDefaultBg = "#f9f9f9" // 选项默认背景

// --- 边框/分割线 ---
const BorderLightColor = "#e8e8e8"  // 轻边框
const BorderMediumColor = "#dddddd" // 中等边框
const SeparatorColor = "#e8e8e8"    // 分割线

// --- 文字颜色 ---
const TextPrimaryColor = "#111111"   // 主要文字（标题）
const TextBodyColor = "#222222"      // 正文文字
const TextSecondaryColor = "#333333" // 次要文字
const TextHintColor = "#555555"      // 提示文字
const TextMutedColor = "#666666"     // 弱化文字
const TextDisabledColor = "#bbbbbb"  // 禁用文字

// --- 按钮颜色 ---
const BtnPrimaryBg = "#418BEC"      // 主按钮背景（蓝色）
const BtnSecondaryBg = "#5D5D5D"    // 次要按钮背景（灰色）
const BtnDisabledBg = "#f8f8f8"     // 禁用按钮背景
const BtnDisabledStroke = "#eeeeee" // 禁用按钮边框

// --- 状态颜色 ---
const ColorCorrectBg = "#E8F5E8"      // 正确背景（绿）
const ColorCorrectBorder = "#06a050"  // 正确边框（深绿）
const ColorWrongBg = "#FFEBEE"        // 错误背景（红）
const ColorWrongBorder = "#ff4d4f"    // 错误边框（红）
const ColorSelectedBg = "#e6f7ff"     // 选中背景（浅蓝）
const ColorSelectedBorder = "#1890ff" // 选中边框（深蓝）

// --- 特殊颜色 ---
const StarColor = "#999999"         // 收藏星标颜色
const AnswerRevealBg = "#e6f7ff"    // 答案展开背景
const SearchHighlightBg = "#5aeb55" // 搜索结果高亮绿

// --- 背景颜色 ---
const ColorBg = "#2B2D30" // 深色模式背景
