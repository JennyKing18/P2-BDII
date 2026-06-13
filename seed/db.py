"""
db.py — Utilidades de operaciones sobre la base de datos.

Funciones de bajo nivel reutilizadas por el orquestador: insercion por lotes,
consulta del maximo id y resincronizacion de secuencias. Mantiene el SQL
repetitivo fuera de la logica de generacion.
"""
from psycopg2.extras import execute_values


def max_id(cur, tabla):
    """
    Devuelve el mayor id presente en una tabla.

    Entradas:
        cur: cursor psycopg2 activo.
        tabla (str): nombre de tabla (interno y confiable, no viene de input
                     externo, por eso es seguro interpolarlo).
    Salidas:
        int: el id maximo, o 0 si la tabla esta vacia.
    Funcionamiento:
        Permite continuar la numeracion de ids despues de los datos que ya
        existan, evitando colisiones con las filas creadas por la API.
    """
    cur.execute(f"SELECT COALESCE(MAX(id), 0) FROM {tabla}")
    return cur.fetchone()[0]


def insertar(cur, sql, filas, page_size=1000):
    """
    Inserta una lista de filas en lotes.

    Entradas:
        cur: cursor psycopg2 activo.
        sql (str): INSERT con el marcador %s donde van los VALUES.
        filas (list[tuple]): filas a insertar.
        page_size (int): tamano de lote para execute_values.
    Salidas:
        Ninguna.
    Funcionamiento:
        execute_values agrupa miles de filas en pocas sentencias (mucho mas
        rapido que un INSERT por fila). No hace commit; lo hace main.py.
    """
    if not filas:
        return
    execute_values(cur, sql, filas, page_size=page_size)


def resincronizar_secuencias(cur, tablas):
    """
    Ajusta las secuencias de auto-incremento al maximo id existente.

    Entradas:
        cur: cursor psycopg2 activo.
        tablas (list[str]): tablas cuyas secuencias reajustar (lista interna
                            y confiable).
    Salidas:
        Ninguna.
    Funcionamiento:
        Como el seed inserta ids explicitos, la secuencia de Postgres queda
        atrasada. Sin este ajuste, el proximo INSERT de la API (que usa la
        secuencia) chocaria con un id ya ocupado. setval la deja en MAX(id).
    """
    for tabla in tablas:
        cur.execute(
            f"SELECT setval(pg_get_serial_sequence('{tabla}', 'id'), "
            f"(SELECT COALESCE(MAX(id), 1) FROM {tabla}))"
        )