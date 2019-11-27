package web

type Status struct {
	Code  int
	Error error
	Body  []byte
}
