package strictgoimports

import (
	"fmt"
	"testing"
)

const (
	localPath = "github.com/momotaro98"

	// file path as test input data
	standard01Path       = "testdata/src/standard01.go"
	standard02Path       = "testdata/src/standard02.go"
	innerCommentLinePath = "testdata/src/inner_comment_line.go"
	starCommentPath      = "testdata/src/star_comment.go"
	singleImportPath     = "testdata/src/single_import.go"
	doubleImportPath     = "testdata/src/double_import.go"
	cgo01Path            = "testdata/src/cgo01.go"
	cgo02Path            = "testdata/src/cgo02.go"
)

func TestStandard(t *testing.T) {
	passCases := []struct {
		name      string
		filePath  string
		localPath string
	}{
		{
			name:      "pass case without local path",
			filePath:  standard01Path,
			localPath: "",
		},
		{
			name:      "pass case with local path",
			filePath:  standard02Path,
			localPath: localPath,
		},
	}
	for _, tt := range passCases {
		t.Run(tt.name, func(t *testing.T) {
			_, poses, actCorrect, _ := Run(tt.filePath, tt.localPath)
			if len(poses) != 0 {
				t.Error("unexpected:", poses)
			}
			if actCorrect != "" {
				t.Error("unexpected:", actCorrect)
			}
		})
	}

	failCases := []struct {
		name               string
		filePath           string
		localPath          string
		expectedLineColumn string
		expectedCorrect    string
	}{
		{
			name:               "fail with local path",
			filePath:           standard01Path,
			localPath:          localPath,
			expectedLineColumn: "7:2",
			expectedCorrect: `import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"

	_ "github.com/momotaro98/strictgoimports"
)`,
		},
		{
			name:               "fail without local path",
			filePath:           standard02Path,
			localPath:          "",
			expectedLineColumn: "8:1",
			expectedCorrect: `import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictgoimports"
)`,
		},
	}

	for _, tt := range failCases {
		t.Run(tt.name, func(t *testing.T) {
			fset, poses, actCorrect, _ := Run(tt.filePath, tt.localPath)
			for _, pos := range poses {
				if p := fset.Position(pos).String(); p != fmt.Sprintf("%s:%s", tt.filePath, tt.expectedLineColumn) {
					t.Error("unexpected:", p)
				}
			}
			if actCorrect != tt.expectedCorrect {
				t.Error("unexpected:", actCorrect)
			}
		})
	}
}

func TestForInnerCommentLine(t *testing.T) {
	tt := struct {
		filePath           string
		localPath          string
		expectedLineColumn string
		expectedCorrect    string
	}{
		innerCommentLinePath,
		"",
		"4:2",
		`import (
	// comment
	// my favorite comment
	_ "fmt"
	_ "strings"

	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictgoimports"
)`,
	}

	fset, poses, actCorrect, _ := Run(tt.filePath, tt.localPath)
	for _, pos := range poses {
		if p := fset.Position(pos).String(); p != fmt.Sprintf("%s:%s", tt.filePath, tt.expectedLineColumn) {
			t.Error("unexpected:", p)
		}
	}
	if actCorrect != tt.expectedCorrect {
		t.Error("unexpected:", actCorrect)
	}
}

func TestForStarCommentLine(t *testing.T) {
	tt := struct {
		filePath  string
		localPath string
	}{
		starCommentPath,
		"",
	}

	defer func() {
		err := recover()
		e, ok := err.(*Err)
		if !ok {
			t.Errorf("got %v\nwant %v", e, "*Err type value")
		}
		t.Log("err:", e)
	}()

	Run(tt.filePath, tt.localPath)
}

func TestForSingleImport(t *testing.T) {
	_, _, diff, _ := Run(singleImportPath, "")
	if diff != "" {
		t.Errorf("got: %+v, expected: empty string", diff)
	}
}

func TestForDoubleImport(t *testing.T) {
	tt := struct {
		filePath  string
		localPath string
	}{
		doubleImportPath,
		"",
	}

	defer func() {
		err := recover()
		e, ok := err.(*Err)
		if !ok {
			t.Errorf("got %v\nwant %v", e, "*Err type value")
		}
		t.Log("err:", e)
	}()

	Run(tt.filePath, tt.localPath)
}

func TestForCGO(t *testing.T) {
	cases := []struct {
		name               string
		filePath           string
		localPath          string
		expectedLineColumn string
		expectedCorrect    string
	}{
		{
			"CGO is Upper",
			cgo01Path,
			"",
			"9:2",
			`import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictgoimports"
)`,
		},
		{
			"CGO is Lower",
			cgo02Path,
			"",
			"4:2",
			`import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictgoimports"
)`,
		},
	}

	for _, tt := range cases {
		fset, poses, actCorrect, _ := Run(tt.filePath, tt.localPath)
		for _, pos := range poses {
			if p := fset.Position(pos).String(); p != fmt.Sprintf("%s:%s", tt.filePath, tt.expectedLineColumn) {
				t.Error("unexpected:", p)
			}
		}
		if actCorrect != tt.expectedCorrect {
			t.Error("unexpected:", actCorrect)
		}
	}
}
