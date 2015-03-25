package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types"
)

type codeEmitter struct {
	w        chan *codeNode
	frameTop int
}

func newCodeEmitter(w chan *codeNode) *codeEmitter {
	return &codeEmitter{
		w:        w,
		frameTop: -1,
	}
}

func (cw codeEmitter) paramNames(params []*ssa.Parameter) string {
	pcode := make([]string, len(params))
	for i, param := range params {
		pcode[i] = sfPrefix(param.Name())
	}

	return strings.Join(pcode, ", ")
}

func (cw codeEmitter) assignment(lhs string, rhs string) string {
	return fmt.Sprintf("%v = %v", lhs, rhs)
}

func (cw codeEmitter) args(args []ssa.Value) string {
	s := ""
	for _, arg := range args {
		s += value(arg, true)
	}

	return s
}

func (cw codeEmitter) stdPrintln(args []ssa.Value) string {
	return fmt.Sprintf("console.log(%v)", cw.args(args))
}

func (cw codeEmitter) initialValue(typ types.Type) string {
	vg := getCG(typ)
	return vg.initialValue()
}

func (cw codeEmitter) varDecl(name, value string) string {
	if value == "" {
		return "var " + name + " "
	}
	return fmt.Sprintf("var %v = %v", sfPrefix(name), value)
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
		s += fmt.Sprintf("%v[%v] = %v; ", TupleVar, i, value(v, false))
	}

	return s + fmt.Sprintf("return %v", TupleVar)
}

func (cw codeEmitter) functionCall(s *frame, name string, args []ssa.Value) string {
	return s.VarDecl(sfPrefix(name) + "(" + cw.args(args) + ")")
}

func (cw codeEmitter) extractionIns(v ssa.Value, index int, s *frame) string {
	t := v.Type().(*types.Tuple).At(index).Type()
	return s.VarDecl(getCG(t).coerce(
		fmt.Sprintf("%v[%v]", v.Name(), index)))
}

func (cw codeEmitter) storeIns(addr, v ssa.Value) string {
	return cw.assignment(
		cw.pointerDeref(sfPrefix(addr.Name()), nil),
		value(v, false))
}

func (cw codeEmitter) write(code string, typ cnType) {
	cw.w <- &codeNode{code, typ}
}

func (cw codeEmitter) writePrelude() func() {
	cw.write("(function() {", BlockOpen)
	cw.write(Prelude, Normal)
	return func() {
		cw.write("\n_main._init(); _main._main()})()", BlockClose)
	}
}

func (cw codeEmitter) writeFuncDecl(fn *ssa.Function) func() {
	cw.write(fmt.Sprintf("function %v(%v) {",
		sfPrefix(fn.Name()),
		cw.paramNames(fn.Params)), BlockOpen)

	for _, param := range fn.Params {
		pn := sfPrefix(param.Name())
		cg := getCG(param.Type())
		cw.write(cw.assignment(pn, cg.coerce(pn)), Normal)
	}

	return cw.writeBC
}

func (cw codeEmitter) writePackageDecl(pkgName string, members map[string]ssa.Member) func() {
	cw.write(fmt.Sprintf("var %v = new function() {", pkgName), BlockOpen)

	return func() {
		cw.write("return {", BlockOpen)
		s := ""
		i := 0
		for _, m := range members {
			name := sfPrefix(m.Name())
			s = fmt.Sprintf(`%v: %v`, name, name)
			if i < len(members)-1 {
				s += ","
			}
			cw.write(s, Normal)
			i++
		}
		cw.writeBC()
		cw.writeBC()
	}
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

func (cw codeEmitter) writePhiIns(ins *ssa.Phi, frameVar string) {
	preds := ins.Block().Preds
	cw.write(fmt.Sprintf("var %v; switch($p) {", frameVar), BlockOpen)
	for i, edge := range ins.Edges {
		cw.write(fmt.Sprintf("case %v: %v = %v; break;",
			preds[i].Index, frameVar, value(edge, false)), Normal)
	}
	cw.writeBC()
}
