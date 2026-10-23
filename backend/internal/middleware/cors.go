package middleware

import (
	"tv/internal/config"

	"github.com/rs/cors"
)

func Cors(cfg *config.Config) *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

}
