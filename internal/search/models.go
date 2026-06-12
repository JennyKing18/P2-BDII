package main

// Product: Modelo de documento indexado en ElasticSearch.
// Representa un item del menú con los campos relevantes para búsqueda.
// ID debe coincidir con el ID del MenuItem en la base de datos principal.
type Product struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}
