"""
config.py — Configuracion de conexion al PostgreSQL del P1.

Centraliza la lectura del .env del proyecto y la creacion de la conexion
psycopg2. Ningun otro modulo lee el .env directamente.
"""
import os
import psycopg2

# .env del proyecto (un nivel arriba de seed/)
RUTA_ENV = os.path.join(os.path.dirname(__file__), "..", ".env")


def cargar_env(ruta=RUTA_ENV):
    """
    Lee un archivo .env y devuelve sus pares clave/valor.

    Entradas:
        ruta (str): ruta al .env. Por defecto el .env del proyecto.
    Salidas:
        dict[str, str]: variables encontradas; vacio si el archivo no existe.
    Funcionamiento:
        Parseo simple linea por linea (KEY=VALUE), ignorando comentarios (#)
        y lineas en blanco. Sin dependencias externas.
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
    Abre una conexion a PostgreSQL usando las variables del .env.

    Entradas:
        Ninguna (lee DB_USER, DB_PASSWORD, DB_NAME, DB_PORT del .env).
    Salidas:
        psycopg2.connection con autocommit desactivado (el control
        transaccional lo lleva main.py).
    Funcionamiento:
        Se conecta a localhost porque docker-compose publica el puerto 5432
        del contenedor de Postgres hacia el host; el seed corre en el host.
        Asume DB_DRIVER=postgres (si el sistema corre en mongo, este seed
        no aplica).
    """
    env = cargar_env()
    conn = psycopg2.connect(
        host="localhost",
        port=env.get("DB_PORT", "5432"),
        user=env.get("DB_USER"),
        password=env.get("DB_PASSWORD"),
        dbname=env.get("DB_NAME"),
    )
    conn.autocommit = False
    return conn