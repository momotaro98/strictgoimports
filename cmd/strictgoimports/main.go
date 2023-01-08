package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	wildcard "github.com/IGLOU-EU/go-wildcard"

	"github.com/momotaro98/strictgoimports"
)

const version = "1.1.2"

const (
	invalidArgumentExitCode = 3
)

var (
	sVersion        = flag.Bool("version", false, "show version")
	write           = flag.Bool("w", false, "write result to (source) file instead of stdout")
	localPaths      = flag.String("local", "", "put imports beginning with this string after 3rd-party packages; comma-separated list")
	excludes        = flag.String("exclude", "", "file names you wanna exclude; wile card is welcome; comma-separated list")
	excludeDirs     = flag.String("exclude-dir", "", "directory names you wanna exclude; wile card is welcome; comma-separated list")
	dontRecurseFlag = flag.Bool("n", false, "don't recursively check paths")
)

func main() {
	flag.Parse()

	if *sVersion {
		fmt.Println("version:", version)
		os.Exit(0)
	}

	if len(flag.Args()) == 0 {
		fmt.Println("missing argument: filepath")
		os.Exit(invalidArgumentExitCode)
	}

	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("Error: %+v\n", err)
		}
	}()

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
	filepath.Walk(rootPath, func(filePath string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error during filesystem walk: %v\n", err)
			return nil
		}
		if fi.IsDir() {
			if *excludeDirs != "" {
				patternDirs := strings.Split(*excludeDirs, ",")
				p := strings.Split(filePath, "/")
				dirName := p[len(p)-1]
				for _, pattern := range patternDirs {
					if wildcard.MatchSimple(pattern, dirName) {
						return filepath.SkipDir
					}
				}
			}
			if filePath != rootPath && (*dontRecurseFlag ||
				filepath.Base(filePath) == "testdata" ||
				filepath.Base(filePath) == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if *excludes != "" {
			patternFiles := strings.Split(*excludes, ",")
			p := strings.Split(filePath, "/")
			fileName := p[len(p)-1]
			for _, pattern := range patternFiles {
				if wildcard.MatchSimple(pattern, fileName) {
					return nil
				}
			}
		}
		if !strings.HasSuffix(filePath, ".go") {
			return nil
		}

		fset, poses, correctImport, fixed := strictgoimports.Run(filePath, *localPaths)

		if *write {
			if len(poses) > 0 {
				// From goimports source code > On Windows, we need to re-set the permissions from the file. See golang/go#38225.
				var perms os.FileMode
				if fi, err := os.Stat(filePath); err == nil {
					perms = fi.Mode() & os.ModePerm
				}
				err = ioutil.WriteFile(filePath, fixed, perms)
				if err != nil {
					return err
				}
			}
		} else {
			for _, pos := range poses {
				fmt.Printf("%s: import not sorted correctly. should be replace to\n%s\n",
					fset.Position(pos), correctImport,
				)
				lintFailed = true
			}
		}

		return nil
	})
	return lintFailed
}
