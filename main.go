package main

import (
	"github.com/lixd96/i-scheduler-extender/pkg/extender"
	"github.com/lixd96/i-scheduler-extender/pkg/server"
	"net/http"
)

var h *server.Handler

func init() {
	h = server.NewHandler(extender.NewExtender())
}

func main() {
	http.HandleFunc("/filter", h.Filter)
	http.HandleFunc("/filter_onlyone", h.FilterOnlyOne) // Filter 接口的一个额外实现
	http.HandleFunc("/priority", h.Prioritize)
	http.HandleFunc("/bind", h.Bind)
	http.ListenAndServe(":8080", nil)
}
