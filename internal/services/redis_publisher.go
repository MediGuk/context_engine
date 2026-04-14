package services

import (
	"context"
	"encoding/json"
	"fmt"

	"context_engine/internal/config"
	"context_engine/internal/utils"
)

// PublishTriage manda el resultado de la IA a una cola de Redis para que Java lo guarde
func PublishTriage(ctx context.Context, userID string, aiJSON string, imageUrl string) error {

	// 1. Desempaquetar lo que nos ha dado el turno interactivo (con datos_faltantes)
	var contextRaw TriageContext
	err := json.Unmarshal([]byte(aiJSON), &contextRaw)
	if err != nil {
		return fmt.Errorf("error parseando JSON de IA para Redis: %v", err)
	}

	// 2. Mapear al contrato FINAL que espera Java (Spring Boot)
	finalReport := JavaTriageRequest{
		Id:                utils.GenerateID(),
		PatientId:         userID,
		SuggestedCategory: contextRaw.SuggestedCategory,
		ResumeClinic:      contextRaw.ResumeClinic,
		ImageUrl:          imageUrl,
		Symptoms:          contextRaw.Symptoms,
		FullTranscript:    contextRaw.FullTranscript,
		CaseTitle:         contextRaw.CaseTitle,
		IsEmergency:       contextRaw.IsEmergency,
	}

	// 3. Volver a empaquetar todo para la cola
	finalJSON, _ := json.Marshal(finalReport)

	// 4. ¡PUSH! Lo metemos en una lista (LPUSH) estilo cola (QUEUE)
	// Java estará escuchando mediante RPOP o un Redis Listener
	queueName := "mediguk:triage:queue"
	err = config.RedisClient.LPush(ctx, queueName, finalJSON).Err()
	if err != nil {
		return fmt.Errorf("error publicando en Redis: %v", err)
	}

	fmt.Printf("🚀 Triage publicado exitosamente en Redis para el usuario [%s]\n", userID)
	return nil
}
