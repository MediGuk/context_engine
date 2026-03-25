package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"context_engine/internal/services" // Importamos a los Cirujanos
)

// TriageContextHandler es el punto final de la ruta POST /api/triage-context
func TriageContextHandler(w http.ResponseWriter, r *http.Request) {

	// 0. Securitires passed from middleware & CORS
	fmt.Println("NUEVA PETICIÓN EN EL HANDLER. Security granted OK.")

	// 1. Le decimos a Go que parsee el "multipart/form-data" que nos manda el navegador.
	// El "10 << 20" significa que le permitimos usar hasta 10 MB de RAM para sostener el archivo.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("❌ Error. El Frontend no ha mandado un form-data válido:", err)
		http.Error(w, "Error leyendo formulario", http.StatusBadRequest)
		return
	}

	// 2. ¿El paciente ha escrito Texto o ha mandado Audio?
	var textPatient string

	// Veiene en campo "text" ???
	textReceived := strings.TrimSpace(r.FormValue("text"))

	if textReceived != "" {
		// ¡VÍA RÁPIDA! El paciente ha escrito. Nos saltamos Whisper.
		fmt.Println("📝 Triaje por texto detectado. Saltando Whisper...")
		textPatient = textReceived

	} else {
		// ¡VÍA AUDIO! Si no hay texto, buscamos el archivo de audio.
		// 2. Extraemos el archivo físicamente de la RAM. 
		// IMPORTANTE: Busca el campo con el nombre que estés usando en tu FormData de Next.js (ej: "file" o "audio")
		// (Te pongo "audio" como ejemplo estándar, cámbialo si en tu Frontend usas form.append('file', blob))
		file, header, err := r.FormFile("audio")
		if err != nil {
			fmt.Println("❌ Error: No hay ni texto ni archivo 'audio' en la petición")
			http.Error(w, "Falta contexto clínico (Texto o Audio)", http.StatusBadRequest)
			return
		}
		// "defer" retrasa la ejecución de esta línea hasta que la función handleUpload termine.
		// Es como decir: "Pase lo que pase, asegúrate de cerrar el archivo en la RAM cuando te vayas para no ahogar la memoria".
		defer file.Close()

		fmt.Printf("🎙️ Audio recibido: %s (%d bytes)\n", header.Filename, header.Size)

		// Llamamos a Whisper para transcribir el audio
		transcribedText, err := services.TranscribeAudio(file, header.Filename)
		if err != nil {
			fmt.Println("❌ Fallo crítico en Whisper:", err)
			http.Error(w, "Error transcribiendo audio", http.StatusInternalServerError)
			return
		}
		fmt.Println("🗣️ Transcripción real obtenida:", transcribedText)
		
		textPatient = transcribedText
	}

	// 3. Llamar al servicio LLaMA 3 (Tanto si era texto como si era audio)
	jsonIAPuro, err := services.AnalyzeTriage(textPatient)
	if err != nil {
		fmt.Println("❌ Fallo crítico en LLaMA:", err)
		http.Error(w, "Error analizando triaje", http.StatusInternalServerError)
		return
	}
	fmt.Println("🤖 Respuesta limpia de IA JSON Crudo:\n", jsonIAPuro)

	// 4. Devolver la magia al Navegador de React
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Código 200 Todo Perfecto
	w.Write([]byte(jsonIAPuro))
}
