package stat

import (
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
)

var RequestCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "request_count",
		Help: "request count",
	},
	[]string{"ip"},
)

func init() {
	prometheus.MustRegister(RequestCounter)
}


func Listen(addr string, ctx context.Context) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(addr, nil)
	}()
}
