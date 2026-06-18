"""
ruteo.py — Heuristica de vecino mas cercano para asignar y ordenar entregas.
"""
from math import radians, sin, cos, asin, sqrt


def haversine(lat1, lon1, lat2, lon2):
    """
    F: Distancia en km entre dos coordenadas (formula de haversine).
    E: lat1, lon1, lat2, lon2 (grados).
    S: distancia en km (float).
    """
    r = 6371.0
    dlat, dlon = radians(lat2 - lat1), radians(lon2 - lon1)
    a = sin(dlat / 2) ** 2 + cos(radians(lat1)) * cos(radians(lat2)) * sin(dlon / 2) ** 2
    return 2 * r * asin(sqrt(a))


def asignar_a_repartidor(repartidores, pedidos, distancia=haversine):
    """
    F: Asigna cada pedido al repartidor mas cercano a su punto de partida.
    E: repartidores (list), pedidos (list), distancia (func opcional).
    S: dict repartidor_id -> lista de pedidos.
    """
    grupos = {r["id"]: [] for r in repartidores}
    for p in pedidos:
        cercano = min(repartidores,
                      key=lambda r: distancia(r["lat"], r["lon"], p["lat"], p["lon"]))
        grupos[cercano["id"]].append(p)
    return grupos


def ordenar_ruta(repartidor, pedidos, distancia=haversine):
    """
    F: Ordena las paradas por vecino mas cercano partiendo del repartidor.
    E: repartidor (dict), pedidos (list), distancia (func opcional).
    S: lista de (pedido, orden_en_ruta, distancia_km desde la parada previa).
    """
    ruta, pendientes = [], list(pedidos)
    lat, lon, orden = repartidor["lat"], repartidor["lon"], 1
    while pendientes:
        sig = min(pendientes, key=lambda p: distancia(lat, lon, p["lat"], p["lon"]))
        d = round(distancia(lat, lon, sig["lat"], sig["lon"]), 3)
        ruta.append((sig, orden, d))
        lat, lon, orden = sig["lat"], sig["lon"], orden + 1
        pendientes.remove(sig)
    return ruta