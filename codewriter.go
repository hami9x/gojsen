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

func valueCode(g *valueCodegen, v ssa.Value) string {
	name, isLiteral := valueName(v.Name())
	vstr := g.value(name, v.Type())
	if !isLiteral {
		return g.coerce(vstr)
	}

	return vstr
}

func valueName(name string) (string, bool) {
	spl := strings.Split(name, ":")
	return spl[0], len(spl) > 1
}

func (cw codeWriter) codeGenerator(typ types.Type) *valueCodegen {
	switch t := typ.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return intCodegen
		case types.String:
			return stringCodegen
		}
	}

	panic("Unhandled")
}

func (cw codeWriter) writeFuncDecl(fn *ssa.Function) func() {
	cw.w <- fmt.Sprintf("function %v(%v) {",
		fn.Name(),
		cw.paramNames(fn.Params))

	for _, param := range fn.Params {
		pn := param.Name()
		cg := cw.codeGenerator(param.Type())
		cw.writeAssignment(pn, cg.coerce(pn))
	}

	return cw.writeBC
}

func (cw codeWriter) writeBC() {
	cw.w <- "}"
}

func (cw codeWriter) writeValue(v ssa.Value) {
	cw.w <- valueCode(cw.codeGenerator(v.Type()), v)
}

func (cw codeWriter) writeArgs(args []ssa.Value) {
	for _, arg := range args {
		cw.writeValue(arg)
	}
}

func (cw codeWriter) writePrintln(args []ssa.Value) {
	cw.w <- "console.log("
	cw.writeArgs(args)
	cw.w <- ")"
}

func (cw codeWriter) writeGlobalWrap() func() {
	cw.w <- "(function() {"
	return func() {
		cw.w <- "})()"
	}
}
