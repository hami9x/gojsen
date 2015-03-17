package main

const (
	TupleVar     = "$tuple"
	VargsVar     = "$vargList"
	VargsSizeVar = "$vargSize"

	Prelude = `
var $tuple = Array(10);
var $vargList = Array(100);
var $vargSize = 0;
`
)
