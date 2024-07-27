package hsp

import (
	"fmt"
)

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
	Code    int
	Msg     string
	Request Request
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s -> %s %s", e.Code, e.Msg, e.Request.Path, e.Request.Method)
}

func NewHttpError(code int, msg string, request Request) *HttpError {
	return &HttpError{Code: code, Msg: msg, Request: request}
}

type WsError struct {
	Msg string
}

func (e *WsError) Error() string {
	return e.Msg
}

func NewWsError(msg string) *WsError {
	return &WsError{Msg: msg}
}
