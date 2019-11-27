package web

import (
	"net/http"
)

type Response struct {
	http.ResponseWriter
	Code int
}

func (response *Response) WriteHeader(code int) {
	response.Code = code
	response.ResponseWriter.WriteHeader(code)
}
