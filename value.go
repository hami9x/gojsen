package main

import (
	"fmt"

	"golang.org/x/tools/go/types"
)

const (
	SafePrefix = "_"
)

func sfPrefix(name string) string {
	return SafePrefix + name
}

type valueCG interface {
	coerce(name string) string
	initialValue() string
}

type stringCG struct{}

func (c stringCG) coerce(name string) string {
	return `""+` + name
}

func (c stringCG) initialValue() string {
	return `""`
}

type intCG struct{}

func (c intCG) coerce(name string) string {
	return name + "|0"
}

func (c intCG) initialValue() string {
	return "0"
}

type boolCG struct{}

func (c boolCG) coerce(name string) string {
	return name
}

func (c boolCG) initialValue() string {
	return "false"
}

type pointerCG struct {
	typ *types.Pointer
}

func (c pointerCG) coerce(name string) string {
	return name
}

func (c pointerCG) initialValue() string {
	elemType := c.typ.Elem()
	elemInit := getCG(elemType).initialValue()
	return fmt.Sprintf("[%v]", elemInit)
}

type nopCG struct{}

func (c nopCG) coerce(name string) string {
	return name
}

func (c nopCG) initialValue() string {
	return "null"
}

func getCG(typ types.Type) valueCG {
	switch t := typ.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return intCG{}
		case types.String:
			return stringCG{}
		case types.Bool:
			return boolCG{}
		}

	case *types.Pointer:
		return pointerCG{t}
	}

	println(fmt.Sprintf("Unhandled type %v", typ.String()))
	return nopCG{}
}
