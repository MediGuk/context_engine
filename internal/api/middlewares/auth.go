package middlewares

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"context_engine/internal/api/context_keys"
	"context_engine/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateJWT envuelve a un Handler para protegerlo (EdDSA + Cookies + Fingerprint)
func ValidateJWT(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// 1. EXTRAER JWT (DE LA CABECERA) Y FINGERPRINT (DE LA COOKIE)

		// A. Extraer JWT del Header Bearer (mejor que split que un hacker puede mandar millones de vacio)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			fmt.Println("🔴 Denegado: Falta token Bearer en cabecera")
			http.Error(w, "Acceso Denegado", http.StatusUnauthorized)
			return
		}
		tokenRecibido := authHeader[7:]

		// B. Extraer lCookie (Fingerprint)
		fgpCookie, errFGP := r.Cookie("fingerprint")
		if errFGP != nil {
			fmt.Println("🔴 Acceso Denegado: Falta cookie de seguridad Fingerprint")
			http.Error(w, "Acceso Denegado", http.StatusUnauthorized)
			return
		}

		// 2. Parse public_key de texto a formato criptográfico de Go
		parsedKey, err := jwt.ParseEdPublicKeyFromPEM([]byte(config.Envs.JwtPublicKey))
		if err != nil {
			fmt.Println("❌ Error de Servidor: Llave pública mal formateada", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		// 3. Token vs public_key (check matematico)
		token, err := jwt.Parse(tokenRecibido, func(t *jwt.Token) (interface{}, error) {
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

		// --- NUEVO: Extraer Identidad del Usuario para Redis ---
		userID, _ := claims["sub"].(string)
		// "sub" es el estándar de JWT que Java rellena con .subject()
		// --------------------------------------------------------

		// Leemos la copia de la huella que escondió Spring Boot
		hashEnJWT, ok := claims["fingerprintHash"].(string)
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

		fmt.Printf("✅ Seguridad pasadísima. JWT de Usuario [%s] validado.\n", userID)

		// 5. Inyectamos el userID en el "contexto"(mochila) de ESTA petición (para que el Handler lo pueda leer)

		// 5.1 Extraer "context"(mochila) de la peticion actual
		originalCtx := r.Context()
		// 5.2 Crear key especial para evitar colisiones
		newCtx := context.WithValue(originalCtx, context_keys.UserIDKey, userID)

		// 5.4 Crear una nueva peticion con el nuevo contexto
		newRequest := r.WithContext(newCtx)
		// 5.5 Llamar al siguiente handler con la nueva peticion
		next.ServeHTTP(w, newRequest)
	}
}
