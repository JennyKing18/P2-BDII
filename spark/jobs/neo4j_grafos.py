import os
import psycopg2
from neo4j import GraphDatabase

# ==========================================
# 1. CONFIGURACIÓN DE CONEXIONES
# ==========================================
PG_HOST = os.getenv("DB_HOST", "db")
PG_PORT = "5432"
PG_USER = os.getenv("DB_USER", "postgres")
PG_PASS = os.getenv("DB_PASSWORD", "postgres")
PG_DB = os.getenv("DB_NAME", "restaurantDB")

NEO4J_URI = "bolt://neo4j_p2:7687"
NEO4J_USER = "neo4j"
NEO4J_PASS = "password123"

def main():
    print("Conectando a bases de datos...")
    pg_conn = psycopg2.connect(host=PG_HOST, port=PG_PORT, user=PG_USER, password=PG_PASS, dbname=PG_DB)
    pg_cur = pg_conn.cursor()
    neo_driver = GraphDatabase.driver(NEO4J_URI, auth=(NEO4J_USER, NEO4J_PASS))

    with neo_driver.session() as session:
        # ==========================================
        # 2. LIMPIEZA Y MODELADO DEL GRAFO
        # ==========================================
        print("Limpiando grafo anterior...")
        session.run("MATCH (n) DETACH DELETE n")

        # Índices para que los MATCH/MERGE por id sean rápidos (clave con miles de pedidos)
        session.run("CREATE INDEX user_id IF NOT EXISTS FOR (u:User) ON (u.id)")
        session.run("CREATE INDEX product_id IF NOT EXISTS FOR (p:Product) ON (p.id)")
        session.run("CREATE INDEX order_id IF NOT EXISTS FOR (o:Order) ON (o.id)")

        print("Migrando Usuarios y Productos...")
        pg_cur.execute("SELECT id, username FROM users")
        usuarios = [{"id": r[0], "username": r[1]} for r in pg_cur.fetchall()]
        session.run("""
            UNWIND $usuarios AS u
            CREATE (:User {id: u.id, username: u.username})
        """, usuarios=usuarios)

        pg_cur.execute("SELECT id, name, category FROM menu_items")
        productos = [{"id": r[0], "name": r[1], "category": r[2]} for r in pg_cur.fetchall()]
        session.run("""
            UNWIND $productos AS p
            CREATE (:Product {id: p.id, name: p.name, category: p.category})
        """, productos=productos)

        print("Migrando Pedidos / Canastas (Usuario -> Pedido -> Productos)...")
        # En 'orders' cada fila es UN producto. Para detectar productos comprados juntos,
        # agrupamos como una misma canasta (Order) todas las filas del mismo usuario en el
        # mismo instante (created_at). Así un Order puede contener varios productos.
        pg_cur.execute("""
            SELECT user_id, created_at, menu_item_id
            FROM orders
            WHERE user_id IS NOT NULL
              AND menu_item_id IS NOT NULL
              AND created_at IS NOT NULL
        """)
        pedidos = [
            {"order_key": f"{r[0]}_{r[1].isoformat()}", "user_id": r[0], "product_id": r[2]}
            for r in pg_cur.fetchall()
        ]
        session.run("""
            UNWIND $pedidos AS p
            MATCH (u:User {id: p.user_id})
            MATCH (prod:Product {id: p.product_id})
            MERGE (o:Order {id: p.order_key})
            MERGE (u)-[:PLACED_ORDER]->(o)
            MERGE (o)-[:CONTAINS_PRODUCT]->(prod)
        """, pedidos=pedidos)

        print("Migrando Recomendaciones (Relación Usuario -> Usuario)...")
        pg_cur.execute("SELECT usuario_origen, usuario_destino FROM recomendaciones")
        recomendaciones = [{"origen": r[0], "destino": r[1]} for r in pg_cur.fetchall()]
        session.run("""
            UNWIND $recomendaciones AS r
            MATCH (u1:User {id: r.origen})
            MATCH (u2:User {id: r.destino})
            MERGE (u1)-[:RECOMMENDS]->(u2)
        """, recomendaciones=recomendaciones)

        # ==========================================
        # 3. EJECUTAR LAS CONSULTAS SOLICITADAS EN EL PDF
        # ==========================================
        print("\n--- 1. LOS 5 PRODUCTOS MÁS COMPRADOS JUNTOS (Patrones de Co-compra) ---")
        q1 = session.run("""
            MATCH (p1:Product)<-[:CONTAINS_PRODUCT]-(o:Order)-[:CONTAINS_PRODUCT]->(p2:Product)
            WHERE p1.name < p2.name
            RETURN p1.name AS Prod1, p2.name AS Prod2, COUNT(DISTINCT o) AS VecesCompradosJuntos
            ORDER BY VecesCompradosJuntos DESC LIMIT 5
        """)
        for record in q1: print(f"{record['Prod1']} + {record['Prod2']} ({record['VecesCompradosJuntos']} veces)")

        print("\n--- 2. USUARIOS INFLUYENTES QUE RECOMIENDAN A OTROS ---")
        q2 = session.run("""
            MATCH (u1:User)-[:RECOMMENDS]->(u2:User)
            RETURN u1.username AS Influencer, COUNT(u2) AS TotalRecomendados
            ORDER BY TotalRecomendados DESC LIMIT 5
        """)
        for record in q2: print(f"{record['Influencer']}: {record['TotalRecomendados']} referidos")

    pg_cur.close()
    pg_conn.close()
    neo_driver.close()
    print("\n¡Proceso finalizado con éxito!")

if __name__ == "__main__":
    main()