package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient es el puntero global a la base de datos en memoria
var RedisClient *redis.Client

// ConnectRedis inicializa la conexión con Redis
func ConnectRedis() {
	fmt.Printf("🔌 Conectando a Redis en %s...\n", Envs.RedisURL)

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     Envs.RedisURL,
		Password: "", // Por ahora sin contraseña
		DB:       0,  // DB por defecto
	})

	// 🧪 Ping de prueba para asegurar que Redis está vivo
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Error crítico: No se pudo conectar a Redis: %v", err)
	}

	fmt.Println("✅ Conexión a Redis establecida con éxito.")
}
