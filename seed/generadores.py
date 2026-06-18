"""
generadores.py — Generadores de datos por entidad.

Cada funcion construye y devuelve listas de filas listas para insertar (tuplas
en el orden de columnas del INSERT correspondiente en main.py). No tocan la
base directamente, salvo actualizar_geo_existentes, que modifica filas
preexistentes.
"""
import random
from datetime import timedelta

from dominio import (ZONAS, CATEGORIAS, PLATOS, NOMBRES_REST, SUFIJOS_REST,
                     INICIO, punto_en, fecha_aleatoria)

# Distribucion de estados (listas "infladas" => muestreo ponderado simple)
ESTADOS_ORDEN = (["completed"] * 72 + ["cancelled"] * 12
                 + ["pending"] * 10 + ["ready_for_pickup"] * 6)
ESTADOS_RESERVA = ["confirmed"] * 70 + ["cancelled"] * 15 + ["pending"] * 15


def actualizar_geo_existentes(cur):
    """
    F: Asigna zona y coordenadas a los usuarios que aun no las tienen.
    E: cur: cursor psycopg2 activo.
    S: int: cantidad de usuarios actualizados.    
    """
    cur.execute("SELECT id FROM users WHERE latitud IS NULL")
    ids = [fila[0] for fila in cur.fetchall()]
    for uid in ids:
        zona = random.choice(list(ZONAS))
        lat, lon = punto_en(zona)
        cur.execute(
            "UPDATE users SET latitud = %s, longitud = %s, zona = %s WHERE id = %s",
            (lat, lon, zona, uid),
        )
    return len(ids)


def generar_restaurantes_y_menus(uid, rid, mid, n_restaurantes):
    """
    F: Genera restaurantes, su usuario admin y sus items de menu.
    E:
        uid (int): primer id de usuario libre.
        rid (int): primer id de restaurante libre.
        mid (int): primer id de menu_item libre.
        n_restaurantes (int): cuantos restaurantes crear.
    S:
        dict con:
            'admins'         list[tuple]              filas para users (admin)
            'restaurantes'   list[tuple]              filas para restaurants
            'menus'          list[tuple]              filas para menu_items
            'items_por_rest' dict[int, list[tuple]]   rid -> [(item_id, precio)]
            'rest_ids'       list[int]                ids de restaurantes creados
            'uid_siguiente'  int                      siguiente id de usuario libre
    """
    admins, restaurantes, menus = [], [], []
    items_por_rest, rest_ids = {}, []

    for i in range(n_restaurantes):
        zona = random.choice(list(ZONAS))
        lat, lon = punto_en(zona)
        creado = INICIO - timedelta(days=random.randint(30, 200))
        nombre = f"{random.choice(NOMBRES_REST)} {random.choice(SUFIJOS_REST)} {i + 1}"

        admins.append((uid, creado, creado, f"admin_rest_{i + 1:03d}",
                       f"admin{i + 1:03d}@seed.local", "", "admin", lat, lon, zona))
        restaurantes.append((rid, creado, creado, nombre, f"{zona}, Costa Rica",
                             f"Restaurante de comida variada en {zona}", uid, lat, lon, zona))

        items_por_rest[rid] = []
        for categoria in random.sample(CATEGORIAS, k=random.randint(4, 7)):
            for plato in random.sample(PLATOS[categoria], k=min(2, len(PLATOS[categoria]))):
                precio = round(random.uniform(1800, 12500), -1)  # redondeo a decena de colones
                menus.append((mid, creado, creado, rid, plato, categoria,
                              f"{plato} — especialidad de la casa", precio))
                items_por_rest[rid].append((mid, precio))
                mid += 1

        rest_ids.append(rid)
        uid += 1
        rid += 1

    return {"admins": admins, "restaurantes": restaurantes, "menus": menus,
            "items_por_rest": items_por_rest, "rest_ids": rest_ids,
            "uid_siguiente": uid}


def generar_clientes(uid, n_clientes):
    """
    F: Genera usuarios con rol 'client'.
    E:
        uid (int): primer id de usuario libre.
        n_clientes (int): cuantos clientes crear.
    S:
        dict con:
            'clientes'     list[tuple]  filas para users
            'clientes_ids' list[int]    ids generados (para ordenes/reservas)    
    """
    clientes, clientes_ids = [], []
    for i in range(n_clientes):
        zona = random.choice(list(ZONAS))
        lat, lon = punto_en(zona)
        creado = INICIO - timedelta(days=random.randint(0, 150))
        clientes.append((uid, creado, creado, f"cliente_{i + 1:04d}",
                         f"cliente{i + 1:04d}@seed.local", "", "client", lat, lon, zona))
        clientes_ids.append(uid)
        uid += 1
    return {"clientes": clientes, "clientes_ids": clientes_ids}


def generar_ordenes(oid, clientes_ids, rest_ids, items_por_rest, n_sesiones):
    """
    F: Genera ordenes agrupadas en "sesiones" de compra.
    E:
        oid (int): primer id de orden libre.
        clientes_ids (list[int]): ids de clientes.
        rest_ids (list[int]): ids de restaurantes.
        items_por_rest (dict): rid -> [(item_id, precio)].
        n_sesiones (int): cuantas sesiones de compra simular.
    S:
        list[tuple]: filas para orders.
    """
    ordenes = []
    for _ in range(n_sesiones):
        cliente = random.choice(clientes_ids)
        rest = random.choice(rest_ids)
        cuando = fecha_aleatoria()
        catalogo = items_por_rest[rest]
        k = min(random.choices([1, 2, 3, 4], weights=[45, 30, 17, 8])[0], len(catalogo))
        for item_id, precio in random.sample(catalogo, k=k):
            ts = cuando + timedelta(minutes=random.randint(0, 3))
            total = round(precio * random.randint(1, 3), 2)  # cantidad implicita 1-3
            ordenes.append((oid, ts, ts, cliente, item_id, rest, total,
                            random.choice(ESTADOS_ORDEN)))
            oid += 1
    return ordenes


def generar_reservas(vid, clientes_ids, rest_ids, n_reservas):
    """
    F: Genera reservaciones.
    E:
        vid (int): primer id de reserva libre.
        clientes_ids (list[int]): ids de clientes.
        rest_ids (list[int]): ids de restaurantes.
        n_reservas (int): cuantas reservas crear.
    S: list[tuple]: filas para reservations.
    """
    reservas = []
    for _ in range(n_reservas):
        creado = fecha_aleatoria()
        fecha_reserva = creado + timedelta(days=random.randint(1, 14))
        reservas.append((vid, creado, creado, random.choice(clientes_ids),
                         random.choice(rest_ids), fecha_reserva,
                         random.choice(ESTADOS_RESERVA)))
        vid += 1
    return reservas


def generar_repartidores(n_repartidores):
    """
    F: Genera repartidores con base geografica.
    E: n_repartidores (int): cuantos repartidores crear.
    S: list[tuple]: filas para repartidores (sin id; la tabla usa BIGSERIAL).    
    """
    repartidores = []
    for i in range(n_repartidores):
        zona = random.choice(list(ZONAS))
        lat, lon = punto_en(zona)
        repartidores.append((f"Repartidor {i + 1:02d}", zona, lat, lon, True))
    return repartidores


def generar_recomendaciones(clientes_ids, n_recomendaciones):
    """
    F: Genera relaciones de recomendacion usuario -> usuario.
    E:
        clientes_ids (list[int]): ids de clientes candidatos.
        n_recomendaciones (int): cuantas relaciones unicas generar.
    S: list[tuple]: filas para recomendaciones (sin id; BIGSERIAL).
    """
    pares = set()
    objetivo = min(n_recomendaciones, len(clientes_ids) * (len(clientes_ids) - 1))
    while len(pares) < objetivo:
        origen, destino = random.sample(clientes_ids, 2)
        pares.add((origen, destino))
    return [(origen, destino, fecha_aleatoria()) for origen, destino in pares]