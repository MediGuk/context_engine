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

	// Extraemos el userID que inyectó el middleware en el contexto
	ctx := r.Context()
	userID, _ := ctx.Value("userID").(string)
	if userID == "" {
		userID = "anonymous"
	}

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

	// --- NUEVO: ¿Y si ha mandado una imagen? ---
	imageUrl := ""
	imageFile, _, err := r.FormFile("image")
	if err == nil {
		defer imageFile.Close()
		// TODO: Implementar subida real a S3. Por ahora simulamos una URL si llega el archivo.
		imageUrl = "https://mediguk-s3.amazonaws.com/temp-placeholder.jpg"
		fmt.Println("📸 Imagen detectada. (S3 Upload Placeholder activo)")
	}

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

	// --- REDIS with context -------------------------------------

	// 1. ¿El paciente dice que ya ha terminado ("isFinal")?
	isFinal := r.FormValue("isFinal") == "true"

	// 2. Recuperamos lo que ya sabíamos de este usuario de la memoria (Redis)
	previousContext, _ := services.GetClinicalContext(ctx, userID)

	if isFinal {
		// A. CASO FINAL: El paciente da el OK.
		if previousContext == "" {
			http.Error(w, "No hay contexto para finalizar", http.StatusBadRequest)
			return
		}

		// Mandamos a la cola de Redis para Spring Boot
		go func() {
			// Ahora pasamos la imageUrl (puede ser vacía "")
			err := services.PublishTriage(ctx, userID, previousContext, imageUrl)
			if err != nil {
				fmt.Println("⚠️ Error asíncrono publicando en Redis:", err)
			}
			// Limpiamos la memoria temporal
			services.DeleteClinicalContext(ctx, userID)
		}()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "finalized", "message": "Triaje enviado a revisión médica"}`))
		return
	}

	// B. CASO INTERACTIVO: Seguimos preguntando
	// Llamar al servicio LLaMA 3 con el texto nuevo Y lo que ya sabíamos
	jsonIAPuro, err := services.AnalyzeTriage(textPatient, previousContext)
	if err != nil {
		fmt.Println("❌ Fallo crítico en LLaMA:", err)
		http.Error(w, "Error analizando triaje", http.StatusInternalServerError)
		return
	}

	// Guardamos el NUEVO contexto actualizado en Redis para la siguiente pregunta
	_ = services.SaveClinicalContext(ctx, userID, jsonIAPuro)

	fmt.Println("🤖 Respuesta interactiva (Contexto Actualizado):\n", jsonIAPuro)
	// ---------------------------------------------------------

	// 4. Devolver la magia al Navegador de React
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Código 200 Todo Perfecto
	w.Write([]byte(jsonIAPuro))
}
