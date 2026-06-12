package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"P1-BASESII/internal/cache"
	"P1-BASESII/internal/database"
	"P1-BASESII/internal/handlers"
	"P1-BASESII/internal/keycloak"
	"P1-BASESII/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main: Punto de entrada de la aplicación.
// Entradas: Ninguna.
// Salidas: Ninguna.
// Funcionalidad: Inicializa repos, BD, rutas y levanta el servidor.
// Casos: Inicio normal, errores de conexión.
func main() {
	godotenv.Load()    // carga .env primero
	database.Connect() // conecta con Postgres
	cache.Connect()

	log.Println("XX DB_DRIVER:", os.Getenv("DB_DRIVER"))
	log.Println("XX MONGO_URI:", os.Getenv("MONGO_URI"))
	log.Println("XX userRepo:", database.UserRepo)

	if os.Getenv("DB_DRIVER") == "mongo" {
		database.ConnectMongo()
	}

	// Inicializar repositorios (una sola vez, después de conectar la BD)
	database.InitRepositories()

	r := gin.Default()

	// ── Endpoints de prueba ───────────────────────────────────────────────
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Sirve, creo"})
	})

	r.GET("/private", keycloak.AuthMiddleware(),
		func(c *gin.Context) { c.JSON(200, gin.H{"message": "ruta privada"}) })

	r.GET("/admin", keycloak.AuthMiddleware(), keycloak.RoleMiddleware("admin"),
		func(c *gin.Context) { c.JSON(200, gin.H{"message": "solo admins"}) })

	// ── Auth (rutas públicas sin JWT) ─────────────────────────────────────

	// Registro de nuevo usuario
	r.POST("/auth/register", func(c *gin.Context) {
		var input struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"` // "admin" o "client"
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := createUserKeycloak(input.Username, input.Email, input.Password, input.Role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		user := models.User{
			Username: input.Username,
			Email:    input.Email,
			Role:     input.Role,
		}
		if err := database.UserRepo.Create(&user); err != nil {
			fmt.Println("Error saving to DB:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving user to DB"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Usuario creado correctamente"})
	})

	// Inicio de sesión — devuelve JWT de Keycloak
	r.POST("/auth/login", func(c *gin.Context) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		data := fmt.Sprintf(
			"grant_type=password&client_id=account&username=%s&password=%s&scope=openid+profile+roles",
			input.Username, input.Password,
		)
		resp, err := http.Post(
			fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
				os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM")),
			"application/x-www-form-urlencoded",
			bytes.NewBufferString(data),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error conectando a Keycloak"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Keycloak says:", string(body))

		var tokenResp map[string]interface{}
		json.Unmarshal(body, &tokenResp)

		if _, ok := tokenResp["access_token"]; !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
			return
		}

		c.JSON(http.StatusOK, tokenResp)
	})

	// ── Rutas autenticadas (JWT válido) ───────────────────────────────────
	auth := r.Group("/").Use(keycloak.AuthMiddleware())
	{
		// Restaurantes
		auth.GET("/restaurants", handlers.GetRestaurants)
		auth.GET("/restaurants/:id", handlers.GetRestaurant)

		// Usuario autenticado
		auth.GET("/users/me", func(c *gin.Context) {
			username := c.GetString("user")
			user, err := database.UserRepo.FindByUsername(username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
				return
			}
			c.JSON(http.StatusOK, user)
		})

		// Reservaciones
		auth.POST("/reservations", handlers.CreateReservation)
		auth.GET("/reservations", handlers.GetReservations)
		auth.GET("/reservations/:id", handlers.GetReservation)
		auth.PUT("/reservations/:id", handlers.UpdateReservation)
		auth.DELETE("/reservations/:id", handlers.DeleteReservation)

		// Menús
		auth.GET("/menus", handlers.GetMenuItems)
		auth.GET("/menus/:id", handlers.GetMenuItem)

		// Órdenes
		auth.POST("/orders", handlers.CreateOrder)
		auth.GET("/orders", handlers.GetOrder)
		auth.GET("/orders/:id", handlers.GetOrder)
		auth.PUT("/orders/:id", handlers.UpdateOrderStatus)
	}

	// ── Rutas solo admin ──────────────────────────────────────────────────
	adminRoutes := r.Group("/").Use(
		keycloak.AuthMiddleware(),
		keycloak.RoleMiddleware("admin"),
	)
	{
		// Usuarios
		adminRoutes.GET("/users", handlers.GetUsers)
		adminRoutes.GET("/users/:id", handlers.GetUser)
		adminRoutes.PUT("/users/:id", handlers.UpdateUser)
		adminRoutes.DELETE("/users/:id", handlers.DeleteUser)

		// Restaurantes
		adminRoutes.POST("/restaurants", handlers.CreateRestaurant)
		adminRoutes.PUT("/restaurants/:id", handlers.UpdateRestaurant)
		adminRoutes.DELETE("/restaurants/:id", handlers.DeleteRestaurant)

		// Menús
		adminRoutes.POST("/menus", handlers.CreateMenuItem)
		adminRoutes.PUT("/menus/:id", handlers.UpdateMenuItem)
		adminRoutes.DELETE("/menus/:id", handlers.DeleteMenuItem)
	}

	r.Run(":" + os.Getenv("PORT"))
}

// ═════════════════════════════════════════════════════════════════════════════
// HELPERS KEYCLOAK
// ═════════════════════════════════════════════════════════════════════════════

// getAdminToken: Obtiene un token de administrador de Keycloak.
// Entradas: Ninguna.
// Salidas: Token de acceso (string), error.
// Funcionalidad: POST a master realm con admin-cli para obtener token admin.
// Casos: Éxito, error de conexión, credenciales inválidas.
func getAdminToken() (string, error) {
	data := fmt.Sprintf(
		"grant_type=password&client_id=admin-cli&username=%s&password=%s",
		os.Getenv("KC_ADMIN"),
		os.Getenv("KC_ADMIN_PASSWORD"),
	)
	resp, err := http.Post(
		fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", os.Getenv("KEYCLOAK_URL")),
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

// createUserKeycloak: Crea un usuario en Keycloak con contraseña y rol.
// Entradas: username, email, password, role (strings).
// Salidas: error.
// Funcionalidad: Crea usuario, asigna contraseña y rol via Admin REST API.
// Casos: Éxito, error en cada paso (creación, password, rol).
func createUserKeycloak(username, email, password, role string) error {
	token, err := getAdminToken()
	if err != nil {
		return fmt.Errorf("no se pudo obtener token admin: %v", err)
	}

	userJSON, _ := json.Marshal(map[string]interface{}{
		"username": username,
		"email":    email,
		"enabled":  true,
	})
	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/admin/realms/%s/users", os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM")),
		bytes.NewBuffer(userJSON),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error creando usuario: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error creando usuario: %s", body)
	}

	id, err := getUserID(username, token)
	if err != nil {
		return fmt.Errorf("error obteniendo ID usuario: %v", err)
	}

	if err := setPassword(id, password, token); err != nil {
		return fmt.Errorf("error asignando contraseña: %v", err)
	}

	if err := assignRole(id, role, token); err != nil {
		return fmt.Errorf("error asignando rol: %v", err)
	}

	return nil
}

// getUserID: Obtiene el ID de un usuario en Keycloak por username.
// Entradas: username (string), token (string).
// Salidas: ID del usuario (string), error.
// Funcionalidad: GET a la Admin API filtrando por username.
// Casos: Encontrado, no encontrado, error de red.
func getUserID(username, token string) (string, error) {
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/admin/realms/%s/users?username=%s",
			os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM"), username),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var users []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &users); err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", fmt.Errorf("usuario no encontrado")
	}
	return users[0].ID, nil
}

// setPassword: Establece la contraseña de un usuario en Keycloak.
// Entradas: userID, password, token (strings).
// Salidas: error.
// Funcionalidad: PUT al endpoint reset-password con temporary=false.
// Casos: Éxito (204), error de Keycloak.
func setPassword(userID, password, token string) error {
	bodyJSON, _ := json.Marshal(map[string]interface{}{
		"type":      "password",
		"value":     password,
		"temporary": false,
	})
	req, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/admin/realms/%s/users/%s/reset-password",
			os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM"), userID),
		bytes.NewBuffer(bodyJSON),
	)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error asignando contraseña: %s", body)
	}
	return nil
}

// assignRole: Asigna un rol de realm a un usuario en Keycloak.
// Entradas: userID, roleName, token (strings).
// Salidas: error.
// Funcionalidad: Obtiene el rol por nombre y lo asigna via role-mappings.
// Casos: Éxito (204), rol no encontrado, error de asignación.
func assignRole(userID, roleName, token string) error {
	req, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/admin/realms/%s/roles/%s",
			os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM"), roleName),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var role struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(body, &role)
	if role.ID == "" {
		return fmt.Errorf("role '%s' not found", roleName)
	}

	roleBody, _ := json.Marshal([]map[string]interface{}{
		{"id": role.ID, "name": role.Name},
	})
	req2, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/admin/realms/%s/users/%s/role-mappings/realm",
			os.Getenv("KEYCLOAK_URL"), os.Getenv("KEYCLOAK_REALM"), userID),
		bytes.NewBuffer(roleBody),
	)
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 204 {
		b, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("error assigning role: %s", b)
	}
	return nil
}
