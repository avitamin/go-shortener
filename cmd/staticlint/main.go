package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	analyzers := []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		waitgroup.Analyzer,
		bodyclose.Analyzer,
		nilerr.Analyzer,
		mainExitAnalyzer,
	}

	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}
	for _, a := range simple.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	multichecker.Main(analyzers...)
}

var mainExitAnalyzer = &analysis.Analyzer{
	Name: "mainexit",
	Doc:  "forbids direct os.Exit calls in main.main",
	Run:  runMainExit,
}

func runMainExit(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		fileName := pass.Fset.Position(file.Package).Filename
		if strings.Contains(fileName, "/.cache/go-build/") {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "Exit" {
					return true
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				obj := pass.TypesInfo.Uses[ident]
				pkgName, ok := obj.(*types.PkgName)
				if !ok || pkgName.Imported() == nil || pkgName.Imported().Path() != "os" {
					return true
				}

				pass.Reportf(call.Pos(), "direct os.Exit call in main.main is forbidden")
				return true
			})
		}
	}

	return nil, nil
}
