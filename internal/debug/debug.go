// Package debug 提供受 GOXCTL_DEBUG 环境变量控制的调试输出。
//
// 与 goxctl 核心共用 GOXCTL_DEBUG：核心 `goxctl --verbose` 会写回该变量并传给被转发的
// 扩展子进程；扩展自身的 `--verbose`/`-v` 也会开启它。
package debug

import (
	"fmt"
	"os"
)

const envKey = "GOXCTL_DEBUG"

var enabled = isTruthy(os.Getenv(envKey))

func isTruthy(v string) bool {
	return v != "" && v != "0" && v != "false"
}

// Enable 显式开启调试，并写回环境变量。
func Enable() {
	enabled = true
	_ = os.Setenv(envKey, "1")
}

// Enabled 返回当前是否开启调试。
func Enabled() bool { return enabled }

// Logf 在调试开启时向 stderr 打印一行（统一 debug: 前缀）。
func Logf(format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
}
