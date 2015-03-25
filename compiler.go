package main

import (
	"fmt"
	"go/token"

	"golang.org/x/tools/go/ssa"
)

type pkgMap map[*ssa.Package]string

type Compiler struct {
	e       *codeEmitter
	pkgMap  pkgMap
	pkgIden map[string]bool
}

func NewCompiler(e *codeEmitter) *Compiler {
	return &Compiler{
		e:       e,
		pkgMap:  make(map[*ssa.Package]string),
		pkgIden: make(map[string]bool),
	}
}

func (c Compiler) compileBuiltinCall(fn *ssa.Builtin, args []ssa.Value, s *frame) {
	switch fn.Name() {
	case "println":
		c.e.write(s.VarDecl(c.e.stdPrintln(args)), Normal)
	}
}

func (c Compiler) compileCall(call *ssa.Call, s *frame) {
	cc := call.Common()
	switch fn := cc.Value.(type) {
	case *ssa.Builtin:
		c.compileBuiltinCall(fn, cc.Args, s)
	case *ssa.Function:
		c.e.write(c.e.functionCall(s, s.RelString(fn), cc.Args), Normal)
	}
}

func (c Compiler) compileUnaryOp(unop *ssa.UnOp, s *frame) {
	x := value(unop.X, false)
	var ass string
	switch unop.Op {
	case token.MUL:
		ass = c.e.pointerDeref(x, unop.Type())
	case token.NOT:
		ass = "!" + x
	default:
	}

	c.e.write(s.VarDecl(ass), Normal)
}

func (c Compiler) compileBinaryOp(binop *ssa.BinOp, s *frame) {
	x, y := value(binop.X, true), value(binop.Y, true)
	var ass string
	switch binop.Op {
	case token.ADD, token.MUL, token.SUB, token.QUO, token.REM:
		ass = getCG(binop.X.Type()).coerce("(" + x + binop.Op.String() + y + ")")
	case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		ass = x + binop.Op.String() + y
	}

	c.e.write(s.VarDecl(ass), Normal)
}

func (c Compiler) compileInstruction(insI ssa.Instruction, s *frame) {
	switch ins := insI.(type) {
	case *ssa.Call:
		c.compileCall(ins, s)
	case *ssa.UnOp:
		c.compileUnaryOp(ins, s)
	case *ssa.BinOp:
		c.compileBinaryOp(ins, s)
	case *ssa.Return:
		c.e.write(c.e.returnIns(ins.Results), Normal)
	case *ssa.Store:
		c.e.write(c.e.storeIns(ins.Addr, ins.Val), Normal)
	case *ssa.Extract:
		c.e.write(c.e.extractionIns(ins.Tuple, ins.Index, s), Normal)
	case *ssa.If:
		c.e.write(c.e.ifIns(ins), BlockClose)
	case *ssa.Jump:
		c.e.write(c.e.jumpIns(ins), BlockClose)
	case *ssa.Phi:
		c.e.writePhiIns(ins, s.LoadVar())
	default:
		//fmt.Printf("Unhandled {%T, %v}\n", insI, insI.String())
		return
	}
}

func (c Compiler) compileBlock(blk *ssa.BasicBlock, s *frame) {
	for _, ins := range blk.Instrs {
		c.compileInstruction(ins, s)
		fmt.Printf("{%T, %v}\n", ins, ins.String())
	}
}

func (c Compiler) compileFunctionDecl(fn *ssa.Function) {
	funcClose := c.e.writeFuncDecl(fn)
	defer funcClose()

	s := newFrame(fn.Package(), c.pkgMap)

	if len(fn.Blocks) == 1 {
		c.compileBlock(fn.Blocks[0], s)
		return
	}

	elClose := c.e.writeExecLoop()
	defer elClose()

	for i, blk := range fn.Blocks {
		caseClose := c.e.writeCase(i, i == len(fn.Blocks)-1)
		c.compileBlock(blk, s)
		ins := blk.Instrs
		if len(ins) > 0 {
			switch ins[len(blk.Instrs)-1].(type) {
			case *ssa.Jump, *ssa.If: // they have switch break already
			case *ssa.Return:
				c.e.write("", BlockClose) // no point breaking after return
			default:
				caseClose()
			}
		}
	}
}

func (c Compiler) compileGlobalDecl(gv *ssa.Global) {
	c.e.write(c.e.varDecl(gv.Name(), c.e.initialValue(gv.Type())), Normal)
}

func (c *Compiler) addPkg(pkg *ssa.Package) {
	i, name := 0, pkg.Object.Name()
	exists := true
	for exists {
		i++
		if i > 1 {
			name = pkg.Object.Name() + fmt.Sprint(i)
		}
		_, exists = c.pkgIden[name]
	}

	c.pkgIden[name] = true
	c.pkgMap[pkg] = name
}

func (c *Compiler) compilePackage(pkg *ssa.Package) {
	declClose := c.e.writePackageDecl(sfPrefix(c.pkgMap[pkg]), pkg.Members)
	defer declClose()

	for _, memI := range pkg.Members {
		switch mem := memI.(type) {
		case *ssa.Function:
			c.compileFunctionDecl(mem)
		case *ssa.Global:
			c.compileGlobalDecl(mem)
		}
	}
}

func (c *Compiler) Compile(prog *ssa.Program) {
	funcClose := c.e.writePrelude()
	defer funcClose()

	for _, pkg := range prog.AllPackages() {
		c.addPkg(pkg)
	}

	for _, pkg := range prog.AllPackages() {
		c.compilePackage(pkg)
	}
}
