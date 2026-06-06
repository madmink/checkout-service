package config

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"
)

func InitializeProfiler(cfg *ApplicationConfig) {
	m := http.NewServeMux()
	srv := http.Server{
		Handler:      m,
		Addr:         fmt.Sprintf(":%d", cfg.Port.Profiler),
		WriteTimeout: cfg.Port.ProfilerTimeout * time.Second,
		ReadTimeout:  cfg.Port.ProfilerTimeout * time.Second,
	}

	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()
}
