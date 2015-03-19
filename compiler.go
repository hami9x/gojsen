package main

import (
	"fmt"

	"go/token"

	"golang.org/x/tools/go/ssa"
)

type Compiler struct {
	e *codeEmitter
}

func (c Compiler) compileBuiltinCall(fn *ssa.Builtin, args []ssa.Value) {
	switch fn.Name() {
	case "println":
		c.e.write(c.e.stdPrintln(args))
	}
}

func (c Compiler) compileCall(call *ssa.Call, stackVar string) {
	c.e.write(c.e.varDecl(stackVar, ""))

	cc := call.Common()
	switch fn := cc.Value.(type) {
	case *ssa.Builtin:
		c.compileBuiltinCall(fn, cc.Args)
	case *ssa.Function:
		c.e.write(c.e.functionCall(fn.Name(), cc.Args))
	}
}

func (c Compiler) compileUnaryOp(unop *ssa.UnOp, stackVar string) {
	x := c.e.coercedValue(unop.X)
	var ass string
	switch unop.Op {
	case token.MUL:
		ass = c.e.pointerDeref(x, unop.Type())
	case token.NOT:
		ass = "!" + x
	default:
	}

	c.e.write(c.e.varDecl(stackVar, ass))
}

func (c Compiler) compileBinaryOp(binop *ssa.BinOp, stackVar string) {
	x, y := c.e.value(binop.X), c.e.value(binop.Y)
	var ass string
	switch binop.Op {
	case token.ADD, token.MUL, token.SUB, token.QUO, token.REM,
		token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		ass = getCG(binop.X.Type()).coerce("(" + x + binop.Op.String() + y + ")")
	}

	c.e.write(c.e.varDecl(stackVar, ass))
}

func (c Compiler) compileReturn(ins *ssa.Return) {
	c.e.write(c.e.functionReturn(ins.Results))
}

func (c Compiler) compileInstruction(insI ssa.Instruction, stackTop *int) {
	switch ins := insI.(type) {
	case *ssa.Call:
		c.compileCall(ins, c.e.loadStackVar(stackTop))
	case *ssa.UnOp:
		c.compileUnaryOp(ins, c.e.loadStackVar(stackTop))
	case *ssa.BinOp:
		c.compileBinaryOp(ins, c.e.loadStackVar(stackTop))
	case *ssa.Return:
		c.e.write(c.e.functionReturn(ins.Results))
	case *ssa.Store:
		c.e.write(c.e.varStore(ins.Addr, ins.Val))
	case *ssa.Extract:
		c.e.write(c.e.extraction(ins.Tuple.Name(), ins.Index, c.e.loadStackVar(stackTop)))
	default:
		//fmt.Printf("Unhandled {%T, %v}\n", insI, insI.String())
		return
	}

	c.e.writeSC()
}

func (c Compiler) compileFunctionDecl(fn *ssa.Function) {
	funcClose := c.e.writeFuncDecl(fn)

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
	c.e.write(c.e.varDecl(gv.Name(), c.e.initialValue(gv.Type())))
	c.e.writeSC()
}

func (c Compiler) Compile(prog *ssa.Program) {
	funcClose := c.e.writePrelude()

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
