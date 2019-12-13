package main

import (
	"golang.org/x/crypto/ssh"
	"io"
)

type serverChannel struct {
	channel ssh.Channel
	src io.Reader
	dst io.Writer
	window chan Window
}

func (s serverChannel) Read(p []byte) (n int, err error) {
	return s.src.Read(p)
}

func (s serverChannel) Write(p []byte) (n int, err error) {
	return s.dst.Write(p)
}
