package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"context_engine/internal/services"
)

// TranscribeHandler se encarga exclusivamente de convertir audio a texto usando Whisper.
// No procesa lógica clínica, solo devuelve la transcripción para que el paciente la revise.
func TranscribeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Parsear el form-data
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Error leyendo archivo de audio", http.StatusBadRequest)
		return
	}

	// 2. Extraer el archivo de audio
	file, header, err := r.FormFile("audio")
	if err != nil {
		fmt.Println("❌ Error en /api/transcribe: No se encontró el campo 'audio'")
		http.Error(w, "Falta el archivo de audio", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Printf("🎙️ Petición de transcripción: %s (%d bytes)\n", header.Filename, header.Size)

	// 3. Llamar al servicio de Whisper (Groq)
	text, err := services.TranscribeAudio(file, header.Filename)
	if err != nil {
		fmt.Printf("❌ Fallo en Whisper: %v\n", err)
		http.Error(w, "Error procesando el audio", http.StatusInternalServerError)
		return
	}

	// 4. Devolver respuesta JSON limpia
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"text": text,
	})
}
