"""
db.py — Lectura/escritura en Postgres para el modulo de enrutamiento.
"""
import os
import psycopg2
from psycopg2.extras import execute_values

RUTA_ENV = os.path.join(os.path.dirname(__file__), "..", ".env")


def _env():
    """
    F: Lee el .env del proyecto a un dict.
    E: ninguna.
    S: dict de variables.
    """
    val = {}
    if os.path.exists(RUTA_ENV):
        with open(RUTA_ENV, encoding="utf-8") as f:
            for ln in f:
                ln = ln.strip()
                if ln and not ln.startswith("#") and "=" in ln:
                    k, _, v = ln.partition("=")
                    val[k.strip()] = v.strip()
    return val


def conectar():
    """
    F: Abre conexion a Postgres (localhost, puerto publicado por compose).
    E: ninguna (usa .env).
    S: conexion psycopg2.
    """
    e = _env()
    return psycopg2.connect(host="localhost", port=e.get("DB_PORT", "5432"),
                            user=e.get("DB_USER"), password=e.get("DB_PASSWORD"),
                            dbname=e.get("DB_NAME"))


def cargar_repartidores(cur):
    """
    F: Trae los repartidores activos con su posicion de partida.
    E: cur.
    S: lista de dicts {id, nombre, lat, lon}.
    """
    cur.execute("SELECT id, nombre, latitud, longitud FROM repartidores WHERE activo")
    return [{"id": r[0], "nombre": r[1], "lat": r[2], "lon": r[3]} for r in cur.fetchall()]


def cargar_pedidos(cur, estado, limite):
    """
    F: Trae pedidos a repartir con la ubicacion del cliente (orders -> users).
    E: cur, estado (str), limite (int).
    S: lista de dicts {pedido_id, cliente_id, lat, lon}.
    """
    cur.execute("""
        SELECT o.id, u.id, u.latitud, u.longitud
        FROM orders o JOIN users u ON u.id = o.user_id
        WHERE o.status = %s AND u.latitud IS NOT NULL
        LIMIT %s
    """, (estado, limite))
    return [{"pedido_id": r[0], "cliente_id": r[1], "lat": r[2], "lon": r[3]}
            for r in cur.fetchall()]


def guardar_asignaciones(cur, filas):
    """
    F: Crea analytics.asignaciones_entrega (idempotente) y reescribe las filas.
    E: cur, filas (tuplas pedido_id, repartidor_id, orden_en_ruta, distancia_km).
    S: ninguna.
    """
    cur.execute("CREATE SCHEMA IF NOT EXISTS analytics")
    cur.execute("""
        CREATE TABLE IF NOT EXISTS analytics.asignaciones_entrega (
            pedido_id     BIGINT,
            repartidor_id BIGINT,
            orden_en_ruta INT,
            distancia_km  DOUBLE PRECISION
        )
    """)
    cur.execute("TRUNCATE analytics.asignaciones_entrega")
    execute_values(cur, """INSERT INTO analytics.asignaciones_entrega
        (pedido_id, repartidor_id, orden_en_ruta, distancia_km) VALUES %s""", filas)