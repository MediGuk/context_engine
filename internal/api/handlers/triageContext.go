package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"context_engine/internal/api/context_keys"
	"context_engine/internal/services" // Importamos a los Cirujanos
)

// TriageContextHandler es el punto final de la ruta POST /api/triage-context
func TriageContextHandler(w http.ResponseWriter, r *http.Request) {

	// Helper para mandar JSON de error limpio
	sendError := func(msg string, code int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(services.TriageResponse{
			Message: msg,
		})
	}

	// 0. Securitires passed from middleware & CORS
	fmt.Println("NUEVA PETICIÓN EN EL HANDLER. Security granted OK.")

	// Extraemos el userID que inyectó el middleware en el contexto
	ctx := r.Context()
	userID, _ := ctx.Value(context_keys.UserIDKey).(string)
	if userID == "" {
		userID = "anonymous"
	}

	// 1. Le decimos a Go que parsee el "multipart/form-data" que nos manda el navegador.
	// El "10 << 20" significa que le permitimos usar hasta 10 MB de RAM para sostener el archivo.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("❌ Error. El Frontend no ha mandado un form-data válido:", err)
		sendError("Error leyendo formulario de triaje", http.StatusBadRequest)
		return
	}

	// 2. ¿Es una petición de REINICIO o CIERRE final?
	newSession := r.FormValue("newSession") == "true"
	if newSession {
		fmt.Printf("⚠️ [SISTEMA] Petición de reinicio. Borrando contexto previo para UserID %s\n", userID)
		_ = services.DeleteClinicalContext(ctx, userID)
		// No salimos, permitimos que se cree el nuevo contexto con el texto/audio que llegue
	}

	// Recuperamos la ficha clínica de Redis (para saber si podemos finalizar)
	contextRaw := services.TriageContext{}
	previousContextJSON, _ := services.GetClinicalContext(ctx, userID)
	if previousContextJSON != "" {
		json.Unmarshal([]byte(previousContextJSON), &contextRaw)
	}

	isFinal := r.FormValue("isFinal") == "true"
	if isFinal {
		// Solo permitimos finalizar si la IA ya ha marcado el triaje como completo
		if !contextRaw.IsComplete {
			sendError("El triaje aún no está listo para enviarse", http.StatusBadRequest)
			return
		}

		// --- NUEVO: ¿Y si ha mandado una imagen final? ---
		imageUrl := ""
		imageFile, _, err := r.FormFile("image")
		if err == nil {
			defer imageFile.Close()
			imageUrl = "https://mediguk-s3.amazonaws.com/temp-placeholder.jpg"
		}

		// Mandamos a la cola de Redis para Spring Boot
		err = services.PublishTriage(ctx, userID, previousContextJSON, imageUrl)
		if err != nil {
			fmt.Println("⚠️ Error publicando en Redis:", err)
			sendError("Error guardando el triaje final", http.StatusInternalServerError)
			return
		}
		_ = services.DeleteClinicalContext(ctx, userID)

		w.Header().Set("Content-Type", "application/json")
		responseFinal := services.TriageResponse{
			IsComplete: true,
			Message:    "Triaje enviado correctamente al médico.",
		}
		json.NewEncoder(w).Encode(responseFinal)
		return
	}

	// 3. ¿El paciente ha escrito Texto o ha mandado Audio? (SOLO PARA TRIAJE INTERACTIVO)
	var textPatient string
	textReceived := strings.TrimSpace(r.FormValue("text"))

	if textReceived != "" {
		fmt.Println("📝 Triaje por texto detectado. Saltando Whisper...")
		textPatient = textReceived
	} else {
		file, header, err := r.FormFile("audio")
		if err != nil {
			fmt.Println("❌ Error: No hay ni texto ni archivo 'audio' en la petición")
			sendError("Falta contexto clínico (Texto o Audio)", http.StatusBadRequest)
			return
		}
		defer file.Close()
		fmt.Printf("🎙️ Audio recibido: %s (%d bytes)\n", header.Filename, header.Size)

		transcribedText, err := services.TranscribeAudio(file, header.Filename)
		if err != nil {
			fmt.Println("❌ Fallo crítico en Whisper:", err)
			sendError("Error transcribiendo audio. Inténtelo de nuevo.", http.StatusInternalServerError)
			return
		}
		textPatient = transcribedText
	}

	// 4. CASO INTERACTIVO: Llamada al Motor Elite de 3 Fases
	responseMessage, err := services.AnalyzeTriageElite(textPatient, &contextRaw)
	if err != nil {
		fmt.Println("❌ Fallo crítico en el Motor Elite:", err)
		sendError("Error analizando triaje: El servicio de IA no responde en este momento", http.StatusInternalServerError)
		return
	}

	// 5. Guardamos el estado ACTUALIZADO en Redis
	updatedJSON, _ := json.Marshal(contextRaw)
	_ = services.SaveClinicalContext(ctx, userID, string(updatedJSON))

	fmt.Printf("🤖 IA responde: [%s] (Complete: %v, Emerg: %v)\n", responseMessage, contextRaw.IsComplete, contextRaw.IsEmergency)

	// 5. Devolver la respuesta "Limpia" al Frontend (Usando Struct Tipado)
	responseObj := services.TriageResponse{
		Question:    responseMessage,
		CaseTitle:   contextRaw.CaseTitle,
		IsComplete:  contextRaw.IsComplete,
		IsEmergency: contextRaw.IsEmergency,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseObj)
}
