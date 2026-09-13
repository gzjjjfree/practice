package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/gzjjjfree/practice/core"
	"github.com/gzjjjfree/practice/gz_theme"
	storagerelated "github.com/gzjjjfree/practice/storageRelated"
	"github.com/gzjjjfree/practice/ui/pages/home"
)

// Compile-time check: ForcedDarkTheme implements fyne.Theme.
//var _ fyne.Theme = (*gz_theme.ForcedDarkTheme)(nil)

// main 程序入口，初始化应用并启动首页。
func main() {
	a := app.NewWithID("com.gzjjjfree.practice")

	// 设置自定义存储目录（用于测试或特殊需求）
	// storage.SetStorageDir("/path/to/custom/storage")

	a.Settings().SetTheme(&gz_theme.ForcedDarkTheme{})

	w := a.NewWindow("题库练习系统")
	w.Resize(fyne.NewSize(400, 750))

	state := core.NewAppState()
	state.SetWindow(w)

	// Set the dialog function for core package
	//core.SetShowCustomConfirm(widgets.ShowCustomConfirm)

	// 扫描本地已有的缓存 json 文件充当 "StoredFiles"
	storagerelated.RefreshLocalFilesList(state)

	// Initialize network layer (loads saved token if exists)
	state.InitNetwork()

	// ==================== 启动时自动加载上次题库 ====================
	lastBankKey := a.Preferences().String("LastOpenedBankKey")
	if lastBankKey != "" {
		storagerelated.LoadAndRenderBank(w, state, lastBankKey)
	}
	// ================================================================

	// 启动时先显示登录界面，登录成功后才进入主界面
	home.ShowLoginPage(w, state, func() {
		home.ShowHome(w, state)
	})

	w.ShowAndRun()
}
