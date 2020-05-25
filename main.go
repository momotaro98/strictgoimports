package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
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
	fset := token.NewFileSet()
	//f, err := parser.ParseFile(fset, path, nil, parser.ParseComments) // Original
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, nil
	}

	// Actual goimport
	cutOutImportLines := func() []string {
		importLines := make([]string, 0, len(f.Imports)*2)
		var cutStarted bool
		file, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		s := bufio.NewScanner(file)
		for s.Scan() {
			text := s.Text()
			if strings.HasPrefix(text, `)`) {
				//importLines = append(importLines, text)
				return importLines
			} else if cutStarted {
				importLines = append(importLines, text)
			} else if strings.HasPrefix(text, `import (`) {
				cutStarted = true
				//importLines = append(importLines, text)
			}
		}
		panic("should not here!")
	}
	importLines := cutOutImportLines()

	// Ideal goimport
	genFileStringRemovedWhitelineInImport := func() string {
		input, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		inputLines := strings.Split(string(input), "\n")

		replacedLines := make([]string, 0, len(inputLines))
		var isInImportBlock bool
		for _, line := range inputLines {
			if isInImportBlock && line == "" {
				continue
			} else {
				replacedLines = append(replacedLines, line)
			}
			if strings.HasPrefix(line, `import (`) {
				isInImportBlock = true
			}
			if isInImportBlock && strings.HasPrefix(line, `)`) {
				isInImportBlock = false
			}
		}
		return strings.Join(replacedLines, "\n")
	}
	idealLines := genFileStringRemovedWhitelineInImport()

	// ii.
	tempFile, err := writeTempFile("", "strict", []byte(idealLines))
	if err != nil {
		panic(err)
	}
	defer os.Remove(tempFile)
	cmd := "goimports"
	// TODO: get local flag
	localPath := `""` // Should be comma separated if there're multiple local paths
	data, err := exec.Command(cmd, "-local", localPath, tempFile).CombinedOutput()
	if err != nil {
		panic(err)
	}

	cutOutImportLinesForIdeal := func(data []byte) []string {
		importLines := make([]string, 0, len(f.Imports)*2)
		var cutStarted bool
		lines := strings.Split(string(data), "\n") // High cost
		for _, line := range lines {
			if strings.HasPrefix(line, `)`) {
				//importLines = append(importLines, line)
				return importLines
			} else if cutStarted {
				importLines = append(importLines, line)
			} else if strings.HasPrefix(line, `import (`) {
				cutStarted = true
				//importLines = append(importLines, line)
			}
		}
		panic("should not here!")
	}
	idealImportLines := cutOutImportLinesForIdeal(data)

	// Compare Actual and Ideal

	matchRealAndIdeal := func(real, ideal []string) (bool, int) {
		var shorter []string
		if len(real) < len(ideal) {
			shorter = real
		} else {
			shorter = ideal
		}
		for i := range shorter {
			if real[i] != ideal[i] {
				return false, i
			}
		}
		return true, -1
	}
	isSame, ImptLineIdx := matchRealAndIdeal(importLines, idealImportLines)
	if isSame {
		return fset, nil, nil
	}
	specifyInvalidLine := func(ImptLineIdx int) (lineNum int, lineText string) {
		file, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		sc := bufio.NewScanner(file)
		var (
			curLineNum    int
			imptPos       int
			importStarted bool
		)
		for sc.Scan() {
			curLineNum++
			line := sc.Text()

			if imptPos == ImptLineIdx {
				return curLineNum, line
			}

			if importStarted {
				imptPos++
			}

			if strings.HasPrefix(line, `import (`) {
				importStarted = true
			}
		}
		panic("invalid")
	}
	lineNum, lineText := specifyInvalidLine(ImptLineIdx)
	fmt.Println(lineNum, lineText)

	return fset, nil, nil // TODO
}

// writeTempFile is from official x/tools/cmd/goimports source code
func writeTempFile(dir, prefix string, data []byte) (string, error) {
	file, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}
