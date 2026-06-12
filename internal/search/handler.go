package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SearchProducts: Busca productos por texto libre en ElasticSearch.
// Endpoint: GET /search/products?q=texto
// Entradas: Query param "q" con el texto a buscar.
// Salidas: 200 + {total, products}, 400 si falta "q", 500 si ES falla.
// Funcionalidad: Multi-match con fuzziness — tolera typos y busca en nombre, categoría y descripción.
func SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'q' requerido"})
		return
	}

	products, total, err := searchByText(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"products": products,
	})
}

// SearchByCategory: Filtra productos por categoría exacta.
// Endpoint: GET /search/products/category/:categoria
// Entradas: Path param "categoria" (ej: "meat", "vegan", "drink").
// Salidas: 200 + {total, products}, 500 si ES falla.
// Funcionalidad: Term query exacto sobre campo keyword — case-sensitive.
func SearchByCategory(c *gin.Context) {
	category := c.Param("categoria")

	products, total, err := searchByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"products": products,
	})
}

// Reindex: Reindexar un batch de productos manualmente.
// Endpoint: POST /search/reindex
// Entradas: Body JSON con array de productos [{id, name, category, description}].
// Salidas: 200 + {message, indexed, total}, 400 si body inválido.
// Funcionalidad: Itera la lista e indexa cada producto — errores individuales se loguean
//
//	pero no abortan el resto (indexa los que pueda).
//
// Casos: Éxito total, éxito parcial (algunos fallaron), body inválido.
func Reindex(c *gin.Context) {
	var products []Product
	if err := c.ShouldBindJSON(&products); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	indexed := 0
	for _, p := range products {
		if err := indexProduct(p); err != nil {
			c.Error(err) // loguea pero no aborta
			continue
		}
		indexed++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reindexado completado",
		"indexed": indexed,
		"total":   len(products),
	})
}

// DeleteFromIndex: Elimina un producto del índice ES por ID.
// Endpoint: DELETE /search/products/:id
// Entradas: Path param "id" (uint).
// Salidas: 200 + mensaje, 400 si ID inválido, 500 si ES falla.
// Funcionalidad: Usado por el API principal al borrar un MenuItem para mantener ES sincronizado.
// Casos: Éxito, ID inválido, error de ES.
func DeleteFromIndex(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := deleteProduct(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Producto %d eliminado del índice", id)})
}
