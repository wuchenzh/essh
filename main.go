package main

import (
	"bufio"
	"os"
	"strings"
)

type Window struct {
	width  int
	height int
}

func main() {
	//加载数据
	hosts := make(map[string][]string)
	f, _ := os.Open("/Users/coolcat/etc/essh/host")
	r := bufio.NewReader(f)
	for {
		h, _, err := r.ReadLine()
		if err != nil {
			break
		}
		hInfo := strings.Split(string(h), " ")

		if len(hInfo) == 4 {
			hosts[hInfo[0]] = hInfo
		}
	}
	sshServer()
	return
}
