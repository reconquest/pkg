package web

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"html/template"

	"github.com/NYTimes/gziphandler"
	"github.com/eknkc/amber"
	"github.com/go-chi/chi"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/reconquest/pkg/stack"
)

type Handler func(*Context) Status

type Web struct {
	*chi.Mux

	templates *template.Template
	resources string

	middlewares []func(Handler) Handler
}

const (
	ctxKeyContext = "web_context"
)

func New() *Web {
	web := Web{
		Mux: chi.NewMux(),
	}

	web.Use(web.recover)
	web.Use(web.log)

	return &web
}

func (web *Web) LoadTemplates(directory string, funcs template.FuncMap) error {
	compiler := amber.New()

	amber.FuncMap["hash"] = web.hash

	for name, function := range funcs {
		amber.FuncMap[name] = function
	}

	compiler.Options.LineNumbers = true
	compiler.Options.PrettyPrint = true

	var tree *template.Template

	err := filepath.Walk(
		directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			name, _ := filepath.Rel(directory, path)
			name = strings.TrimSuffix(name, filepath.Ext(name))

			if tree == nil {
				tree = template.New(name)
			} else {
				tree = tree.New(name)
			}

			log.Debugf(nil, "loading template: %s -> %s", path, name)

			err = compiler.ParseFile(path)
			if err != nil {
				return karma.Format(
					err,
					"error while parsing template: %s",
					path,
				)
			}

			tree, err = compiler.CompileWithTemplate(tree)
			if err != nil {
				return karma.Format(
					err,
					"error while compiling template: %s",
					path,
				)
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	web.templates = tree

	return nil
}

func (web *Web) Use(middleware func(Handler) Handler) {
	web.middlewares = append(web.middlewares, middleware)
}

func (web *Web) SetResourcesDir(dir string) error {
	path, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	web.resources = path

	return nil
}

func (web *Web) ServeTemplate(name string) http.HandlerFunc {
	return web.ServeFunc(
		func(context *Context) Status {
			return context.Render(name)
		},
	)
}

func (web *Web) ServeTemplateWithData(
	name string,
	data map[string]interface{},
) http.HandlerFunc {
	return web.ServeFunc(
		func(context *Context) Status {
			for key, value := range data {
				context.Set(key, value)
			}

			return context.Render(name)
		},
	)
}

func (web *Web) ServeFunc(handler Handler) http.HandlerFunc {
	return func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		web.serve(writer, request, handler)
	}
}

func (web *Web) ServeDirectory(dir string, prefix string) http.HandlerFunc {
	handler := gziphandler.GzipHandler(
		http.StripPrefix(
			prefix,
			http.FileServer(
				http.Dir(dir),
			),
		),
	)

	return func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if strings.HasSuffix(request.URL.Path, "/") {
			http.NotFound(writer, request)
			return
		}

		handler.ServeHTTP(
			writer,
			request,
		)
	}
}

func (web *Web) ServeResources(prefix string) http.HandlerFunc {
	return web.ServeDirectory(web.resources, prefix)
}

func (web *Web) Static(path string) {
	path = strings.TrimSuffix(path, "/") + "/"

	web.Mux.Get(path+"*", web.ServeResources(path))
}

func (web *Web) log(handler Handler) Handler {
	return func(context *Context) Status {
		start := time.Now()

		status := handler(context)

		duration := time.Now().Sub(start)

		logger := func(message string, args ...interface{}) {
			log.Debugf(nil, message, args...)
		}

		if status.Error != nil {
			logger = func(message string, args ...interface{}) {
				if status.Code >= 500 {
					log.Errorf(status.Error, message, args...)
				} else {
					log.Warningf(status.Error, message, args...)
				}
			}
		}

		request := context.GetRequest()

		logger(
			"{http} %v %4v %v | %.5f %v",
			status.Code,
			request.Method,
			request.URL.String(),
			duration.Seconds(),
			request.RemoteAddr,
		)

		return status
	}
}

func (web *Web) recover(handler Handler) Handler {
	return func(context *Context) Status {
		defer func() {
			if err := recover(); err != nil {
				request := context.GetRequest()

				dump, _ := httputil.DumpRequest(request, false)

				err := karma.
					Describe("client", request.RemoteAddr).
					Describe("request", strings.TrimSpace(string(dump))).
					Describe("stack", stack.Get(3)).
					Reason(err)

				log.Errorf(err, "panic while serving %s", request.URL)
			}
		}()

		return handler(context)
	}
}

func (web *Web) serve(
	writer http.ResponseWriter,
	request *http.Request,
	endpoint Handler,
) {
	response := &response{
		real:   writer,
		Code:   http.StatusOK,
		header: http.Header{},
		buffer: bytes.NewBuffer(nil),
	}

	context := NewContext(response, request, web.templates)

	handler := chain(web.middlewares, endpoint)

	context.status = handler(context)
	if context.status.Code != 0 {
		response.Code = context.status.Code
	}

	err := response.flush()
	if err != nil {
		log.Errorf(err, "error while flushing response: %s", request.URL)
	}
}

func chain(middlewares []func(Handler) Handler, endpoint Handler) Handler {
	if len(middlewares) == 0 {
		return endpoint
	}

	handler := middlewares[len(middlewares)-1](endpoint)
	for i := len(middlewares) - 2; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

func (web *Web) hash(filename string) string {
	path := filepath.Join(
		web.resources,
		strings.TrimPrefix(filename, "/static/"),
	)

	file, err := os.Open(path)
	if err != nil {
		log.Errorf(
			err,
			"{template} hashsum: unable file %s (%s)",
			path,
			filename,
		)

		return "error"
	}

	hasher := sha512.New()

	io.Copy(hasher, file)

	hash := hex.EncodeToString(hasher.Sum(nil))

	return hash[:6]
}

func (web *Web) RedirectSlashes(exclude ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if len(exclude) > 0 {
				for _, prefix := range exclude {
					if strings.HasPrefix(r.URL.Path, prefix) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			var path string
			ctx := chi.RouteContext(r.Context())
			if ctx.RoutePath == "" {
				match := chi.NewRouteContext()
				if len(r.URL.Path) > 0 && r.URL.Path[len(r.URL.Path)-1] != '/' {
					if web.Match(match, r.Method, r.URL.Path+"/") {
						if r.URL.RawQuery != "" {
							path = r.URL.Path + "/?" + r.URL.RawQuery
						} else {
							path = r.URL.Path + "/"
						}
						http.Redirect(w, r, path, 301)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
