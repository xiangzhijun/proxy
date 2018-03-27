package main

import (
	"fmt"
	"net"
)

func main() {
	fmt.Println("server start")
	server_add := "127.0.0.1:3000"
	tcp_add, _ := net.ResolveTCPAddr("tcp", server_add)
	l, _ := net.ListenTCP("tcp", tcp_add)
	for {
		conn, _ := l.AcceptTCP()
		buf := make([]byte, 100)
		conn.Read(buf)
		fmt.Println("new conn:" + string(buf))
		buf = []byte("hello,what's you name?")
		local_add := conn.LocalAddr()
		remote_add := conn.RemoteAddr()
		fmt.Println("local addr:" + local_add.String() + ",remote addr:" + remote_add.String())
		conn.Write(buf)
	}
}
