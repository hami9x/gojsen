package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types"
)

type codeEmitter struct {
	w        chan string
	stackTop int
}

func newCodeEmitter(w chan string) *codeEmitter {
	return &codeEmitter{
		w:        w,
		stackTop: -1,
	}
}

func (cw codeEmitter) paramNames(params []*ssa.Parameter) string {
	pcode := make([]string, len(params))
	for i, param := range params {
		pcode[i] = param.Name()
	}

	return strings.Join(pcode, ", ")
}

func (cw codeEmitter) assignment(lhs string, rhs string) string {
	return fmt.Sprintf("%v = %v;", lhs, rhs)
}

func (cw codeEmitter) coercedValue(v ssa.Value) string {
	name, isLiteral := parseValue(v.Name())
	vstr := name
	if !isLiteral {
		return getCG(v.Type()).coerce(vstr)
	}

	return vstr
}

func (cw codeEmitter) value(v ssa.Value) string {
	name, _ := parseValue(v.Name())
	return name
}

func parseValue(name string) (string, bool) {
	spl := strings.Split(name, ":")
	return spl[0], len(spl) > 1
}

func (cw codeEmitter) args(args []ssa.Value, coerce bool) string {
	s := ""
	for _, arg := range args {
		if coerce {
			s += cw.coercedValue(arg)
		} else {
			s += cw.value(arg)
		}
	}

	return s
}

func (cw codeEmitter) stdPrintln(args []ssa.Value) string {
	return fmt.Sprintf("console.log(%v)", cw.args(args, false))
}

func (cw codeEmitter) initialValue(typ types.Type) string {
	vg := getCG(typ)
	return vg.initialValue()
}

func (cw codeEmitter) varDecl(name, value string) string {
	return fmt.Sprintf("var %v = %v", name, value)
}

func (cw codeEmitter) pointerDeref(pointerValue string, elemType types.Type) string {
	deref := fmt.Sprintf("%v[0]", pointerValue)
	if elemType == nil {
		return deref
	}

	return getCG(elemType).coerce(deref)
}

func (cw codeEmitter) functionReturn(returns []ssa.Value) string {
	if len(returns) == 0 {
		return ""
	}

	s := ""
	for i, v := range returns {
		s += fmt.Sprintf("%v[%v] = %v;", TupleVar, i, cw.coercedValue(v))
	}

	return s + fmt.Sprintf("return %v", TupleVar)
}

func (cw codeEmitter) functionCall(name string, args []ssa.Value) string {
	return name + "(" + cw.args(args, true) + ")"
}

func (cw codeEmitter) extraction(name string, index int, stackVar string) string {
	return cw.varDecl(stackVar, fmt.Sprintf("%v[%v]", name, index))
}

func (cw codeEmitter) varStore(addr, value ssa.Value) string {
	return cw.assignment(
		cw.pointerDeref(addr.Name(), nil),
		cw.coercedValue(value))
}

func (cw *codeEmitter) loadStackVar(stackTop *int) string {
	*stackTop++
	return fmt.Sprintf("t%v", *stackTop)
}

func (cw codeEmitter) write(code string) {
	if code != "" {
		cw.w <- code
	}
}

func (cw codeEmitter) writePrelude() func() {
	cw.w <- "(function() {"
	cw.w <- Prelude
	return func() {
		cw.w <- "\ninit(); main()})()"
	}
}

func (cw codeEmitter) writeSC() {
	cw.w <- ";"
}

func (cw codeEmitter) writeFuncDecl(fn *ssa.Function) func() {
	cw.w <- fmt.Sprintf("function %v(%v) {",
		fn.Name(),
		cw.paramNames(fn.Params))

	for _, param := range fn.Params {
		pn := param.Name()
		cg := getCG(param.Type())
		cw.write(cw.assignment(pn, cg.coerce(pn)))
	}

	return cw.writeBC
}

func (cw codeEmitter) writeBC() {
	cw.w <- "}"
}
