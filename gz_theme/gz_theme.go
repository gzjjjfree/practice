package gz_theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/gzjjjfree/practice/core"
)

// ForcedDarkTheme implements fyne.Theme to force a dark foreground / light background theme.
type ForcedDarkTheme struct{}

//var _ fyne.Theme = (*ForcedDarkTheme)(nil)

// Color returns forced theme colors.
func (f *ForcedDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForeground:
		return core.HexColor(core.TextBodyColor)
	case theme.ColorNameBackground:
		return core.HexColor(core.PageBgColor)
	case theme.ColorNameInputBackground:
		return core.HexColor(core.InputBgColor)
	case theme.ColorNameButton:
		return core.HexColor(core.BtnPrimaryBg)
	case theme.ColorNameOverlayBackground:
		return core.HexColor(core.CardBgColor)
	case theme.ColorNameMenuBackground:
		return core.HexColor(core.CardBgColor)
	case theme.ColorNameHover:
		return core.HexColor(core.CardBgColor)
	case theme.ColorNameFocus:
		return core.HexColor(core.BtnPrimaryBg)
	case theme.ColorNameScrollBar:
		return core.HexColor(core.CardBgColor)
	case theme.ColorNameSeparator:
		return core.HexColor(core.BorderLightColor)
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

// Font delegates to the default theme.
func (f *ForcedDarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon delegates to the default theme.
func (f *ForcedDarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size delegates to the default theme.
func (f *ForcedDarkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// noShadowTheme 自定义局部主题，专门用于抹除滚动条边缘的渐变阴影
type NoShadowTheme struct {
	fyne.Theme
}

func (t *NoShadowTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 拦截阴影颜色请求，直接返回完全透明
	if name == theme.ColorNameShadow {
		return color.Transparent
	}
	// 其他所有颜色正常跟随系统默认主题
	return t.Theme.Color(name, variant)
}
