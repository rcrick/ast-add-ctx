package main

import (
	"bytes"
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
	"go/token"
)

func main() {
	srcCode := `
package test



// Added new context.Context parameter for downstream call
func changedFn(ctx context.Context) {
	fmt.Println("Do some important work...")
	// Now also make a downstream call
	makeDownstreamRequest(ctx, "Some important data!") 
}

// TODO: func needsctx1(ctx context.Context, n int)
func needsctx1(n int) {
	if true {
		// TODO: changedFn(ctx)
		changedFn()
	}
}

// TODO: func needsctx2(ctx context.Context) bool
func needsctx2() bool {
	for index := 0; index < 3; index++ {
		needsctx1(ctx, 1)
	}
	return true
}

// TODO: func needsctx3(ctx context.Context)
func needsctx3() {
	if needsctx2(ctx) {
		changedFn(ctx)
	}
}

type SS struct{}

// TODO: func (rec *SS) save(ctx context.Context, s string, n int) 
func (rec *SS) save(s string, n int) {
	// TODO: needsctx1(ctx, 2)
	needsctx1(2)
}
`
	file, err := decorator.Parse(srcCode)
	if err != nil {
		panic(err)
	}
	addImport(file)
	applyFunc := func(c *dstutil.Cursor) bool {
		node := c.Node()
		switch n := node.(type) {
		case *dst.FuncDecl:
			addCtxParam(n)
		case *dst.CallExpr:
			addCtxArg(n)
		case *dst.GenDecl:
			//addImport(n)
		}

		return true
	}
	dstutil.Apply(file, applyFunc, nil)
	fmt.Println(FormatNode(*file))
}

func addCtxArg(node *dst.CallExpr) {
	if node.Args == nil {
		node.Args = []dst.Expr{&dst.Ident{Name: "ctx"}}
	} else {
		existCtxArg := false
		for _, arg := range node.Args {
			switch n := arg.(type) {
			case *dst.Ident:
				if n.Name == "ctx" {
					existCtxArg = true
					break
				}

			}
		}
		if !existCtxArg {
			node.Args = append([]dst.Expr{&dst.Ident{Name: "ctx"}}, node.Args...)
		}
	}
}

func addCtxParam(node *dst.FuncDecl) {
	existCtxP := false
	for _, param := range node.Type.Params.List {
		t, ok := param.Type.(*dst.SelectorExpr)
		if !ok {
			continue
		}
		if t.X.(*dst.Ident).Name == "context" && t.Sel.Name == "Context" {
			existCtxP = true
			break
		}
	}
	if !existCtxP {
		newCtxParam := &dst.Field{
			Names: []*dst.Ident{&dst.Ident{Name: "ctx"}},
			Type: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "context"},
				Sel: &dst.Ident{Name: "Context"},
			},
		}
		node.Type.Params.List = append([]*dst.Field{newCtxParam}, node.Type.Params.List...)
	}
}

func addImport(file *dst.File) {
	emptyImport := true
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *dst.GenDecl:
			if node.Tok == token.IMPORT {
				emptyImport = false
				existCtx := false
				for _, spec := range node.Specs {
					switch n := spec.(type) {
					case *dst.ImportSpec:
						if n.Path.Value == "\"context\"" {
							existCtx = true
							break
						}
					}
				}
				if !existCtx {
					node.Specs = append([]dst.Spec{&dst.ImportSpec{Path: &dst.BasicLit{Value: "\"context\""}}}, node.Specs...)
				}
			}

		}
	}
	if emptyImport {
		file.Decls = append([]dst.Decl{&dst.GenDecl{Tok: token.IMPORT, Specs: []dst.Spec{&dst.ImportSpec{Path: &dst.BasicLit{Value: "\"context\""}}}}}, file.Decls...)
	}
}

func FormatNode(file dst.File) string {
	var buf bytes.Buffer
	decorator.Fprint(&buf, &file)
	return buf.String()
}
