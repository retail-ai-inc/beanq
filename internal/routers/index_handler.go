package routers

import (
	"net/http"
)

type Index struct {
}

func NewIndex() *Index {
	return &Index{}
}

func (t *Index) Home(w http.ResponseWriter, r *http.Request) {
}
