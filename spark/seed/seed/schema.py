"""
schema.py — Extensiones de esquema del Proyecto 2 (idempotentes).

DataDefinitionLanguage (DDL) agrega sobre la base del P1:
  - columnas de geolocalizacion en users y restaurants
  - tablas nuevas: repartidores y recomendaciones
"""

DDL_P2 = """
-- Esquema de salida para los resultados de Spark/OLAP
CREATE SCHEMA IF NOT EXISTS analytics;

-- Geolocalizacion sobre entidades existentes (la API las ignora)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS latitud  DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitud DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS zona     TEXT;

ALTER TABLE restaurants
    ADD COLUMN IF NOT EXISTS latitud  DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS longitud DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS zona     TEXT;

-- Repartidores: insumo del modulo de enrutamiento y de los geonodos de Neo4j
CREATE TABLE IF NOT EXISTS repartidores (
    id         BIGSERIAL PRIMARY KEY,
    nombre     TEXT NOT NULL,
    zona       TEXT NOT NULL,
    latitud    DOUBLE PRECISION NOT NULL,
    longitud   DOUBLE PRECISION NOT NULL,
    activo     BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Recomendaciones usuario->usuario: insumo del grafo Neo4j.
-- Datos sinteticos; la API nunca las crea.
CREATE TABLE IF NOT EXISTS recomendaciones (
    id              BIGSERIAL PRIMARY KEY,
    usuario_origen  BIGINT NOT NULL REFERENCES users(id),
    usuario_destino BIGINT NOT NULL REFERENCES users(id),
    fecha           TIMESTAMPTZ NOT NULL,
    UNIQUE (usuario_origen, usuario_destino)
);
"""


def aplicar_esquema(cur):
    """
    F: Aplica las extensiones de esquema del P2.
    E: cur: cursor psycopg2 activo.
    S: -    
    """
    cur.execute(DDL_P2)