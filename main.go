package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/gz_theme"
	"github.com/gzjjjfree/practice/ui/pages"
)

// Compile-time check: ForcedDarkTheme implements fyne.Theme.
var _ fyne.Theme = (*gz_theme.ForcedDarkTheme)(nil)

// main 程序入口，初始化应用并启动首页。
func main() {
	a := app.NewWithID("com.gzjjjfree.practice")

	// 设置自定义存储目录（用于测试或特殊需求）
	// storage.SetStorageDir("/path/to/custom/storage")

	a.Settings().SetTheme(&gz_theme.ForcedDarkTheme{})

	w := a.NewWindow("题库练习系统")
	w.Resize(fyne.NewSize(400, 750))

	state := &core.AppState{
		UserAnswers: make(map[string][]string),
		Submitted:   make(map[string]bool),
		StoredFiles: []string{},
		ModeRecords: make(map[string]map[string]string),
	}

	// 扫描本地已有的缓存 json 文件充当 "StoredFiles"
	pages.RefreshLocalFilesList(state)

	// ==================== 启动时自动加载上次题库 ====================
	lastBankKey := a.Preferences().String("LastOpenedBankKey")
	if lastBankKey != "" {
		pages.LoadAndRenderBank(w, state, lastBankKey)
	}
	// ================================================================

	pages.ShowHome(w, state)
	w.ShowAndRun()
}
