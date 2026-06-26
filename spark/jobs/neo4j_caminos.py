"""
neo4j_caminos.py — Caminos mínimos entre ubicaciones para reparto eficiente.

Se traen las ubicaciones reales desde Postgres: repartidores activos
(puntos de partida) y clientes con pedidos (puntos de entrega).
Luego se construye en Neo4j una RED DE CERCANÍA: cada ubicación se conecta a sus
K vecinos más cercanos con una arista :CONECTA ponderada por la distancia
en km (fórmula de haversine). 
Se calcula el camino de menor distancia con Dijkstra.

Requiere el plugin APOC en Neo4j (ya activo: NEO4J_PLUGINS=["apoc"]).
"""
import os
from math import radians, sin, cos, asin, sqrt
import psycopg2
from neo4j import GraphDatabase

# ==========================================
# 1. CONFIGURACIÓN
# ==========================================
PG_HOST = os.getenv("DB_HOST", "db")
PG_PORT = os.getenv("DB_PORT", "5432")
PG_USER = os.getenv("DB_USER", "postgres")
PG_PASS = os.getenv("DB_PASSWORD", "postgres")
PG_DB = os.getenv("DB_NAME", "restaurantDB")

NEO4J_URI = "bolt://neo4j_p2:7687"
NEO4J_USER = "neo4j"
NEO4J_PASS = "password123"

K_VECINOS = 6        # cada ubicación se conecta con sus K vecinos más cercanos
MAX_CLIENTES = 150   # tope de puntos de entrega para que el grafo sea manejable


def haversine(lat1, lon1, lat2, lon2):
    """Distancia en km entre dos coordenadas (grados)."""
    r = 6371.0
    dlat, dlon = radians(lat2 - lat1), radians(lon2 - lon1)
    a = sin(dlat / 2) ** 2 + cos(radians(lat1)) * cos(radians(lat2)) * sin(dlon / 2) ** 2
    return 2 * r * asin(sqrt(a))


def cargar_ubicaciones():
    """Repartidores activos + clientes con pedidos, todos con coordenadas."""
    conn = psycopg2.connect(host=PG_HOST, port=PG_PORT, user=PG_USER,
                            password=PG_PASS, dbname=PG_DB)
    cur = conn.cursor()
    ubic = []

    cur.execute("""
        SELECT id, nombre, latitud, longitud
        FROM repartidores
        WHERE activo AND latitud IS NOT NULL AND longitud IS NOT NULL
    """)
    for r in cur.fetchall():
        ubic.append({"id": f"REP_{r[0]}", "nombre": r[1], "tipo": "Repartidor",
                     "lat": float(r[2]), "lon": float(r[3])})

    cur.execute("""
        SELECT DISTINCT u.id, u.username, u.latitud, u.longitud
        FROM users u JOIN orders o ON o.user_id = u.id
        WHERE u.latitud IS NOT NULL AND u.longitud IS NOT NULL
        LIMIT %s
    """, (MAX_CLIENTES,))
    for r in cur.fetchall():
        ubic.append({"id": f"CLI_{r[0]}", "nombre": r[1], "tipo": "Cliente",
                     "lat": float(r[2]), "lon": float(r[3])})

    cur.close()
    conn.close()
    return ubic


def construir_aristas(ubic, k):
    """kNN: conecta cada ubicación con sus k vecinos más cercanos (peso = km)."""
    aristas = []
    for a in ubic:
        vecinos = sorted(
            ((haversine(a["lat"], a["lon"], b["lat"], b["lon"]), b)
             for b in ubic if b["id"] != a["id"]),
            key=lambda x: x[0]
        )[:k]
        for km, b in vecinos:
            aristas.append({"desde": a["id"], "hacia": b["id"], "km": round(km, 3)})
    return aristas


def main():
    print("Cargando ubicaciones desde Postgres...")
    ubic = cargar_ubicaciones()
    repartidores = [u for u in ubic if u["tipo"] == "Repartidor"]
    print(f"  {len(ubic)} ubicaciones ({len(repartidores)} repartidores, "
          f"{len(ubic) - len(repartidores)} clientes)")
    if not ubic:
        print("No hay ubicaciones con coordenadas. Revisa las tablas repartidores/users.")
        return

    print(f"Construyendo red de cercanía (k={K_VECINOS} vecinos)...")
    aristas = construir_aristas(ubic, K_VECINOS)
    print(f"  {len(aristas)} conexiones ponderadas por km")

    driver = GraphDatabase.driver(NEO4J_URI, auth=(NEO4J_USER, NEO4J_PASS))
    with driver.session() as s:
       
        print("Reconstruyendo grafo de ubicaciones en Neo4j...")
        s.run("MATCH (n:Ubicacion) DETACH DELETE n")
        s.run("CREATE INDEX ubicacion_id IF NOT EXISTS FOR (u:Ubicacion) ON (u.id)")

        s.run("""
            UNWIND $ubic AS u
            CREATE (:Ubicacion {id: u.id, nombre: u.nombre, tipo: u.tipo,
                                lat: u.lat, lon: u.lon})
        """, ubic=ubic)

        s.run("""
            UNWIND $aristas AS a
            MATCH (x:Ubicacion {id: a.desde})
            MATCH (y:Ubicacion {id: a.hacia})
            MERGE (x)-[c:CONECTA]->(y)
            SET c.km = a.km
        """, aristas=aristas)

        # ==========================================
        # 2. CONSULTA: CAMINOS MÍNIMOS PARA REPARTO (Dijkstra)
        # ==========================================
        origen = repartidores[0] if repartidores else ubic[0]
        print(f"\n--- ENTREGAS MÁS COSTOSAS DESDE '{origen['nombre']}' "
              f"(ruta óptima por km) ---")
        filas = s.run("""
            MATCH (o:Ubicacion {id: $origen})
            MATCH (d:Ubicacion {tipo: 'Cliente'}) WHERE d <> o
            CALL apoc.algo.dijkstra(o, d, 'CONECTA', 'km') YIELD path, weight
            RETURN d.nombre AS cliente,
                   round(weight, 2) AS km_optimo,
                   length(path) AS saltos,
                   [n IN nodes(path) | n.nombre] AS ruta
            ORDER BY km_optimo DESC
            LIMIT 5
        """, origen=origen["id"]).data()

        if not filas:
            print("No se hallaron rutas (ningún cliente alcanzable desde el origen).")
        else:
            for i, f in enumerate(filas, 1):
                print(f"{i}. {f['cliente']}: {f['km_optimo']} km en {f['saltos']} saltos")

            top = filas[0]
            print("\n--- RUTA ÓPTIMA A LA ENTREGA MÁS LEJANA (camino mínimo) ---")
            print("  " + "  ->  ".join(top["ruta"]))
            print(f"  Distancia total mínima: {top['km_optimo']} km")

    driver.close()
    print("\n¡Cálculo de caminos mínimos finalizado!")


if __name__ == "__main__":
    main()
