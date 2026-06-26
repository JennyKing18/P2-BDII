"""
verificar_dw.py — Comprobacion de solo lectura del Data Warehouse.

 lista tablas, cuenta filas de dims/hechos y ejecuta las 5
vistas OLAP para confirmar que devuelven datos. Sirve como evidencia de que el DW funciona.

"""
from pyspark.sql import SparkSession

WAREHOUSE_DIR = "/home/jovyan/work/spark-warehouse"
DW_DATABASE = "restaurant_dw"

TABLAS = ["dim_usuario", "dim_restaurante", "dim_producto", "dim_tiempo",
          "fact_pedidos", "fact_reservas"]

VISTAS = ["v_ingresos_mes_categoria", "v_actividad_clientes_zona",
          "v_frecuencia_uso_clientes", "v_patrones_reservas_zona",
          "v_estadisticas_pedidos_estado"]


def main():
    spark = SparkSession.builder \
        .appName("P2-VerificacionDW") \
        .config("spark.sql.catalogImplementation", "hive") \
        .config("spark.sql.warehouse.dir", WAREHOUSE_DIR) \
        .enableHiveSupport() \
        .getOrCreate()
    spark.sparkContext.setLogLevel("WARN")
    spark.sql(f"USE {DW_DATABASE}")

    print("=== Conteo de filas por tabla ===")
    for t in TABLAS:
        n = spark.sql(f"SELECT COUNT(*) AS n FROM {t}").collect()[0]["n"]
        print(f"  {t:<18} {n:>10,} filas")

    print("\n=== Ejecucion real de las 5 vistas OLAP ===")
    for v in VISTAS:
        n = spark.sql(f"SELECT COUNT(*) AS n FROM {v}").collect()[0]["n"]
        print(f"\n--- {v}  ({n:,} filas) ---")
        spark.sql(f"SELECT * FROM {v} LIMIT 5").show(truncate=False)

    spark.stop()


if __name__ == "__main__":
    main()
