package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Guardar claves para no usar leugo os.Getenv
type AppConfig struct {
	Port         string
	FrontURL     string
	RedisURL     string
	GroqKey      string
	JwtPublicKey string
}

// Variable global en memoria para acceder desde cualquier lado
var Envs AppConfig

// Carga los .env y ASEGURA que existen al cargar
func LoadEnv() {

	// 1. Cargar el archivo .env antes de hacer nada
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Aviso: No se encontró el archivo .env, o no se pudo cargar. Se usarán variables del sistema.")
	}

	// 2. Extraer las variables
	Envs = AppConfig{
		Port:         os.Getenv("PORT"),
		FrontURL:     os.Getenv("FRONT_URL"),
		RedisURL:     os.Getenv("REDIS_URL"),
		GroqKey:      os.Getenv("GROQ_API_KEY"),
		JwtPublicKey: os.Getenv("JWT_PUBLIC_KEY"),
	}

	// 3. VALIDAR variables al ARRANCAR

	// Puerto default en caso de null
	if Envs.Port == "" {
		Envs.Port = "8090"
	}

	// Front url null
	if Envs.FrontURL == "" {
		Envs.FrontURL = "*"
	}

	// Redis default
	if Envs.RedisURL == "" {
		Envs.RedisURL = "localhost:6379"
	}

	// GroqKey null
	if Envs.GroqKey == "" {
		log.Fatal("❌ ERROR CRÍTICO: GROQ_API_KEY no está configurada. El servidor se niega a arrancar.")
	}
	// JwtPublicKey null
	if Envs.JwtPublicKey == "" {
		log.Fatal("❌ ERROR CRÍTICO: JWT_PUBLIC_KEY (Llave de Spring Boot) no está configurada. El servidor se niega a arrancar.")
	}
	// Limpiamos llave RSA de saltos \n
	Envs.JwtPublicKey = strings.ReplaceAll(Envs.JwtPublicKey, `\n`, "\n")

	log.Println("✅ Sistema de Configuración y Seguridad inicializado con éxito.")
}
