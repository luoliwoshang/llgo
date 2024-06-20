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

package clang

import (
	"io"
	"os"
	"os/exec"
)

// -----------------------------------------------------------------------------

// Cmd represents a clang command.
type Cmd struct {
	app string

	Stdout io.Writer
	Stderr io.Writer
}

// New creates a new clang command.
func New(app string) *Cmd {
	if app == "" {
		app = "clang"
	}
	// 例如，当你在程序中使用 fmt.Println() 或 fmt.Printf() 等函数时，除非另外指定，否则输出默认是发送到 os.Stdout。
	return &Cmd{app, os.Stdout, os.Stderr}
}

// 执行指定指令
func (p *Cmd) Exec(args ...string) error {
	cmd := exec.Command(p.app, args...)
	cmd.Stdout = p.Stdout
	cmd.Stderr = p.Stderr
	return cmd.Run()
}

// -----------------------------------------------------------------------------
