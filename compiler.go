package main

import (
	"fmt"

	"go/token"

	"golang.org/x/tools/go/ssa"
)

type Compiler struct {
	*codeWriter
}

func (c Compiler) compileBuiltinCall(call *ssa.Builtin, args []ssa.Value) {
	switch call.Name() {
	case "println":
		c.writePrintln(args)
	}
}

func (c Compiler) compileCall(call *ssa.Call) {
	cc := call.Common()
	switch call := cc.Value.(type) {
	case *ssa.Builtin:
		c.compileBuiltinCall(call, cc.Args)
	}
}

func (c Compiler) compileUnaryOp(unop *ssa.UnOp, stackTop *int) {
	x := c.coercedValue(unop.X)
	var ass string
	switch unop.Op {
	case token.MUL:
		ass = c.pointerDeref(x, unop.Type())
	case token.NOT:
		ass = "!" + x
	default:
	}

	c.writeVarDecl(c.loadStackVar(stackTop), ass)
}

func (c Compiler) compileBinaryOp(binop *ssa.BinOp, stackTop *int) {
	x, y := c.value(binop.X), c.value(binop.Y)
	var ass string
	switch binop.Op {
	case token.ADD, token.MUL, token.SUB, token.QUO, token.REM,
		token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		ass = getCG(binop.X.Type()).coerce("(" + x + binop.Op.String() + y + ")")
	}

	c.writeVarDecl(c.loadStackVar(stackTop), ass)
}

func (c Compiler) compileInstruction(insI ssa.Instruction, stackTop *int) {
	switch ins := insI.(type) {
	case *ssa.Call:
		c.compileCall(ins)
	case *ssa.UnOp:
		c.compileUnaryOp(ins, stackTop)
	case *ssa.BinOp:
		c.compileBinaryOp(ins, stackTop)
	default:
		//fmt.Printf("Unhandled {%T, %v}\n", insI, insI.String())
		return
	}

	c.writeSC()
}

func (c Compiler) compileFunctionDecl(fn *ssa.Function) {
	funcClose := c.writeFuncDecl(fn)

	stackTop := -1
	i := 0
	for i < len(fn.Blocks) {
		blk := fn.Blocks[i]
		for _, ins := range blk.Instrs {
			c.compileInstruction(ins, &stackTop)
			fmt.Printf("{%T, %v}\n", ins, ins.String())
		}

		i++
	}

	funcClose()
}

func (c Compiler) compileGlobalDecl(gv *ssa.Global) {
	//println(gv.Type().Underlying().String())
	c.writeVarDecl(gv.Name(), c.initialValue(gv.Type()))
	c.writeSC()
}

func (c Compiler) Compile(prog *ssa.Program) {
	funcClose := c.writeUniversalWrap()

	for _, pkg := range prog.AllPackages() {
		for _, memI := range pkg.Members {
			switch mem := memI.(type) {
			case *ssa.Function:
				c.compileFunctionDecl(mem)
			case *ssa.Global:
				c.compileGlobalDecl(mem)
			}
		}
	}

	funcClose()
}
