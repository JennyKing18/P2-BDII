"""
conexion.py — Utilidades de conexion Spark <-> PostgreSQL.

Crea la SparkSession con el driver JDBC de Postgres y ofrece helpers para leer
y escribir tablas. Reutilizable por todos los jobs de spark/jobs/.
"""
from pyspark.sql import SparkSession

# Driver JDBC de Postgres que Spark descarga de Maven al iniciar la sesion.
PAQUETE_JDBC = "org.postgresql:postgresql:42.7.3"


def crear_sesion(nombre="P2-Spark"):
    """
    Crea (o reutiliza) una SparkSession con el driver JDBC de Postgres.

    Entradas:
        nombre (str): nombre de la aplicacion Spark.
    Salidas:
        SparkSession lista para leer/escribir por JDBC.
    Funcionamiento:
        spark.jars.packages hace que Spark baje el driver de Postgres desde
        Maven la primera vez (requiere internet en el contenedor).
    """
    return (SparkSession.builder
            .appName(nombre)
            .config("spark.jars.packages", PAQUETE_JDBC)
            .getOrCreate())


def _propiedades(usuario, clave):
    """
    Arma el dict de propiedades JDBC.

    Entradas:
        usuario (str), clave (str): credenciales de Postgres.
    Salidas:
        dict con user, password y driver.
    """
    return {"user": usuario, "password": clave, "driver": "org.postgresql.Driver"}


def leer_tabla(spark, url, tabla, usuario, clave):
    """
    Lee una tabla de Postgres como DataFrame.

    Entradas:
        spark: SparkSession.
        url (str): URL JDBC (jdbc:postgresql://host:puerto/bd).
        tabla (str): nombre de tabla, o una subconsulta entre parentesis.
        usuario, clave (str): credenciales.
    Salidas:
        DataFrame con el contenido de la tabla.
    """
    return spark.read.jdbc(url=url, table=tabla, properties=_propiedades(usuario, clave))


def escribir_tabla(df, url, tabla, usuario, clave, modo="overwrite"):
    """
    Escribe un DataFrame a una tabla de Postgres.

    Entradas:
        df: DataFrame a persistir.
        url (str): URL JDBC.
        tabla (str): tabla destino (ej. 'analytics.tendencias_categoria_mes').
        usuario, clave (str): credenciales.
        modo (str): 'overwrite' (idempotente, recrea) u 'append'.
    Salidas:
        Ninguna.
    Funcionamiento:
        'overwrite' deja el job idempotente: re-correrlo reemplaza el resultado
        en vez de duplicarlo (clave para el DAG de Airflow).
    """
    (df.write
       .jdbc(url=url, table=tabla, mode=modo, properties=_propiedades(usuario, clave)))