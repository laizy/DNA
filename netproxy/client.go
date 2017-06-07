package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

func sender(conn net.Conn) {
	rand.Seed(int64(time.Now().Nanosecond()))
	words := fmt.Sprintln("hello world!", rand.Int31())
	conn.Write([]byte(words))
	fmt.Println("send over:", words)

}

func main() {
	server := "127.0.0.1:9090"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("connect success")
	sender(conn)

}
