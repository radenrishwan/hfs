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
	Code string
	Msg  string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("HTTP %s: %s", e.Code, e.Msg)
}

func NewHttpError(code string, msg string) *HttpError {
	return &HttpError{Code: code, Msg: msg}
}
