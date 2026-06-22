"""
config.py — Configuracion de conexion al PostgreSQL del P1.

Centraliza la lectura del .env del proyecto y la creacion de la conexion
psycopg2. Ningun otro modulo lee el .env directamente.
"""
import os
import psycopg2

RUTA_ENV = os.path.join(os.path.dirname(__file__), "..", ".env")


def cargar_env(ruta=RUTA_ENV):
    """
    F: Lee un archivo .env y devuelve sus pares clave/valor.
    E: ruta (str): ruta al .env. Por defecto el .env del proyecto.
    S: dict[str, str]: variables encontradas; vacio si el archivo no existe.    
    """
    valores = {}
    if not os.path.exists(ruta):
        return valores
    with open(ruta, encoding="utf-8") as f:
        for linea in f:
            linea = linea.strip()
            if not linea or linea.startswith("#") or "=" not in linea:
                continue
            clave, _, valor = linea.partition("=")
            valores[clave.strip()] = valor.strip()
    return valores


def conectar():
    """
    F: Abre una conexion a PostgreSQL.

    Prioriza las variables de entorno del proceso (las que se pasan con
    `docker exec -e ...` o el env_file de compose) y usa el .env del
    proyecto solo como respaldo.

    E: -
    S:
        psycopg2.connection con autocommit desactivado (el control
        transaccional lo lleva main.py).
    """
    env = cargar_env()

    def valor(clave, defecto=None):
        return os.environ.get(clave, env.get(clave, defecto))

    conn = psycopg2.connect(
        host=valor("DB_HOST", "localhost"),
        port=valor("DB_PORT", "5432"),
        user=valor("DB_USER"),
        password=valor("DB_PASSWORD"),
        dbname=valor("DB_NAME"),
    )
    conn.autocommit = False
    return conn