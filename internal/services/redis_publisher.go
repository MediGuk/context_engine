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
	// Aquí se filtran los 'datos_faltantes' automáticamente al no estar en la estructura JavaTriageRequest
	finalReport := JavaTriageRequest{
		Id:                utils.GenerateID(), // ID ÚNICO via utilitario
		PatientId:         userID,             // ID del Paciente (Verificado por JWT!)
		SuggestedCategory: contextRaw.SuggestedCategory, // Categoria sugerida por la IA
		ResumeClinic:      contextRaw.ResumeClinic, // Resumen clinico del paciente
		ImageUrl:          imageUrl, // URL de S3 enviada por Go
		ExtraData:         contextRaw.ExtraData, // Todo detalles tecnicos que ia ha detectado
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
