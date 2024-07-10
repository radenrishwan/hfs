package hsp

import (
	"log/slog"
	"net"
	"strings"
)

func parseRequest(conn net.Conn) Request {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		slog.Error("Error while reading from connection", "ERROR", err)
	}

	stringBuf := string(buf)
	var request Request

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

	request.Conn = conn
	request.Cookie = parseCookie(request.Headers["Cookie"])

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
