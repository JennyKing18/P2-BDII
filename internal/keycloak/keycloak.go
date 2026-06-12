package keycloak

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// getPublicKey: Obtiene la clave pública RSA de Keycloak.
// Entradas: Ninguna.
// Salidas: Clave pública RSA, error.
// Funcionalidad: Consulta el endpoint JWKS de Keycloak y decodifica la clave.
// Casos: Éxito, error de conexión, no hay claves.
func getPublicKey() (*rsa.PublicKey, error) {
	url := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
		os.Getenv("KEYCLOAK_URL"),
		os.Getenv("KEYCLOAK_REALM"),
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Keys []struct {
			N   string `json:"n"`
			E   string `json:"e"`
			Use string `json:"use"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Keys) == 0 {
		return nil, errors.New("no keys found")
	}

	var key struct{ N, E string }
	for _, k := range result.Keys {
		if k.Use == "sig" {
			key.N = k.N
			key.E = k.E
			break
		}
	}
	if key.N == "" {
		return nil, errors.New("no signing key found")
	}

	nBytes, _ := base64.RawURLEncoding.DecodeString(key.N)
	eBytes, _ := base64.RawURLEncoding.DecodeString(key.E)

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}

// AuthMiddleware: Middleware para validar JWT de Keycloak.
// Entradas: Ninguna (devuelve HandlerFunc).
// Salidas: HandlerFunc de Gin.
// Funcionalidad: Verifica el token JWT y extrae claims.
// Casos: Token válido, inválido, faltante.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Falta Authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			c.JSON(401, gin.H{"error": "Formato inválido"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		publicKey, err := getPublicKey()
		if err != nil {
			c.JSON(500, gin.H{"error": "Error obteniendo clave pública"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("algoritmo inválido")
			}
			return publicKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Token inválido"})
			fmt.Println("JWT Error:", err)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(401, gin.H{"error": "Claims inválidos"})
			c.Abort()
			return
		}

		// Validar issuer
		expectedIssuer := fmt.Sprintf("%s/realms/%s",
			os.Getenv("KEYCLOAK_EXTERNAL_URL"),
			os.Getenv("KEYCLOAK_REALM"),
		)

		if claims["iss"] != expectedIssuer {
			c.JSON(401, gin.H{"error": "Issuer inválido"})
			c.Abort()
			return
		}

		fmt.Println("Full claims:", claims)
		// Extraer roles
		var roles []string
		if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
			if r, ok := realmAccess["roles"].([]interface{}); ok {
				for _, role := range r {
					roles = append(roles, role.(string))
				}
			}
		}
		if resourceAccess, ok := claims["resource_access"].(map[string]interface{}); ok {
			for _, clientRoles := range resourceAccess {
				if clientMap, ok := clientRoles.(map[string]interface{}); ok {
					if r, ok := clientMap["roles"].([]interface{}); ok {
						for _, role := range r {
							roles = append(roles, role.(string))
						}
					}
				}
			}
		}
		c.Set("user_sub", claims["sub"])            // ID único de Keycloak
		c.Set("user", claims["preferred_username"]) // username
		c.Set("roles", roles)

		c.Next()
	}
}

// ---------------- ROLE MIDDLEWARE ----------------
// Verifica roles
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {

		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(403, gin.H{"error": "No hay roles"})
			c.Abort()
			return
		}
		fmt.Println(" Roles received:", roles)
		fmt.Println("Required role:", requiredRole)

		for _, r := range roles.([]string) {
			if r == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(403, gin.H{"error": "No tienes permisos"})
		c.Abort()
	}
}
