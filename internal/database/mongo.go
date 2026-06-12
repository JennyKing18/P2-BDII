package database

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var MongoClient *mongo.Client

// ConnectMongo: Inicializa el cliente MongoDB.
// Entradas: Ninguna (lee MONGO_URI del entorno).
// Salidas: Ninguna.
// Funcionalidad: Conecta al cluster sharded via mongos router con autenticación.
// Casos: Conexión exitosa, panic si falla.
func ConnectMongo() {
	uri := os.Getenv("MONGO_URI")
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		panic("MongoDB connection failed: " + err.Error())
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		panic("MongoDB ping failed: " + err.Error())
	}
	MongoClient = client
}
