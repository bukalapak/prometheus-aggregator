package handler

import (
	"fmt"
	"net/http"

	pr "github.com/bukalapak/prometheus-aggregator"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//pake protor bukan registrymanager karena bakal beda map nya
//karena ga bisa kasih pointer ke interface :(
type Handler struct {
	Protor *pr.Protor
}

func New(protor *pr.Protor) *Handler {
	return &Handler{
		Protor: protor,
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, "ok")
}

func (h *Handler) EndPoint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		str := r.URL.Path[1:]
		if str == "favicon.ico" {
			return
		}
		PRegistry, err := h.Protor.RegistryManager.FindRegistry(str)
		if err != nil {
			fmt.Fprint(w, "End Point not exist")
			return
		}
		h := promhttp.HandlerFor(PRegistry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
}
