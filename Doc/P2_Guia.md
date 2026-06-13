# ⚙️ Updates P2

Se agregan Modulo de Seed que maneja:
- Repartidores
- Recomendaciones
- data de 8 meses para spark (random)

Ambas entidades se mantienen fuera del Modelo por desición técnica. Agregarlas al modelo implicaría escribir esta funcionalidad para las repos de Mongo y Postgres, su respectiva migración y afectaría el coverage actual. Tal que existen como tablas dentro de la BD sin que el GORM sepa de ellas. (Por ello no se puede hacer CRUD de estas).

Esto implica que solo podemos cambiar-interactuar con ellas por medio de SQL puro o scripts (no tienen ruta), sin embargo, la data SI esta conectada es el caso de ´recomendaciones´ por medio de ´FK a users´. Mientras que ´repartidores´ solo comparte "geografía" no se conecta con BD. 

## ¿Por qué esta arquitectura funciona para Neo4j y OLAP?

La clave es que **Neo4j y OLAP son capas analíticas de solo lectura**: no
necesitan CRUD ni la API, solo leer los datos. Por eso mantener las nuevas
entidades fuera del modelo GORM no les cuesta nada.

Todo (tablas P1 + extensiones P2) vive en el mismo
PostgreSQL. Así Airflow extrae de un solo lugar, Spark lee por JDBC y el cargador
de Neo4j toma todo de la misma BD, sin servicios ni conectores extra.

**Para Neo4j (relaciones):** un grafo se arma a partir de *relaciones entre
datos*, y eso es justo lo que ya tenemos como filas:
- `orders` agrupadas en sesiones (mismo cliente + restaurante + momento) → arista
  de **co-compra** entre productos.
- `recomendaciones` con FK a `users` → arista **usuario→recomienda→usuario**.
- coordenadas en `users`/`restaurants` + `repartidores` → **geonodos** y rutas.
El cargador de grafo solo necesita la data y sus relaciones; no le importa que
GORM no las maneje.

# ⚙️ Guía de Comandos
```
# 1. Crear el esquema de salida una vez (Spark crea tablas, no esquemas)
docker exec -it postgres_db_v2 psql -U <user> -d <db> -c "CREATE SCHEMA IF NOT EXISTS analytics;"

# 2. Levantar Spark
docker compose -f docker-compose.p2.yml up -d spark

# 3. Correr el análisis (la 1ra vez baja el driver JDBC de Maven — tarda ~1 min)
docker exec -it spark_p2 python /home/jovyan/work/jobs/tendencias_consumo.py `
  --url jdbc:postgresql://host.docker.internal:5432/<db> `
  --usuario <user> --clave <pass>
```

## 📗Dependencias (mas adelante mover al docker para tenerlo automatico)
- py -m pip install psycopg2-binary
- py -m pip install pyspark


