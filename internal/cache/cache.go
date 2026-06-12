package cache

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var ctx = context.Background()

// Expirations: políticas de expiración por tipo de dato.
const (
	TTLRestaurants = 24 * time.Hour // Restaurantes y menús cambian poco
	TTLMenus       = 24 * time.Hour
	TTLOrders      = 5 * time.Minute // Órdenes cambian frecuentemente
	TTLSearch      = 1 * time.Hour   // Búsquedas: balance entre frescura y performance
)

// Connect: Inicializa el cliente Redis.
// Entradas: Ninguna (lee REDIS_URL del entorno).
// Salidas: Ninguna.
// Funcionalidad: Parsea la URL y conecta al servidor Redis.
// Casos: Conexión exitosa, panic si falla.
func Connect() {
	opts, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic("Redis connection failed: " + err.Error())
	}
	Client = redis.NewClient(opts)
	if err := Client.Ping(ctx).Err(); err != nil {
		panic("Redis ping failed: " + err.Error())
	}
}

// Set: Guarda un valor serializado en Redis.
// Entradas: key (string), value (any), ttl (duración).
// Salidas: error.
// Funcionalidad: Serializa a JSON y almacena con expiración.
// Casos: Éxito, error de serialización, error de Redis.
func Set(key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, ttl).Err()
}

// Get: Obtiene y deserializa un valor de Redis.
// Entradas: key (string), dest (puntero al tipo esperado).
// Salidas: error (redis.Nil si no existe).
// Funcionalidad: Busca la clave y deserializa el JSON al destino.
// Casos: Hit, miss (redis.Nil), error de Redis.
func Get(key string, dest any) error {
	data, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return err // redis.Nil = cache miss
	}
	return json.Unmarshal(data, dest)
}

// Delete: Elimina una o varias claves de Redis.
// Entradas: keys (variadic string).
// Salidas: error.
// Funcionalidad: Invalida claves específicas del caché.
// Casos: Éxito, clave inexistente (no es error), error de Redis.
func Delete(keys ...string) error {
	return Client.Del(ctx, keys...).Err()
}

// DeleteByPattern: Elimina todas las claves que coincidan con un patrón.
// Entradas: pattern (string, e.g. "restaurants:*").
// Salidas: error.
// Funcionalidad: Escanea y elimina claves en lote — usado para invalidación masiva.
// Casos: Éxito sin claves, éxito con claves borradas, error de scan.
func DeleteByPattern(pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := Client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			Client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
