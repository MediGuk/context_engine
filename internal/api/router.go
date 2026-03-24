package api

import (
	"net/http"
	
	"context_engine/internal/api/handlers" //TriageContextHandler
)
// SetupRouter crea el mapa de rutas y lo devuelve al main
func SetupRouter() *http.ServeMux {
	// 1. Creamos un nuevo enrutador limpio de Go
	mux := http.NewServeMux()

	// 2. Registrar TODAS las rutas
	mux.HandleFunc("POST /api/triage-context", handlers.TriageContextHandler)
	// .... mas rutas .....
	
	// 3. Devolvemos el enrutador listo para usar
	return mux
}