package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-martini/martini"
)

// PrintDebug prints message if DEBUG=true
func PrintDebug(msg interface{}) {
	if os.Getenv("DEBUG") != "" {
		fmt.Println(msg)
	}
}

/*
	The following functions are to avoid tons of output in the logs
	from people accessing the "currently running tests" webpage
*/

var ignoredRoutes = []string{"/v1/status/runs"}

func ignoreRoute(path string) bool {
	for _, route := range ignoredRoutes {
		if path == route {
			return true
		}
	}
	return false
}

// CustomLogger prints HTTP server info for Martini while ignoring certain routes
func CustomLogger() martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context, log *log.Logger) {
		if ignoreRoute(req.URL.Path) {
			return
		}

		start := time.Now()

		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}

		log.Printf("Started %s %s for %s", req.Method, req.URL.Path, addr)

		rw := res.(martini.ResponseWriter)
		c.Next()

		log.Printf("Completed %v %s in %v\n", rw.Status(), http.StatusText(rw.Status()), time.Since(start))
	}
}

// CreateClassicMartini returns a new ClassicMartini instance with a custom logger
func CreateClassicMartini() *martini.ClassicMartini {
	r := martini.NewRouter()
	m := martini.New()
	m.Use(CustomLogger())
	m.Use(martini.Recovery())
	m.Use(martini.Static("public"))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)

	return &martini.ClassicMartini{Martini: m, Router: r}
}
