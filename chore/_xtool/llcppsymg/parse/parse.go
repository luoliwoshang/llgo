package parse

import (
	"errors"
	"strconv"
	"strings"

	"github.com/goplus/llgo/c"
	"github.com/goplus/llgo/c/clang"
	"github.com/goplus/llgo/chore/_xtool/llcppsymg/clangutils"
)

type SymbolInfo struct {
	GoName    string
	ProtoName string
}

type Context struct {
	namespaceName string
	className     string
	prefixes      []string
	symbolMap     map[string]*SymbolInfo
	currentFile   string
	nameCounts    map[string]int
}

func newContext(prefixes []string) *Context {
	return &Context{
		prefixes:   prefixes,
		symbolMap:  make(map[string]*SymbolInfo),
		nameCounts: make(map[string]int),
	}
}

func (c *Context) setNamespaceName(name string) {
	c.namespaceName = name
}

func (c *Context) setClassName(name string) {
	c.className = name
}

func (c *Context) setCurrentFile(filename string) {
	c.currentFile = filename
}

func (c *Context) removePrefix(str string) string {
	for _, prefix := range c.prefixes {
		if strings.HasPrefix(str, prefix) {
			return strings.TrimPrefix(str, prefix)
		}
	}
	return str
}

func toTitle(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func toCamel(originName string) string {
	if originName == "" {
		return ""
	}
	subs := strings.Split(string(originName), "_")
	name := ""
	for _, sub := range subs {
		name += toTitle(sub)
	}
	return name
}

// 1. remove prefix from config
// 2. convert to camel case
func (c *Context) toGoName(name string) string {
	name = c.removePrefix(name)
	return toCamel(name)
}

func (c *Context) genGoName(name string) string {
	class := c.toGoName(c.className)
	name = c.toGoName(name)

	var baseName string
	if class == "" {
		baseName = name
	} else {
		baseName = c.genMethodName(class, name)
	}

	return c.addSuffix(baseName)
}

func (c *Context) genMethodName(class, name string) string {
	prefix := "(*" + class + ")."
	if class == name {
		return prefix + "Init"
	}
	if name == "~"+class {
		return prefix + "Dispose"
	}
	return prefix + name
}

func (p *Context) genProtoName(cursor clang.Cursor) string {
	displayName := cursor.DisplayName()
	defer displayName.Dispose()

	scopingParts := clangutils.BuildScopingParts(cursor.SemanticParent())

	var builder strings.Builder
	for _, part := range scopingParts {
		builder.WriteString(part)
		builder.WriteString("::")
	}

	builder.WriteString(c.GoString(displayName.CStr()))
	return builder.String()
}

func (c *Context) addSuffix(name string) string {
	c.nameCounts[name]++
	count := c.nameCounts[name]
	if count > 1 {
		return name + "__" + strconv.Itoa(count-1)
	}
	return name
}

var context = newContext([]string{})

func collectFuncInfo(cursor clang.Cursor) {
	cursorStr := cursor.String()
	symbol := cursor.Mangling()

	name := c.GoString(cursorStr.CStr())
	symbolName := c.GoString(symbol.CStr())
	if len(symbolName) >= 1 && symbolName[0] == '_' {
		symbolName = symbolName[1:]
	}
	defer symbol.Dispose()
	defer cursorStr.Dispose()

	context.symbolMap[symbolName] = &SymbolInfo{
		GoName:    context.genGoName(name),
		ProtoName: context.genProtoName(cursor),
	}
}

func visit(cursor, parent clang.Cursor, clientData c.Pointer) clang.ChildVisitResult {
	switch cursor.Kind {
	case clang.CursorNamespace, clang.CursorClassDecl:
		nameStr := cursor.String()
		defer nameStr.Dispose()

		name := c.GoString(nameStr.CStr())
		if cursor.Kind == clang.CursorNamespace {
			context.setNamespaceName(name)
		} else {
			context.setClassName(name)
		}

		clang.VisitChildren(cursor, visit, nil)

		if cursor.Kind == clang.CursorNamespace {
			context.setNamespaceName("")
		} else {
			context.setClassName("")
		}

	case clang.CursorCXXMethod, clang.CursorFunctionDecl, clang.CursorConstructor, clang.CursorDestructor:
		loc := cursor.Location()
		var file clang.File
		var line, column c.Uint

		loc.SpellingLocation(&file, &line, &column, nil)
		filename := file.FileName()
		defer filename.Dispose()

		isCurrentFile := c.Strcmp(filename.CStr(), c.AllocaCStr(context.currentFile)) == 0
		isPublicMethod := (cursor.CXXAccessSpecifier() == clang.CXXPublic) && cursor.Kind == clang.CursorCXXMethod || cursor.Kind == clang.CursorConstructor || cursor.Kind == clang.CursorDestructor

		if isCurrentFile && (cursor.Kind == clang.CursorFunctionDecl || isPublicMethod) {
			collectFuncInfo(cursor)
		}
	}

	return clang.ChildVisit_Continue
}

func ParseHeaderFile(filepaths []string, prefixes []string, isCpp bool) (map[string]*SymbolInfo, error) {
	context = newContext(prefixes)
	index := clang.CreateIndex(0, 0)
	for _, filename := range filepaths {
		_, unit, err := clangutils.CreateTranslationUnit(&clangutils.Config{
			File:  filename,
			Temp:  false,
			IsCpp: isCpp,
			Index: index,
		})
		if err != nil {
			return nil, errors.New("Unable to parse translation unit for file " + filename)
		}

		cursor := unit.Cursor()
		context.setCurrentFile(filename)
		clang.VisitChildren(cursor, visit, nil)
		unit.Dispose()
	}
	index.Dispose()
	return context.symbolMap, nil
}
