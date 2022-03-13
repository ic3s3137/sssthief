package main

import (
	"github.com/google/goterm/term"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"
)

type ptyReader struct {
	pty      *term.PTY
	ReadLock chan struct{}
	rules    [][]*regexp.Regexp
	isClose  bool
}

//func (p *ptyReader)Reader(pty *term.PTY){
//	return p
//}
func (p *ptyReader) Reader(pty *term.PTY) {
	p.pty = pty
	p.ReadLock = make(chan struct{})
}
func (p *ptyReader) AddRule(rule ...*regexp.Regexp) {
	p.rules = append(p.rules, rule)
}

//func (p *ptyreadineByte
func (p *ptyReader) Readline() string {
	var lineByte []byte
	for {
		v, err := p.pty.ReadByte()
		if err != nil {
			//p.Done()
			//p.Close()
			p.isClose = true
			return ""
		}
		lineByte = append(lineByte, v)
		if v == 10 || v == 13 {
			return string(lineByte)
		}
		for _, rl := range p.rules {
			var notMatch bool
			for _, r := range rl {
				if !r.MatchString(string(lineByte)) {
					notMatch = true
					break
				}
			}
			if !notMatch {
				return string(lineByte)
			}
		}
	}
}
func (p *ptyReader) IsClose() bool {
	return p.isClose
}
func (p *ptyReader) Close() {
	p.pty.Close()
	<-p.ReadLock
}

func ExecIfErr(err error) {
	if err != nil {
		ExecScript(cmd)
		os.Exit(1)
	}
}
func CheckErr(err error) {
	if err != nil {
		os.Exit(1)
	}
}
func WritePassword(content string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	file.WriteString(timeStr + " " + program + " " + content + "\n")
}
func GetPassword() string {
	bytePassword, _ := terminal.ReadPassword(fd)

	return string(bytePassword)
}
func ExecScript(content string) {
	ioutil.WriteFile(scriptName, []byte(content), 777)
	c := exec.Command("/bin/bash", scriptName)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	c.Run()
	os.Remove(scriptName)
}
