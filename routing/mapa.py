"""
mapa.py — Render del mapa de rutas con folium.
"""
import folium

COLORES = ["red", "blue", "green", "purple", "orange", "darkred", "cadetblue",
           "darkgreen", "darkblue", "darkpurple", "pink", "gray", "black"]


def dibujar(repartidores, rutas, salida):
    """
    F: Dibuja repartidores, paradas y la polilinea de cada ruta en un mapa HTML.
    E: repartidores (list), rutas (dict rep_id -> [(pedido,orden,dist)]), salida (path).
    S: ninguna (guarda el HTML).
    """
    centro_lat = sum(r["lat"] for r in repartidores) / len(repartidores)
    centro_lon = sum(r["lon"] for r in repartidores) / len(repartidores)
    m = folium.Map(location=[centro_lat, centro_lon], zoom_start=11)

    por_id = {r["id"]: r for r in repartidores}
    for i, (rep_id, ruta) in enumerate(rutas.items()):
        color = COLORES[i % len(COLORES)]
        rep = por_id[rep_id]
        puntos = [(rep["lat"], rep["lon"])] + [(p["lat"], p["lon"]) for p, _, _ in ruta]

        folium.Marker([rep["lat"], rep["lon"]], tooltip=f"{rep['nombre']} (inicio)",
                      icon=folium.Icon(color=color, icon="home")).add_to(m)
        for p, orden, _ in ruta:
            folium.CircleMarker([p["lat"], p["lon"]], radius=4, color=color, fill=True,
                                tooltip=f"Pedido {p['pedido_id']} (parada {orden})").add_to(m)
        folium.PolyLine(puntos, color=color, weight=2, opacity=0.7).add_to(m)

    m.save(salida)