package middlewares

import (
	"context_engine/internal/config"
	"net/http"
)

// EnableCORS es el escudo exterior de tu servidor. Filtra TODO lo que entra.
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. Las cabeceras mágicas para el Navegador
		w.Header().Set("Access-Control-Allow-Origin", config.Envs.FrontURL) // Ej: http://localhost:3000
		w.Header().Set("Access-Control-Allow-Credentials", "true")          // Para las cookies
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Authorization for Bearer JWT

		// 2. Gestionar el famoso "Preflight" (Petición fantasma OPTIONS del navegador)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 3. Dejamos pasar la petición
		next.ServeHTTP(w, r)
	})
}
