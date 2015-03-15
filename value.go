package main

import (
	"golang.org/x/tools/go/types"
)

type valueCodegen struct {
	coerce func(name string) string
	value  func(name string, typ types.Type) string
}

func valNoopFunc(name string, typ types.Type) string {
	return name
}

var (
	stringCodegen = &valueCodegen{
		coerce: func(valueStr string) string {
			return `""+` + valueStr
		},
		value: valNoopFunc,
	}

	intCodegen = &valueCodegen{
		coerce: func(name string) string {
			return name + "|0"
		},
		value: valNoopFunc,
	}
)
