"""
reindex_es.py — Reindexa el catalogo de productos en ElasticSearch.

Lee menu_items desde PostgreSQL y los envia (POST batch) al microservicio
search-service, que los indexa en ElasticSearch. Es la tarea final del DAG de
Airflow: "Reindexado de ElasticSearch si cambia el catalogo de productos".

"""
import os
import json
import urllib.request
import urllib.error
import psycopg2

PG_HOST = os.getenv("DB_HOST", "db")
PG_PORT = os.getenv("DB_PORT", "5432")
PG_USER = os.getenv("DB_USER", "admin_jenny")
PG_PASS = os.getenv("DB_PASSWORD", "abcdef")
PG_DB = os.getenv("DB_NAME", "restaurantDB")

SEARCH_URL = os.getenv("SEARCH_URL", "http://search-service:8080/reindex")


def cargar_productos():
    """Lee el catalogo (menu_items) desde Postgres como lista de dicts."""
    conn = psycopg2.connect(host=PG_HOST, port=PG_PORT, user=PG_USER,
                            password=PG_PASS, dbname=PG_DB)
    cur = conn.cursor()
    cur.execute("SELECT id, name, category, description FROM menu_items")
    productos = [
        {
            "id": r[0],
            "name": r[1] or "",
            "category": r[2] or "",
            "description": r[3] or "",
        }
        for r in cur.fetchall()
    ]
    cur.close()
    conn.close()
    return productos


def reindexar(productos):
    """POST del batch de productos al search-service. Devuelve la respuesta."""
    cuerpo = json.dumps(productos).encode("utf-8")
    req = urllib.request.Request(
        SEARCH_URL, data=cuerpo,
        headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.status, resp.read().decode("utf-8")


def main():
    print(f"Leyendo catalogo desde Postgres ({PG_HOST}:{PG_PORT}/{PG_DB})...")
    productos = cargar_productos()
    print(f"  {len(productos)} productos en el catalogo")

    if not productos:
        print("No hay productos para reindexar (corriste el seed?). Nada que hacer.")
        return

    print(f"Reindexando en ElasticSearch via {SEARCH_URL}...")
    try:
        status, body = reindexar(productos)
    except urllib.error.HTTPError as e:
        raise Exception(f"search-service respondio {e.code}: {e.read().decode('utf-8')}")
    except urllib.error.URLError as e:
        raise Exception(f"No se pudo contactar al search-service: {e.reason}")

    print(f"Respuesta HTTP {status}: {body}")
    if status not in (200, 201):
        raise Exception(f"Fallo el reindexado (HTTP {status}).")
    print("Reindexado completado con exito!")


if __name__ == "__main__":
    main()
