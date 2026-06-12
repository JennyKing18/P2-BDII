#!/bin/bash

# ==============================================================================
# Script de Orquestación: Despliegue de Microservicios y MongoDB Distribuido
# Propósito: Aprovisionar contenedores, inicializar Replica Sets, configurar RBAC 
#            y aplicar reglas de particionamiento (Sharding).
# ==============================================================================

# Definición de colores
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
MAGENTA='\033[1;35m'
NC='\033[0m' # No Color

echo -e "${CYAN}  Levantando Infraestructura del Restaurante...${NC}"


# 1. Levantar todo el ecosistema de Docker en segundo plano
docker compose up -d --build

echo -e "\n${YELLOW}[+] Esperando 15 segundos a que los contenedores arranquen...${NC}"
sleep 15

# 2. Iniciar los Replica Sets (bypasseando la seguridad localmente)
echo -e "\n${GREEN}[+] Configurando Cerebro y Shards...${NC}"
docker exec mongo-config-1 mongosh --quiet --eval "rs.initiate({ _id: 'configRS', configsvr: true, members: [{ _id: 0, host: 'mongo-config-1:27017' }, { _id: 1, host: 'mongo-config-2:27017' }, { _id: 2, host: 'mongo-config-3:27017' }] })" > /dev/null 2>&1
docker exec mongo-shard1-1 mongosh --quiet --eval "rs.initiate({ _id: 'shard1RS', members: [{ _id: 0, host: 'mongo-shard1-1:27017' }, { _id: 1, host: 'mongo-shard1-2:27017' }, { _id: 2, host: 'mongo-shard1-3:27017' }] })" > /dev/null 2>&1
docker exec mongo-shard2-1 mongosh --quiet --eval "rs.initiate({ _id: 'shard2RS', members: [{ _id: 0, host: 'mongo-shard2-1:27017' }, { _id: 1, host: 'mongo-shard2-2:27017' }, { _id: 2, host: 'mongo-shard2-3:27017' }] })" > /dev/null 2>&1

echo -e "${YELLOW}[+] Nodos iniciados. Esperando a que elijan a sus líderes (15 seg)...${NC}"
sleep 15

# 3. Crear el Usuario Administrador en el router local
echo -e "\n${GREEN}[+] Creando Usuario de Seguridad...${NC}"
docker exec mongo-router mongosh --quiet --eval "db.getSiblingDB('admin').createUser({user: 'admin', pwd: 'password123', roles: ['root']})" > /dev/null 2>&1

# 4. Refrescar el script de setup para que detecte al usuario y termine el sharding
echo -e "\n${CYAN}[+] Autenticación lista. Finalizando configuración de Sharding...${NC}"
docker compose restart mongo-setup


echo -e "${MAGENTA} ¡Proyecto listo!${NC}"
echo -e "${MAGENTA} API de Go ya puede conectarse a MongoDB.${NC}"
