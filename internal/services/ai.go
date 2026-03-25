package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"context_engine/internal/config"
)

// TranscribeAudio recibe el archivo físico de RAM y le pide a Whisper que lo pase a texto.
func TranscribeAudio(file io.Reader, filename string) (string, error) {
	fmt.Printf("🚀 Enviando audio a Whisper (Groq)...\n")

	// 1. Crear un "Bolsa de RAM temporal" (&requestBody) vacía para fabricar nuestra propia carta Multipart
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 2. Escribir el parámetro principal: ¿Qué modelo de transcripción queremos usar?
	// "whisper-large-v3-turbo" o "whisper-large-v3" son los que tienen latencia ridicula en Groq
	_ = writer.WriteField("model", "whisper-large-v3-turbo")
	_ = writer.WriteField("language", "es") // Fuerza español para latencia cero

	// 3. Crear el "agujero" (file) en la carta donde irá el archivo de audio
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("Error al crear campo file para audio: %v", err)
	}

	// 4. ¡El MOTOR! Volcamos el río original de bytes (nuestro audio) dentro de la carta.
	// io.Copy copia directamente de RAM a RAM sin tocar el disco duro.
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("error volcando audio a la RAM: %v", err)
	}
	writer.Close() // Cerramos la carta sellándola. Ya está lista para enviarse.

	// 5. Preparar Petición HTTP a Groq
	whisperURL := "https://api.groq.com/openai/v1/audio/transcriptions"
	req, err := http.NewRequest("POST", whisperURL, &requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType()) //Header dice a Groq tipo de carta que mandamos
	req.Header.Set("Authorization", "Bearer "+config.Envs.GroqKey) //Header dice a Groq nuestra llave

	// 6. ¡DISPARAR! Hacemos la llamada real a internet
	client := &http.Client{} // Creamos un cliente HTTP nativo
	groqRes, err := client.Do(req)
	if err != nil || groqRes.StatusCode != 200 {
		return "", fmt.Errorf("error de Groq Whisper (HTTP %d): %v", groqRes.StatusCode, err)
	}
	defer groqRes.Body.Close() //No cerrar hasta haber enviado

	// 7. Extraer solo el texto útil de la respuesta JSON
	whisperResponseBody, _ := io.ReadAll(groqRes.Body)
	var result struct {
		Text string `json:"text"`
	}
	//Desempaquetar el JSON
	if err := json.Unmarshal(whisperResponseBody, &result); err != nil {
		return "", err
	}

	return result.Text, nil
}

//***********************************************************************************************************
// AnalyzeTriage recibe el texto (del paciente o de Whisper) y le pide a LLaMA un JSON estructurado
func AnalyzeTriage(patientText string) (string, error) {
	fmt.Printf("🧠 Enviando a LLaMA 3...\n")

	// 1. Prepara Prompt estricto (se le pide que devuelva JSON)
	prompt := buildLlamaPrompt(patientText)

	// 2. Fabricar el JSON para enviar a la API de Chat (LLaMA)
	payload := map[string]interface{}{
		"model": "llama-3.3-70b-versatile",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"}, // ¡Forzar salida JSON perfecta!
		"temperature":     0.0, // Cero alucinaciones (robotico y exacto)
	}
	payloadBytes, _ := json.Marshal(payload)

	// 3. Disparar Petición HTTP a Groq
	llamaURL := "https://api.groq.com/openai/v1/chat/completions"
	req, _ := http.NewRequest("POST", llamaURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+config.Envs.GroqKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{} // Usamos el mismo cliente HTTP de antes
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return "", fmt.Errorf("error de Groq LLaMA (HTTP %d): %v", res.StatusCode, err)
	}
	defer res.Body.Close()

	// 4. Extraer el texto del JSON gigante de la IA de forma segura
	bodyBytes, _ := io.ReadAll(res.Body)
	var output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	// Parseamos el JSON y cuidaod eroro si lista está vacia
	if err := json.Unmarshal(bodyBytes, &output); err != nil || len(output.Choices) == 0 {
		return "", fmt.Errorf("respuesta de la IA corrupta o vacía") // Return error
	}

	return output.Choices[0].Message.Content, nil // Return respuesta limpia
}

// Funcion privada para prompt perfecto
func buildLlamaPrompt(input string) string {
	return `Eres un asistente médico experto en triaje. Revisa esto: "` + input + `". 
Genera un resultado en estricto formato JSON con esta estructura:
{
  "urgency": "Alta|Media|Baja",
  "recommended_specialty": "Nombre especialidad",
  "summary": "Resumen claro..."
}`
}
