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
		c.e.write(c.e.stdPrintln(args), Normal)
	}
}

func (c Compiler) compileCall(call *ssa.Call, stackVar string) {

	cc := call.Common()
	switch fn := cc.Value.(type) {
	case *ssa.Builtin:
		c.compileBuiltinCall(fn, cc.Args)
	case *ssa.Function:
		c.e.write(c.e.functionCall(stackVar, fn.Name(), cc.Args), Normal)
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

	c.e.write(c.e.varDecl(stackVar, ass), Normal)
}

func (c Compiler) compileBinaryOp(binop *ssa.BinOp, stackVar string) {
	x, y := c.e.value(binop.X), c.e.value(binop.Y)
	var ass string
	switch binop.Op {
	case token.ADD, token.MUL, token.SUB, token.QUO, token.REM:
		ass = getCG(binop.X.Type()).coerce("(" + x + binop.Op.String() + y + ")")
	case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		ass = x + binop.Op.String() + y
	}

	c.e.write(c.e.varDecl(stackVar, ass), Normal)
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
		c.e.write(c.e.returnIns(ins.Results), Normal)
	case *ssa.Store:
		c.e.write(c.e.storeIns(ins.Addr, ins.Val), Normal)
	case *ssa.Extract:
		c.e.write(c.e.extractionIns(ins.Tuple, ins.Index, c.e.loadStackVar(stackTop)), Normal)
	case *ssa.If:
		c.e.write(c.e.ifIns(ins), BlockClose)
	case *ssa.Jump:
		c.e.write(c.e.jumpIns(ins), BlockClose)
	case *ssa.Phi:
		c.e.writePhiIns(ins, c.e.loadStackVar(stackTop))
	default:
		//fmt.Printf("Unhandled {%T, %v}\n", insI, insI.String())
		return
	}
}

func (c Compiler) compileBlock(blk *ssa.BasicBlock, stackTop *int) {
	for _, ins := range blk.Instrs {
		c.compileInstruction(ins, stackTop)
		fmt.Printf("{%T, %v}\n", ins, ins.String())
	}
}

func (c Compiler) compileFunctionDecl(fn *ssa.Function) {
	funcClose := c.e.writeFuncDecl(fn)
	defer funcClose()

	stackTop := -1
	if len(fn.Blocks) == 1 {
		c.compileBlock(fn.Blocks[0], &stackTop)
		return
	}

	elClose := c.e.writeExecLoop()
	defer elClose()

	for i, blk := range fn.Blocks {
		caseClose := c.e.writeCase(i, i == len(fn.Blocks)-1)
		c.compileBlock(blk, &stackTop)
		ins := blk.Instrs
		if len(ins) > 0 {
			switch ins[len(blk.Instrs)-1].(type) {
			case *ssa.Jump, *ssa.If: // do nothing
			case *ssa.Return:
				c.e.write("", BlockClose)
			default:
				caseClose()
			}
		}
	}
}

func (c Compiler) compileGlobalDecl(gv *ssa.Global) {
	c.e.write(c.e.varDecl(gv.Name(), c.e.initialValue(gv.Type())), Normal)
}

func (c Compiler) Compile(prog *ssa.Program) {
	funcClose := c.e.writePrelude()
	defer funcClose()

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
}
