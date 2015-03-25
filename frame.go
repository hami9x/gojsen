package main

import (
	"fmt"

	"golang.org/x/tools/go/ssa"
)

type frame struct {
	top    int
	pkgMap pkgMap
	pkg    *ssa.Package
}

func newFrame(pkg *ssa.Package, pkgMap pkgMap) *frame {
	return &frame{
		top:    -1,
		pkgMap: pkgMap,
		pkg:    pkg,
	}
}

func (s *frame) LoadVar() string {
	s.top++
	return fmt.Sprintf("t%v", s.top)
}

func (s *frame) VarDecl(value string) string {
	return fmt.Sprintf("var %v = %v", s.LoadVar(), value)
}

func (s *frame) RelString(fn *ssa.Function) string {
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
