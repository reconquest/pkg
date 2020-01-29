package web

import (
	"bytes"
	"io"
	"net/http"
)

var _ http.ResponseWriter = (*response)(nil)

type response struct {
	real   http.ResponseWriter
	Code   int
	header http.Header
	buffer *bytes.Buffer
}

func (response *response) WriteHeader(code int) {
	response.Code = code
}

func (response *response) Header() http.Header {
	return response.header
}

func (response *response) Write(buffer []byte) (int, error) {
	return response.buffer.Write(buffer)
}

func (response *response) flush() error {
	for key, _ := range response.header {
		response.real.Header().Set(key, response.header.Get(key))
	}
	response.real.WriteHeader(response.Code)

	_, err := io.Copy(response.real, response.buffer)
	return err
}
