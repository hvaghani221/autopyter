package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"

	"github.com/hvaghani221/autopyter/code"
	"github.com/hvaghani221/autopyter/internal/kernel"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFs embed.FS

func main() {
	address := flag.String("address", "127.0.0.1:8080", "Address to listen on")
	kernelAddr := flag.String("kernelhost", "127.0.0.1:8888", "kernel host address")
	token := flag.String("token", "ab17a9eb56a95a0bb5af1befa3772368339592c3192da431", "API token")
	debug := flag.Bool("debug", false, "Debug mode")

	flag.Parse()
	kernel.InitKernel(*kernelAddr, *token, *debug)

	r := mux.NewRouter()

	c := code.NewCode(templateFs, *debug)
	defer c.Close()
	r.HandleFunc("/", c.PageHandler)
	r.Methods("GET").Path("/page/clip").HandlerFunc(c.ClipHandler)
	r.Methods("DELETE").Path("/page/clip/{ID}").HandlerFunc(c.ClipDeleteHandler)

	r.Methods("GET").Path("/page/code").HandlerFunc(c.CodeHandler)
	r.Methods("DELETE").Path("/page/code/{ID}").HandlerFunc(c.CodeDeleteHandler)

	r.Methods("POST").Path("/page/execute/{ID}").HandlerFunc(c.ExecuteHandler)

	r.Methods("GET").Path("/page/result/{ID}").HandlerFunc(c.ResultHandler)

	r.Methods("GET").Path("/page/select/{ID}").HandlerFunc(c.SelectHandler)

	r.Methods("GET").Path("/page/state").HandlerFunc(c.StateHander)
	r.Methods("GET").Path("/page/reset").HandlerFunc(c.StateResetHander)
	r.Methods("DELETE").Path("/page/state/{ID}").HandlerFunc(c.StateDeleteHander)

	r.Methods("GET").Path("/page/editor").HandlerFunc(c.EditorHandler)
	r.Methods("POST").Path("/page/editor/execute").HandlerFunc(c.EditorExecuteHandler)

	// Static files
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFiles)))

	errchan := make(chan error)
	go func() { errchan <- http.ListenAndServe(*address, r) }()

	url := "http://" + *address
	fmt.Println("Open URL: " + url)

	shutdownChan := make(chan os.Signal, 2)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// _ = webbrowser.Open(url)
	select {
	case <-shutdownChan:
		log.Println("Shutting down")
		break
	case err := <-errchan:
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

// func dynamicFileServer(fs fs.FS) http.Handler {
// 	return http.FileServer(http.FS(fs))
// }

func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.RequestURI)
			next.ServeHTTP(w, r)
		})
	}
}
