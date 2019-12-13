package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func readHosts() [][]string {
	var hosts [][]string
	f, _ := os.Open("/Users/coolcat/etc/essh/host")
	r := bufio.NewReader(f)
	for {
		h, _, err := r.ReadLine()
		if err != nil {
			break
		}
		hInfo := strings.Split(string(h), " ")

		if len(hInfo) == 4 {
			hosts = append(hosts, hInfo)
		}
	}
	return hosts
}

func sshServer() {
	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		// Remove to disable password auth.
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			if c.User() == "coolcat" && string(pass) == "1" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}
	privateBytes, err := ioutil.ReadFile("/Users/coolcat/.ssh/id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)
	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2022")
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	for {
		nConn, err := listener.Accept()
		go func() {
			if err != nil {
				log.Fatal("failed to accept incoming connection: ", err)
			}

			// Before use, a handshake must be performed on the incoming
			// net.Conn.
			_, chans, reqs, err := ssh.NewServerConn(nConn, config)
			if err != nil {
				log.Fatal("failed to handshake: ", err)
			}
			//log.Printf("logged in with key %s", conn.Permissions.Extensions["pubkey-fp"])

			// The incoming Request channel must be serviced.
			go ssh.DiscardRequests(reqs)

			// Service the incoming Channel channel.
			for newChannel := range chans {
				// Channels have a type, depending on the application level
				// protocol intended. In the case of a shell, the type is
				// "session" and ServerShell may be used to present a simple
				// terminal interface.
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
					continue
				}
				channel, requests, err := newChannel.Accept()
				if err != nil {
					log.Fatalf("Could not accept channel: %v", err)
				}
				windowSize := make(chan Window, 10)
				//捕获req信息 主要处理窗口更改
				go func(in <-chan *ssh.Request) {
					for req := range in {
						switch req.Type {
						case "shell":
							req.Reply(req.Type == "shell", nil)
						case "pty-req":
							sendWindowChange(req.Payload[req.Payload[3]+4:], windowSize)
							req.Reply(true, nil)
						case "window-change":
							sendWindowChange(req.Payload, windowSize)
						}
					}
				}(requests)
				go func() {
					hosts := readHosts()
					//定义交互地址 0为服务端 1为目的主机
					place := 0
					//创建缓冲区域
					sr, sw := io.Pipe()
					cr, cw := io.Pipe()
					sC := serverChannel{dst:channel, src:sr}
					cC := serverChannel{dst:channel, src:cr}
					buf := make([]byte, 32 * 1024)
					//根据place传入不同channel
					go func() {
						for {
							nr, er := channel.Read(buf)
							if nr > 0 {
								if place == 0 {
									nw, ew := sw.Write(buf[0:nr])
									if ew != nil {
										err = ew
										break
									}
									if nr != nw {
										err = io.ErrShortWrite
										break
									}
								} else {
									nw, ew := cw.Write(buf[0:nr])
									if ew != nil {
										err = ew
										break
									}
									if nr != nw {
										err = io.ErrShortWrite
										break
									}
								}
							}
							if er != nil {
								if er != io.EOF {
									err = er
								}
								break
							}
						}
						fmt.Print("结束拷贝")
						return
					}()
					term := terminal.NewTerminal(sC, "> ")
					term.Write([]byte("欢迎使用essh!\n"))
					for {
						line, err := term.ReadLine()
						fmt.Println(line)
						if err != nil {
							break
						}
						switch line {
						case "exit":
							nConn.Close()
						case "ls":
							for k, v := range hosts {
								term.Write([]byte(strconv.Itoa(k) + " " + v[0] + "\n"))
							}
						default:
							id, err := strconv.Atoi(line)
							if err != nil {
								term.Write([]byte("错误输入\n"))
								continue
							}
							if id >= len(hosts) {
								term.Write([]byte("错误ID\n"))
								continue
							}
							h := hosts[id]
							place = 1
							sshClient(h[2], h[3], h[0]+":"+h[1], windowSize, cC)
							//手动结束sshClient中io.Copy
							cw.Write([]byte("1"))
							place = 0
						}
					}
				}()
			}
		}()
	}
}

func sendWindowChange(b []byte, wc chan<- Window) error {
	w, h := int(binary.BigEndian.Uint32(b)), int(binary.BigEndian.Uint32(b[4:]))
	wc <- Window{width: w, height: h}
	return nil
}
