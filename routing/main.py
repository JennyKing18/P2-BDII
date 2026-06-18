"""
main.py — Orquesta el ruteo: carga datos, asigna, ordena, guarda y dibuja.
Uso: python routing/main.py [--estado ready_for_pickup] [--limite 80]
"""
import argparse
import os

import db
import ruteo
import mapa


def main():
    """
    F: Asigna pedidos a repartidores (vecino mas cercano), guarda asignaciones y mapa.
    E: args CLI (--estado, --limite, --salida).
    S: ninguna (escribe analytics.asignaciones_entrega y un HTML).
    """
    p = argparse.ArgumentParser(description="Enrutamiento de entregas (vecino mas cercano)")
    p.add_argument("--estado", default="ready_for_pickup")
    p.add_argument("--limite", type=int, default=80)
    p.add_argument("--salida", default=os.path.join(os.path.dirname(__file__), "rutas.html"))
    args = p.parse_args()

    conn = db.conectar()
    cur = conn.cursor()
    try:
        repartidores = db.cargar_repartidores(cur)
        pedidos = db.cargar_pedidos(cur, args.estado, args.limite)

        grupos = ruteo.asignar_a_repartidor(repartidores, pedidos)
        por_id = {r["id"]: r for r in repartidores}

        rutas, filas = {}, []
        for rep_id, lista in grupos.items():
            if not lista:
                continue
            ruta = ruteo.ordenar_ruta(por_id[rep_id], lista)
            rutas[rep_id] = ruta
            for pedido, orden, dist in ruta:
                filas.append((pedido["pedido_id"], rep_id, orden, dist))

        db.guardar_asignaciones(cur, filas)
        conn.commit()

        if rutas:
            mapa.dibujar(repartidores, rutas, args.salida)

        print(f"Pedidos ruteados: {len(filas)} | repartidores con ruta: {len(rutas)}")
        print(f"Mapa: {args.salida}")
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    main()