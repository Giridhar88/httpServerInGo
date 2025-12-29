package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"proj/internal/request"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		defer f.Close()

		str := ""
		for {
			data := make([]byte, 8)
			n, err := f.Read(data)
			if err != nil {
				break
			}
			data = data[:n]
			if i := bytes.IndexByte(data, '\n'); i != -1 {
				str += string(data[:i])
				data = data[i+1:]
				out <- str
				str = ""
			}
			str += string(data)
		}
		if len(str) != 0 {
			out <- str
		}
	}()

	return out
}

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
