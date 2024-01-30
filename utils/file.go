package utils

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func ExistsFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return !fi.IsDir()
}

func GetGOPATH() string {
	goPath := os.Getenv("GOPATH")
	// If there are many path in GOPATH, pick up the first one.
	if GoPath := strings.TrimSpace(goPath); GoPath != "" {
		return GoPath
	}
	// GOPATH not set through environment variables, try to get one by executing "go env GOPATH"
	output, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	goPath = strings.TrimSpace(string(output))
	if len(goPath) == 0 {
		buildContext := build.Default
		goPath = buildContext.GOPATH
	}

	if len(goPath) == 0 {
		panic("GOPATH not found")
	}
	return goPath
}

func SearchGoMod(cwd string) (string, string, bool) {
	for {
		path := filepath.Join(cwd, "go.mod")
		data, err := ioutil.ReadFile(path)
		if err == nil {
			re := regexp.MustCompile(`^\s*module\s+(\S+)\s*`)
			for _, line := range strings.Split(string(data), "\n") {
				m := re.FindStringSubmatch(line)
				if m != nil {
					return m[1], cwd, true
				}
			}
			panic(fmt.Sprintf("module name don't exist in file %s", path))
		}

		if !os.IsNotExist(err) {
			return "", "", false
		}
		parentCwd := filepath.Dir(cwd)
		if parentCwd == cwd {
			break
		}
		cwd = parentCwd
	}
	return "", "", false
}

// InitGoMod TODO
func InitGoMod(path, module string) error {
	if ExistsFile("go.mod") {
		return nil
	}

	cmd := exec.Command("go", "mod", "init", module)
	return cmd.Run()
}
