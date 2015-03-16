package main

import "golang.org/x/tools/go/ssa"

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

func (c Compiler) compileInstruction(insI ssa.Instruction) {
	switch ins := insI.(type) {
	case *ssa.Call:
		c.compileCall(ins)
	}
}

func (c Compiler) compileFunctionDecl(fn *ssa.Function) {
	funcClose := c.writeFuncDecl(fn)

	i := 0
	for i < len(fn.Blocks) {
		blk := fn.Blocks[i]
		for _, ins := range blk.Instrs {
			c.compileInstruction(ins)
		}

		i++
	}

	funcClose()
}

func (c Compiler) compileGlobalDecl(gv *ssa.Global) {
	//println(gv.Type().Underlying().String())
	c.writeVarDecl(gv.Name(), gv.Type(), nil)
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
