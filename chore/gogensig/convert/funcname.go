package convert

import "strings"

type GoFuncName struct {
	recvName string
	funcName string
}

func NewGoFuncName(name string) *GoFuncName {
	l := strings.Split(name, ".")
	if len(l) < 2 {
		return &GoFuncName{funcName: name}
	}
	return &GoFuncName{recvName: l[0], funcName: l[1]}
}

func (p *GoFuncName) HasReceiver() bool {
	return len(p.recvName) > 0
}
