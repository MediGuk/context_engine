package handlers

import (
	"fmt"
	"net/http"
)

//Mayuscula para que sea publico y cualquiera lo utilice
func TriageContextHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "El contexto de triaje se esta ejecutando...")

	// Aquí en el futuro llamaremos a internal/services/ai.go
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok", "message": "Contexto clínico recibido correctamente"}`)
}