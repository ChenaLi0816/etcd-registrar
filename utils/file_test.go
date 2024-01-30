package utils

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestExistsfile(t *testing.T) {
	fmt.Println(ExistsFile("./Makefile"))
	fmt.Println(ExistsFile("file.go"))
	fmt.Println(ExistsFile("E:/project/GoProject/src/etcd-registrar/Makefile"))
}

func TestFilePath(t *testing.T) {
	fmt.Println(filepath.Split("/ok/ok.file"))
	fmt.Println("dir:", filepath.Dir("/"))

}

func TestGetGOPATH(t *testing.T) {
	fmt.Println(GetGOPATH())
}

func TestSearchGoMod(t *testing.T) {
	//p, _ := filepath.Abs(".")
	p := "E:\\project\\GoProject\\src"
	fmt.Println(SearchGoMod(p))

}

func TestJoin(t *testing.T) {
	fmt.Println(filepath.Join(GetGOPATH(), "src"))

}
