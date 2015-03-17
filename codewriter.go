package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types"
)

type codeWriter struct {
	w        chan string
	stackTop int
}

func newCodeWriter(w chan string) *codeWriter {
	return &codeWriter{
		w:        w,
		stackTop: -1,
	}
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

func (cw codeWriter) coercedValue(v ssa.Value) string {
	name, isLiteral := parseValue(v.Name())
	vstr := name
	if !isLiteral {
		return getCG(v.Type()).coerce(vstr)
	}

	return vstr
}

func (cw codeWriter) value(v ssa.Value) string {
	name, _ := parseValue(v.Name())
	return name
}

func parseValue(name string) (string, bool) {
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

func (cw codeWriter) writeArgs(args []ssa.Value) {
	for _, arg := range args {
		cw.w <- cw.value(arg)
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
		cw.w <- "\ninit(); main()})()"
	}
}

func (cw codeWriter) writeSC() {
	cw.w <- ";"
}

func (cw codeWriter) initialValue(typ types.Type) string {
	vg := getCG(typ)
	return vg.initialValue()
}

func (cw codeWriter) writeVarDecl(name, value string) {
	cw.w <- fmt.Sprintf("var %v = %v", name, value)
}

func (cw codeWriter) pointerDeref(pointerValue string, elemType types.Type) string {
	deref := fmt.Sprintf("%v[0]", pointerValue)
	return getCG(elemType).coerce(deref)
}

func (cw codeWriter) write(code string) {
	cw.w <- code
}

func (cw *codeWriter) loadStackVar(stackTop *int) string {
	*stackTop++
	return fmt.Sprintf("t%v", *stackTop)
}
