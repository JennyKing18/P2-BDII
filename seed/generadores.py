"""
generadores.py — Generadores de datos por entidad.

Cada funcion construye y devuelve listas de filas listas para insertar (tuplas
en el orden de columnas del INSERT correspondiente en main.py). No tocan la
base directamente, salvo actualizar_geo_existentes, que modifica filas
preexistentes. La "forma" realista (geo, fechas) viene de dominio.py.
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
    Asigna zona y coordenadas a los usuarios que aun no las tienen.

    Entradas:
        cur: cursor psycopg2 activo.
    Salidas:
        int: cantidad de usuarios actualizados.
    Funcionamiento:
        Los usuarios creados por la API antes del seed no tienen geo. Se les
        asigna una zona aleatoria y un punto dentro de ella para que entren en
        los analisis por ubicacion sin quedar nulos.
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
    Genera restaurantes, su usuario admin y sus items de menu.

    Entradas:
        uid (int): primer id de usuario libre.
        rid (int): primer id de restaurante libre.
        mid (int): primer id de menu_item libre.
        n_restaurantes (int): cuantos restaurantes crear.
    Salidas:
        dict con:
            'admins'         list[tuple]              filas para users (admin)
            'restaurantes'   list[tuple]              filas para restaurants
            'menus'          list[tuple]              filas para menu_items
            'items_por_rest' dict[int, list[tuple]]   rid -> [(item_id, precio)]
            'rest_ids'       list[int]                ids de restaurantes creados
            'uid_siguiente'  int                      siguiente id de usuario libre
    Funcionamiento:
        Cada restaurante recibe un admin propio y entre 4 y 7 categorias, con
        hasta 2 platos por categoria. Restaurante y admin comparten zona y se
        crean con fecha anterior a INICIO (existen antes de la actividad).
        items_por_rest se devuelve para que el generador de ordenes elija
        platos validos del restaurante correcto (clave para la co-compra).
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
    Genera usuarios con rol 'client'.

    Entradas:
        uid (int): primer id de usuario libre.
        n_clientes (int): cuantos clientes crear.
    Salidas:
        dict con:
            'clientes'     list[tuple]  filas para users
            'clientes_ids' list[int]    ids generados (para ordenes/reservas)
    Funcionamiento:
        Cada cliente recibe zona, coordenadas y una fecha de alta dispersa en
        los 5 meses previos a INICIO.
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
    Genera ordenes agrupadas en "sesiones" de compra.

    Entradas:
        oid (int): primer id de orden libre.
        clientes_ids (list[int]): ids de clientes.
        rest_ids (list[int]): ids de restaurantes.
        items_por_rest (dict): rid -> [(item_id, precio)].
        n_sesiones (int): cuantas sesiones de compra simular.
    Salidas:
        list[tuple]: filas para orders.
    Funcionamiento:
        Cada sesion = un cliente + un restaurante + un instante, con 1 a 4
        items comprados juntos (sesgado a 1-2). Que varios items compartan
        cliente/restaurante/momento es lo que habilita el analisis de co-compra
        en Neo4j. El P1 no tiene ordenes multi-item, asi que la "sesion" es
        nuestra definicion operativa de "comprados juntos".
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
    Genera reservaciones.

    Entradas:
        vid (int): primer id de reserva libre.
        clientes_ids (list[int]): ids de clientes.
        rest_ids (list[int]): ids de restaurantes.
        n_reservas (int): cuantas reservas crear.
    Salidas:
        list[tuple]: filas para reservations.
    Funcionamiento:
        created_at es cuando se hizo la reserva (sesgado a horas pico); date es
        la fecha reservada, 1 a 14 dias despues.
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
    Genera repartidores con base geografica.

    Entradas:
        n_repartidores (int): cuantos repartidores crear.
    Salidas:
        list[tuple]: filas para repartidores (sin id; la tabla usa BIGSERIAL).
    Funcionamiento:
        Cada repartidor tiene zona y punto de partida dentro de ella, que el
        modulo de enrutamiento usa como origen de la ruta.
    """
    repartidores = []
    for i in range(n_repartidores):
        zona = random.choice(list(ZONAS))
        lat, lon = punto_en(zona)
        repartidores.append((f"Repartidor {i + 1:02d}", zona, lat, lon, True))
    return repartidores


def generar_recomendaciones(clientes_ids, n_recomendaciones):
    """
    Genera relaciones de recomendacion usuario -> usuario.

    Entradas:
        clientes_ids (list[int]): ids de clientes candidatos.
        n_recomendaciones (int): cuantas relaciones unicas generar.
    Salidas:
        list[tuple]: filas para recomendaciones (sin id; BIGSERIAL).
    Funcionamiento:
        Toma pares distintos de clientes hasta juntar n relaciones unicas (un
        set evita duplicados origen->destino). Insumo del analisis "usuarios
        que recomiendan a otros" en Neo4j. El objetivo se acota al maximo de
        pares posibles por si n excede las combinaciones disponibles.
    """
    pares = set()
    objetivo = min(n_recomendaciones, len(clientes_ids) * (len(clientes_ids) - 1))
    while len(pares) < objetivo:
        origen, destino = random.sample(clientes_ids, 2)
        pares.add((origen, destino))
    return [(origen, destino, fecha_aleatoria()) for origen, destino in pares]