# Fyne 智能刷题与题库管理客户端

一个基于 Go 语言和 **Fyne (v2.6+)** 跨平台 GUI 框架开发的本地题库练习与管理系统。专门针对个人开发者和高频刷题需求设计，具备移动端手势兼容性与本地沙盒持久化能力。

```go
// 本项目基于 Fyne 2.6+ 的最新特性开发（如原生 widget.Label 的 SizeName 和 Selectable 属性）
import "fyne.io/fyne/v2"
```

---

## 🌟 核心特性

- **📱 跨平台智能滑动翻页**：自研 `TouchInterceptor` 探测器，完美解决手机端（Android/iOS）与桌面端（WSL/Windows）的滚动冲突。
  - _向左滑动_ = 上一题
  - _向右滑动_ = 下一题
  - _上下滑动_ = 丝滑浏览长题干与多选项
- **⚡ “金蝉脱壳”局部刷新引擎**：全面重构 UI 渲染机制，彻底消除传统 Fyne 应用切题、重绘时常见的屏幕闪烁与画面抖动问题。
- **🛠️ 智能文件覆盖与自增管理**：读取同名题库时支持高级安全交互：
  - _取消_：放弃导入。
  - _覆盖_：物理销毁旧主文件及历史痕迹，按最新时间戳落盘。
  - _新增_：自动分配如 `题库名(1).xlsx` 的自增别名。
- **🧹 影子文件级联删除**：执行“删除题库”时，一键深度清理本地沙盒内所有关联的 `错题集`、`收藏集` 及 `答题进度记录`，绝不留存垃圾文件。
- **🔒 智能答题锁死拦截**：单选/判断题点击即刻判定，答错自动标绿正确答案并永久锁定选项；多选题提交后全盘锁死，防止重复点击污染答题记录。
- **🎨 现代响应式 UI**：完美模拟微信小程序答题卡视觉，选项之间采用微米级透明卡片撑开呼吸间距，支持字号大小切换与长文本自动折行。

---

## 🛠️ 项目架构与技术栈

- **开发语言**：Go (Golang)
- **GUI 框架**：Fyne v2.6.x+ (强烈建议不要使用低于 2.6 的版本，否则无法通过编译)
- **数据交换**：JSON (本地沙盒存储)
- **自动化流水线**：GitHub Actions + `softprops/action-gh-release@v2` (实现多端多架构交叉编译及自动发版)

---

## 🚀 快速开始

### 1. 环境准备

确保你的本地环境已配置好 Go 1.21+ 及 C 编译器（Fyne 依赖 cgo）：

- **Windows**: 安装 MSYS2 并配置 gcc。
- **Ubuntu/WSL**: `sudo apt install go-gstreamer1.0-dev libgl1-mesa-dev libx11-dev libxi-dev libxcursor-dev libxrandr-dev libinerama-dev`

### 2. 获取代码与运行

```bash
git clone <你的项目仓库地址>
cd <项目文件夹>
go mod tidy
go run .
```

---

## 📂 本地沙盒文件规范

系统运行在 Fyne 的独立沙盒存储目录中，文件遵循以下严格的命名规范：

- **主题库文件**：`[文件类型]Data_[原始文件名]_[13位毫秒级时间戳].json`
- **衍生影子文件**：`[原始文件名]_[错题集|收藏集|答题记录].json`

> **注意**：系统内置白名单过滤机制，刷新主页题库列表时，所有的衍生影子文件都会被自动拦截，确保题库展示界面的纯净。

---

## 🤖 自动化部署 (GitHub Actions)

项目内置了持续集成方案，推行带有 `v*` 的 Git Tag 时将自动触发编译并发布到 GitHub Release：

```yaml
# 工作流权限安全加固配置
permissions:
  contents: write # 显式声明写入权限

# ...
- name: Create GitHub Release
  uses: softprops/action-gh-release@v2
  with:
    files: release_assets/*
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} # 无需手动在 Secrets 中配置，系统会自动托管注入
```

---

## 🤝 贡献指引

1. **事件处理**：若要扩展特殊手势，请参考 `TouchInterceptor` 结构体，通过鸭子类型实现 `fyne.Draggable` 或 `fyne.Scrollable` 接口。
2. **布局规整**：使用网格布局（如 `GridWithColumns`）时，为防止按钮被强行拉伸导致文字偏离，请将按钮文字放置在 `container.NewCenter` 中解耦尺寸束缚。
