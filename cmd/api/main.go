package main

import (
	"fmt"      //manipular textos y pintar en consola
	"log"      //manejar errores
	"net/http" //manejar peticiones web

	"context_engine/internal/api"             //importamos el router y asi ....
	"context_engine/internal/api/middlewares" // Importamos el CORS
	"context_engine/internal/config"          //importar el env y asi ...
)

//fmt.Println() -> para imprimir en consola normal
//fmt.Fprintln() -> para imprimir en la respuesta HTTP
//fmt.Printf() -> para imprimir con formato %s, %d, %f, etc EXTRA.....

func main() {
	// 1. Cargar y validar variables de entorno
	config.LoadEnv()

	// 2. Todas las rutas de api (variable router en new objeto api)
	router := api.SetupRouter()

	// 3. Encender el servidor en el puerto 8090
	fmt.Println("🚀 Servidor arrancando en http://localhost:8090 ...")

	// Envolvemos el router entero dentro de EnableCORS
	err := http.ListenAndServe(":8090", middlewares.EnableCORS(router))
	if err != nil {
		log.Fatal("❌ Error fatal:", err)
	}
}
