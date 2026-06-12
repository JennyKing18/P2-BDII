# 🍔 Reserva Inteligente de Restaurante con Docker
Plataforma de API para administrar restaurantes con reservaciones inteligentes. Permite crear usuarios tipo **Administrador** y **Cliente** para gestionar restaurantes, menús, órdenes y reservaciones.
 
Construida sobre arquitectura de microservicios en Docker con autenticación federada (Keycloak + JWT), balanceo de carga (NGINX), persistencia dual (MongoDB sharded cluster **o** PostgreSQL), caché TTL (Redis) y búsqueda full-text (Elasticsearch).
 
---
 
## 📋 Tabla de Contenidos
 
- [🍔 Reserva Inteligente de Restaurante con Docker](#-reserva-inteligente-de-restaurante-con-docker)
  - [📋 Tabla de Contenidos](#-tabla-de-contenidos)
  - [🛠 Requisitos Previos](#-requisitos-previos)
  - [🧠 Reglas de Negocio — Caché](#-reglas-de-negocio--caché)
    - [Política de TTL](#política-de-ttl)
  - [📁 Estructura del Proyecto](#-estructura-del-proyecto)
  - [🔐 Variables de Entorno](#-variables-de-entorno)
  - [🚀 Ejecutar el Proyecto Completo](#-ejecutar-el-proyecto-completo)
    - [Con PostgreSQL (default)](#con-postgresql-default)
    - [Con MongoDB Sharded](#con-mongodb-sharded)
    - [Detener los servicios](#detener-los-servicios)
    - [⚠️ Consideraciones al cambiar de motor](#️-consideraciones-al-cambiar-de-motor)
  - [📖 Documentación API](#-documentación-api)
  - [✅ Tests y Coverage](#-tests-y-coverage)

## 🛠 Requisitos Previos
 
| Herramienta | Versión mínima | Notas |
|-------------|---------------|-------|
| Docker + Docker Compose | 24.x / v2.x | Todo corre en contenedores |
| Go | 1.22+ | Solo para desarrollo/tests fuera de Docker |
| Postman | cualquiera | Opcional — para probar el flujo de requests |
 
---
 
## 🧠 Reglas de Negocio — Caché
 
El sistema usa el patrón **Cache-Aside** con Redis:
 
- **Read:** revisa Redis → cache miss → consulta la BD → guarda resultado en Redis
- **Write:** actualiza la BD → elimina todas las llaves afectadas de Redis (invalidación)
![Diagrama Redis](doc/redisD.png)
 
### Política de TTL
 
| Recurso        | TTL   | Justificación |
|----------------|-------|---------------|
| Restaurants    | 24h   | Baja frecuencia de actualización — los restaurantes son estables |
| Menu Items     | 24h   | El menú cambia en batch, no continuamente |
| Orders         | 5 min | Alta frecuencia de actualización |
| Search Results | 1h    | Balance entre frescura de datos y performance |
 
---
 
## 📁 Estructura del Proyecto
 
```
P1-BASESII/
├── cmd/main.go                     # Entrypoint de la API
├── internal/
│   ├── cache/cache.go              # Lógica de caché Redis (Cache-Aside)
│   ├── database/
│   │   ├── database.go             # Inicialización de conexión
│   │   ├── factory.go              # Factory: elige Postgres o Mongo según DB_DRIVER
│   │   └── mongo.go                # Setup cliente MongoDB
│   ├── handlers/                   # HTTP handlers por recurso
│   │   ├── menu_handler.go
│   │   ├── order_handler.go
│   │   ├── reservation_handler.go
│   │   ├── restaurant_handler.go
│   │   └── user_handler.go
│   ├── keycloak/keycloak.go        # Validación JWT + extracción de roles
│   ├── models/models.go            # Structs de dominio compartidos
│   ├── repository/
│   │   ├── interface.go            # IRepository — contrato común
│   │   ├── mongo/
│   │   │   ├── entities.go         # Mapeo de documentos MongoDB
│   │   │   └── repos.go            # Implementación MongoDB del repository
│   │   └── postgres/
│   │       ├── entities.go         # Mapeo de tablas PostgreSQL
│   │       └── repos.go            # Implementación PostgreSQL del repository
│   └── search/                     # Microservicio de búsqueda (servicio independiente)
│       ├── Dockerfile
│       ├── elastic.go
│       ├── handler.go
│       ├── main.go
│       ├── models.go
│       ├── go.mod
│       └── go.sum
├── mongo-cluster/
│   ├── configs/                    # Archivos .conf para cada nodo MongoDB
│   │   ├── configsvr.conf
│   │   ├── mongos.conf
│   │   ├── shard1.conf
│   │   └── shard2.conf
│   ├── keyfile                     # Clave de autenticación interna del cluster
│   └── setup/init-cluster.sh      # Script de inicialización del cluster sharded
├── nginx/nginx.conf                # Configuración del Load Balancer
├── doc/
│   ├── Guia_Comandos.md
│   └── redisD.png
├── docker-compose.yml              # Stack completo (PostgreSQL por defecto)
├── docker-compose.mongo.yml        # Override para MongoDB sharded
├── dockerfile                      # Imagen de la API Go
├── restaurant-system-realm.json    # Exportación del realm de Keycloak
├── start-project.sh                # Script de inicio rápido
├── .env.example
└── .env                            # Variables de entorno (NO commitear)
```
 
---
## 🔐 Variables de Entorno
 
```bash
cp .env.example .env
# Editar .env con tus valores antes de levantar el proyecto
```
 
Variables clave en `.env`:
 
```env
# Motor de base de datos: "postgres" | "mongo"
DB_DRIVER=postgres
 
# PostgreSQL
POSTGRES_USER=appuser
POSTGRES_PASSWORD=changeme
POSTGRES_DB=app_db
POSTGRES_DSN=postgresql://appuser:changeme@postgres:5432/app_db?sslmode=disable
 
# MongoDB
MONGO_URI=mongodb://mongo-router:27017
MONGO_DB=appdb
 
# Redis
REDIS_URL=redis://redis:6379
 
# Keycloak
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_REALM=restaurant-system
KEYCLOAK_CLIENT_ID=api-client
KEYCLOAK_CLIENT_SECRET=changeme
 
# Elasticsearch
ELASTICSEARCH_URL=http://elasticsearch:9200
```
 
---

## 🚀 Ejecutar el Proyecto Completo
 
### Con PostgreSQL (default)
 
```bash
# 1. Configurar variables de entorno
cp .env.example .env
 
# 2. Levantar todos los servicios
docker compose up --build -d
 
# 3. Verificar que todo está corriendo
docker compose ps
```
 
### Con MongoDB Sharded
 
```bash
# 1. Configurar variables (DB_DRIVER=mongo en .env)
cp .env.example .env
 
# 2. Levantar el stack con el override de MongoDB
docker compose -f docker-compose.yml -f docker-compose.mongo.yml up --build -d
 
# 3. Inicializar el cluster sharded (solo la primera vez)
bash mongo-cluster/setup/init-cluster.sh
```
 
También podés usar el script de inicio rápido incluido que hace todo en un paso:
 
```bash
bash start-project.sh
```
### Detener los servicios
 
```bash
docker compose down          # Para y elimina contenedores (datos en volúmenes se conservan)
docker compose down -v       # CUIDADO: elimina también los volúmenes (borra datos)
docker compose stop          # Solo pausa los contenedores
```
 ### ⚠️ Consideraciones al cambiar de motor
 
| Aspecto | Detalle |
|---------|---------|
| **Datos** | Cada motor tiene su propio almacenamiento. Los datos no se migran automáticamente. |
| **Caché** | Limpiar Redis al cambiar para evitar datos obsoletos: `docker exec redis redis-cli FLUSHALL` |
| **Keycloak** | Siempre usa PostgreSQL para su propia BD, sin importar el `DB_DRIVER` de la app. |
| **Migraciones** | PostgreSQL aplica migraciones SQL al arrancar. MongoDB usa esquema flexible (sin migraciones). |
 
---
## 📖 Documentación API
 
El archivo `doc/swagger.json` contiene la especificación completa de la API.
 
**Para visualizarla:**
 
- **VS Code:** instalar la extensión [Swagger Viewer](https://marketplace.visualstudio.com/items?itemName=Arjun.swagger-viewer) → abrir `swagger.json` → `Shift+Alt+P` → Preview Swagger
- **Swagger UI online:** pegar el contenido en [editor.swagger.io](https://editor.swagger.io)
---
 
## ✅ Tests y Coverage
 
```bash
# Correr todos los tests con verbose
go test ./internal/tests/... -v
 
# Ver coverage de los handlers
go test ./internal/tests -coverpkg=Tarea1-BasesII/internal/handlers
 
# Output esperado:
# ok   Tarea1-BasesII/internal/tests  (cached)  coverage: 90.2% of statements in Tarea1-BasesII/internal/handlers
 
# Generar reporte HTML de coverage
go test ./internal/tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
 
---
 