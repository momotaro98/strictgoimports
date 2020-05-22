package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const (
	invalidArgumentExitCode = 3
)

var (
	dontRecurseFlag = flag.Bool("n", false, "don't recursively check paths")
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("missing argument: filepath")
		os.Exit(invalidArgumentExitCode)
	}

	var lintFailed bool
	for _, path := range flag.Args() {
		rootPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("Error finding absolute path: %s", err)
			os.Exit(invalidArgumentExitCode)
		}
		if walk(rootPath) {
			lintFailed = true
		}
	}

	if lintFailed {
		os.Exit(1)
	}
}

func walk(rootPath string) bool {
	var lintFailed bool
	filepath.Walk(rootPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error during filesystem walk: %v\n", err)
			return nil
		}
		if fi.IsDir() {
			if path != rootPath && (*dontRecurseFlag ||
				filepath.Base(path) == "testdata" ||
				filepath.Base(path) == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset, _, ineff := RunLint(path)
		for _, id := range ineff {
			fmt.Printf("%s\n", fset.Position(id.Pos()))
			lintFailed = true
		}
		return nil
	})
	return lintFailed
}

func RunLint(path string) (*token.FileSet, []*ast.CommentGroup, []*ast.Ident) {
	return nil, nil, nil // TODO: Implement core program
}
