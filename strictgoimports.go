package strictgoimports

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/tools/imports"
)

type Err struct {
	Pos     token.Pos
	Message string
	FileSet *token.FileSet
}

func (e *Err) Error() string {
	return fmt.Sprintf("Message: %s. Position: %s", e.Message, e.FileSet.Position(e.Pos).String())
}

func Run(filePath, localPaths string) (fileSet *token.FileSet, pos []token.Pos, correctImport string, fixed []byte) {
	fileSet = token.NewFileSet()
	f, err := parser.ParseFile(fileSet, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, nil, "", nil
	}

	realLines := buildImportLines(filePath, f, fileSet)
	idealLines, idealSrc := buildIdeal(filePath, localPaths, fileSet)

	// Compare Real import lines and Ideal import lines
	if isSame, ImptLineIdx := func(real, ideal ImportLines) (bool, int) {
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
	}(realLines, idealLines); !isSame {
		return fileSet, []token.Pos{realLines[ImptLineIdx].pos}, idealLines.String(), idealSrc
	}

	return fileSet, nil, "", nil
}

type ImportLines []*imptLine

func (ils ImportLines) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, len(ils)*50))
	buf.WriteString("import (\n")
	for i := range ils {
		if !ils[i].isWhiteLine() {
			buf.WriteString("\t")
		}
		if nm := ils[i].name; nm != "" {
			buf.WriteString(nm)
			buf.WriteString(" ")
		}
		buf.WriteString(ils[i].path)
		if c := ils[i].comment; c != "" {
			if !ils[i].isCommentLine() {
				buf.WriteString(" ")
			}
			buf.WriteString("//")
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

func (l *imptLine) isWhiteLine() bool {
	return l.name == "" && l.path == "" && l.comment == ""
}

func (l *imptLine) isCommentLine() bool {
	return l.name == "" && l.path == "" && l.comment != ""
}

// buildImportLines returns empty length list when
// the target code has no `import` or single path `import "path"`
func buildImportLines(filePath string, f *ast.File, fileSet *token.FileSet) ImportLines {
	const starCommentErrMsg = "there's star comment (/* */) in import lines"

	raisePanicWithPositionAndMessage := func(pos token.Pos, message string) {
		err := &Err{
			Pos:     pos,
			Message: message,
			FileSet: fileSet,
		}
		panic(err)
	}

	findImportPosition := func() token.Pos {
		const (
			cgoValue = `"C"`
		)
		var (
			pos                     = token.Pos(0)
			alreadyGot1stImportDecl bool
		)
		for i, d := range f.Decls {
			switch v := d.(type) {
			case *ast.GenDecl:
				if v.Tok == token.IMPORT {
					if alreadyGot1stImportDecl {
						if f.Imports[len(f.Imports)-1].Path.Value != cgoValue {
							raisePanicWithPositionAndMessage(v.TokPos, "there's more than one `import` declaration")
						}
					}
					if f.Imports[i].Path.Value != cgoValue {
						pos = v.TokPos
						alreadyGot1stImportDecl = true
					}
				}
			}
		}
		if int(pos) == 0 {
			panic("no `import` declaration")
		}
		return pos
	}
	importPosition := findImportPosition() // `import` Position below `import "C"`

	work := func() []*imptLine {
		lines := make([]*imptLine, 0, len(f.Imports)*2) // *2 means considering white lines in `import ()`

		var isImportedLinesStarted bool
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		var (
			curFileLineNum    int // will be imptLine.fileLineNum
			nextImportSpecIdx int // will be imptLine.name, path, pos
			nextLinePosition  token.Pos
		)
		s := bufio.NewScanner(file)
		for s.Scan() {
			curFileLineNum++

			var (
				name, path, comment string
				pos                 token.Pos
			)

			text := s.Text()

			if strings.Contains(text, `*/import (`) {
				pos := importPosition
				raisePanicWithPositionAndMessage(pos, starCommentErrMsg)
			}
			if isImportedLinesStarted {
				if strings.Contains(text, "/*") || strings.Contains(text, `*/`) {
					raisePanicWithPositionAndMessage(nextLinePosition, starCommentErrMsg)
				}
			}

			// skip for CGO
			if strings.HasPrefix(text, `import "C"`) {
				nextImportSpecIdx++
				continue
			}

			if isImportedLinesStarted && strings.HasPrefix(text, `)`) {
				return lines
			} else if strings.HasPrefix(text, `import (`) {
				isImportedLinesStarted = true

				impPos := importPosition
				nextLinePosition = token.Pos(1 + int(impPos) + len(text) + 1)
			} else if isImportedLinesStarted {
				if text == "" || strings.HasPrefix(text, "\t//") {
					pos = nextLinePosition
				} else {
					if nm := f.Imports[nextImportSpecIdx].Name; nm != nil {
						name = f.Imports[nextImportSpecIdx].Name.Name
					}
					path = f.Imports[nextImportSpecIdx].Path.Value
					pos = f.Imports[nextImportSpecIdx].Pos()

					nextImportSpecIdx++
				}

				if c := strings.SplitN(text, "//", 2); len(c) == 2 {
					comment = c[1]
				}

				nextLinePosition = token.Pos(int(nextLinePosition) + len(text) + 1)

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

func buildIdeal(filePath, localPaths string, fileSet *token.FileSet) (ImportLines, []byte) {
	genFileStringRemovedWhiteLineAndSortedCommentLineInImport := func() string {
		input, err := ioutil.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		fileAllLines := strings.Split(string(input), "\n")

		replacedFileAllLines := make([]string, 0, len(fileAllLines))
		importCommentLines := make([]string, 0, len(fileAllLines)/10)
		importPathLines := make([]string, 0, len(fileAllLines)/10)

		var isInImportBlock bool
		for _, line := range fileAllLines {
			if isInImportBlock && strings.HasPrefix(line, `)`) {
				isInImportBlock = false

				replacedFileAllLines = append(replacedFileAllLines, importCommentLines...)
				replacedFileAllLines = append(replacedFileAllLines, importPathLines...)
			}

			if isInImportBlock {
				if line == "" {
					continue // remove white line
				} else if isInImportBlock && strings.HasPrefix(line, "\t//") {
					importCommentLines = append(importCommentLines, line)
				} else {
					importPathLines = append(importPathLines, line)
				}
			} else {
				replacedFileAllLines = append(replacedFileAllLines, line)
			}

			if strings.HasPrefix(line, `import (`) {
				isInImportBlock = true
			}
		}

		return strings.Join(replacedFileAllLines, "\n")
	}
	idealLinesString := genFileStringRemovedWhiteLineAndSortedCommentLineInImport()

	genIdealFileData := func(srcStr string) []byte {
		src := []byte(srcStr)

		tempFile, err := writeTempFile("", "strict", src)
		if err != nil {
			panic(err)
		}
		defer os.Remove(tempFile)

		opt := &imports.Options{
			TabWidth:  8,
			TabIndent: true,
			Comments:  true,
			Fragment:  true,
		}
		imports.LocalPrefix = localPaths
		imports.Debug = false
		idealFileData, err := imports.Process(tempFile, src, opt)
		if err != nil {
			panic(err)
		}

		return idealFileData
	}
	idealFileData := genIdealFileData(idealLinesString)

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

		return buildImportLines(tempFile, f, fileSet)
	}
	return genIdealLines(idealFileData), idealFileData
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
