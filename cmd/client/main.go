package main

import (
	"fmt"
	"net"
)

func main() {
	server_add := "127.0.0.1:3000"
	tcp_add, _ := net.ResolveTCPAddr("tcp", server_add)
	conn, err := net.DialTCP("tcp", nil, tcp_add)
	if err != nil {
		fmt.Println(err)
		return
	}
	buf := []byte("hello,xiangzhijun")
	conn.Write(buf)
	conn.Read(buf)
	fmt.Println("server response:" + string(buf))
}
