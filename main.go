package main

import (
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

//goo:embed static
// var staticFiles embed.FS

//goo:embed templates
// var templateFs embed.FS

func main() {
	address := flag.String("address", ":8080", "Address to listen on")
	kernelAddr := flag.String("kernelhost", "localhost:8888", "kernel host address")
	token := flag.String("token", "ab17a9eb56a95a0bb5af1befa3772368339592c3192da431", "API token")

	flag.Parse()
	kernel.InitKernel(*kernelAddr, *token)

	r := mux.NewRouter()

	c := code.NewCode(os.DirFS("."))
	defer c.Close()
	r.HandleFunc("/", c.PageHandler)
	r.Methods("GET").Path("/page/clip").HandlerFunc(c.ClipHandler)
	r.Methods("DELETE").Path("/page/clip/{ID}").HandlerFunc(c.ClipDeleteHandler)

	r.Methods("GET").Path("/page/code").HandlerFunc(c.CodeHandler)
	r.Methods("DELETE").Path("/page/code/{ID}").HandlerFunc(c.CodeDeleteHandler)
	r.Methods("POST").Path("/page/execute/{ID}").HandlerFunc(c.ExecuteHandler)
	r.Methods("GET").Path("/page/result/{ID}").HandlerFunc(c.ResultHandler)

	// Static files
	// r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFiles)))
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(os.DirFS("."))))

	errchan := make(chan error)
	go func() { errchan <- http.ListenAndServe(*address, r) }()

	url := "http://localhost" + *address
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
