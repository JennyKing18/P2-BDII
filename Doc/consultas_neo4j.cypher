// ============================================================================
// Neo4J — Estructura del grafo y consultas Cypher
// 
//
// La CARGA de datos la hacen los scripts neo4j_grafos.py y neo4j_caminos.py
// (extraen de PostgreSQL). Este archivo documenta los ÍNDICES (estructura) y
// reúne las CONSULTAS para ejecutarlas/validarlas en el Neo4j Browser.
// ============================================================================

// ----------------------------------------------------------------------------
// 1. ESTRUCTURA — indices del grafo
// ----------------------------------------------------------------------------

// Subgrafo de consumo y recomendaciones
CREATE INDEX user_id    IF NOT EXISTS FOR (u:User)    ON (u.id);
CREATE INDEX product_id IF NOT EXISTS FOR (p:Product) ON (p.id);
CREATE INDEX order_id   IF NOT EXISTS FOR (o:Order)   ON (o.id);

// Subgrafo de rutas (geonodos)
CREATE INDEX ubicacion_id IF NOT EXISTS FOR (u:Ubicacion) ON (u.id);

// Modelo (referencia):
//   (:User {id, username})
//   (:Product {id, name, category})
//   (:Order {id})                         id = "<user_id>_<created_at>" (canasta)
//   (:User)-[:PLACED_ORDER]->(:Order)
//   (:Order)-[:CONTAINS_PRODUCT]->(:Product)
//   (:User)-[:RECOMMENDS]->(:User)
//   (:Ubicacion {id, nombre, tipo, lat, lon})   tipo = 'Repartidor' | 'Cliente'
//   (:Ubicacion)-[:CONECTA {km}]->(:Ubicacion)  red kNN (k=6), peso = distancia haversine

// ----------------------------------------------------------------------------
// 2. CONSULTAS EXIGIDAS POR EL ENUNCIADO (sección 5)
// ----------------------------------------------------------------------------

// 2.1 — Los 5 productos más comprados juntos (co-compra)
MATCH (p1:Product)<-[:CONTAINS_PRODUCT]-(o:Order)-[:CONTAINS_PRODUCT]->(p2:Product)
WHERE p1.name < p2.name
RETURN p1.name AS Prod1,
       p2.name AS Prod2,
       COUNT(DISTINCT o) AS VecesCompradosJuntos
ORDER BY VecesCompradosJuntos DESC
LIMIT 5;

// 2.2 — Usuarios que recomiendan a otros (influyentes)
MATCH (u1:User)-[:RECOMMENDS]->(u2:User)
RETURN u1.username AS Influencer,
       COUNT(u2) AS TotalRecomendados
ORDER BY TotalRecomendados DESC
LIMIT 5;

// 2.3 — Caminos mínimos entre ubicaciones para reparto eficiente (Dijkstra/APOC)

MATCH (o:Ubicacion {id: 'REP_1'})
MATCH (d:Ubicacion {tipo: 'Cliente'}) WHERE d <> o
CALL apoc.algo.dijkstra(o, d, 'CONECTA', 'km') YIELD path, weight
RETURN d.nombre AS cliente,
       round(weight, 2) AS km_optimo,
       length(path)     AS saltos,
       [n IN nodes(path) | n.nombre] AS ruta
ORDER BY km_optimo DESC
LIMIT 5;

// ----------------------------------------------------------------------------
// 3. CONSULTAS AUXILIARES (exploración / evidencia)
// ----------------------------------------------------------------------------

// 3.1 — Vista general del grafo de consumo
MATCH (u:User)-[:PLACED_ORDER]->(o:Order)-[:CONTAINS_PRODUCT]->(p:Product)
RETURN u, o, p LIMIT 50;

// 3.2 — Red de ubicaciones (geonodos) con sus distancias
MATCH (a:Ubicacion)-[c:CONECTA]->(b:Ubicacion)
RETURN a, c, b LIMIT 50;

// 3.3 — Camino mínimo concreto entre dos ubicaciones (dibuja la ruta)
MATCH (o:Ubicacion {nombre:'Repartidor 01'}), (d:Ubicacion {nombre:'cliente_0319'})
CALL apoc.algo.dijkstra(o, d, 'CONECTA', 'km') YIELD path, weight
RETURN path, weight;

// 3.4 — Conteo de nodos por tipo (evidencia de carga)
MATCH (n) RETURN labels(n) AS tipo, count(*) AS cantidad ORDER BY cantidad DESC;
