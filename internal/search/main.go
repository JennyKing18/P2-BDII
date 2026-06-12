package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main: Punto de entrada del microservicio de búsqueda.
// Entradas: Ninguna.
// Salidas: Ninguna.
// Funcionalidad: Conecta a ES, asegura el índice products y levanta el servidor HTTP.
// Casos: Inicio normal, fatal si ES no responde al arrancar.
func main() {
	godotenv.Load()

	if err := connectES(); err != nil {
		log.Fatalf("No se pudo conectar a ElasticSearch: %v", err)
	}
	log.Println("ElasticSearch conectado")

	if err := ensureIndex(); err != nil {
		log.Fatalf("Error creando índice: %v", err)
	}
	log.Println("Índice 'products' listo")

	r := gin.Default()

	// ── Health check ──────────────────────────────────────────────────────
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "search-service OK"})
	})

	// ── Búsqueda ──────────────────────────────────────────────────────────
	r.GET("/products", SearchProducts)                       // ?q=texto — búsqueda textual
	r.GET("/products/category/:categoria", SearchByCategory) // filtro por categoría exacta

	// ── Indexación ────────────────────────────────────────────────────────
	r.POST("/reindex", Reindex)                // reindexar batch manualmente
	r.DELETE("/products/:id", DeleteFromIndex) // eliminar producto del índice

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Search service corriendo en :%s", port)
	r.Run(":" + port)
}
