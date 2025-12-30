package main

import (
	"fmt"
	"log"
	"net"
	"proj/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("error", err)
	}
	fmt.Println("tcp listener started on port 42069")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("error", err)
		}
		req, _ := request.RequestFromReader(conn)
		log.Println("HttpVersion", string(req.RequestLine.HttpVersion))
		log.Println(req.Headers)
		log.Println("Method:", req.RequestLine.Method)
	}

}
