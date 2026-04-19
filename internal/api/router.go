package api

import (
	"net/http"
	
	"context_engine/internal/api/handlers" //TriageContextHandler
	"context_engine/internal/api/middlewares" //ValidateJWT
)
// SetupRouter crea el mapa de rutas y lo devuelve al main
func SetupRouter() *http.ServeMux {
	// 1. Creamos un nuevo enrutador limpio de Go
	mux := http.NewServeMux()

	// 2. Registrar TODAS las rutas con middleware y POST o GET para el filtros !!!
	mux.HandleFunc("POST /api/triage-context", middlewares.ValidateJWT(handlers.TriageContextHandler))
	mux.HandleFunc("POST /api/transcribe", middlewares.ValidateJWT(handlers.TranscribeHandler))
	// .... mas rutas .....

	// 3. Devolvemos el enrutador listo para usar
	return mux
}