package web

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"

	"github.com/reconquest/pkg/log"

	"github.com/go-chi/chi"
	"github.com/reconquest/karma-go"
	"github.com/xtgo/uuid"
)

type Context struct {
	*karma.Context

	id        string
	writer    http.ResponseWriter
	request   *http.Request
	templates *template.Template
	status    Status

	data map[string]interface{}
}

type ContextKey string

func NewContext(
	writer http.ResponseWriter,
	request *http.Request,
	templates *template.Template,
) *Context {
	id := hex.EncodeToString(uuid.NewRandom().Bytes())

	return &Context{
		Context: karma.Describe("request_id", id),

		id:        id,
		writer:    writer,
		request:   request,
		templates: templates,
	}
}

func (context *Context) Get(name string) interface{} {
	return context.data[name]
}

func (context *Context) Set(name string, value interface{}) *Context {
	if context.data == nil {
		context.data = map[string]interface{}{}
	}

	context.data[name] = value

	return context
}

func (context *Context) GetStatus() Status {
	return context.status
}

func (context *Context) SetStatus(status Status) {
	context.status = status
}

func (context *Context) Write(body []byte) (int, error) {
	return context.writer.Write(body)
}

func (context *Context) GetURL() *url.URL {
	return context.request.URL
}

func (context *Context) GetURLParam(key string) string {
	return chi.URLParam(context.request, key)
}

func (context *Context) GetQueryParam(key string) string {
	return context.request.URL.Query().Get(key)
}

func (context *Context) GetRequest() *http.Request {
	return context.request
}

func (context *Context) GetResponseWriter() http.ResponseWriter {
	return context.writer
}

func (context *Context) SetResponseWriter(writer http.ResponseWriter) {
	context.writer = writer
}

func (context *Context) GetBody() io.ReadCloser {
	return context.request.Body
}

func (context *Context) GetID() string {
	return context.id
}

func (context *Context) Describe(key string, value string) *Context {
	context.Context = context.Context.Describe(key, value)

	return context
}

func (context *Context) Render(name string) Status {
	if context.templates == nil {
		return context.InternalError(
			errors.New("no templates"),
			"unable to render template: %s",
			name,
		)
	}

	context.GetResponseWriter().Header().Set("Content-Type", "text/html; charset=utf-8")

	err := context.templates.ExecuteTemplate(
		context.writer,
		name,
		context.data,
	)
	if err != nil {
		return context.InternalError(err, "unable to execute template")
	}

	return context.OK()
}

func (context *Context) OK() Status {
	return Status{
		Code: http.StatusOK,
	}
}

func (context *Context) Redirect(location string) Status {
	context.writer.Header().Set("location", location)

	return Status{
		Code: http.StatusFound,
	}
}

func (context *Context) NotFound() Status {
	context.writer.WriteHeader(http.StatusNotFound)

	return Status{
		Error: context.
			Describe("status", fmt.Sprint(http.StatusNotFound)).
			Format(
				nil,
				"not found: %q",
				context.GetURL(),
			),
		Code: http.StatusNotFound,
	}
}

func (context *Context) InternalError(
	err error,
	message string,
	values ...interface{},
) Status {
	return context.Error(
		http.StatusInternalServerError,
		err,
		message,
		values...,
	)
}

func (context *Context) BadRequest(
	err error,
	message string,
	values ...interface{},
) Status {
	return context.Error(
		http.StatusBadRequest,
		err,
		message,
		values...,
	)
}

func (context *Context) Error(
	code int,
	err error,
	message string,
	values ...interface{},
) Status {
	// log error but do not show it
	log.Errorf(err, message, values...)

	status := Status{
		Code:  code,
		Error: context.Format(nil, message, values...),
	}

	context.writer.WriteHeader(code)

	// do not send nested error to http client
	err = json.NewEncoder(context.writer).Encode(struct {
		RequestID string `json:"request_id"`
		Error     string `json:"error"`
	}{
		RequestID: context.id,
		Error:     fmt.Sprintf(message, values...),
	})
	if err != nil {
		return Status{
			Code: http.StatusInternalServerError,
			Error: karma.Format(
				err,
				"unable to marshal error",
			),
		}
	}

	return status
}
