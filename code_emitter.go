package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types"
)

type codeEmitter struct {
	w        chan *codeNode
	stackTop int
}

func newCodeEmitter(w chan *codeNode) *codeEmitter {
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
	return fmt.Sprintf("%v = %v", lhs, rhs)
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
	if value == "" {
		return "var " + name + " "
	}
	return fmt.Sprintf("var %v = %v", name, value)
}

func (cw codeEmitter) pointerDeref(pointerValue string, elemType types.Type) string {
	deref := fmt.Sprintf("%v[0]", pointerValue)
	if elemType == nil {
		return deref
	}

	return getCG(elemType).coerce(deref)
}

func (cw codeEmitter) returnIns(returns []ssa.Value) string {
	if len(returns) == 0 {
		return "return"
	}

	s := ""
	for i, v := range returns {
		s += fmt.Sprintf("%v[%v] = %v;", TupleVar, i, cw.coercedValue(v))
	}

	return s + fmt.Sprintf("return %v", TupleVar)
}

func (cw codeEmitter) functionCall(stackVar string, name string, args []ssa.Value) string {
	return cw.varDecl(stackVar, name+"("+cw.args(args, true)+")")
}

func (cw codeEmitter) extractionIns(v ssa.Value, index int, stackVar string) string {
	t := v.Type().(*types.Tuple).At(index).Type()
	return cw.varDecl(stackVar,
		getCG(t).coerce(
			fmt.Sprintf("%v[%v]", v.Name(), index)))
}

func (cw codeEmitter) storeIns(addr, value ssa.Value) string {
	return cw.assignment(
		cw.pointerDeref(addr.Name(), nil),
		cw.coercedValue(value))
}

func (cw *codeEmitter) loadStackVar(stackTop *int) string {
	*stackTop++
	return fmt.Sprintf("t%v", *stackTop)
}

func (cw codeEmitter) write(code string, typ cnType) {
	cw.w <- &codeNode{code, typ}
}

func (cw codeEmitter) writePrelude() func() {
	cw.write("(function() {", BlockOpen)
	cw.write(Prelude, Normal)
	return func() {
		cw.write("\ninit(); main()})()", BlockClose)
	}
}

func (cw codeEmitter) writeFuncDecl(fn *ssa.Function) func() {
	cw.write(fmt.Sprintf("function %v(%v) {",
		fn.Name(),
		cw.paramNames(fn.Params)), BlockOpen)

	for _, param := range fn.Params {
		pn := param.Name()
		cg := getCG(param.Type())
		cw.write(cw.assignment(pn, cg.coerce(pn)), Normal)
	}

	return cw.writeBC
}

func (cw codeEmitter) writeExecLoop() func() {
	cw.write("var $l = 0, $p = 0", Normal)
	cw.write("while (1) switch($l) {", BlockOpen)
	return cw.writeBC
}

func (cw codeEmitter) writeCase(i int, last bool) func() {
	cw.write(fmt.Sprintf("case %v:", i), BlockOpen)
	return func() {
		if !last {
			cw.write(fmt.Sprintf("$l = %v; $p = %v; break;",
				i+1, i), BlockClose)
		} else {
			cw.write("", BlockClose)
		}
	}
}

func (cw codeEmitter) ifIns(s *ssa.If) string {
	tblock := s.Block().Succs[0].Index
	fblock := s.Block().Succs[1].Index

	return fmt.Sprintf("$l = (%v) ? %v : %v; break",
		s.Cond.Name(), tblock, fblock)
}

func (cw codeEmitter) jumpIns(ins *ssa.Jump) string {
	return fmt.Sprintf("$l = %v; $p = %v; break;",
		ins.Block().Succs[0].Index, ins.Block().Index)
}

func (cw codeEmitter) writeBC() {
	cw.write("}", BlockClose)
}

func (cw codeEmitter) writePhiIns(ins *ssa.Phi, stackVar string) {
	preds := ins.Block().Preds
	cw.write(fmt.Sprintf("var %v; switch($p) {", stackVar), BlockOpen)
	for i, edge := range ins.Edges {
		cw.write(fmt.Sprintf("case %v: %v = %v; break;",
			preds[i].Index, stackVar, cw.value(edge)), Normal)
	}
	cw.writeBC()
}
