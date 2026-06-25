# 🚀 Ejecutar el proyecto desde 0 (recién clonado)

**Prerrequisitos:** Docker Desktop **corriendo**, Python 3 (`py`), Git.

```bash
# 1. Clonar y configurar el entorno
cp .env.example .env        # llenar DB_USER, DB_PASSWORD, DB_NAME, etc.
                            # (COMPOSE_PROFILES=app,analytics ya viene seteado)

# 2. Levantar el stack (build de la API Go + pull de imágenes — la 1ra vez tarda)
docker compose up -d --build

# 3. Dependencias Python (el seed y el ruteo corren desde el host)
py -m pip install psycopg2-binary folium

# 4. Cargar datos de prueba
py seed/main.py

# 5. Análisis Spark crudo (esquema analytics + 3 análisis)
docker exec -it postgres_db_v2 psql -U admin_jenny -d restaurantDB -c "CREATE SCHEMA IF NOT EXISTS analytics;"
docker exec -it spark_p2 spark-submit --packages org.postgresql:postgresql:42.7.3 /home/jovyan/work/jobs/tendencias_consumo.py --usuario admin_jenny --clave abcdef
docker exec -it spark_p2 spark-submit --packages org.postgresql:postgresql:42.7.3 /home/jovyan/work/jobs/horarios_pico.py --usuario admin_jenny --clave abcdef
docker exec -it spark_p2 spark-submit --packages org.postgresql:postgresql:42.7.3 /home/jovyan/work/jobs/crecimiento_mensual.py --usuario admin_jenny --clave abcdef

# 6. Data Warehouse (Hive) + servirlo para Metabase
docker exec -it spark_p2 spark-submit --packages org.postgresql:postgresql:42.7.3 /home/jovyan/work/jobs/etl_hive.py --usuario admin_jenny --clave abcdef
docker compose --profile dw up -d spark-thrift     # OJO: no correr el ETL mientras esto esté arriba

# 7. Neo4j (grafo co-compra + recomendaciones)
docker exec -it spark_p2 pip install neo4j psycopg2-binary
docker exec -e DB_HOST=db -e DB_USER=admin_jenny -e DB_PASSWORD=abcdef -e DB_NAME=restaurantDB -it spark_p2 python /home/jovyan/work/jobs/neo4j_grafos.py

# 8. Enrutamiento (rutas + mapa)
py routing/main.py          # genera routing/rutas.html y analytics.asignaciones_entrega
```

**9. Conectar las visualizaciones en Metabase** (localhost:3000 → crear cuenta admin):
- **PostgreSQL** → host `db`, port `5432`, db `restaurantDB`, user/pass del `.env` (tablas `analytics.*` + mapa de `users`).
- **Spark SQL** → host `spark-thrift`, port `10000`, db `restaurant_dw`, user `spark` (el DW Hive).

**Accesos:**
| Servicio | URL |
|---|---|
| API (vía Nginx) | http://localhost:8000 |
| Keycloak | http://localhost:8082 |
| Metabase | http://localhost:3000 |
| Jupyter (Spark) | http://localhost:8888 |
| Airflow | http://localhost:8081 (admin/admin) |
| Neo4j Browser | http://localhost:7474 (neo4j/password123) |
| Spark Thrift (JDBC) | localhost:10000 |

> Para apagar todo: `docker compose --profile dw --profile analytics down` (agregá `-v` para borrar volúmenes/datos).

---


# ⚡Comandos Spark
Verificar que el spark crudo funcione: 

`
docker exec -it postgres_db_v2 psql -U admin_jenny -d restaurantDB -c "SELECT count(*) FROM analytics.tendencias_categoria_mes;" `

##  🧠 Metabase + DW (Hive vía Spark Thrift)
Usando**Spark Thrift Server** que expone el DW por JDBC. Metabase lo consulta con su driver Spark SQL (nativo en v0.50). El `spark/conf/hive-site.xml` fija el metastore a una ruta persistida y
compartida entre el ETL y el Thrift Server.

```
# 1. Construir el DW (con el thrift APAGADO)
docker exec -it spark_p2 spark-submit --packages org.postgresql:postgresql:42.7.3 \
  /home/jovyan/work/jobs/etl_hive.py --usuario <user> --clave <pass>

# 2. Levantar el Thrift Server (sirve el DW en :10000)
docker compose --profile dw up -d spark-thrift

# 3. (opcional) probar el DW sin Metabase
docker exec -it spark_thrift_p2 /usr/local/spark/bin/beeline \
  -u "jdbc:hive2://localhost:10000" -e "USE restaurant_dw; SHOW TABLES;"
```
Conexión en Metabase: **Add database → Spark SQL** → host `spark-thrift`, port `10000`,
database `restaurant_dw`, user `spark` (sin password).

## 📗Dependencias (mas adelante mover al docker para tenerlo automatico)
- py -m pip install psycopg2-binary
- py -m pip install folium psycopg2-binary
- py -m pip install pyspark

# 🔎 Comandos de verificar
Thrift Server arriba
```
docker exec spark_thrift_p2 /usr/local/spark/bin/beeline -u "jdbc:hive2://localhost:10000/restaurant_dw" -e "SHOW TABLES;"

# Apagar Thrift
docker stop spark_thrift_p2

# Encender Thrift
docker start spark_thrift_p2

# Apagar Airflow (no es necesario para el ETL, pero si querés)
docker stop airflow_p2

# Encender Airflow
docker start airflow_p2
```

Datos Folium
```
docker exec postgres_db_v2 psql -U admin_jenny -d restaurantDB -c "SELECT * FROM analytics.asignaciones_entrega LIMIT 20;" 
```