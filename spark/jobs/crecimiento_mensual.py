"""
crecimiento_mensual.py — Analisis Spark #3: crecimiento mensual.
% de crecimiento mes a mes con SparkSQL (LAG). Escribe analytics.crecimiento_mensual.
"""
import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from lib.conexion import crear_sesion, leer_tabla, escribir_tabla  # noqa: E402

from pyspark.sql import functions as F  # noqa: E402


def cargar_ordenes(spark, url, usuario, clave):
    """
    F: Lee orders 'completed' y deriva el mes (yyyy-MM).
    E: spark, url, usuario, clave.
    S: DataFrame: mes, total.
    """
    ordenes = leer_tabla(spark, url, "orders", usuario, clave)
    return (ordenes.filter(F.col("status") == "completed")
            .select(F.date_format("created_at", "yyyy-MM").alias("mes"), F.col("total")))


def crecimiento_mensual(spark, df):
    """
    F: Ingresos/ordenes por mes y % de crecimiento vs mes anterior (LAG).
    E: spark, df de cargar_ordenes.
    S: DataFrame: mes, ordenes, ingresos, ingresos_prev, crecimiento_pct.
    """
    df.createOrReplaceTempView("ventas")
    return spark.sql("""
        WITH mensual AS (
            SELECT mes, COUNT(*) AS ordenes, ROUND(SUM(total), 2) AS ingresos
            FROM ventas GROUP BY mes
        )
        SELECT mes, ordenes, ingresos,
               LAG(ingresos) OVER (ORDER BY mes) AS ingresos_prev,
               ROUND(100 * (ingresos - LAG(ingresos) OVER (ORDER BY mes))
                     / LAG(ingresos) OVER (ORDER BY mes), 2) AS crecimiento_pct
        FROM mensual
        ORDER BY mes
    """)


def main():
    """
    F: Calcula crecimiento mensual y persiste en analytics.
    E: args CLI (--url, --usuario, --clave, --esquema-salida).
    S: Ninguna (escribe 1 tabla).
    """
    p = argparse.ArgumentParser(description="Crecimiento mensual (Spark)")
    p.add_argument("--url", default=os.getenv("JDBC_URL", "jdbc:postgresql://db:5432/restaurantDB"))
    p.add_argument("--usuario", default=os.getenv("DB_USER"))
    p.add_argument("--clave", default=os.getenv("DB_PASSWORD"))
    p.add_argument("--esquema-salida", default="analytics")
    args = p.parse_args()

    spark = crear_sesion("P2-CrecimientoMensual")
    spark.sparkContext.setLogLevel("WARN")

    df = cargar_ordenes(spark, args.url, args.usuario, args.clave)
    resultado = crecimiento_mensual(spark, df)
    resultado.show(24, truncate=False)

    escribir_tabla(resultado, args.url, f"{args.esquema_salida}.crecimiento_mensual", args.usuario, args.clave)
    spark.stop()


if __name__ == "__main__":
    main()