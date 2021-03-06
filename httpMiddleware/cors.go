package httpMiddleware

import (
	"github.com/rs/cors"
	"net/http"
)

var c = cors.New(cors.Options{
	AllowedOrigins:   []string{"*"},
	AllowCredentials: true,
	Debug:            false,
	AllowedHeaders:   []string{"Content-Type", "Authorization"},
	AllowedMethods:   []string{"GET", "POST", "PATCH"},
})

func HandleCors(h http.Handler) http.Handler {
	return c.Handler(h)
}
