package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	BuildDir string
	Port     string
}

func New(buildDir, port string) *Server {
	return &Server{
		BuildDir: buildDir,
		Port:     port,
	}
}

func (s *Server) cleanURLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index"
		}

		if strings.HasSuffix(path, "/") {
			path = strings.TrimSuffix(path, "/")
		}

		htmlPath := filepath.Join(s.BuildDir, path+".html")

		if _, err := os.Stat(htmlPath); err == nil {
			http.ServeFile(w, r, htmlPath)
			return
		}

		// Try dir/index.html (e.g. /blog or /blog.html -> /blog/index.html)
		dirPath := strings.TrimSuffix(path, ".html")
		indexPath := filepath.Join(s.BuildDir, dirPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		staticPath := filepath.Join(s.BuildDir, path)
		if _, err := os.Stat(staticPath); err == nil {
			http.ServeFile(w, r, staticPath)
			return
		}

		// Serve custom 404 page if available
		notFoundPath := filepath.Join(s.BuildDir, "404.html")
		if data, err := os.ReadFile(notFoundPath); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			w.Write(data)
			return
		}

		http.NotFound(w, r)
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.cleanURLHandler())

	fmt.Printf("Serving on http://localhost:%s\n", s.Port)
	return http.ListenAndServe(":"+s.Port, nil)
}

func (s *Server) StartWithLogging() {
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}