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
		fset, poses, correctImport := Run(path)
		for _, pos := range poses {
			fmt.Printf("%s: import not sorted correctly. should be replace to %s\n", fset.Position(pos), correctImport)
			lintFailed = true
		}
		return nil
	})
	return lintFailed
}

type ImportLines []*imptLine

func buildImportLines(filePath string, f *ast.File) ImportLines {
	work := func() []*imptLine {
		lines := make([]*imptLine, 0, len(f.Imports)*2) // *2 means considering white lines in `import ()`

		var cutStarted bool
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		var (
			curNum           int // will be imptLine.num
			curFileLineNum   int // will be imptLine.fileLineNum
			curImportSpecIdx int // will be imptLine.name, path, pos
		)
		s := bufio.NewScanner(file)
		for s.Scan() {
			curFileLineNum++

			var (
				name, path string
				pos        token.Pos
			)

			text := s.Text()
			if strings.HasPrefix(text, `)`) {
				return lines
			} else if strings.HasPrefix(text, `import (`) {
				cutStarted = true
			} else if cutStarted {
				curNum++

				if text == "" { // white line in `import ()`
					// The following code considers only case where there's one white line, which means
					// the target code must be formatted by gofmt in advance.
					pos = f.Imports[curImportSpecIdx].Pos() - 2 // take "\t" and "\n" from head of next ImportSpec
				} else {
					if nm := f.Imports[curImportSpecIdx].Name; nm != nil {
						name = f.Imports[curImportSpecIdx].Name.Name
					}
					path = f.Imports[curImportSpecIdx].Path.Value
					pos = f.Imports[curImportSpecIdx].Pos()

					curImportSpecIdx++
				}

				lines = append(lines, &imptLine{
					num:         curNum,
					name:        name,
					path:        path,
					fileLineNum: curFileLineNum,
					pos:         pos,
				})
			}
		}
		panic("panic!")
	}

	return work()
}

type imptLine struct {
	num  int    // import () の中で何番目の行数目か。1始まりである。
	name string // import ( mypackage "github.com/user/package" ) の mypackage
	path string // 上記の "github.com/user/package"
	// comment string // comment out text at the line // if it's needed, let's implement TODO: Needed for telling correctImport string with comment
	fileLineNum int
	pos         token.Pos
}

//func (l imptLine) isWhiteline() bool {
//	return l.name != "" && l.path != ""
//}

func Run(path string) (fileSet *token.FileSet, pos []token.Pos, correctImport string) {
	fileSet = token.NewFileSet()
	f, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, ""
	}

	// Build import lines for real
	realLines := buildImportLines(path, f)

	// Actual goimport
	//cutOutImportLines := func() []string {
	//	importLines := make([]string, 0, len(f.Imports)*2)
	//	var cutStarted bool
	//	file, err := os.Open(path)
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer file.Close()
	//	s := bufio.NewScanner(file)
	//	for s.Scan() {
	//		text := s.Text()
	//		if strings.HasPrefix(text, `)`) {
	//			return importLines
	//		} else if cutStarted {
	//			importLines = append(importLines, text)
	//		} else if strings.HasPrefix(text, `import (`) {
	//			cutStarted = true
	//		}
	//	}
	//	panic("should not here!")
	//}
	//importLines := cutOutImportLines()

	// Ideal goimport
	// i.
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
	idealLinesString := genFileStringRemovedWhitelineInImport()

	// ii.
	genIdealFileData := func(src string) []byte {
		tempFile, err := writeTempFile("", "strict", []byte(src))
		if err != nil {
			panic(err)
		}
		defer os.Remove(tempFile)

		cmd := "goimports"
		// TODO: get local flag
		localPath := `""` // Should be comma separated if there're multiple local paths

		idealFileData, err := exec.Command(cmd, "-local", localPath, tempFile).CombinedOutput()
		if err != nil {
			panic(err)
		}

		return idealFileData
	}
	idealFileData := genIdealFileData(idealLinesString)

	// Build import lines for real
	genIdealLines := func(src []byte) ImportLines {
		tempFile, err := writeTempFile("", "strict", src)
		if err != nil {
			panic(err)
		}
		defer os.Remove(tempFile)

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, tempFile, nil, parser.ImportsOnly)
		if err != nil {
			panic(err)
		}

		// Build import lines for real
		return buildImportLines(tempFile, f)
	}
	idealLines := genIdealLines(idealFileData)

	//cutOutImportLinesForIdeal := func(data []byte) []string {
	//	importLines := make([]string, 0, len(f.Imports)*2)
	//	var cutStarted bool
	//	lines := strings.Split(string(data), "\n") // High cost
	//	for _, line := range lines {
	//		if strings.HasPrefix(line, `)`) {
	//			return importLines
	//		} else if cutStarted {
	//			importLines = append(importLines, line)
	//		} else if strings.HasPrefix(line, `import (`) {
	//			cutStarted = true
	//		}
	//	}
	//	panic("should not here!")
	//}
	//idealImportLines := cutOutImportLinesForIdeal(idealFileData)

	// Compare Actual and Ideal

	matchRealAndIdeal := func(real, ideal ImportLines) (bool, int) {
		var shorter ImportLines
		if len(real) < len(ideal) {
			shorter = real
		} else {
			shorter = ideal
		}
		for i := range shorter {
			if real[i].fileLineNum != ideal[i].fileLineNum || real[i].path != ideal[i].path {
				return false, i
			}
		}
		return true, -1
	}
	isSame, ImptLineIdx := matchRealAndIdeal(realLines, idealLines)
	if isSame {
		return fileSet, nil, ""
	}

	//specifyInvalidLine := func(ImptLineIdx int) (lineNum int, lineText string) {
	//	// TODO: Use my builder
	//	file, err := os.Open(path)
	//	if err != nil {
	//		panic(err)
	//	}
	//	defer file.Close()
	//	sc := bufio.NewScanner(file)
	//	var (
	//		curLineNum    int
	//		imptPos       int
	//		importStarted bool
	//	)
	//	for sc.Scan() {
	//		curLineNum++
	//		line := sc.Text()
	//
	//		if imptPos == ImptLineIdx {
	//			return curLineNum, line
	//		}
	//
	//		if importStarted {
	//			imptPos++
	//		}
	//
	//		if strings.HasPrefix(line, `import (`) {
	//			importStarted = true
	//		}
	//	}
	//	panic("invalid")
	//}
	targetLine := realLines[ImptLineIdx]

	return fileSet, []token.Pos{targetLine.pos}, "" // TODO: assemble correctImport string with ImportLines.String()
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
