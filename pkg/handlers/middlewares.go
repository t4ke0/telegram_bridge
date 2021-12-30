package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/t4ke0/telegram_bridge/pkg/db"
)

// LogginMiddleware
func LogginMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %15s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

// AuthorizeTokenMiddleware
func AuthorizeTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		if token = strings.TrimSpace(r.Header.Get(tokenHeaderKey)); token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		defer errorHandler(w)
		conn, err := db.New()
		if err != nil {
			panic(err)
		}

		defer conn.Close()

		ok, err := conn.AuthorizeToken(token)
		if err != nil {
			panic(err)
		}

		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Middleware
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Middlewares
type Middlewares []Middleware

// Chain
func (ms Middlewares) Chain(root http.HandlerFunc) http.HandlerFunc {
	if len(ms) == 0 {
		return root
	}

	h := root

	i := len(ms) - 1
	for ; i >= 0; i-- {
		h = ms[i](h)
	}

	return h
}
