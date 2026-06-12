package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

var esClient *elasticsearch.Client

// connectES: Inicializa y verifica la conexión con ElasticSearch.
// Entradas: Ninguna (usa variable de entorno ES_URL).
// Salidas: error si ES no responde.
func connectES() error {
	cfg := elasticsearch.Config{
		Addresses: []string{os.Getenv("ES_URL")},
	}

	var err error
	esClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("error creando cliente ES: %w", err)
	}

	res, err := esClient.Ping()
	if err != nil || res.IsError() {
		return fmt.Errorf("ElasticSearch no responde en %s", os.Getenv("ES_URL"))
	}

	return nil
}

// ensureIndex: Crea el índice "products" en ES si no existe.
// Entradas: Ninguna.
// Salidas: error si falla la creación.
// Funcionalidad: Define mapping con tipos adecuados para búsqueda textual y filtro por categoría.
//   - name/description: "text" → búsqueda full-text con fuzziness
//   - category: "keyword" → filtro exacto
//
// Casos: Índice ya existe (skip), creado exitosamente, error.
func ensureIndex() error {
	res, err := esClient.Indices.Exists([]string{"products"})
	if err != nil {
		return fmt.Errorf("error verificando índice: %w", err)
	}
	if res.StatusCode == 200 {
		return nil // ya existe, no hacer nada
	}

	mapping := `{
		"mappings": {
			"properties": {
				"id":          { "type": "integer" },
				"name":        { "type": "text" },
				"category":    { "type": "keyword" },
				"description": { "type": "text" }
			}
		}
	}`

	res, err = esClient.Indices.Create(
		"products",
		esClient.Indices.Create.WithBody(bytes.NewBufferString(mapping)),
	)
	if err != nil || res.IsError() {
		return fmt.Errorf("error creando índice products: %v", err)
	}

	return nil
}

// indexProduct: Indexa un producto en ElasticSearch.
// Entradas: Product a indexar.
// Salidas: error si falla la serialización o el insert en ES.
// Funcionalidad: Asigna descripción default si viene vacía, luego indexa por ID de documento.
// Casos: Éxito, error de serialización, error de ES.
func indexProduct(p Product) error {
	if p.Description == "" {
		p.Description = "Producto sin descripción"
	}

	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	res, err := esClient.Index(
		"products",
		bytes.NewReader(data),
		esClient.Index.WithDocumentID(fmt.Sprintf("%d", p.ID)),
		esClient.Index.WithContext(context.Background()),
	)
	if err != nil || res.IsError() {
		return fmt.Errorf("error indexando producto %d: %v", p.ID, err)
	}

	return nil
}

// deleteProduct: Elimina un producto del índice ES por ID.
// Entradas: id uint del producto a eliminar.
// Salidas: error si falla.
// Funcionalidad: Hard delete del documento en ES — no afecta la DB principal.
// Casos: Éxito, documento no existía (no error), error de ES.
func deleteProduct(id uint) error {
	res, err := esClient.Delete(
		"products",
		fmt.Sprintf("%d", id),
		esClient.Delete.WithContext(context.Background()),
	)
	if err != nil || res.IsError() {
		return fmt.Errorf("error eliminando producto %d de ES: %v", id, err)
	}
	return nil
}

// searchByText: Busca productos por texto libre en nombre, categoría y descripción.
// Entradas: query string con el texto a buscar.
// Salidas: []Product, total de resultados, error.
// Funcionalidad: Multi-match con fuzziness AUTO para tolerar typos.
//   - name tiene peso x3 (más relevante que descripción o categoría)
//
// Casos: Con resultados, sin resultados, error de ES.
func searchByText(query string) ([]Product, int, error) {
	body := fmt.Sprintf(`{
		"query": {
			"multi_match": {
				"query": %q,
				"fields": ["name^3", "category^2", "description"],
				"fuzziness": "AUTO"
			}
		}
	}`, query)

	res, err := esClient.Search(
		esClient.Search.WithIndex("products"),
		esClient.Search.WithBody(bytes.NewBufferString(body)),
	)
	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("error buscando productos: %v", err)
	}
	defer res.Body.Close()

	return parseHits(res.Body)
}

// searchByCategory: Filtra productos por categoría exacta.
// Entradas: category string (debe coincidir exactamente con el valor indexado).
// Salidas: []Product, total, error.
// Funcionalidad: Term query sobre campo keyword → match exacto, case-sensitive.
// Casos: Con resultados, sin resultados, error de ES.
func searchByCategory(category string) ([]Product, int, error) {
	body := fmt.Sprintf(`{
		"query": {
			"term": {
				"category": %q
			}
		}
	}`, category)

	res, err := esClient.Search(
		esClient.Search.WithIndex("products"),
		esClient.Search.WithBody(bytes.NewBufferString(body)),
	)
	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("error filtrando por categoría: %v", err)
	}
	defer res.Body.Close()

	return parseHits(res.Body)
}

// parseHits: Convierte la respuesta cruda de ES en lista de Products.
// Entradas: Body del response ES (io.Reader).
// Salidas: []Product, total de hits, error de deserialización.
func parseHits(body interface{ Read([]byte) (int, error) }) ([]Product, int, error) {
	var raw map[string]interface{}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, 0, err
	}

	hits := raw["hits"].(map[string]interface{})
	total := int(hits["total"].(map[string]interface{})["value"].(float64))
	innerHits := hits["hits"].([]interface{})

	var results []Product
	for _, h := range innerHits {
		src := h.(map[string]interface{})["_source"]
		data, _ := json.Marshal(src)
		var p Product
		json.Unmarshal(data, &p)
		results = append(results, p)
	}

	return results, total, nil
}
