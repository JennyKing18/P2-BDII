from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta
import docker

# ==========================================
# CONFIGURACIÓN
# ==========================================
SPARK_CONTAINER = "spark_p2"

# spark-submit con el driver JDBC de Postgres
SPARK_SUBMIT = "/usr/local/spark/bin/spark-submit --packages org.postgresql:postgresql:42.7.3"

# Conexión a Postgres POR NOMBRE DE RED DE DOCKER ('db')
JDBC_URL = "jdbc:postgresql://db:5432/restaurantDB"
DB_USER = "admin_jenny"
DB_PASS = "abcdef"
DB_ARGS = f"--url {JDBC_URL} --usuario {DB_USER} --clave {DB_PASS}"

default_args = {
    'owner': 'estudiante',
    'depends_on_past': False,
    'start_date': datetime(2023, 1, 1),
    'retries': 1,
    'retry_delay': timedelta(minutes=2),
}

dag = DAG(
    '01_pipeline_datos_restaurante',
    default_args=default_args,
    description='Pipeline OLAP: Extrae, transforma con Spark, carga en DW y reindexa ElasticSearch',
    schedule_interval='@daily',  # Se ejecuta automáticamente una vez al día
    catchup=False,
    tags=['proyecto2', 'olap', 'spark', 'elasticsearch']
)


# ==========================================
# HELPERS: ejecutar comandos DENTRO del contenedor spark_p2
# ==========================================
def _exec_en_spark(comando):
    """Ejecuta un comando dentro del contenedor de Spark (equivale a docker exec)."""
    client = docker.from_env()
    try:
        contenedor = client.containers.get(SPARK_CONTAINER)
    except docker.errors.NotFound:
        raise Exception(f"¡Error! El contenedor '{SPARK_CONTAINER}' no está corriendo.")

    print(f"$ {comando}")
    exit_code, output = contenedor.exec_run(comando)

    print("----- SALIDA -----")
    print(output.decode('utf-8', errors='replace'))
    print("------------------")

    if exit_code != 0:
        raise Exception(f"El comando falló con código {exit_code} en {SPARK_CONTAINER}. Revisa los logs.")


def correr_spark_job(ruta, args=""):
    """Lanza un job de Spark con spark-submit + driver JDBC."""
    _exec_en_spark(f"{SPARK_SUBMIT} {ruta} {args}".strip())


def correr_python(ruta):
    """Corre un script de Python plano dentro de spark_p2 (sin Spark)."""
    _exec_en_spark(f"python {ruta}")


# ==========================================
# DEFINICIÓN DE LAS TAREAS DEL DAG
# ==========================================

# Tarea A: Extracción desde Postgres + Carga en el Data Warehouse (Hive)
tarea_etl_dw = PythonOperator(
    task_id='1_extraccion_transformacion_carga_DW',
    python_callable=correr_spark_job,
    op_kwargs={'ruta': '/home/jovyan/work/jobs/etl_hive.py', 'args': DB_ARGS},
    dag=dag,
)

# Tarea B: Análisis de Tendencias de Consumo con Spark (DataFrames + SparkSQL)
tarea_tendencias = PythonOperator(
    task_id='2_analisis_tendencias_spark',
    python_callable=correr_spark_job,
    op_kwargs={'ruta': '/home/jovyan/work/jobs/tendencias_consumo.py', 'args': DB_ARGS},
    dag=dag,
)

# Tarea C: Reindexado del catálogo de productos en ElasticSearch
tarea_reindexar = PythonOperator(
    task_id='3_reindexar_elasticsearch',
    python_callable=correr_python,
    op_kwargs={'ruta': '/home/jovyan/work/jobs/reindex_es.py'},
    dag=dag,
)

# ==========================================
# ORDEN DE EJECUCIÓN (Dependencias)
# ==========================================
# Llenar DW -> calcular tendencias -> reindexar búsqueda
tarea_etl_dw >> tarea_tendencias >> tarea_reindexar
