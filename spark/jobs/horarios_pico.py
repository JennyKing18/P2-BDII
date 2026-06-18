"""
horarios_pico.py — Analisis Spark #2: horarios pico.
Escribe analytics.ordenes_por_hora y analytics.ordenes_por_hora_dia.
"""
import argparse
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from lib.conexion import crear_sesion, leer_tabla, escribir_tabla  # noqa: E402

from pyspark.sql import functions as F  # noqa: E402


def cargar_ordenes(spark, url, usuario, clave):
    """
    F: Lee orders y deriva hora (0-23) y dia de semana (1=dom..7=sab).
    E: spark, url, usuario, clave.
    S: DataFrame con hora, dia_semana, total, status.
    """
    ordenes = leer_tabla(spark, url, "orders", usuario, clave)
    return ordenes.select(
        F.hour("created_at").alias("hora"),
        F.dayofweek("created_at").alias("dia_semana"),
        F.col("total"),
        F.col("status"),
    )


def ordenes_por_hora(df):
    """
    F: Cuenta ordenes e ingresos por hora del dia.
    E: df de cargar_ordenes.
    S: DataFrame: hora, ordenes, ingresos.
    """
    return (df.groupBy("hora")
            .agg(F.count("*").alias("ordenes"),
                 F.round(F.sum("total"), 2).alias("ingresos"))
            .orderBy("hora"))


def ordenes_por_hora_dia(df):
    """
    F: Matriz dia-de-semana x hora con conteo de ordenes (heatmap).
    E: df de cargar_ordenes.
    S: DataFrame: dia_semana, hora, ordenes.
    """
    return (df.groupBy("dia_semana", "hora")
            .agg(F.count("*").alias("ordenes"))
            .orderBy("dia_semana", "hora"))


def main():
    """
    F: Calcula picos por hora y por hora/dia, persiste en analytics.
    E: args CLI (--url, --usuario, --clave, --esquema-salida).
    S: Ninguna (escribe 2 tablas).
    """
    p = argparse.ArgumentParser(description="Horarios pico (Spark)")
    p.add_argument("--url", default=os.getenv("JDBC_URL", "jdbc:postgresql://db:5432/restaurantDB"))
    p.add_argument("--usuario", default=os.getenv("DB_USER"))
    p.add_argument("--clave", default=os.getenv("DB_PASSWORD"))
    p.add_argument("--esquema-salida", default="analytics")
    args = p.parse_args()

    spark = crear_sesion("P2-HorariosPico")
    spark.sparkContext.setLogLevel("WARN")

    df = cargar_ordenes(spark, args.url, args.usuario, args.clave)
    df.cache()

    por_hora = ordenes_por_hora(df)
    por_hora_dia = ordenes_por_hora_dia(df)
    por_hora.show(24, truncate=False)
    por_hora_dia.show(12, truncate=False)

    escribir_tabla(por_hora, args.url, f"{args.esquema_salida}.ordenes_por_hora", args.usuario, args.clave)
    escribir_tabla(por_hora_dia, args.url, f"{args.esquema_salida}.ordenes_por_hora_dia", args.usuario, args.clave)
    spark.stop()


if __name__ == "__main__":
    main()