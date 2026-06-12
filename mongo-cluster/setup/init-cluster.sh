#!/bin/bash

# ==============================================================================
# Script de Inicialización: MongoDB Sharded Cluster
# Propósito: Registrar los shards en el enrutador (mongos) y configurar
#            la estrategia de partición para las colecciones de la aplicación.
# ==============================================================================

echo "Verificando disponibilidad y credenciales del Router (mongos)..."

# Espera hasta que el enrutador acepte conexiones autenticadas
until mongosh mongodb://admin:password123@mongo-router:27017/admin --quiet --eval "db.adminCommand('ping').ok" > /dev/null 2>&1; do
  sleep 5
done

echo "[OK] Router operativo y autenticación exitosa."


# --- Registro de Nodos de Datos ---

echo "Registrando Replica Sets como Shards..."

# Se utiliza try/catch para garantizar idempotencia (evita errores si el script se ejecuta más de una vez)
mongosh mongodb://admin:password123@mongo-router:27017/admin --quiet --eval '
try { sh.addShard("shard1RS/mongo-shard1-1:27017,mongo-shard1-2:27017,mongo-shard1-3:27017") } catch(e) {}
try { sh.addShard("shard2RS/mongo-shard2-1:27017,mongo-shard2-2:27017,mongo-shard2-3:27017") } catch(e) {}
' > /dev/null


# --- Configuración de Base de Datos y Colecciones ---

echo "Aplicando estrategia de Sharding a la base de datos 'restaurantDB'..."

# Se habilita sharding a nivel de BD y se aplica particionamiento por hash en colecciones
# El uso de 'hashed' garantiza una distribución equitativa de los documentos entre los shards
mongosh mongodb://admin:password123@mongo-router:27017/admin --quiet --eval '
sh.enableSharding("restaurantDB");
sh.shardCollection("restaurantDB.restaurants", { _id: "hashed" });
sh.shardCollection("restaurantDB.products", { _id: "hashed" });
sh.shardCollection("restaurantDB.reservations", { _id: "hashed" });
sh.shardCollection("restaurantDB.users", { _id: "hashed" });
' > /dev/null

echo "[OK] Clúster de MongoDB particionado y asegurado correctamente."