package gz_theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/gzjjjfree/practice/core"
)

// ForcedDarkTheme implements fyne.Theme to force a dark foreground / light background theme.
type ForcedDarkTheme struct{}

var _ fyne.Theme = (*ForcedDarkTheme)(nil)

// Color returns forced theme colors.
func (f *ForcedDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForeground:
		return core.HexColor(core.TextBodyColor)
	case theme.ColorNameBackground:
		return core.HexColor(core.ColorBg)
	case theme.ColorNameInputBackground:
		return core.HexColor(core.InputBgColor)
	case theme.ColorNameButton:
		return core.HexColor(core.BtnPrimaryBg)
	case theme.ColorNameOverlayBackground:
		return core.HexColor(core.CardBgColor)
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
