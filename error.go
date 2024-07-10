package hsp

import "fmt"

type ServerError struct {
	Msg string
}

func (e *ServerError) Error() string {
	return e.Msg
}

func NewServerError(msg string) *ServerError {
	return &ServerError{Msg: msg}
}

type HandlingError struct {
	Msg string
}

func (e *HandlingError) Error() string {
	return e.Msg
}

func NewHandlingError(msg string) *HandlingError {
	return &HandlingError{Msg: msg}
}

type HttpError struct {
	Code    string
	Msg     string
	Request Request
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP %s: %s -> %s %s", e.Code, e.Msg, e.Request.Path, e.Request.Method)
}

func NewHttpError(code string, msg string, request Request) *HttpError {
	return &HttpError{Code: code, Msg: msg, Request: request}
}
