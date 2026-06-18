"""
tendencias_consumo.py — Analisis Spark #1: tendencias de consumo.

Calcula como evoluciona el consumo en el tiempo a partir de las ordenes,
cruzando con el catalogo para tener la categoria de cada producto. Produce dos
tablas en el esquema 'analytics' que alimentan los dashboards y el Data Warehouse.

Demuestra:
  - DataFrames (API funcional) -> ventas por categoria y mes
  - SparkSQL (consulta SQL)    -> top de productos por mes

Parametrizado para que el DAG de Airflow lo invoque con spark-submit.
"""
import argparse
import os
import sys

# Permite "from lib.conexion import ..." sin importar desde donde se corra.
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from lib.conexion import crear_sesion, leer_tabla, escribir_tabla  # noqa: E402

from pyspark.sql import functions as F  # noqa: E402


def cargar_ordenes_con_categoria(spark, url, usuario, clave):
    """
    F: Carga las ordenes cruzadas con la categoria de su producto.
    E:
        spark: SparkSession.
        url, usuario, clave: parametros JDBC.
    S:
        DataFrame con: mes (yyyy-MM), categoria, producto, total, status, creado.
    """
    ordenes = leer_tabla(spark, url, "orders", usuario, clave)
    items = leer_tabla(spark, url, "menu_items", usuario, clave)

    return (ordenes.alias("o")
            .join(items.alias("m"), F.col("o.menu_item_id") == F.col("m.id"))
            .select(
                F.date_format("o.created_at", "yyyy-MM").alias("mes"),
                F.col("m.category").alias("categoria"),
                F.col("m.name").alias("producto"),
                F.col("o.total").alias("total"),
                F.col("o.status").alias("status"),
                F.col("o.created_at").alias("creado"),
            ))


def ventas_por_categoria_mes(df):
    """
    F: Ingresos y volumen por categoria y mes (estilo DataFrames).
    E: df: DataFrame de cargar_ordenes_con_categoria.
    S: DataFrame: mes, categoria, ingresos, ordenes, ticket_promedio.
    """
    return (df.filter(F.col("status") == "completed")
            .groupBy("mes", "categoria")
            .agg(
                F.round(F.sum("total"), 2).alias("ingresos"),
                F.count("*").alias("ordenes"),
                F.round(F.avg("total"), 2).alias("ticket_promedio"),
            )
            .orderBy("mes", "categoria"))


def top_productos_por_mes(spark, df, n=10):
    """
    F: Top N productos mas consumidos por mes (estilo SparkSQL).
    E:
        spark: SparkSession.
        df: DataFrame de cargar_ordenes_con_categoria.
        n (int): cuantos productos por mes.
    S:
        DataFrame: mes, producto, categoria, unidades, ingresos, ranking.
    """
    df.filter(F.col("status") == "completed").createOrReplaceTempView("consumo")
    return spark.sql(f"""
        WITH ranking AS (
            SELECT mes, producto, categoria,
                   COUNT(*)             AS unidades,
                   ROUND(SUM(total), 2) AS ingresos,
                   ROW_NUMBER() OVER (PARTITION BY mes ORDER BY COUNT(*) DESC) AS ranking
            FROM consumo
            GROUP BY mes, producto, categoria
        )
        SELECT mes, producto, categoria, unidades, ingresos, ranking
        FROM ranking
        WHERE ranking <= {n}
        ORDER BY mes, ranking
    """)


def main():
    """
    F: Orquesta el analisis: carga, calcula y persiste en 'analytics'.
    E: -         
    S: -         
    """
    parser = argparse.ArgumentParser(description="Tendencias de consumo (Spark)")
    parser.add_argument("--url", default=os.getenv(
        "JDBC_URL", "jdbc:postgresql://host.docker.internal:5432/restaurantDB"))
    parser.add_argument("--usuario", default=os.getenv("DB_USER"))
    parser.add_argument("--clave", default=os.getenv("DB_PASSWORD"))
    parser.add_argument("--esquema-salida", default="analytics")
    parser.add_argument("--top", type=int, default=10)
    args = parser.parse_args()

    spark = crear_sesion("P2-TendenciasConsumo")
    spark.sparkContext.setLogLevel("WARN")

    df = cargar_ordenes_con_categoria(spark, args.url, args.usuario, args.clave)
    df.cache()  # se reutiliza en los dos analisis

    ventas = ventas_por_categoria_mes(df)
    top = top_productos_por_mes(spark, df, args.top)

    print("== Ventas por categoria y mes (muestra) ==")
    ventas.show(12, truncate=False)
    print("== Top productos por mes (muestra) ==")
    top.show(12, truncate=False)

    escribir_tabla(ventas, args.url, f"{args.esquema_salida}.tendencias_categoria_mes",
                   args.usuario, args.clave)
    escribir_tabla(top, args.url, f"{args.esquema_salida}.top_productos_mes",
                   args.usuario, args.clave)

    print("Tablas escritas en el esquema:", args.esquema_salida)
    spark.stop()

if __name__ == "__main__":
    main()