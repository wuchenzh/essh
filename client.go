package main

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
)

func sshClient(username, password, addr string, windowSize <-chan Window, userChan io.ReadWriter) error {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// Connect to ssh server
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Print("unable to connect: ", err)
		return err
	}

	// Create a session
	session, err := conn.NewSession()
	defer conn.Close()
	defer session.Close()
	if err != nil {
		log.Print("unable to create session: ", err)
		return err
	}

	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	// Request pseudo terminal
	//fmt.Printf("terminal %d %d\n", windowSize.width, windowSize.height)
	if err := session.RequestPty("xterm", 42, 169, modes); err != nil {
		return err
	}
	//监测窗口改变
	go func() {
		for {
			select {
			case w := <-windowSize:
				_ = session.WindowChange(w.height, w.width)
			}
		}
	}()
	sshIn, _ := session.StdinPipe()
	sshOut, _ := session.StdoutPipe()
	sshErr, _ := session.StderrPipe()
	go io.Copy(sshIn, userChan)
	go io.Copy(userChan, sshOut)
	go io.Copy(userChan, sshErr)
	session.Shell()
	session.Wait()
	return nil
}
