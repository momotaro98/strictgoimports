package strictimportsort

import (
	"bufio"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func Run(filePath, localPaths string) (fileSet *token.FileSet, pos []token.Pos, correctImport string) {
	fileSet = token.NewFileSet()
	f, err := parser.ParseFile(fileSet, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, ""
	}

	// Real part
	realLines := buildImportLines(filePath, f)

	// Ideal part
	// i.
	genFileStringRemovedWhitelineInImport := func() string {
		input, err := ioutil.ReadFile(filePath)
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

		idealFileData, err := exec.Command(
			"goimports",
			"-local", localPaths, tempFile).CombinedOutput()
		if err != nil {
			panic(err)
		}

		return idealFileData
	}
	idealFileData := genIdealFileData(idealLinesString)
	// iii.
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

		return buildImportLines(tempFile, f)
	}
	idealLines := genIdealLines(idealFileData)

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

	return fileSet, []token.Pos{realLines[ImptLineIdx].pos}, idealLines.String()
}

type ImportLines []*imptLine

func (ils ImportLines) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, len(ils)*50))
	buf.WriteString("import (\n")
	for i := range ils {
		if !ils[i].isWhiteline() {
			buf.WriteString("\t")
		}
		if nm := ils[i].name; nm != "" {
			buf.WriteString(nm)
			buf.WriteString(" ")
		}
		buf.WriteString(ils[i].path)
		if c := ils[i].comment; c != "" {
			buf.WriteString(" //")
			buf.WriteString(ils[i].comment)
		}
		buf.WriteString("\n")
	}
	buf.WriteString(")")
	return buf.String()
}

type imptLine struct {
	name        string    // mp of `import ( mp "github.com/user/package" )`
	path        string    // "github.com/user/package" above
	comment     string    // comment out text at the line
	fileLineNum int       // The number of the file's line
	pos         token.Pos // File offset in the file. first char of the line
}

func (l *imptLine) isWhiteline() bool {
	return l.name == "" && l.path == ""
}

// buildImportLines returns empty length list when
// the target code has no `import` or single path `import "path"`
func buildImportLines(filePath string, f *ast.File) ImportLines {
	work := func() []*imptLine {
		lines := make([]*imptLine, 0, len(f.Imports)*2) // *2 means considering white lines in `import ()`

		var isImportedLinesStarted bool
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		var (
			curFileLineNum   int // will be imptLine.fileLineNum
			curImportSpecIdx int // will be imptLine.name, path, pos
		)
		s := bufio.NewScanner(file)
		for s.Scan() {
			curFileLineNum++

			var (
				name, path, comment string
				pos                 token.Pos
			)

			text := s.Text()
			if strings.HasPrefix(text, `)`) {
				return lines
			} else if strings.HasPrefix(text, `import (`) {
				isImportedLinesStarted = true
			} else if isImportedLinesStarted {
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

					if c := strings.SplitN(text, "//", 2); len(c) == 2 {
						comment = c[1]
					}

					curImportSpecIdx++
				}

				lines = append(lines, &imptLine{
					name:        name,
					path:        path,
					comment:     comment,
					fileLineNum: curFileLineNum,
					pos:         pos,
				})
			}
		}
		if isImportedLinesStarted {
			panic("panic!")
		}
		return lines
	}

	return work()
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
