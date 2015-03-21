package main

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

type stack struct {
	top    int
	pkgMap pkgMap
	pkg    *ssa.Package
}

func newStack(pkg *ssa.Package, pkgMap pkgMap) *stack {
	return &stack{
		top:    -1,
		pkgMap: pkgMap,
		pkg:    pkg,
	}
}

func isStackVar(name string) bool {
	r := []rune(name)
	if r[0] != 't' {
		return false
	}

	for i := 1; i < len(name); i++ {
		if r[i] < '0' || r[i] > '9' {
			return false
		}
	}

	return true
}

func (s *stack) LoadVar() string {
	s.top++
	return fmt.Sprintf("t%v", s.top)
}

func (s *stack) VarDecl(value string) string {
	return fmt.Sprintf("var %v = %v", s.LoadVar(), value)
}

func (s *stack) RelString(fn *ssa.Function) string {
	pkg := fn.Package()
	if s.pkg == pkg {
		return fn.Name()
	}

	if pkgName := s.pkgMap[pkg]; pkgName != "" {
		return pkgName + "." + sfPrefix(fn.Name())
	}

	panic("No such package.")
	return ""
}
