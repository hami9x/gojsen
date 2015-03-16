package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types"
)

type codeWriter struct {
	w chan string
}

func (cw codeWriter) paramNames(params []*ssa.Parameter) string {
	pcode := make([]string, len(params))
	for i, param := range params {
		pcode[i] = param.Name()
	}

	return strings.Join(pcode, ", ")
}

func (cw codeWriter) writeAssignment(lhs string, rhs string) {
	cw.w <- fmt.Sprintf("%v = %v;", lhs, rhs)
}

func valueCode(cg valueCG, v ssa.Value) string {
	name, isLiteral := valueName(v.Name())
	vstr := name
	if !isLiteral {
		return cg.coerce(vstr)
	}

	return vstr
}

func valueName(name string) (string, bool) {
	spl := strings.Split(name, ":")
	return spl[0], len(spl) > 1
}

func (cw codeWriter) writeFuncDecl(fn *ssa.Function) func() {
	cw.w <- fmt.Sprintf("function %v(%v) {",
		fn.Name(),
		cw.paramNames(fn.Params))

	for _, param := range fn.Params {
		pn := param.Name()
		cg := getCG(param.Type())
		cw.writeAssignment(pn, cg.coerce(pn))
	}

	return cw.writeBC
}

func (cw codeWriter) writeBC() {
	cw.w <- "}"
}

func (cw codeWriter) writeValue(v ssa.Value) {
	cw.w <- valueCode(getCG(v.Type()), v)
}

func (cw codeWriter) writeArgs(args []ssa.Value) {
	for _, arg := range args {
		cw.writeValue(arg)
	}
}

func (cw codeWriter) writePrintln(args []ssa.Value) {
	cw.w <- "console.log("
	cw.writeArgs(args)
	cw.w <- ");"
}

func (cw codeWriter) writeUniversalWrap() func() {
	cw.w <- "(function() {"
	return func() {
		cw.w <- "\nmain()})()"
	}
}

func (cw codeWriter) writeVarDecl(name string, typ types.Type, v ssa.Value) {
	cw.w <- fmt.Sprintf("var %v = ", name)
	vg := getCG(typ)
	if v != nil {
		cw.w <- valueCode(getCG(typ), v)
	} else {
		cw.w <- vg.initialValue()
	}
	cw.w <- ";"
}
