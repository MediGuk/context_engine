package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"text/template"

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
	req.Header.Set("Content-Type", writer.FormDataContentType())   //Header dice a Groq tipo de carta que mandamos
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

// AnalyzeTriageElite orquesta las 3 fases del triaje inteligente
func AnalyzeTriageElite(patientInput string, context *TriageContext) ([]string, error) {
	// --- REGISTRO DE TRANSCRIPT (Caja Negra) ---
	turnKey := "INITIAL"
	if len(context.LastTurnQuestions) > 0 {
		turnKey = strings.Join(context.LastTurnQuestions, " | ")
	}
	context.FullTranscript = append(context.FullTranscript, TranscriptEntry{
		Question: turnKey,
		Answer:   patientInput,
	})

	fmt.Printf("🖋️ Fase 1: Extractor (Detección de síntomas)...\n")
	
	// --- FASE 1: KEY EXTRACTOR ---
	extractorPrompt, err := renderTemplate("key_extractor.txt", ExtractorData{
		PreviousContext: symptomsToText(context.Symptoms),
		QuestionHistory: context.QuestionHistory,
		DeniedSymptoms:  context.DeniedSymptoms,
		PatientInput:    patientInput,
	})
	if err != nil {
		return nil, err
	}

	extractorJSON, err := callLLM(extractorPrompt, true) // Forzamos JSON
	if err != nil {
		return nil, err
	}
	fmt.Printf("🔍 [DEBUG] Extractor RAW JSON: %s\n", extractorJSON)

	// Parsear resultado del Extractor
	var extraction struct {
		Symptoms       []Symptom `json:"symptoms"`
		DeniedSymptoms []string  `json:"denied_symptoms"`
		CaseTitle      string    `json:"case_title"`
		IsEmerg        bool      `json:"is_emergency"`
	}
	json.Unmarshal([]byte(extractorJSON), &extraction)

	// --- LÓGICA DE FUSIÓN (GO) ---
	context.Symptoms = mergeSymptoms(context.Symptoms, extraction.Symptoms)
	context.DeniedSymptoms = mergeDeniedSymptoms(context.DeniedSymptoms, extraction.DeniedSymptoms)
	
	// Limpieza: Si un síntoma ahora está presente, lo quitamos de negados
	for _, s := range extraction.Symptoms {
		context.DeniedSymptoms = removeFromSlice(context.DeniedSymptoms, s.Key)
	}

	context.IsEmergency = extraction.IsEmerg
	
	// Solo actualizamos el título si la IA ha detectado uno (no sobreescribimos con vacío)
	if extraction.CaseTitle != "" {
		context.CaseTitle = extraction.CaseTitle
	}

	fmt.Printf("📦 [DEBUG] Contexto Clínico tras Merge:\n")
	for _, s := range context.Symptoms {
		fmt.Printf("   - [%s]: Atributos: %+v\n", s.Key, s.Attributes)
	}
	fmt.Printf("🚫 [DEBUG] Síntomas Descartados: %v\n", context.DeniedSymptoms)
	fmt.Printf("📜 [DEBUG] Historial de Preguntas: %v\n", context.QuestionHistory)

	fmt.Printf("🗣️ Fase 2: Protocolo (Generación de respuesta)...\n")
	
	// --- FASE 2: PROTOCOLO GENERAL ---
	protocolPrompt, err := renderTemplate("general_protocol.txt", ProtocolData{
		CurrentContext:  symptomsToText(context.Symptoms),
		QuestionHistory: context.QuestionHistory,
		DeniedSymptoms:  context.DeniedSymptoms,
	})
	if err != nil {
		return nil, err
	}

	protocolJSON, err := callLLM(protocolPrompt, true)
	if err != nil {
		return nil, err
	}
	fmt.Printf("🔍 [DEBUG] Protocol RAW JSON: %s\n", protocolJSON)

	var protocolData struct {
		Questions  []string `json:"questions"`
		IsComplete bool     `json:"is_complete"`
	}
	json.Unmarshal([]byte(protocolJSON), &protocolData)

	context.IsComplete = protocolData.IsComplete
	// Añadimos las nuevas preguntas al historial acumulado
	if len(protocolData.Questions) > 0 {
		context.QuestionHistory = append(context.QuestionHistory, protocolData.Questions...)
		context.LastTurnQuestions = protocolData.Questions // Guardamos para el próximo turno del Transcript
	} else {
		context.LastTurnQuestions = nil
	}

	// --- FASE 3: SUMMARIZER (Opcional si IsComplete) ---
	if context.IsComplete {
		fmt.Printf("📋 Fase 3: Resumidor (Informe Final)...\n")
		summarizerPrompt, err := renderTemplate("clinical_summarizer.txt", SummarizerData{
			SymptomsContext: symptomsToText(context.Symptoms),
		})
		if err == nil {
			summaryJSON, err := callLLM(summarizerPrompt, true)
			if err == nil {
				var summaryData struct {
					ResumeClinic      string `json:"resume_clinic"`
					SuggestedCategory string `json:"suggested_category"`
				}
				json.Unmarshal([]byte(summaryJSON), &summaryData)
				context.ResumeClinic = summaryData.ResumeClinic
				context.SuggestedCategory = summaryData.SuggestedCategory
			}
		}
	}

	return protocolData.Questions, nil
}

// --- UTILIDADES INTERNAS ---

func callLLM(prompt string, forceJSON bool) (string, error) {
	payload := map[string]interface{}{
		"model":       "llama-3.3-70b-versatile",
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.0,
	}
	if forceJSON {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+config.Envs.GroqKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return "", fmt.Errorf("error Groq: %v", err)
	}
	defer res.Body.Close()

	var output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(res.Body).Decode(&output)
	if len(output.Choices) == 0 {
		return "", fmt.Errorf("ia vacia")
	}
	return output.Choices[0].Message.Content, nil
}

func renderTemplate(filename string, data interface{}) (string, error) {
	tmpl, err := template.ParseFiles("internal/prompts/" + filename)
	if err != nil {
		return "", fmt.Errorf("error cargando prompt %s: %v", filename, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func symptomsToText(symptoms []Symptom) string {
	if len(symptoms) == 0 {
		return "Ninguno detectado."
	}
	res := ""
	for _, s := range symptoms {
		attrs := ""
		if len(s.Attributes) > 0 {
			attrStrings := []string{}
			for k, v := range s.Attributes {
				attrStrings = append(attrStrings, fmt.Sprintf("%s: %s", k, v))
			}
			attrs = fmt.Sprintf(" | ATRIBUTOS: {%s}", strings.Join(attrStrings, ", "))
		}
		res += fmt.Sprintf("- [%s]: %s (Evidencia: %s)\n", s.Key, attrs, s.RawEvidence)
	}
	return res
}

func mergeSymptoms(old []Symptom, new []Symptom) []Symptom {
	sMap := make(map[string]Symptom)
	for _, s := range old {
		sMap[s.Key] = s
	}
	for _, s := range new {
		existing, found := sMap[s.Key]
		if found {
			// Fusionamos atributos
			if existing.Attributes == nil {
				existing.Attributes = make(map[string]string)
			}
			for k, v := range s.Attributes {
				existing.Attributes[k] = v
			}
			// Actualizamos el resto de info
			existing.RawEvidence = s.RawEvidence
			sMap[s.Key] = existing
		} else {
			sMap[s.Key] = s
		}
	}
	res := []Symptom{}
	for _, v := range sMap {
		res = append(res, v)
	}
	return res
}

// mergeDeniedSymptoms fusiona las negaciones asegurando que no haya duplicados
func mergeDeniedSymptoms(old []string, new []string) []string {
	dMap := make(map[string]bool)
	for _, s := range old {
		dMap[s] = true
	}
	for _, s := range new {
		dMap[s] = true
	}
	res := []string{}
	for k := range dMap {
		res = append(res, k)
	}
	return res
}

// removeFromSlice elimina un valor de un slice de strings
func removeFromSlice(slice []string, val string) []string {
	res := []string{}
	for _, s := range slice {
		if s != val {
			res = append(res, s)
		}
	}
	return res
}
