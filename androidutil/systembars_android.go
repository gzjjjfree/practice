//go:build android

package androidutil

/*
#include <stdint.h>
void setLightSystemBars(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx);
*/
import "C"

import "fyne.io/fyne/v2/driver"

// SetLightSystemBars 强制状态栏/导航栏使用深色图标，与强制浅色主题搭配。
// 幂等，可重复调用。
func SetLightSystemBars() {
	_ = driver.RunNative(func(ctx interface{}) error {
		if ac, ok := ctx.(*driver.AndroidContext); ok {
			C.setLightSystemBars(C.uintptr_t(ac.VM), C.uintptr_t(ac.Env), C.uintptr_t(ac.Ctx))
		}
		return nil
	})
}
