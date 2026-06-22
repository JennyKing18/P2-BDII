"""
dominio.py — Conocimiento del dominio para generar datos realistas.
Reune las constantes (zonas, categorias, platos, pesos temporales)
"""
import random
from datetime import datetime, timedelta

# Centros aproximados (lat, lon) de zonas de la Gran Area Metropolitana
ZONAS = {
    "Cartago":      (9.8644, -83.9194),
    "San Jose":     (9.9281, -84.0907),
    "Escazu":       (9.9189, -84.1398),
    "Heredia":      (9.9981, -84.1166),
    "Alajuela":     (10.0162, -84.2117),
    "Curridabat":   (9.9155, -84.0334),
    "Tres Rios":    (9.9061, -83.9883),
    "Desamparados": (9.8977, -84.0625),
}

CATEGORIAS = ["meat", "vegan", "drink", "dessert", "seafood", "pasta", "breakfast"]

# Platos por categoria (para nombres de menu_items con sentido)
PLATOS = {
    "meat":      ["Casado con bistec", "Chuleta ahumada", "Lomito", "Pollo asado", "Costilla BBQ"],
    "vegan":     ["Bowl de quinoa", "Gallo pinto vegano", "Ensalada cesar vegana", "Wrap de tofu"],
    "drink":     ["Fresco de cas", "Cafe chorreado", "Batido de mora", "Te frio", "Horchata"],
    "dessert":   ["Tres leches", "Flan de coco", "Brownie", "Granizado"],
    "seafood":   ["Ceviche de corvina", "Arroz con camarones", "Pescado entero"],
    "pasta":     ["Lasagna", "Fetuccini alfredo", "Ravioles"],
    "breakfast": ["Gallo pinto", "Huevos rancheros", "Tostadas francesas"],
}

NOMBRES_REST = ["La Esquina", "El Fogon", "Sabor Tico", "Don Carlos", "La Terraza",
                "El Jardin", "Casa Vieja", "Rincon", "Delicias", "El Patio"]
SUFIJOS_REST = ["del Valle", "Central", "Express", "Gourmet", "Criollo", "2000",
                "de la Montana", "del Parque"]

# Ventana historica (>6 meses para que se vea el "crecimiento mensual")
INICIO = datetime(2025, 11, 1)
FIN = datetime(2026, 6, 11)
DIAS_TOTALES = (FIN - INICIO).days

# Pesos por hora: picos de almuerzo (12-13) y cena (19-20)
PESO_HORA = {11: 5, 12: 14, 13: 12, 14: 6, 15: 3, 16: 3, 17: 5,
             18: 9, 19: 13, 20: 11, 21: 6, 22: 3}
# Pesos por dia de semana (lun=0 .. dom=6): fin de semana mas fuerte
PESO_DIA = [7, 6, 7, 8, 12, 14, 10]


def punto_en(zona):
    """
    F: Genera un punto (lat, lon) disperso alrededor del centro de una zona.
    E: zona (str): clave existente en ZONAS.
    S: tuple[float, float]: (latitud, longitud)
    """
    lat, lon = ZONAS[zona]
    return (round(lat + random.uniform(-0.02, 0.02), 6),
            round(lon + random.uniform(-0.02, 0.02), 6))


def fecha_aleatoria():
    """
    F: Genera una fecha/hora sesgada hacia meses recientes y horas pico.
    E: -
    S: datetime dentro de la ventana [INICIO, FIN].
    """
    while True:
        dia = random.randint(0, DIAS_TOTALES - 1)
        crecimiento = 1.0 + 1.2 * (dia / DIAS_TOTALES)
        if random.random() < crecimiento / 2.2:
            break
    base = INICIO + timedelta(days=dia)
    if random.random() > PESO_DIA[base.weekday()] / 14:
        base += timedelta(days=(4 - base.weekday()) % 7)  # acerca al viernes
    hora = random.choices(list(PESO_HORA), weights=list(PESO_HORA.values()))[0]
    return base.replace(hour=hora, minute=random.randint(0, 59),
                        second=random.randint(0, 59))