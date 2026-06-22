"""
etl_hive.py — Poblamiento del Data Warehouse OLAP en Hive.
Extrae de PostgreSQL, transforma a Modelo Estrella y carga en Apache Hive.
"""
import os
import shutil
import argparse
from pyspark.sql import SparkSession
from pyspark.sql import functions as F

WAREHOUSE_DIR = "/home/jovyan/work/spark-warehouse"
DW_DATABASE = "restaurant_dw"

def crear_sesion_hive(app_name="P2-DataWarehouse"):
    """Crea una sesión de Spark con soporte nativo para el Metastore de Hive."""
    return SparkSession.builder \
        .appName(app_name) \
        .config("spark.sql.catalogImplementation", "hive") \
        .config("spark.sql.warehouse.dir", WAREHOUSE_DIR) \
        .enableHiveSupport() \
        .getOrCreate()

def reiniciar_dw(spark, database=DW_DATABASE, warehouse_dir=WAREHOUSE_DIR):
    """
    Deja el Data Warehouse en un estado limpio para que el ETL sea
    re-ejecutable (idempotente).

    El warehouse se monta como volumen (persiste), pero el metastore Derby
    vive dentro del contenedor (se reinicia). Esa desincronizacion provoca
    'LOCATION_ALREADY_EXISTS'. Borramos ambos lados antes de recrear:
      1. El registro en el metastore (DROP DATABASE ... CASCADE).
      2. La carpeta fisica .db por si quedaron archivos huerfanos.
    """
    spark.sql(f"DROP DATABASE IF EXISTS {database} CASCADE")
    db_dir = os.path.join(warehouse_dir, f"{database}.db")
    if os.path.isdir(db_dir):
        shutil.rmtree(db_dir)

def leer_pg(spark, url, tabla, usuario, clave):
    """Lee una tabla desde PostgreSQL usando JDBC."""
    return spark.read.format("jdbc") \
        .option("url", url) \
        .option("dbtable", tabla) \
        .option("user", usuario) \
        .option("password", clave) \
        .option("driver", "org.postgresql.Driver") \
        .load()

def columna_segura(df, nombre_columna, valor_default="Desconocida"):
    """
    Verifica si una columna existe en el origen (ej: 'zona' que se inserta por seed).
    Si existe la devuelve, si no, devuelve un valor por defecto.
    """
    if nombre_columna in df.columns:
        return F.col(nombre_columna)
    return F.lit(valor_default).alias(nombre_columna)

def main():
    parser = argparse.ArgumentParser(description="ETL Data Warehouse (Hive)")
    parser.add_argument("--url", default=os.getenv("JDBC_URL", "jdbc:postgresql://db:5432/restaurantDB"))
    parser.add_argument("--usuario", default=os.getenv("DB_USER"))
    parser.add_argument("--clave", default=os.getenv("DB_PASSWORD"))
    args = parser.parse_args()

    spark = crear_sesion_hive()
    spark.sparkContext.setLogLevel("WARN")

    print("=== 1. Extrayendo datos de PostgreSQL ===")
    users_df = leer_pg(spark, args.url, "users", args.usuario, args.clave)
    rests_df = leer_pg(spark, args.url, "restaurants", args.usuario, args.clave)
    items_df = leer_pg(spark, args.url, "menu_items", args.usuario, args.clave)
    orders_df = leer_pg(spark, args.url, "orders", args.usuario, args.clave)
    reservas_df = leer_pg(spark, args.url, "reservations", args.usuario, args.clave)

    print("=== 2. Creando Base de Datos en Hive ===")
    reiniciar_dw(spark)
    spark.sql(f"CREATE DATABASE IF NOT EXISTS {DW_DATABASE}")
    spark.sql(f"USE {DW_DATABASE}")

    print("=== 3. Transformando y Cargando Dimensiones en Hive ===")
    
    # DIM_USUARIO (uso columna_segura para 'zona' en caso de que no se haya corrido el seed)
    dim_usuario = users_df.select(
        F.col("id").alias("user_id"), 
        "username", 
        "email", 
        "role", 
        columna_segura(users_df, "zona")
    )
    dim_usuario.write.mode("overwrite").saveAsTable("dim_usuario")

    # DIM_RESTAURANTE
    dim_restaurante = rests_df.select(
        F.col("id").alias("restaurant_id"), 
        "name", 
        columna_segura(rests_df, "zona")
    )
    dim_restaurante.write.mode("overwrite").saveAsTable("dim_restaurante")

    # DIM_PRODUCTO
    dim_producto = items_df.select(
        F.col("id").alias("product_id"), 
        "name", 
        columna_segura(items_df, "category", "Sin Categoria").alias("category"), 
        "price"
    )
    dim_producto.write.mode("overwrite").saveAsTable("dim_producto")

    # DIM_TIEMPO
    fechas_ordenes = orders_df.select("created_at")
    fechas_reservas = reservas_df.select("created_at")
    todas_fechas = fechas_ordenes.union(fechas_reservas).dropDuplicates()

    dim_tiempo = todas_fechas.select(
        F.date_format("created_at", "yyyyMMddHH").alias("id_tiempo"),
        F.to_date("created_at").alias("fecha"),
        F.year("created_at").alias("anio"),
        F.month("created_at").alias("mes"),
        F.dayofmonth("created_at").alias("dia"),
        F.hour("created_at").alias("hora"),
        F.dayofweek("created_at").alias("dia_semana")
    ).dropDuplicates(["id_tiempo"])
    dim_tiempo.write.mode("overwrite").saveAsTable("dim_tiempo")

    print("=== 4. Transformando y Cargando Tablas de Hechos en Hive ===")
    
    # FACT_PEDIDOS
    fact_pedidos = orders_df.select(
        F.col("id").alias("order_id"),
        F.col("user_id"),
        F.col("menu_item_id").alias("product_id"),
        F.col("restaurant_id"),
        F.date_format("created_at", "yyyyMMddHH").alias("id_tiempo"),
        F.col("total"),
        F.col("status"),
        F.year("created_at").alias("anio"),
        F.month("created_at").alias("mes")
    )
    spark.conf.set("spark.sql.sources.partitionOverwriteMode", "dynamic")
    fact_pedidos.write.mode("overwrite").partitionBy("anio", "mes").saveAsTable("fact_pedidos")

    # FACT_RESERVAS
    fact_reservas = reservas_df.select(
        F.col("id").alias("reservation_id"),
        F.col("user_id"),
        F.col("restaurant_id"),
        F.date_format("created_at", "yyyyMMddHH").alias("id_tiempo"),
        F.col("status"),
        F.year("created_at").alias("anio"),
        F.month("created_at").alias("mes")
    )
    fact_reservas.write.mode("overwrite").partitionBy("anio", "mes").saveAsTable("fact_reservas")

    print("=== 5. Creando los 5 Cubos/Vistas OLAP obligatorios ===")
    
    # 1. Por Tiempo y Tipo de Producto
    spark.sql("""
        CREATE OR REPLACE VIEW v_ingresos_mes_categoria AS
        SELECT t.anio, t.mes, p.category AS tipo_producto,
               ROUND(SUM(f.total), 2) AS ingresos_totales, COUNT(f.order_id) AS cantidad_pedidos
        FROM fact_pedidos f
        JOIN dim_tiempo t ON f.id_tiempo = t.id_tiempo
        JOIN dim_producto p ON f.product_id = p.product_id
        WHERE f.status = 'completed'
        GROUP BY t.anio, t.mes, p.category
    """)

    # 2. Por Ubicación (Zona)
    spark.sql("""
        CREATE OR REPLACE VIEW v_actividad_clientes_zona AS
        SELECT u.zona AS zona_cliente, r.zona AS zona_restaurante,
               COUNT(f.order_id) AS total_pedidos, ROUND(SUM(f.total), 2) AS volumen_dinero
        FROM fact_pedidos f
        JOIN dim_usuario u ON f.user_id = u.user_id
        JOIN dim_restaurante r ON f.restaurant_id = r.restaurant_id
        GROUP BY u.zona, r.zona
    """)

    # 3. Por Frecuencia de Uso
    spark.sql("""
        CREATE OR REPLACE VIEW v_frecuencia_uso_clientes AS
        SELECT t.anio, t.mes, u.username,
               COUNT(f.order_id) AS frecuencia_compras, ROUND(SUM(f.total), 2) AS total_gastado
        FROM fact_pedidos f
        JOIN dim_tiempo t ON f.id_tiempo = t.id_tiempo
        JOIN dim_usuario u ON f.user_id = u.user_id
        GROUP BY t.anio, t.mes, u.username
        ORDER BY frecuencia_compras DESC
    """)

    # 4. Patrones de reservas (Ubicación + Tiempo)
    spark.sql("""
        CREATE OR REPLACE VIEW v_patrones_reservas_zona AS
        SELECT r.zona AS zona_restaurante, t.dia_semana, t.hora, f.status,
               COUNT(f.reservation_id) AS cantidad_reservas
        FROM fact_reservas f
        JOIN dim_tiempo t ON f.id_tiempo = t.id_tiempo
        JOIN dim_restaurante r ON f.restaurant_id = r.restaurant_id
        GROUP BY r.zona, t.dia_semana, t.hora, f.status
    """)

    # 5. Estadísticas Completados vs Cancelados
    spark.sql("""
        CREATE OR REPLACE VIEW v_estadisticas_pedidos_estado AS
        SELECT t.anio, t.mes, f.status AS estado_pedido,
               COUNT(f.order_id) AS cantidad_pedidos, ROUND(SUM(f.total), 2) AS dinero_involucrado
        FROM fact_pedidos f
        JOIN dim_tiempo t ON f.id_tiempo = t.id_tiempo
        GROUP BY t.anio, t.mes, f.status
    """)

    print("=== ¡Data Warehouse y Vistas OLAP creados exitosamente en Hive! ===")
    
    print("\n--- Muestra del Cubo 1 (Ingresos por Mes y Categoría) ---")
    spark.sql("SELECT * FROM v_ingresos_mes_categoria LIMIT 5").show()

    spark.stop()

if __name__ == "__main__":
    main()