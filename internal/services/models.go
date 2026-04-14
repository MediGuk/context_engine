package services

// TranscriptEntry representa un par pregunta-respuesta literal
type TranscriptEntry struct {
	Question string `json:"question"` // La pregunta o bloque de preguntas realizadas por la IA
	Answer   string `json:"answer"`   // La respuesta literal del paciente
}

// Symptom representa un hallazgo clínico individual con su evidencia
type Symptom struct {
	Key           string            `json:"key"`            // Ej: "fiebre"
	RawEvidence   string            `json:"raw_evidence"`   // Fragmento de texto original
	Attributes    map[string]string `json:"attributes,omitempty"` // Atributos generales (onset, location, etc)
}

// TriageContext es la ficha clínica que vive en Redis y orquesta la charla entre fases
type TriageContext struct {
	CaseTitle         string    `json:"case_title"`       // Título dinámico del caso (ej: "Dolor abdominal agudo")
	Symptoms          []Symptom `json:"symptoms"`           // Lista de síntomas detectados
	IsEmergency       bool      `json:"is_emergency"`       // Flag de urgencia vital (Phase 1)
	QuestionHistory   []string  `json:"question_history"`   // Historial de todas las preguntas realizadas
	LastTurnQuestions []string  `json:"last_turn_questions"` // Preguntas del último turno (para el transcript)
	DeniedSymptoms    []string  `json:"denied_symptoms"`    // Síntomas que el paciente ha descartado (negativos)
	FullTranscript    []TranscriptEntry `json:"full_transcript"` // El mapa de historial literal
	IsComplete        bool      `json:"is_complete"`        // ¿Triaje terminado? (Phase 2)
	SuggestedCategory string    `json:"suggested_category"` // Especialidad detectada (Phase 2/3)
	ResumeClinic      string    `json:"resume_clinic"`      // Resumen profesional generado (Phase 3)
}

// JavaTriageRequest es el contrato FINAL con el backend de Java (Spring Boot)
type JavaTriageRequest struct {
	Id                string         `json:"id"`        // ID de la petición de triaje
	PatientId         string         `json:"patientId"` // ID del paciente
	SuggestedCategory string         `json:"suggestedCategory"`
	ResumeClinic      string         `json:"resumeClinic"`
	ImageUrl          string         `json:"imageUrl,omitempty"` // URL de S3 enviada por Go
	Symptoms          []Symptom         `json:"symptoms"`           // Enviamos la lista completa para el dashboard médico
	FullTranscript    []TranscriptEntry `json:"fullTranscript"`      // El historial literal de charla
	CaseTitle         string            `json:"caseTitle"`         // Título dinámico del caso
	IsEmergency       bool              `json:"isEmergency"`       // Flag de urgencia vital
}

// --- ESTRUCTURAS PARA TEMPLATES (TYPED DATA) ---

// ExtractorData define los campos para el prompt del key_extractor.txt
type ExtractorData struct {
	PreviousContext string
	QuestionHistory []string
	DeniedSymptoms  []string
	PatientInput    string
}

// ProtocolData define los campos para el prompt del general_protocol.txt
type ProtocolData struct {
	CurrentContext  string
	QuestionHistory []string
	DeniedSymptoms  []string
}

// SummarizerData define los campos para el prompt del clinical_summarizer.txt
type SummarizerData struct {
	SymptomsContext string
}

// TriageResponse es el contrato de salida hacia el Frontend (React/Next.js)
type TriageResponse struct {
	Questions   []string `json:"questions"`    // Lista de preguntas que debe mostrar el front
	CaseTitle   string   `json:"case_title"`   // Título dinámico (ej: "Dolor abdominal")
	IsComplete  bool     `json:"is_complete"`  // Flag para saber si hemos terminado
	IsEmergency bool     `json:"is_emergency"` // Flag de urgencia vital
	Message     string   `json:"message,omitempty"` // Mensaje opcional (ej: para errores o avisos)
}
