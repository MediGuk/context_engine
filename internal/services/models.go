package services

// Manejamos INTERNAMENTE (Go + AI + Frontend)
// Contiene la memoria de lo que falta para seguir preguntando.
type TriageContext struct {
	ExtraData         map[string]any `json:"extraData"`
	SuggestedCategory string         `json:"suggestedCategory"`
	DatosFaltantes    []string       `json:"datos_faltantes"`
	ResumeClinic   	  string         `json:"resumeClinic"`
}

// Contrato FINAL con Java (Spring Boot)
type JavaTriageRequest struct {
	Id                string         `json:"id"`                 // ID of triage request
	PatientId         string         `json:"patientId"`          // ID of patient
	SuggestedCategory string         `json:"suggestedCategory"`
	ResumeClinic   	  string         `json:"resumeClinic"`
	ImageUrl          string         `json:"imageUrl,omitempty"` // URL de S3
	ExtraData         map[string]any `json:"extraData"`         // anatomSite, fever, etc.
}
