package strictimportsort

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	const (
		localPath = "github.com/momotaro98"
		data01    = "testdata/src/testdata01.go"
		data02    = "testdata/src/testdata02.go"
	)

	passCases := []struct {
		name      string
		filePath  string
		localPath string
	}{
		{
			name:      "pass case without local path",
			filePath:  data01,
			localPath: "",
		},
		{
			name:      "pass case with local path",
			filePath:  data02,
			localPath: localPath,
		},
	}
	for _, tt := range passCases {
		t.Run(tt.name, func(t *testing.T) {
			_, poses, actCorrect := Run(tt.filePath, tt.localPath)
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
			filePath:           data01,
			localPath:          localPath,
			expectedLineColumn: "7:2",
			expectedCorrect: `import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"

	_ "github.com/momotaro98/strictimportsort"
)`,
		},
		{
			name:               "fail without local path",
			filePath:           data02,
			localPath:          "",
			expectedLineColumn: "7:1",
			expectedCorrect: `import (
	_ "fmt"

	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictimportsort"
)`,
		},
	}

	for _, tt := range failCases {
		t.Run(tt.name, func(t *testing.T) {
			fset, poses, actCorrect := Run(tt.filePath, tt.localPath)
			for _, pos := range poses {
				if p := fset.Position(pos).String(); p != fmt.Sprintf("%s:%s", tt.filePath, tt.expectedLineColumn) {
					t.Error("unexpected:", p)
				}
				if actCorrect != tt.expectedCorrect {
					t.Error("unexpected:", actCorrect)
				}
			}
		})
	}

}
