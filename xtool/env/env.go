/*
 * Copyright (c) 2024 The GoPlus Authors (goplus.org). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package env

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func ExpandEnv(s string) string {
	return expandEnvWithCmd(s)
}

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
	return os.Expand(expanded, os.Getenv)
}
