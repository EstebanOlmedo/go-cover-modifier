package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	edit "github.com/EstebanOlmedo/go-cover-modifier/internal"
)

func main() {
	os.Exit(runMain())
}

func runMain() (exitCode int) {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(args []string) error {
	for _, filename := range args {
		_, err := process(filename)
		if err != nil {
			return fmt.Errorf("while processing %s: %v", filename, err)
		}
	}
	return nil
}

type eraser struct {
	markedNodes map[ast.Node]bool
	fset        *token.FileSet
	buffer      *edit.Buffer
	root        *ast.File
}

func process(filename string) ([]byte, error) {
	fset := token.NewFileSet()
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(fset, filename, content, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	e := eraser{
		markedNodes: make(map[ast.Node]bool),
		fset:        fset,
		buffer:      edit.NewBuffer(content),
	}

	ast.Walk(e, f)
	return e.buffer.Bytes(), nil
}

func (e *eraser) offset(pos token.Pos) int {
	return e.fset.Position(pos).Offset
}

func (e *eraser) processBlock(blck *ast.BlockStmt, isFuncBlock bool) {
	var erase []int
	processedBlck := false
	for i, stmt := range blck.List {
		switch m := stmt.(type) {
		case *ast.IfStmt:
			ast.Walk(e, m)
			processedBlck = false
		case *ast.SwitchStmt:
			ast.Walk(e, m)
			processedBlck = false
		default:
			if !processedBlck {
				erase = append(erase, i)
				processedBlck = true
			}
		}
	}
	// If the length of the slice is 0, then the block is empty and
	// there's nothing to erase. If the length of the slice is 1, that
	// means there's at least one statement, but no control changing
	// sequences, then the covered statement can't be erased
	if len(erase) < 2 {
		return
	}
	// Erase all statement, but the last one
	erase = erase[:len(erase)-1]

	// If this block is the body of a function, then the first 3
	// statements will be metadata written by go tool cover, use this to
	// skip them.
	if isFuncBlock {
		erase[0] += 3
	}
	for _, idx := range erase {
		if _, ok := blck.List[idx].(*ast.AssignStmt); ok {
			e.buffer.Delete(
				e.offset(blck.List[idx].Pos()),
				e.offset(blck.List[idx].End()+1), // +1 to erase ';'
			)
		}
	}
}

func (e eraser) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *ast.FuncDecl:
		e.processBlock(v.Body, true /* isFuncBlock */)
		return nil
	case *ast.IfStmt:
		e.processBlock(v.Body, false /* isFuncBlock */)
		ast.Walk(e, v.Else)
		return nil
	case *ast.SwitchStmt:
		for _, stmt := range v.Body.List {
			// Generate a virtual block around the body of
			// the case statement, and then traverse it
			if cs, ok := stmt.(*ast.CaseClause); ok {
				vb := &ast.BlockStmt{
					List: cs.Body,
				}
				e.processBlock(vb, false /* isFuncBlock */)
			}
		}
	case *ast.BlockStmt:
		e.processBlock(v, false /* isFuncBlock */)
		return nil
	}
	return e
}
