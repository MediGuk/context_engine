package main

import (
	"fmt" //manipular textos y pintar en consola
	"log" //manejar errores
	"net/http" //manejar peticiones web
)

//fmt.Println() -> para imprimir en consola normal
//fmt.Fprintln() -> para imprimir en la respuesta HTTP
//fmt.Printf() -> para imprimir con formato %s, %d, %f, etc EXTRA.....

func main() {
	// 1. Configuramos una ruta de prueba súper tonta
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "¡El servidor de Go está vivo bro!")
	})
	// 2. Encendemos el motor en el puerto 8080
	fmt.Println("🚀 Servidor arrancando en http://localhost:8090 ...")
	
	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatal("❌ Error fatal:", err)
	}
}
