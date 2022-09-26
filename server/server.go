package server

import (
	"net/http"

	"github.com/milzero/toys/handler"
)

func Start(endpoint string) error {
	handler.Handler()
	err := http.ListenAndServe(endpoint, nil)
	return err
}
