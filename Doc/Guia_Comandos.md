# ⚙️ Guía de Comandos

## ⚠️ Problemas Conocidos

- **Volúmenes al reiniciar:** Al hacer `compose down` + `up` no se está cargando el volumen correctamente. Por ahora usar `start`/`stop` o `compose up --build` para hacer pruebas con datos sin que se borren.
- **Keycloak realm:** Se supone que con `--import-realm` en el build de Keycloak debería cargar automáticamente lo que está en PostgreSQL, pero no está jalando. (En progreso.)

---

## Redis

Verificar caché con Postman — `GET http://localhost/restaurants`
O bien si se tiene al app desktop RedisInsight se puede vizualizar
- Primera request → va a BD, tarda más
- Segunda request → sale desde caché, más rápido

```bash
docker exec -it redis_cache redis-cli   # Entrar al container de Redis
KEYS *                                  # Ver keys cacheadas
                                        # ⚠️ Solo usar si hay pocas keys, sino buscar específico
```

---

## Search Service

```bash
# Verificar que está arriba
docker exec search-service wget -qO- http://localhost:8080/ping

# Ver logs
docker logs search-service
```

---
## Postgres
Entrar al shell:
```bash
docker exec -it postgres_db_v2 psql -U admin_jenny -d restaurantDB -c "SELECT * FROM menu_items;"
```
## MongoDB (Compose principal)

Levantar y Configurar shards:
```bash
./start-project.sh
docker compose restart mongo-setup   # En caso de fallo
```

Ver si el sharding se configuró bien:
```bash
docker logs mongo-setup -f
```

Entrar al shell y verificar datos:
```bash
docker exec -it mongo-router mongosh \
  "mongodb://admin:password123@mongo-router:27017/restaurantDB?authSource=admin"

# Dentro del shell:
db.users.find().pretty()
show collections
```

Verificar estado del sharding:
```bash
docker exec -it mongo-router mongosh --port 27017 --eval "sh.status()"
```

## Replicas
docker compose up --scale api=3 -d

## K8s

kubectl apply -f k8s/
kubectl get pods
kubectl get services
---

## General (Debug)

```bash
# Ver logs/prints de la API (últimas 20 líneas)
docker compose logs api --tail 20
```
