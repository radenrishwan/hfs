package hsp

import (
	"log/slog"
	"net"
	"strconv"
	"strings"
)

func parseRequest(conn net.Conn) (request Request) {
	buf := make([]byte, 1024)
	request.Conn = conn

	_, err := conn.Read(buf)
	if err != nil {
		slog.Error("Error while reading from connection", "ERROR", err)

		return request
	}

	stringBuf := string(buf)

	sp := strings.Split(stringBuf, "\r\n")

	requestLine := strings.Split(sp[0], " ")
	request.Method = requestLine[0]
	request.Path = requestLine[1]
	request.Version = requestLine[2]

	request.Headers = make(map[string]string)
	for i := 1; i < len(sp); i++ {
		if sp[i] == "" {
			request.Body = sp[i+1]
			break
		}

		header := strings.Split(sp[i], ": ")
		request.Headers[header[0]] = header[1]
	}

	// check if cookie exists in Headers
	if request.Headers["Cookie"] != "" {
		request.Cookie = parseCookie(request.Headers["Cookie"])
	}

	return request
}

func parseCookie(cookie string) map[string]string {
	cookieMap := make(map[string]string)
	cookies := strings.Split(cookie, "; ")

	for _, c := range cookies {
		cookie := strings.Split(c, "=")
		cookieMap[cookie[0]] = cookie[1]
	}

	return cookieMap
}

func headerString(headers map[string]string) string {
	var headerString string
	for key, value := range headers {
		headerString += key + ": " + value + "\r\n"
	}

	return headerString
}

func writeResponse(response *Response, conn net.Conn) {
	// check if header has a content-type
	if _, ok := response.Headers["Content-Type"]; !ok {
		response.Headers["Content-Type"] = "text/plain"
	}

	// add content length to Headers
	response.Headers["Content-Length"] = strconv.Itoa(len(response.Body))

	// check if code is 0
	if response.Code == 0 {
		response.Code = 200
	}

	conn.Write([]byte(
		"HTTP/1.1 " + strconv.Itoa(response.Code) + "\r\n" +
			headerString(response.Headers) +
			"\r\n" +
			response.Body,
	))
}
