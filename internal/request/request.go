package request

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type Request struct {
	RequestLine RequestLine
	Headers     map[string]string
	Body        string
	state       parserState
}

var ERROR_BAD_START_LINE = fmt.Errorf("error in the start line")
var INCOMPLETE_START_LINE = fmt.Errorf("start line is incomplete")
var ERROR_UNSUPPORTED_HTTP_VERSION = fmt.Errorf("the http version is not supported :P")
var ERROR_NO_CRLF = fmt.Errorf("crlf not found yet")
var ERROR_INVALID_CONTENTLEN = fmt.Errorf("content length is invalid")
var ERROR_IN_HEADER = fmt.Errorf("corrupted header")
var SEPARATOR = []byte("\r\n")

type parserState string

const (
	StateInit            parserState = "init"
	StateDone            parserState = "done"
	StateRequestLineDone parserState = "request line parsed"
	StateHeadersDone     parserState = "headers have been parsed"
	StateBodyDone        parserState = "body has been parsed"
)

func (r *RequestLine) validHTTP() bool {
	return r.HttpVersion == "1.1"
}

func (r *Request) Get(key string) (string, bool) {
	if key, exists := r.Headers[key]; exists {
		return key, true
	} else {
		return "", false
	}
}

func newRequest() *Request {
	return &Request{
		state: StateInit,
	}
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		switch r.state {
		case StateInit:
			rl, n, err := parseRequestLine(data[read:])
			if err != nil {
				return 0, err
			}
			if n == 0 {
				break outer
			}
			r.RequestLine = *rl
			read += n
			r.state = StateRequestLineDone
			continue

		case StateRequestLineDone:
			h, n, err := parseHeaders(data[read:])
			if err != nil {
				return read, err
			}

			if n == 0 {
				return read, nil
			}

			r.Headers = h
			read += n
			r.state = StateHeadersDone
			continue

		case StateHeadersDone:
			cl, exists := r.Get("content-length")
			if !exists {
				cl = "0"
			}
			clen, err := strconv.Atoi(cl)
			if err != nil {
				return 0, err
			}
			if clen == 0 {
				r.state = StateDone
				continue
			}
			current := ""
			current += string(data[read:])
			r.Body += current
			read += len(current)
			if len(r.Body) >= clen {
				r.state = StateDone
			} else {
				return read, nil
			}

			continue

		case StateDone:
			break outer
		}
	}
	return read, nil
}

func (r *Request) done() bool {
	return r.state == StateDone
}

func parseRequestLine(b []byte) (*RequestLine, int, error) {

	idx := bytes.Index(b, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}
	startLine := b[:idx]
	read := idx + len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))

	httpParts := bytes.Split(parts[2], []byte("/"))
	if len(httpParts) != 2 || string(httpParts[0]) != "HTTP" || string(httpParts[1]) != "1.1" {
		return nil, 0, ERROR_UNSUPPORTED_HTTP_VERSION
	}

	if len(parts) < 3 {
		return nil, 0, ERROR_BAD_START_LINE
	}

	rl := &RequestLine{string(httpParts[1]), string(parts[1]), string(parts[0])}
	if !rl.validHTTP() {
		return nil, 0, ERROR_UNSUPPORTED_HTTP_VERSION
	}
	return rl, read, nil
}

func parseHeaders(b []byte) (map[string]string, int, error) {
	headers := make(map[string]string)
	he := append(SEPARATOR, SEPARATOR...)
	n := bytes.Index(b, he)

	if n == -1 {
		return nil, 0, nil
	}

	n += len(he)

	rawHeaders := b[:n]
	hl := bytes.Split(rawHeaders, SEPARATOR)
	for i := 0; i < len(hl); i++ {
		if len(hl[i]) == 0 {
			continue
		}
		ei := strings.Index(string(hl[i]), ":")
		if ei == -1 {
			return nil, 0, ERROR_IN_HEADER
		}
		e := []string{string(hl[i])[:ei], string(hl[i])[ei+1:]}

		if strings.Contains(e[0], " ") {
			return nil, 0, ERROR_IN_HEADER
		}
		if len(e[0]) == 0 {
			return nil, 0, ERROR_IN_HEADER
		}
		k, v := strings.TrimSpace(e[0]), strings.TrimSpace(e[1])

		r := regexp.MustCompile("^[A-Za-z0-9!#$%&'*+\\-.^_|~`]+$")
		if !r.MatchString(k) {
			return nil, 0, ERROR_IN_HEADER
		}
		k = strings.ToLower(k)
		if _, exists := headers[k]; exists {
			headers[k] += ", " + v
		} else {
			headers[k] = v
		}
	}

	return headers, n, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()
	buf := make([]byte, 1024)
	bufIdx := 0
	for !request.done() {
		n, err := reader.Read(buf[bufIdx:])

		if err != nil {
			return nil, err
		}
		bufIdx += n
		readN, err := request.parse(buf[:bufIdx])
		if err != nil {
			return nil, err
		}
		copy(buf, buf[readN:bufIdx])
		bufIdx -= readN

	}
	return request, nil
}
