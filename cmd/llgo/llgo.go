/*
 * Copyright (c) 2023 The GoPlus Authors (goplus.org). All rights reserved.
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

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/qiniu/x/log"

	"github.com/goplus/llgo/cmd/internal/base"
	"github.com/goplus/llgo/cmd/internal/build"
	"github.com/goplus/llgo/cmd/internal/clean"
	"github.com/goplus/llgo/cmd/internal/help"
	"github.com/goplus/llgo/cmd/internal/install"
	"github.com/goplus/llgo/cmd/internal/run"
)

func mainUsage() {
	help.PrintUsage(os.Stderr, base.Llgo)
	os.Exit(2)
}

func init() { // main 函数执行之前完成
	flag.Usage = mainUsage //  flag.Usage 用于打印命令行参数的使用方法
	base.Llgo.Commands = []*base.Command{
		build.Cmd,
		install.Cmd,
		run.Cmd,
		run.CmpTestCmd,
		clean.Cmd,
	}
}

func main() {
	// 标志（Flags）：
	// 标志通常以 - 或 -- 开头。它们用于指定配置选项或开关某些功能。例如，在命令行工具中，你可能会看到类似 -v 或 --verbose 这样的标志，用来启动详细模式。
	flag.Parse() //当使用 flag.Parse() 函数解析命令行输入后，所有被识别的标志（和它们的值）都会从命令行输入中去除，剩下的就是非标志参数
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage() //当没有提供任何参数时，flag.Usage() 函数会打印出命令行工具的使用方法
	}
	log.SetFlags(log.Ldefault &^ log.LstdFlags) //TODO: 设置日志级别

	base.CmdName = args[0] // for error messages 移除标志后的第一个参数即为命令
	if args[0] == "help" {
		help.Help(os.Stderr, args[1:])
		return
	}

BigCmdLoop:
	for bigCmd := base.Llgo; ; {
		for _, cmd := range bigCmd.Commands {
			if cmd.Name() != args[0] {
				continue
			}
			args = args[1:]            //获得命令后的所有参数
			if len(cmd.Commands) > 0 { //TODO: 什么时候会有子命令
				bigCmd = cmd
				if len(args) == 0 {
					help.PrintUsage(os.Stderr, bigCmd)
					os.Exit(2)
				}
				if args[0] == "help" {
					help.Help(os.Stderr, append(strings.Split(base.CmdName, " "), args[1:]...))
					return
				}
				base.CmdName += " " + args[0]
				continue BigCmdLoop
			}
			if !cmd.Runnable() {
				continue
			}
			cmd.Run(cmd, args) // 调用对应的指令，并且传递参数
			return
		}
		helpArg := ""
		if i := strings.LastIndex(base.CmdName, " "); i >= 0 {
			helpArg = " " + base.CmdName[:i]
		}
		fmt.Fprintf(os.Stderr, "llgo %s: unknown command\nRun 'llgo help%s' for usage.\n", base.CmdName, helpArg)
		os.Exit(2)
	}
}
