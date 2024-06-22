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

package build

import (
	"archive/zip"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/tools/go/ssa"

	"github.com/goplus/llgo/cl"
	"github.com/goplus/llgo/internal/packages"
	"github.com/goplus/llgo/ssa/abi"
	"github.com/goplus/llgo/xtool/clang"
	"github.com/goplus/llgo/xtool/env"

	llssa "github.com/goplus/llgo/ssa"
	clangCheck "github.com/goplus/llgo/xtool/clang/check"
)

type Mode int

const (
	ModeBuild Mode = iota
	ModeInstall
	ModeRun
	ModeCmpTest
)

const (
	debugBuild = packages.DebugPackagesLoad
)

func needLLFile(mode Mode) bool {
	return mode != ModeBuild
}

type Config struct {
	// 存放构建后的产物的路径
	BinPath string
	AppExt  string // ".exe" on Windows, empty on Unix
	OutFile string // only valid for ModeBuild when len(pkgs) == 1

	//TODO: only valid for ModeRun
	RunArgs []string
	Mode    Mode
}

func NewDefaultConf(mode Mode) *Config {
	bin := os.Getenv("GOBIN")
	if bin == "" {

		// 获得GOPATH的路径
		gopath, err := envGOPATH()
		if err != nil {
			panic(fmt.Errorf("cannot get GOPATH: %v", err))
		}
		bin = filepath.Join(gopath, "bin")
	}
	conf := &Config{
		// 获得GOPATH/bin 作为编译的二进制文件存放的路径
		BinPath: bin,
		Mode:    mode,
		AppExt:  DefaultAppExt(),
	}
	return conf
}

func envGOPATH() (string, error) {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "go"), nil
}

func DefaultAppExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// -----------------------------------------------------------------------------

const (
	loadFiles   = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles
	loadImports = loadFiles | packages.NeedImports
	loadTypes   = loadImports | packages.NeedTypes | packages.NeedTypesSizes
	loadSyntax  = loadTypes | packages.NeedSyntax | packages.NeedTypesInfo
)

func Do(args []string, conf *Config) {

	//解析一些运行时的参数
	//llgo run -v . -> flags:[-v] patterns:[.] verbose:true
	flags, patterns, verbose := ParseArgs(args, buildFlags)
	cfg := &packages.Config{
		// TODO:
		Mode:       loadSyntax | packages.NeedDeps | packages.NeedModule | packages.NeedExportFile,
		BuildFlags: flags,
		Fset:       token.NewFileSet(),
	}

	// TODO:
	if len(overlayFiles) > 0 {
		cfg.Overlay = make(map[string][]byte)
		for file, src := range overlayFiles {
			overlay := unsafe.Slice(unsafe.StringData(src), len(src))
			cfg.Overlay[filepath.Join(runtime.GOROOT(), "src", file)] = overlay
		}
	}

	llssa.Initialize(llssa.InitAll)

	prog := llssa.NewProgram(nil)
	// 获得类型对应的一些尺寸信息
	sizes := prog.TypeSizes

	//设置一个缓存器
	dedup := packages.NewDeduper()

	//默认为执行 .
	if patterns == nil {
		patterns = []string{"."}
	}

	// 为每个包加载依赖，处理类型内存信息等（go工具链）
	// 获得的initial包含的包就是编译过程的入口
	initial, err := packages.LoadEx(dedup, sizes, cfg, patterns...)
	check(err)
	// initial[0].ExportFile是 "/Users/zhangzhiyang/Library/Caches/go-build/49/49972779d0e9962d2935ca4-d"
	// 指向包含了编译后的包类型信息的文件的路径。这个文件是在编译过程中生成的，包含了包的公开接口信息但不包含实现细节。它使得其他包在编译时可以仅依据这个文件进行，而不需要包含完整的源代码。

	// CompiledGoFiles是 "/Users/zhangzhiyang/Documents/Code/goplus/llgo/_demo/hello/hello.go"
	// CompiledGoFiles考虑了实际参与编译的go文件，比如有的时候在某些平台会忽略的文件，GoFile会包含这些文件，而CompiledGoFiles只包含参与编译的go文件
	mode := conf.Mode
	if len(initial) == 1 && len(initial[0].CompiledGoFiles) > 0 {
		//TODO: 为什么要这么处理
		if mode == ModeBuild {
			mode = ModeInstall
		}
	} else if mode == ModeRun {
		// 如果没有包，并且没有进行编译的go文件，那么就报错
		if len(initial) > 1 {
			fmt.Fprintln(os.Stderr, "cannot run multiple packages")
		} else {
			fmt.Fprintln(os.Stderr, "no Go files in matched packages")
		}
		return
	}

	// 生成了一个包含替代包路径的列表,并且再加载那些包的路径，通过LoadEx加载
	// 这里altPkgPaths会和pattern一样的方式进行加载
	// TODO: 为什么是这些包被转换了？AltPkg怎么得出这个结果的
	// [
	// 	"github.com/goplus/llgo/internal/runtime",
	// 	"github.com/goplus/llgo/internal/lib/internal/bytealg",
	// 	"github.com/goplus/llgo/internal/lib/runtime",
	// 	"github.com/goplus/llgo/internal/lib/sync/atomic",
	// 	"github.com/goplus/llgo/internal/lib/sync",
	// 	"github.com/goplus/llgo/internal/lib/internal/reflectlite",
	// 	"github.com/goplus/llgo/internal/lib/errors",
	// 	"github.com/goplus/llgo/internal/lib/syscall"
	// ]
	altPkgPaths := altPkgs(initial, llssa.PkgRuntime)                  //TODO: 转换一些Pkg?，比如 "github.com/goplus/llgo/internal/lib/errors"  "github.com/goplus/llgo/internal/runtime"
	altPkgs, err := packages.LoadEx(dedup, sizes, cfg, altPkgPaths...) // 加载指定路径的包
	check(err)

	//TODO:
	noRt := 1
	prog.SetRuntime(func() *types.Package {
		noRt = 0
		return altPkgs[0].Types
	})
	prog.SetPython(func() *types.Package {
		return dedup.Check(llssa.PkgPython).Types
	})

	// 初始化一个ssa程序（go），但是不执行build
	progSSA := ssa.NewProgram(initial[0].Fset, ssaBuildMode)

	//TODO:
	patches := make(cl.Patches, len(altPkgPaths))

	// 执行ssa.Program的build，并遍历所有的包，获得其中需要打补丁（patch）的包
	altSSAPkgs(progSSA, patches, altPkgs[1:], verbose)

	ctx := &context{progSSA, prog, dedup, patches, make(map[string]none), initial, mode}

	//构建所有包
	pkgs := buildAllPkgs(ctx, initial, verbose)

	var llFiles []string
	dpkg := buildAllPkgs(ctx, altPkgs[noRt:], verbose)
	for _, pkg := range dpkg {
		if !strings.HasSuffix(pkg.ExportFile, ".ll") {
			continue
		}
		llFiles = append(llFiles, pkg.ExportFile)
	}

	//如果不只是构建，那么进行链接并且执行
	if mode != ModeBuild {
		nErr := 0
		for _, pkg := range initial {
			if pkg.Name == "main" {
				nErr += linkMainPkg(pkg, pkgs, llFiles, conf, mode, verbose)
			}
		}
		if nErr > 0 {
			os.Exit(nErr)
		}
	}
}

func setNeedRuntimeOrPyInit(pkg *packages.Package, needRuntime, needPyInit bool) {
	v := []byte{'0', '0'}
	if needRuntime {
		v[0] = '1'
	}
	if needPyInit {
		v[1] = '1'
	}
	pkg.ID = string(v) // just use pkg.ID to mark it needs runtime
}

func isNeedRuntimeOrPyInit(pkg *packages.Package) (needRuntime, needPyInit bool) {
	if len(pkg.ID) == 2 {
		return pkg.ID[0] == '1', pkg.ID[1] == '1'
	}
	return
}

const (
	ssaBuildMode = ssa.SanityCheckFunctions | ssa.InstantiateGenerics
)

type context struct {
	//go的ssa.Program
	progSSA *ssa.Program
	//llgo的ssa program
	prog  llssa.Program
	dedup packages.Deduper //TODO:
	//一个go程序中需要打补丁的包 string -> ssa.Package
	patches cl.Patches
	built   map[string]none //TODO:
	initial []*packages.Package
	mode    Mode
}

func buildAllPkgs(ctx *context, initial []*packages.Package, verbose bool) (pkgs []*aPackage) {
	prog := ctx.prog
	pkgs, errPkgs := allPkgs(ctx, initial, verbose)
	for _, errPkg := range errPkgs {
		for _, err := range errPkg.Errors {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Fprintln(os.Stderr, "cannot build SSA for package", errPkg)
	}
	built := ctx.built
	for _, aPkg := range pkgs {
		//获得ssa的包
		pkg := aPkg.Package
		//如果在built中，那么就不需要生成导出文件了
		if _, ok := built[pkg.PkgPath]; ok {
			pkg.ExportFile = ""
			continue
		}
		built[pkg.PkgPath] = none{}
		switch kind, param := cl.PkgKindOf(pkg.Types); kind {
		case cl.PkgDeclOnly:
			// skip packages that only contain declarations
			// and set no export file
			pkg.ExportFile = ""
		case cl.PkgLinkIR, cl.PkgLinkExtern, cl.PkgPyModule:
			if isPkgInLLGo(pkg.PkgPath) {
				pkg.ExportFile = concatPkgLinkFiles(pkg, verbose)
			} else {
				// panic("todo")
				// TODO(xsw): support packages out of llgo
				pkg.ExportFile = ""
			}
			if kind == cl.PkgLinkExtern { // need to be linked with external library
				// format: ';' separated alternative link methods. e.g.
				//   link: $LLGO_LIB_PYTHON; $(pkg-config --libs python3-embed); -lpython3
				expd := ""
				altParts := strings.Split(param, ";")
				for _, param := range altParts {
					expd = strings.TrimSpace(env.ExpandEnv(strings.TrimSpace(param)))
					if len(expd) > 0 {
						break
					}
				}
				if expd == "" {
					panic(fmt.Sprintf("'%s' cannot locate the external library", param))
				}

				command := ""
				if expd[0] == '-' {
					command += " " + expd
				} else {
					linkFile := expd
					dir, lib := filepath.Split(linkFile)
					command = " -l " + lib
					if dir != "" {
						command += " -L " + dir[:len(dir)-1]
					}
				}
				if err := clangCheck.CheckLinkArgs(command); err != nil {
					panic(fmt.Sprintf("test link args '%s' failed\n\texpanded to: %s\n\tresolved to: %v\n\terror: %v", param, expd, command, err))
				}
				if isSingleLinkFile(pkg.ExportFile) {
					pkg.ExportFile = command + " " + pkg.ExportFile
				} else {
					pkg.ExportFile = command + pkg.ExportFile
				}
			}
		default:
			buildPkg(ctx, aPkg, verbose)
			setNeedRuntimeOrPyInit(pkg, prog.NeedRuntime, prog.NeedPyInit)
		}
	}
	return
}

func linkMainPkg(pkg *packages.Package, pkgs []*aPackage, llFiles []string, conf *Config, mode Mode, verbose bool) (nErr int) {
	// 被编译的那个包的路径
	pkgPath := pkg.PkgPath

	// 包指明的可执行文件名字
	name := path.Base(pkgPath)

	//TODO: 什么时候这里会有OutFile不为空
	app := conf.OutFile
	if app == "" {
		app = filepath.Join(conf.BinPath, name+conf.AppExt)
	} //获得编译的程序的位置
	const N = 6

	//一个ll文件需要占用一个args，并且还有其他额外的链接的文件 TODO: 为什么是N个
	args := make([]string, N, len(pkg.Imports)+len(llFiles)+(N+1))
	args[0] = "-o"

	//输出文件的位置及名称
	args[1] = app
	//TODO:
	args[2] = "-Wno-override-module"
	//TODO:
	args[3] = "-Xlinker"
	if runtime.GOOS == "darwin" { // ld64.lld (macOS)
		// TODO: 下面两个参数是干啥的
		args[4] = "-dead_strip"
		args[5] = "" // It's ok to leave it empty, as we can assume libpthread is built-in on macOS.
	} else { // ld.lld (Unix), lld-link (Windows), wasm-ld (WebAssembly)
		// TODO: 下面两个参数是干啥的
		args[4] = "--gc-sections"
		args[5] = "-lpthread" // libpthread is built-in since glibc 2.34 (2021-08-01); we need to support earlier versions.
	}
	//args[6] = "-fuse-ld=lld" // TODO(xsw): to check lld exists or not
	//args[7] = "-O2"
	needRuntime := false
	needPyInit := false

	//pkg的export file为 "/Users/zhangzhiyang/Library/Caches/go-build/f1/f1034b28cafe60f86ff5c134f28d99bd8403810da0f706763be089151c0ec43a-d.ll"
	packages.Visit([]*packages.Package{pkg}, nil, func(p *packages.Package) {
		if p.ExportFile != "" { // skip packages that only contain declarations
			// 将ExportFile处理为参数
			args = appendLinkFiles(args, p.ExportFile)
			need1, need2 := isNeedRuntimeOrPyInit(p)
			if !needRuntime {
				needRuntime = need1
			}
			if !needPyInit {
				needPyInit = need2
			}
		}
	})

	// 获得编译的那个包对应的llgo的包
	var aPkg *aPackage
	for _, v := range pkgs {
		if v.Package == pkg { // found this package
			aPkg = v
			break
		}
	}

	//TODO:
	dirty := false

	// 需要链接runtime链接
	if needRuntime && llFiles != nil {
		for _, file := range llFiles {
			// 将需要编译和链接的文件添加到参数中
			args = appendLinkFiles(args, file)
		}
	} else {
		dirty = true
		fn := aPkg.LPkg.FuncOf(cl.RuntimeInit)
		fn.MakeBody(1).Return()
	}
	if needPyInit {
		dirty = aPkg.LPkg.PyInit()
	}

	if dirty && needLLFile(mode) {
		lpkg := aPkg.LPkg
		os.WriteFile(pkg.ExportFile, []byte(lpkg.String()), 0644)
	}

	if verbose || mode != ModeRun {
		fmt.Fprintln(os.Stderr, "#", pkgPath)
	}

	// 在执行完成后，返回link时候的错误
	defer func() {
		if e := recover(); e != nil {
			nErr = 1
		}
	}()

	// TODO(xsw): show work
	// 输出clang编译的参数
	if verbose {
		fmt.Fprintln(os.Stderr, "clang", args)
	}

	//新建一个clang的处理，并且执行对应参数
	err := clang.New("").Exec(args...)
	check(err)

	switch mode {
	case ModeRun:
		cmd := exec.Command(app, conf.RunArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		if s := cmd.ProcessState; s != nil {
			os.Exit(s.ExitCode())
		}
	case ModeCmpTest:
		cmpTest("", pkgPath, app, conf.RunArgs)
	}
	return
}

func buildPkg(ctx *context, aPkg *aPackage, verbose bool) {
	pkg := aPkg.Package
	pkgPath := pkg.PkgPath
	if debugBuild || verbose {
		fmt.Fprintln(os.Stderr, pkgPath)
	}
	if canSkipToBuild(pkgPath) {
		pkg.ExportFile = ""
		return
	}
	var syntax = pkg.Syntax
	if altPkg := aPkg.AltPkg; altPkg != nil {
		syntax = append(syntax, altPkg.Syntax...)
	}
	showDetail := verbose && pkgExists(ctx.initial, pkg)
	if showDetail {
		llssa.SetDebug(llssa.DbgFlagAll)
		cl.SetDebug(cl.DbgFlagAll)
	}
	ret, err := cl.NewPackageEx(ctx.prog, ctx.patches, aPkg.SSA, syntax)
	if showDetail {
		llssa.SetDebug(0)
		cl.SetDebug(0)
	}
	check(err)
	if needLLFile(ctx.mode) {
		pkg.ExportFile += ".ll"
		os.WriteFile(pkg.ExportFile, []byte(ret.String()), 0644)
		if debugBuild || verbose {
			fmt.Fprintf(os.Stderr, "==> Export %s: %s\n", aPkg.PkgPath, pkg.ExportFile)
		}
	}
	aPkg.LPkg = ret
}

const (
	altPkgPathPrefix = abi.PatchPathPrefix
)

// 通过packages.Visit来访问当前编译的包通过DFS搜索到的直接和间接的依赖包，并将其中能进行替换的包返回出来
func altPkgs(initial []*packages.Package, alts ...string) []string {
	// 遍历一个包以及它的所有直接和间接依赖
	packages.Visit(initial, nil, func(p *packages.Package) {
		// 存在类型并且不包含类型错误：p.IllTyped = true
		if p.Types != nil && !p.IllTyped {
			if _, ok := hasAltPkg[p.PkgPath]; ok {
				alts = append(alts, altPkgPathPrefix+p.PkgPath)
			}
		}
	})
	return alts
}

func altSSAPkgs(prog *ssa.Program, patches cl.Patches, alts []*packages.Package, verbose bool) { //TODO: patches 代表?
	packages.Visit(alts, nil, func(p *packages.Package) {
		if p.Types != nil && !p.IllTyped {
			if debugBuild || verbose {
				log.Println("==> BuildSSA", p.PkgPath)
			}

			//go工具链的ssa表达
			pkgSSA := prog.CreatePackage(p.Types, p.Syntax, p.TypesInfo, true)

			// altPkgPathPrefix = "github.com/goplus/llgo/internal/lib/"
			// "github.com/goplus/llgo/internal/lib/internal/bytealg"
			if strings.HasPrefix(p.PkgPath, altPkgPathPrefix) {
				path := p.PkgPath[len(altPkgPathPrefix):]

				//lib后的路径
				patches[path] = pkgSSA
				if debugBuild || verbose {
					log.Println("==> Patching", path)
				}
			}
		}
	})
	prog.Build()
}

type aPackage struct {

	// 存在 export file "/Users/zhangzhiyang/Library/Caches/go-build/9a/9a87da144dcfb1906be0685b4aafe0fbe907b74c3d794f9e62d0e270eadcce44-d.ll"
	*packages.Package
	SSA    *ssa.Package
	AltPkg *packages.Cached
	LPkg   llssa.Package
}

func allPkgs(ctx *context, initial []*packages.Package, verbose bool) (all []*aPackage, errs []*packages.Package) {
	prog := ctx.progSSA
	built := ctx.built
	packages.Visit(initial, nil, func(p *packages.Package) {
		if p.Types != nil && !p.IllTyped {
			pkgPath := p.PkgPath
			if _, ok := built[pkgPath]; ok || strings.HasPrefix(pkgPath, altPkgPathPrefix) {
				return
			}
			var altPkg *packages.Cached
			var ssaPkg = createSSAPkg(prog, p, verbose)
			if _, ok := hasAltPkg[pkgPath]; ok {
				if altPkg = ctx.dedup.Check(altPkgPathPrefix + pkgPath); altPkg == nil {
					return
				}
			}
			all = append(all, &aPackage{p, ssaPkg, altPkg, nil})
		} else {
			errs = append(errs, p)
		}
	})
	return
}

func createSSAPkg(prog *ssa.Program, p *packages.Package, verbose bool) *ssa.Package {
	pkgSSA := prog.ImportedPackage(p.PkgPath)
	if pkgSSA == nil {
		if debugBuild || verbose {
			log.Println("==> BuildSSA", p.PkgPath)
		}
		pkgSSA = prog.CreatePackage(p.Types, p.Syntax, p.TypesInfo, true)
		pkgSSA.Build() // TODO(xsw): build concurrently
	}
	return pkgSSA
}

var (
	// TODO(xsw): complete build flags
	buildFlags = map[string]bool{
		"-C":         true,  // -C dir: Change to dir before running the command
		"-a":         false, // -a: force rebuilding of packages that are already up-to-date
		"-n":         false, // -n: print the commands but do not run them
		"-p":         true,  // -p n: the number of programs to run in parallel
		"-race":      false, // -race: enable data race detection
		"-cover":     false, // -cover: enable coverage analysis
		"-covermode": true,  // -covermode mode: set the mode for coverage analysis
		"-v":         false, // -v: print the names of packages as they are compiled
		"-work":      false, // -work: print the name of the temporary work directory and do not delete it when exiting
		"-x":         false, // -x: print the commands
		"-tags":      true,  // -tags 'tag,list': a space-separated list of build tags to consider satisfied during the build
		"-pkgdir":    true,  // -pkgdir dir: install and load all packages from dir instead of the usual locations
		"-ldflags":   true,  // --ldflags 'flag list': arguments to pass on each go tool link invocation
	}
)

// 以标识为切割，比如llgo run -v .   run会在一开始在main函数中就处理掉了，而剩下的-v 和 .会被该函数处理为flags 和 patterns
func ParseArgs(args []string, swflags map[string]bool) (flags, patterns []string, verbose bool) {
	n := len(args)
	for i := 0; i < n; i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") { // 处理标志
			checkFlag(arg, &i, &verbose, swflags)
		} else {
			flags, patterns = args[:i], args[i:] //以标志为切割
			return
		}
	}
	flags = args
	return
}

func SkipFlagArgs(args []string) int {
	n := len(args)
	for i := 0; i < n; i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") { //TODO:
			checkFlag(arg, &i, nil, buildFlags)
		} else {
			return i
		}
	}
	return -1
}

func checkFlag(arg string, i *int, verbose *bool, swflags map[string]bool) {
	if pos := strings.IndexByte(arg, '='); pos > 0 {
		if verbose != nil && arg == "-v=true" {
			*verbose = true
		}
	} else if hasarg, ok := swflags[arg]; ok {
		if hasarg {
			*i++
		} else if verbose != nil && arg == "-v" {
			*verbose = true
		}
	} else {
		panic("unknown flag: " + arg)
	}
}

var (
	rootDir string
)

// TODO: 了解这个变量有啥用
func llgoRoot() string {
	if rootDir == "" {
		root := os.Getenv("LLGOROOT")
		if root == "" {
			panic("todo: LLGOROOT not set")
		}
		rootDir, _ = filepath.Abs(root)
	}
	return rootDir
}

// 将一些带编译参数的处理为args
func appendLinkFiles(args []string, file string) []string {
	if isSingleLinkFile(file) {
		return append(args, file)
	}
	//处理类似于这种的路径 -L/opt/homebrew/Cellar/bdw-gc/8.2.6/lib -lgc -lpthread /Users/zhangzhiyang/Documents/Code/goplus/llgo/c/bdwgc/llgo_autogen.ll
	return append(args, strings.Split(file[1:], " ")...)
}

func isSingleLinkFile(ret string) bool {
	return len(ret) > 0 && ret[0] != ' '
}

func concatPkgLinkFiles(pkg *packages.Package, verbose bool) string {
	var b strings.Builder
	var ret string
	var n int
	llgoPkgLinkFiles(pkg, "", func(linkFile string) {
		if n == 0 {
			ret = linkFile
		} else {
			b.WriteByte(' ')
			b.WriteString(linkFile)
		}
		n++
	}, verbose)
	if n > 1 {
		b.WriteByte(' ')
		b.WriteString(ret)
		return b.String()
	}
	return ret
}

// const LLGoFiles = "file1; file2; ..."
func llgoPkgLinkFiles(pkg *packages.Package, llFile string, procFile func(linkFile string), verbose bool) {
	if o := pkg.Types.Scope().Lookup("LLGoFiles"); o != nil {
		val := o.(*types.Const).Val()
		if val.Kind() == constant.String {
			clFiles(constant.StringVal(val), pkg, procFile, verbose)
		}
	}
	unzipPkgLinkFiles(pkg.PkgPath, llFile, procFile)
}

// files = "file1; file2; ..."
func clFiles(files string, pkg *packages.Package, procFile func(linkFile string), verbose bool) {
	dir := filepath.Dir(pkg.GoFiles[0])
	expFile := pkg.ExportFile
	for _, file := range strings.Split(files, ";") {
		cFile := filepath.Join(dir, strings.TrimSpace(file))
		clFile(cFile, expFile, procFile, verbose)
	}
}

func clFile(cFile, expFile string, procFile func(linkFile string), verbose bool) {
	llFile := expFile + filepath.Base(cFile) + ".ll"
	args := []string{"-emit-llvm", "-S", "-o", llFile, "-c", cFile}
	if verbose {
		fmt.Fprintln(os.Stderr, "clang", args)
	}
	err := clang.New("").Exec(args...)
	check(err)
	procFile(llFile)
}

func unzipPkgLinkFiles(pkgPath string, llFile string, procFile func(linkFile string)) {
	dir := llgoRoot() + pkgPath[len(llgoModPath):] + "/"
	if llFile == "" {
		llFile = "llgo_autogen.ll"
	}
	llPath := dir + llFile
	llaPath := llPath + "a"
	zipf, err := zip.OpenReader(llaPath)
	if err != nil {
		procFile(llPath)
		return
	}
	defer zipf.Close()

	for _, f := range zipf.File {
		procFile(dir + f.Name)
	}
	if _, err := os.Stat(llPath); os.IsNotExist(err) {
		for _, f := range zipf.File {
			decodeFile(dir+f.Name, f)
		}
	}
}

const (
	llgoModPath = "github.com/goplus/llgo"
)

func isPkgInLLGo(pkgPath string) bool {
	return isPkgInMod(pkgPath, llgoModPath)
}

func isPkgInMod(pkgPath, modPath string) bool {
	if strings.HasPrefix(pkgPath, modPath) {
		suffix := pkgPath[len(modPath):]
		return suffix == "" || suffix[0] == '/'
	}
	return false
}

func decodeFile(outFile string, zipf *zip.File) (err error) {
	f, err := zipf.Open()
	if err != nil {
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err == nil {
		err = os.WriteFile(outFile, data, 0644)
	}
	return
}

func pkgExists(initial []*packages.Package, pkg *packages.Package) bool {
	for _, v := range initial {
		if v == pkg {
			return true
		}
	}
	return false
}

func canSkipToBuild(pkgPath string) bool {
	if _, ok := hasAltPkg[pkgPath]; ok {
		return false
	}
	switch pkgPath {
	case "unsafe":
		return true
	default:
		return strings.HasPrefix(pkgPath, "internal/") ||
			strings.HasPrefix(pkgPath, "runtime/internal/")
	}
}

type none struct{}

// 需要进行转换的lib包名，会被转换为 "github.com/goplus/llgo/internal/lib/" + pkgPath
var hasAltPkg = map[string]none{
	"errors":               {},
	"internal/abi":         {},
	"internal/bytealg":     {},
	"internal/reflectlite": {},
	"io":                   {},
	"io/fs":                {},
	"math":                 {},
	"math/cmplx":           {},
	"reflect":              {},
	"sync":                 {},
	"sync/atomic":          {},
	"syscall":              {},
	"os":                   {},
	"runtime":              {},
}

var overlayFiles = map[string]string{
	"math/exp_amd64.go": "package math;",
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// -----------------------------------------------------------------------------
