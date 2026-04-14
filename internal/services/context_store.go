package services

import (
	"context"
	"fmt"
	"time"

	"context_engine/internal/config"
)

const contextPrefix = "triage:context:"
const contextTTL = 30 * time.Minute

// GetClinicalContext recupera el progreso del triaje guardado en Redis
func GetClinicalContext(ctx context.Context, userID string) (string, error) {
	key := contextPrefix + userID
	val, err := config.RedisClient.Get(ctx, key).Result()
	if err != nil {
		// Si no existe, devolvemos un string vacío sin error crítico
		fmt.Printf("📂 [REDIS] No hay contexto previo para %s\n", userID)
		return "", nil
	}
	fmt.Printf("📂 [REDIS] Contexto RECUPERADO para %s: %s\n", userID, val)
	return val, nil
}

// SaveClinicalContext guarda el estado actual del triaje para el siguiente turno
func SaveClinicalContext(ctx context.Context, userID string, contextJSON string) error {
	key := contextPrefix + userID
	fmt.Printf("💾 [REDIS] Guardando Contexto para %s: %s\n", userID, contextJSON)
	err := config.RedisClient.Set(ctx, key, contextJSON, contextTTL).Err()
	if err != nil {
		return fmt.Errorf("error guardando contexto en Redis: %v", err)
	}
	return nil
}

// DeleteClinicalContext limpia la memoria tras enviar el informe a Java
func DeleteClinicalContext(ctx context.Context, userID string) error {
	key := contextPrefix + userID
	return config.RedisClient.Del(ctx, key).Err()
}
