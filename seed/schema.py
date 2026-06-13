"""
schema.py — Extensiones de esquema del Proyecto 2 (idempotentes).

Aqui vive TODO el DDL que el P2 agrega sobre la base del P1:
  - columnas de geolocalizacion en users y restaurants
  - tablas nuevas: repartidores y recomendaciones

Decision de diseno: estas estructuras NO se agregan al modelo de dominio Go.
El app Go es el sistema OLTP de origen y se mantiene intacto; estas son
extensiones de la capa analitica del P2 (las consumen Spark, Neo4j y el modulo
de enrutamiento, no la API). GORM AutoMigrate es aditivo: ignora columnas y
tablas que no conoce, asi que la API sigue funcionando igual.
"""

# DDL idempotente: se puede correr multiples veces sin error.
DDL_P2 = """
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
    Aplica las extensiones de esquema del P2.

    Entradas:
        cur: cursor psycopg2 activo.
    Salidas:
        Ninguna.
    Funcionamiento:
        Ejecuta el DDL idempotente (IF NOT EXISTS): es seguro correrlo en cada
        ejecucion del seed. No hace commit.
    """
    cur.execute(DDL_P2)