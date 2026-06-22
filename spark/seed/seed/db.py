"""
db.py — Utilidades de operaciones sobre la base de datos.

Funciones de bajo nivel reutilizadas por el orquestador: insercion por lotes,
consulta del maximo id y resincronizacion de secuencias. Mantiene el SQL
repetitivo fuera de la logica de generacion.
"""
from psycopg2.extras import execute_values


def max_id(cur, tabla):
    """
    F: Devuelve el mayor id presente en una tabla.
    E:
        cur: cursor psycopg2 activo.
        tabla (str): nombre de tabla 
    S: int: el id maximo, o 0 si la tabla esta vacia.    
    """
    cur.execute(f"SELECT COALESCE(MAX(id), 0) FROM {tabla}")
    return cur.fetchone()[0]


def insertar(cur, sql, filas, page_size=1000):
    """
    F: Inserta una lista de filas en lotes.

    E:
        cur: cursor psycopg2 activo.
        sql (str): INSERT con el marcador %s donde van los VALUES.
        filas (list[tuple]): filas a insertar.
        page_size (int): tamano de lote para execute_values.
    S: -           
    """
    if not filas:
        return
    execute_values(cur, sql, filas, page_size=page_size)


def resincronizar_secuencias(cur, tablas):
    """
    F: Ajusta las secuencias de auto-incremento al maximo id existente.

    E:
        cur: cursor psycopg2 activo.
        tablas (list[str]): tablas cuyas secuencias reajustar
    S: -        
    """
    for tabla in tablas:
        cur.execute(
            f"SELECT setval(pg_get_serial_sequence('{tabla}', 'id'), "
            f"(SELECT COALESCE(MAX(id), 1) FROM {tabla}))"
        )