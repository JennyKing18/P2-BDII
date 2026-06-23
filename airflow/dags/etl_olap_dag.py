from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta
import docker
import requests

# 1. Configuración básica de cómo y cuándo se ejecuta el pipeline
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
    description='Pipeline OLAP: Extrae, transforma con Spark, carga en DW y actualiza ElasticSearch',
    schedule_interval='@daily', # Se ejecuta automáticamente una vez al día
    catchup=False,
    tags=['proyecto2', 'olap', 'spark', 'elasticsearch']
)

# 2. Función auxiliar para ejecutar scripts DENTRO del contenedor de Spark
def ejecutar_script_spark(ruta_script):
    client = docker.from_env()
    # Buscamos el contenedor llamado 'spark_p2'
    try:
        contenedor_spark = client.containers.get('spark_p2')
    except docker.errors.NotFound:
        raise Exception("¡Error! El contenedor 'spark_p2' no está corriendo.")

    print(f"Ejecutando script en Spark: {ruta_script}...")
    
    # Esto equivale a hacer un 'docker exec' en la terminal
    exit_code, output = contenedor_spark.exec_run(f"python {ruta_script}")
    
    print("----- RESULTADO DE SPARK -----")
    print(output.decode('utf-8'))
    print("------------------------------")
    
    if exit_code != 0:
        raise Exception(f"El script de Spark falló con código {exit_code}. Revisa los logs.")

# 3. Función para avisarle a tu API en Go que debe reindexar
def reindexar_elasticsearch():
    print("Llamando al microservicio search-service para reindexar...")
    # Como Airflow y search-service están en la misma red de Docker, usamos el nombre del contenedor
    url = "http://search-service:8080/reindex"
    
    response = requests.post(url)
    
    print(f"Código de respuesta HTTP: {response.status_code}")
    if response.status_code not in [200, 201]:
        raise Exception(f"Fallo al reindexar ElasticSearch. Respuesta: {response.text}")
    print("¡Reindexado completado con éxito!")

# ==========================================
# DEFINICIÓN DE LAS TAREAS DEL DAG
# ==========================================

# Tarea A: Extracción desde Postgres y Carga en Data Warehouse (Hive) usando tu script
tarea_etl_dw = PythonOperator(
    task_id='1_extraccion_transformacion_carga_DW',
    python_callable=ejecutar_script_spark,
    # NOTA: Asegúrate de que esta ruta sea la correcta dentro de la carpeta /home/jovyan/work
    op_kwargs={'ruta_script': '/home/jovyan/work/etl_hive.py'},
    dag=dag,
)

# Tarea B: Análisis de Tendencias de Consumo con Spark
tarea_tendencias = PythonOperator(
    task_id='2_analisis_tendencias_spark',
    python_callable=ejecutar_script_spark,
    # En tu comentario dijiste que usabas "jobs/tendencias_consumo.py"
    op_kwargs={'ruta_script': '/home/jovyan/work/jobs/tendencias_consumo.py'},
    dag=dag,
)

# Tarea C: Reindexado del catálogo en ElasticSearch
tarea_reindexar = PythonOperator(
    task_id='3_reindexar_elasticsearch',
    python_callable=reindexar_elasticsearch,
    dag=dag,
)

# ==========================================
# ORDEN DE EJECUCIÓN (Dependencias)
# ==========================================
# Primero llenamos el DW -> Luego calculamos tendencias -> Por último reindexamos búsqueda
tarea_etl_dw >> tarea_tendencias >> tarea_reindexar