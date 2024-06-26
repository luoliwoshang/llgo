package env

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// 将$(cmd) 替换为 cmd 的执行结果，并将其中的环境变量替换为实际变量
func ExpandEnv(s string) string {
	return expandEnvWithCmd(s)
}

// 将$(cmd) 替换为 cmd 的执行结果，并将其中的环境变量替换为实际变量
func expandEnvWithCmd(s string) string {
	re := regexp.MustCompile(`\$\(([^)]+)\)`)
	expanded := re.ReplaceAllStringFunc(s, func(m string) string {
		cmd := re.FindStringSubmatch(m)[1]
		var out []byte
		var err error
		if runtime.GOOS == "windows" {
			out, err = exec.Command("cmd", "/C", cmd).Output()
		} else {
			out, err = exec.Command("sh", "-c", cmd).Output()
		}
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	})
	// 	如果你的字符串中包含了类似 ${PATH} 或 $HOME 这样的环境变量，例如：
	// expanded := "-L$HOME/opt/homebrew/Cellar/cjson/1.7.18/lib -lcjson"
	// result := os.Expand(expanded, os.Getenv)
	// 这时，os.Expand 会将 $HOME 替换成环境变量 HOME 的当前值。如果 HOME 环境变量的值是 /Users/username，则 result 将会是：
	// "-L/Users/username/opt/homebrew/Cellar/cjson/1.7.18/lib -lcjson"
	return os.Expand(expanded, os.Getenv)
}
