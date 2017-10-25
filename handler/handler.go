package handler

import (
	"fmt"
	"net/http"

	pr "github.com/bukalapak/prometheus-aggregator"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	pProtor *pr.Protor
}

func NewHandler(pProtor *pr.Protor) *Handler {
	return &Handler{
		pProtor: pProtor,
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, "ok")
}

func (h *Handler) EndPoint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		PRegistry, err := h.pProtor.AskForRegistry(r.URL.Path)
		if err != nil {
			fmt.Fprint(w, "End Point not exist")
			return
		}
		h := promhttp.HandlerFor(PRegistry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)

	})
}
