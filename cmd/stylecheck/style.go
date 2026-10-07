package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type diagnostic struct {
	path    string
	line    int
	rule    string
	message string
}

type checker struct {
	files         *token.FileSet
	path          string
	lines         []string
	diagnostics   []diagnostic
	named         map[*ast.StructType]bool
	cobra         map[*ast.FuncLit]bool
	cobraPackages map[string]bool
}

func checkSource(path string, data []byte) []diagnostic {
	c := checker{files: token.NewFileSet(), path: path, lines: strings.Split(string(data), "\n")}
	file, err := parser.ParseFile(c.files, path, data, parser.ParseComments)
	if err != nil {
		return []diagnostic{{path: path, line: 1, rule: "syntax", message: err.Error()}}
	}

	if ast.IsGenerated(file) {
		return nil
	}

	formatted, err := format.Source(data)
	if err == nil && !bytes.Equal(data, formatted) {
		c.diagnostics = append(c.diagnostics, diagnostic{path: path, line: 1, rule: "gofmt", message: "run gofmt on this file"})
	}

	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !allowedComment(comment.Text) && !c.cgoPreamble(file, group) {
				c.report(comment.Pos(), "comments", "use names and structure instead of prose comments")
			}
		}
	}

	c.named = make(map[*ast.StructType]bool)
	c.cobra = make(map[*ast.FuncLit]bool)
	c.cobraPackages = make(map[string]bool)
	for _, imported := range file.Imports {
		if imported.Path.Value == `"github.com/spf13/cobra"` {
			name := "cobra"
			if imported.Name != nil {
				name = imported.Name.Name
			}

			c.cobraPackages[name] = true
		}
	}

	ast.Inspect(file, c.collectNamed)
	ast.Inspect(file, c.checkNode)
	return c.diagnostics
}

func (c *checker) collectNamed(node ast.Node) bool {
	switch value := node.(type) {
	case *ast.TypeSpec:
		if structure, ok := value.Type.(*ast.StructType); ok {
			c.named[structure] = true
		}
	case *ast.CompositeLit:
		selector, ok := value.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Command" {
			break
		}

		name, ok := selector.X.(*ast.Ident)
		if ok && c.cobraPackages[name.Name] {
			c.collectCobraHandlers(value)
		}
	}

	return true
}

func (c *checker) collectCobraHandlers(command *ast.CompositeLit) {
	for _, element := range command.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		name, ok := field.Key.(*ast.Ident)
		if !ok || name.Name != "RunE" {
			continue
		}

		if function, ok := field.Value.(*ast.FuncLit); ok {
			c.cobra[function] = true
		}
	}
}

func (c *checker) checkNode(node ast.Node) bool {
	switch value := node.(type) {
	case *ast.StructType:
		if !c.named[value] && len(value.Fields.List) > 0 {
			c.report(value.Pos(), "named-struct", "declare a named type for this struct")
		}
	case *ast.FuncLit:
		if !c.cobra[value] && (len(value.Body.List) > 3 || c.functionLines(value) > 5) {
			c.report(value.Pos(), "named-function", "move this multi-step function to a named function")
		}
	case *ast.BlockStmt:
		c.checkStatements(value.List)
	case *ast.CaseClause:
		c.checkStatements(value.Body)
	case *ast.CommClause:
		c.checkStatements(value.Body)
	}

	return true
}

func (c *checker) functionLines(function *ast.FuncLit) int {
	start := c.files.Position(function.Pos()).Line
	end := c.files.Position(function.End()).Line
	count := 0
	for line := start - 1; line < end; line++ {
		if strings.TrimSpace(c.lines[line]) != "" {
			count++
		}
	}

	return count
}

func allowedComment(text string) bool {
	return strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "//nolint") || strings.HasPrefix(text, "// @")
}

func (c *checker) cgoPreamble(file *ast.File, group *ast.CommentGroup) bool {
	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			continue
		}

		for _, specification := range imports.Specs {
			value := specification.(*ast.ImportSpec)
			if value.Path.Value == `"C"` && (imports.Doc == group || value.Doc == group) {
				return true
			}
		}
	}

	return false
}

func (c *checker) report(position token.Pos, rule, message string) {
	c.diagnostics = append(c.diagnostics, diagnostic{path: c.path, line: c.files.Position(position).Line, rule: rule, message: message})
}

func (c *checker) checkStatements(statements []ast.Stmt) {
	for index, statement := range statements {
		if index+1 == len(statements) {
			continue
		}

		next := statements[index+1]
		end := c.files.Position(statement.End()).Line
		start := c.files.Position(next.Pos()).Line
		switch statement.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			if !c.hasBlankLine(end, start) {
				c.report(next.Pos(), "block-spacing", "add a blank line after the preceding control-flow block")
			}
		}

		assignment, ok := statement.(*ast.AssignStmt)
		if !ok {
			continue
		}

		check, ok := next.(*ast.IfStmt)
		if !ok || check.Init != nil || !c.hasBlankLine(end, start) {
			continue
		}

		if checksAssignedError(assignment, check.Cond) {
			c.report(check.Pos(), "error-spacing", "keep the error assignment and its check together")
		}
	}
}

func (c *checker) hasBlankLine(end, start int) bool {
	for line := end; line < start-1; line++ {
		if strings.TrimSpace(c.lines[line]) == "" {
			return true
		}
	}

	return false
}

func checksAssignedError(assignment *ast.AssignStmt, condition ast.Expr) bool {
	comparison, ok := condition.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.NEQ {
		return false
	}

	name, ok := comparison.X.(*ast.Ident)
	nilValue, isNil := comparison.Y.(*ast.Ident)
	if !ok || !isNil || nilValue.Name != "nil" || (name.Name != "err" && !strings.HasSuffix(name.Name, "Err")) {
		return false
	}

	for _, expression := range assignment.Lhs {
		if identifier, ok := expression.(*ast.Ident); ok && identifier.Name == name.Name {
			return true
		}
	}

	return false
}
