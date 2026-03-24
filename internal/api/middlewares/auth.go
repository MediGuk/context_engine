package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"context_engine/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// ValidateJWT envuelve a un Handler para protegerlo (EdDSA + Cookies + Fingerprint)
func ValidateJWT(next http.HandlerFunc) http.HandlerFunc {
	
	return func(w http.ResponseWriter, r *http.Request) {
		
		// 1. Extraer el Pasaporte y la Huella desde las Cookies
		// (Asegúrate de que los nombres coinciden con las cookies de Spring Boot)
		jwtCookie, errJWT := r.Cookie("accessToken")
		fgpCookie, errFGP := r.Cookie("fingerprint")
		
		if errJWT != nil || errFGP != nil {
			fmt.Println("🔴 Acceso Denegado: Faltan cookies de seguridad")
			http.Error(w, "Acceso Denegado", http.StatusUnauthorized)
			return
		}

		// 2. Parseamos la clave pública de texto a formato criptográfico de Go
		parsedKey, err := jwt.ParseEdPublicKeyFromPEM([]byte(config.Envs.RSAPublicKey))
		if err != nil {
			fmt.Println("❌ Error de Servidor: Llave pública mal formateada", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		// 3. Comprobamos matemáticamente el token contra nuestra llave pública
		token, err := jwt.Parse(jwtCookie.Value, func(t *jwt.Token) (interface{}, error) {
			//Protección extra: confirmar que nadie nos ha colado un token firmado con otro algoritmo
			if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
				return nil, fmt.Errorf("método de firma hackeado")
			}
			return parsedKey, nil
		})

		if err != nil || !token.Valid {
			fmt.Println("🔴 Acceso bloqueado: Token falso o caducado.")
			http.Error(w, "Token inválido", http.StatusUnauthorized)
			return
		}

		// 4. EL SEGURO ANTIRROBO: Validar si la Huella de la Cookie es del dueño del JWT
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Token dañado", http.StatusUnauthorized)
			return
		}

		// Leemos la copia de la huella que escondió Spring Boot
		hashEnJWT, ok := claims["fgp_hash"].(string)
		if !ok {
			fmt.Println("🔴 Denegado: Token sin tecnología Fingerprint")
			http.Error(w, "Token sin seguridad", http.StatusUnauthorized)
			return
		}

		// Trituramos (SHA256) la cookie real que nos acaba de mandar el paciente
		hashCreator := sha256.New()
		hashCreator.Write([]byte(fgpCookie.Value))
		hashCalculado := hex.EncodeToString(hashCreator.Sum(nil))

		// Si las dos huellas no son idénticas... ¡Patada en la puerta!
		if hashCalculado != hashEnJWT {
			fmt.Println("🚨 ATAQUE BLOQUEADO: La huella real no cuadra con la del JWT")
			http.Error(w, "🔴 Denegado: Posible robo de sesión", http.StatusUnauthorized)
			return
		}

		// 5. ¡VISTA BUENA! Todo cuadra matemáticamente
		fmt.Println("✅ Seguridad pasadísima. JWT Ed25519 y Huella coinciden.")
		next.ServeHTTP(w, r)
	}
}
