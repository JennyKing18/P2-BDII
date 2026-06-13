"""
main.py — Orquestador del seed del Proyecto 2.

Flujo: conectar -> aplicar esquema P2 -> generar entidades en memoria ->
insertar por lotes -> resincronizar secuencias -> commit. Imprime un resumen.

Uso:
    python seed/main.py            # respeta el guard anti-duplicados
    python seed/main.py --force    # regenera aunque ya existan datos

Requisitos: la base del P1 debe estar corriendo y la API haber arrancado al
menos una vez (para que GORM AutoMigrate haya creado las tablas base).
"""
import random
import sys

import config
import db
import schema
import generadores

# Semilla fija => datasets reproducibles entre corridas y entre companeras
random.seed(42)

# Tamano del escenario (ajustar aqui para mas/menos volumen)
N_RESTAURANTES = 40
N_CLIENTES = 1200
N_SESIONES = 25000      # ~55-60k ordenes (1-4 items por sesion)
N_RESERVAS = 15000
N_REPARTIDORES = 25
N_RECOMENDACIONES = 3000

UMBRAL_GUARD = 10000    # si ya hay mas ordenes que esto, se asume seed aplicado


def main():
    """
    Punto de entrada: ejecuta el seed completo en una sola transaccion.

    Entradas:
        Ninguna (argv: --force para saltar el guard).
    Salidas:
        Ninguna (imprime un resumen; hace commit o rollback).
    Funcionamiento:
        Envuelve todo en try/except: si algo falla, rollback para no dejar la
        base a medias. El guard evita duplicar datos en corridas repetidas.
    """
    conn = config.conectar()
    cur = conn.cursor()
    try:
        # ── Guard anti-duplicados ────────────────────────────────────────
        if db.max_id(cur, "orders") > UMBRAL_GUARD and "--force" not in sys.argv:
            print("Ya hay datos de seed (>10k ordenes). Usa --force para regenerar.")
            return

        # ── Esquema P2 (idempotente) ─────────────────────────────────────
        schema.aplicar_esquema(cur)

        # ── Geo a usuarios preexistentes ─────────────────────────────────
        actualizados = generadores.actualizar_geo_existentes(cur)

        # ── Ids iniciales (continuar despues de lo existente) ────────────
        uid = db.max_id(cur, "users") + 1
        rid = db.max_id(cur, "restaurants") + 1
        mid = db.max_id(cur, "menu_items") + 1

        # ── Generacion en memoria ────────────────────────────────────────
        rest = generadores.generar_restaurantes_y_menus(uid, rid, mid, N_RESTAURANTES)
        cli = generadores.generar_clientes(rest["uid_siguiente"], N_CLIENTES)
        ordenes = generadores.generar_ordenes(
            db.max_id(cur, "orders") + 1, cli["clientes_ids"],
            rest["rest_ids"], rest["items_por_rest"], N_SESIONES)
        reservas = generadores.generar_reservas(
            db.max_id(cur, "reservations") + 1, cli["clientes_ids"],
            rest["rest_ids"], N_RESERVAS)
        repartidores = generadores.generar_repartidores(N_REPARTIDORES)
        recomendaciones = generadores.generar_recomendaciones(
            cli["clientes_ids"], N_RECOMENDACIONES)

        # ── Insercion por lotes ──────────────────────────────────────────
        usuarios = rest["admins"] + cli["clientes"]
        db.insertar(cur, """INSERT INTO users
            (id, created_at, updated_at, username, email, password, role,
             latitud, longitud, zona) VALUES %s ON CONFLICT (id) DO NOTHING""", usuarios)
        db.insertar(cur, """INSERT INTO restaurants
            (id, created_at, updated_at, name, address, description, admin_id,
             latitud, longitud, zona) VALUES %s""", rest["restaurantes"])
        db.insertar(cur, """INSERT INTO menu_items
            (id, created_at, updated_at, restaurant_id, name, category,
             description, price) VALUES %s""", rest["menus"])
        db.insertar(cur, """INSERT INTO orders
            (id, created_at, updated_at, user_id, menu_item_id, restaurant_id,
             total, status) VALUES %s""", ordenes)
        db.insertar(cur, """INSERT INTO reservations
            (id, created_at, updated_at, user_id, restaurant_id, date, status)
            VALUES %s""", reservas)
        db.insertar(cur, """INSERT INTO repartidores
            (nombre, zona, latitud, longitud, activo) VALUES %s""", repartidores)
        db.insertar(cur, """INSERT INTO recomendaciones
            (usuario_origen, usuario_destino, fecha) VALUES %s
            ON CONFLICT DO NOTHING""", recomendaciones)

        # ── Resincronizar secuencias (insertamos ids explicitos) ─────────
        db.resincronizar_secuencias(
            cur, ["users", "restaurants", "menu_items", "orders", "reservations"])

        conn.commit()
        print("Seed aplicado correctamente:")
        print(f"  usuarios geo actualizados : {actualizados}")
        print(f"  usuarios nuevos           : {len(usuarios)}")
        print(f"  restaurantes              : {len(rest['restaurantes'])}")
        print(f"  menu_items                : {len(rest['menus'])}")
        print(f"  ordenes                   : {len(ordenes)}")
        print(f"  reservas                  : {len(reservas)}")
        print(f"  repartidores              : {len(repartidores)}")
        print(f"  recomendaciones           : {len(recomendaciones)}")
    except Exception:
        conn.rollback()
        raise
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()