package handlers

import (
	"P1-BASESII/internal/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ==========================================
// REQUISITO DE RÚBRICA: Pruebas de Integración
// Validar el correcto funcionamiento entre los servicios (Flujo Completo)
// ==========================================
func TestIntegration_CompleteFlow(t *testing.T) {
	// Inyectamos Mocks
	database.UserRepo = &MockUserRepo{}
	database.RestaurantRepo = &MockRestaurantRepo{}
	database.MenuRepo = &MockMenuRepo{}
	database.OrderRepo = &MockOrderRepo{}

	// 1. Iniciamos el motor usando la función del archivo principal
	r, mr := setupRouter()
	defer mr.Close()

	r.POST("/restaurants", CreateRestaurant)
	r.POST("/menus", CreateMenuItem)
	r.POST("/orders", CreateOrder)
	r.PUT("/orders/:id", UpdateOrderStatus)

	// PASO 1: Admin crea un Restaurante
	req1, _ := http.NewRequest("POST", "/restaurants", bytes.NewBuffer([]byte(`{"name": "Restaurante Integracion"}`)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-User", "admin_user")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Errorf("Integración Fallida en Crear Restaurante: %d", w1.Code)
	}

	// PASO 2: Admin agrega un Plato al Menú
	req2, _ := http.NewRequest("POST", "/menus", bytes.NewBuffer([]byte(`{"restaurant_id": 1, "name": "Plato Int", "price": 10}`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User", "admin_user")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 201 {
		t.Errorf("Integración Fallida en Crear Menú: %d", w2.Code)
	}

	// PASO 3: Cliente crea una Orden para ese Plato
	req3, _ := http.NewRequest("POST", "/orders", bytes.NewBuffer([]byte(`{"menu_item_id": 1}`)))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-User", "client_user") // Cambiamos de rol
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != 201 {
		t.Errorf("Integración Fallida en Crear Orden: %d", w3.Code)
	}

	// PASO 4: Admin marca la Orden como Completada
	req4, _ := http.NewRequest("PUT", "/orders/1", bytes.NewBuffer([]byte(`{"status": "completed"}`)))
	req4.Header.Set("Content-Type", "application/json")
	req4.Header.Set("X-User", "admin_user")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != 200 {
		t.Errorf("Integración Fallida en Actualizar Orden: %d", w4.Code)
	}
}

// ==========================================
// REQUISITO DE RÚBRICA: Validar uso de datos LLM
// ==========================================
func TestIntegration_LLMDataProcessing(t *testing.T) {
	database.UserRepo = &MockUserRepo{}
	database.RestaurantRepo = &MockRestaurantRepo{}
	database.MenuRepo = &MockMenuRepo{}

	r, mr := setupRouter()
	defer mr.Close()

	r.POST("/menus", CreateMenuItem)

	// Iteramos sobre el array generado por LLM para ingresarlo al sistema
	for _, item := range LLMGeneratedMenuItems {
		// Asignamos el ID del restaurante mockeado (1) a los datos del LLM
		payload := map[string]interface{}{
			"restaurant_id": 1,
			"name":          item.Name,
			"category":      item.Category,
			"description":   item.Description,
			"price":         item.Price,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/menus", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User", "admin_user")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Falló la integración de datos LLM con el plato %s: Esperado 201, se obtuvo %d", item.Name, w.Code)
		}
	}
}
