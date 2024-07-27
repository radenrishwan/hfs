package hfs

import "strconv"

type Response struct {
	Code int
	// you need to assign a headers map if you create response from [Response],
	// please use [NewResponse] instead to avoid nil headers
	Headers map[string]string
	Body    string
}

func NewResponse() *Response {
	return &Response{
		Code:    200,
		Headers: make(map[string]string),
	}
}

func (r *Response) AddHeader(key, value string) {
	r.Headers[key] = value
}

func (r *Response) SetBody(body string) {
	r.Body = body
}

func (r *Response) SetCode(code int) {
	r.Code = code
}

func (r *Response) SetCookie(key, value, path string, maxAge int) {
	r.Headers["Set-Cookie"] = key + "=" + value + "; Path=" + path + "; Max-Age=" + strconv.Itoa(maxAge)
}