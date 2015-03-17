package main

import (
	"fmt"

	"go/token"

	"golang.org/x/tools/go/ssa"
)

type Compiler struct {
	*codeWriter
}

func (c Compiler) compileBuiltinCall(fn *ssa.Builtin, args []ssa.Value) {
	switch fn.Name() {
	case "println":
		c.writePrintln(args)
	}
}

func (c Compiler) compileCall(call *ssa.Call, stackVar string) {
	c.writeVarDecl(stackVar, "")

	cc := call.Common()
	switch fn := cc.Value.(type) {
	case *ssa.Builtin:
		c.compileBuiltinCall(fn, cc.Args)
	case *ssa.Function:
		c.writeFunctionCall(fn.Name(), cc.Args)
	}
}

func (c Compiler) compileUnaryOp(unop *ssa.UnOp, stackVar string) {
	x := c.coercedValue(unop.X)
	var ass string
	switch unop.Op {
	case token.MUL:
		ass = c.pointerDeref(x, unop.Type())
	case token.NOT:
		ass = "!" + x
	default:
	}

	c.writeVarDecl(stackVar, ass)
}

func (c Compiler) compileBinaryOp(binop *ssa.BinOp, stackVar string) {
	x, y := c.value(binop.X), c.value(binop.Y)
	var ass string
	switch binop.Op {
	case token.ADD, token.MUL, token.SUB, token.QUO, token.REM,
		token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		ass = getCG(binop.X.Type()).coerce("(" + x + binop.Op.String() + y + ")")
	}

	c.writeVarDecl(stackVar, ass)
}

func (c Compiler) compileReturn(ins *ssa.Return) {
	c.writeReturn(ins.Results)
}

func (c Compiler) compileInstruction(insI ssa.Instruction, stackTop *int) {
	switch ins := insI.(type) {
	case *ssa.Call:
		c.compileCall(ins, c.loadStackVar(stackTop))
	case *ssa.UnOp:
		c.compileUnaryOp(ins, c.loadStackVar(stackTop))
	case *ssa.BinOp:
		c.compileBinaryOp(ins, c.loadStackVar(stackTop))
	case *ssa.Return:
		c.writeReturn(ins.Results)
	case *ssa.Store:
		c.writeStore(ins.Addr, ins.Val)
	case *ssa.Extract:
		c.writeExtract(ins.Tuple.Name(), ins.Index, c.loadStackVar(stackTop))
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
	funcClose := c.writePrelude()

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
