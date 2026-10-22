#!/usr/bin/env python3
import json
import subprocess
from collections import Counter

print("=" * 70)
print("RELATÓRIO DE ANÁLISE DE LOGS & TESTES DE CARGA - PROJETO NOVA")
print("=" * 70)

# 1. Análise Caddy (Proxy HTTPS / Terminação de Conexão)
cmd_caddy = ["docker", "compose", "logs", "caddy", "--since", "30m"]
out_caddy = subprocess.check_output(cmd_caddy, cwd="/opt/projeto-nova").decode("utf-8", errors="ignore")

caddy_logs = []
for line in out_caddy.splitlines():
    if "http.log.access" in line:
        idx = line.find("{")
        if idx != -1:
            try:
                caddy_logs.append(json.loads(line[idx:]))
            except Exception:
                pass

print(f"\n[1] TRÁFEGO PROCESSADO PELO CADDY (Últimos 30 min):")
print(f"Total de requisições registradas: {len(caddy_logs)}")

status_caddy = Counter(l.get("status") for l in caddy_logs)
print("Distribuição de Status HTTP:")
for s, c in sorted(status_caddy.items()):
    pct = (c / len(caddy_logs)) * 100 if caddy_logs else 0
    print(f"  HTTP {s}: {c} ({pct:.1f}%)")

uris_caddy = Counter(l.get("request", {}).get("uri", "").split("?")[0] for l in caddy_logs)
print("\nEndpoints mais requisitados:")
for u, count in uris_caddy.most_common(8):
    print(f"  {count}x {u}")

print("\nDetalhamento por Status HTTP:")
for st in [200, 404, 429]:
    st_uris = Counter(l.get("request", {}).get("uri", "").split("?")[0] for l in caddy_logs if l.get("status") == st)
    if st_uris:
        print(f"  HTTP {st}:")
        for u, count in st_uris.most_common(5):
            print(f"    {count}x {u}")

caddy_durations = [l.get("duration", 0) * 1000 for l in caddy_logs if l.get("duration")]
if caddy_durations:
    caddy_durations.sort()
    avg_d = sum(caddy_durations) / len(caddy_durations)
    p50 = caddy_durations[int(len(caddy_durations) * 0.50)]
    p95 = caddy_durations[int(len(caddy_durations) * 0.95)]
    p99 = caddy_durations[int(len(caddy_durations) * 0.99)]
    max_d = max(caddy_durations)
    min_d = min(caddy_durations)
    print(f"\nLatência no Caddy (Tempo Total do Proxy):")
    print(f"  Média: {avg_d:.2f} ms")
    print(f"  Mínima: {min_d:.2f} ms | Mediana (p50): {p50:.2f} ms")
    print(f"  p95: {p95:.2f} ms | p99: {p99:.2f} ms | Máxima: {max_d:.2f} ms")

# 2. Análise Backend-API (Go Runtime / PostgreSQL / Redis)
cmd_api = ["docker", "compose", "logs", "backend-api", "--since", "30m"]
out_api = subprocess.check_output(cmd_api, cwd="/opt/projeto-nova").decode("utf-8", errors="ignore")

api_logs = []
for line in out_api.splitlines():
    if "http_request" in line:
        idx = line.find("{")
        if idx != -1:
            try:
                api_logs.append(json.loads(line[idx:]))
            except Exception:
                pass

print(f"\n[2] PROCESSAMENTO NO BACKEND REST (Go Core):")
print(f"Total de requisições capturadas pela API: {len(api_logs)}")

status_api = Counter(l.get("status") for l in api_logs)
print("Distribuição de Status na API:")
for s, c in sorted(status_api.items()):
    pct = (c / len(api_logs)) * 100 if api_logs else 0
    print(f"  HTTP {s}: {c} ({pct:.1f}%)")

api_durations = [l.get("duration", 0) / 1_000_000 for l in api_logs if l.get("duration")] # nanosegundos -> ms
if api_durations:
    api_durations.sort()
    avg_api = sum(api_durations) / len(api_durations)
    p50_api = api_durations[int(len(api_durations) * 0.50)]
    p95_api = api_durations[int(len(api_durations) * 0.95)]
    p99_api = api_durations[int(len(api_durations) * 0.99)]
    max_api = max(api_durations)
    min_api = min(api_durations)
    print(f"\nLatência Interna da API (Go + DB + Redis):")
    print(f"  Média: {avg_api:.2f} ms")
    print(f"  Mínima: {min_api:.2f} ms | Mediana (p50): {p50_api:.2f} ms")
    print(f"  p95: {p95_api:.2f} ms | p99: {p99_api:.2f} ms | Máxima: {max_api:.2f} ms")

# 3. Análise dos Recursos de Hardware dos Containers (Docker Stats)
print(f"\n[3] CONSUMO DE RECURSOS DOS CONTAINERS (docker stats):")
cmd_stats = ["docker", "stats", "--no-stream", "--format", "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"]
out_stats = subprocess.check_output(cmd_stats, cwd="/opt/projeto-nova").decode("utf-8", errors="ignore")
print(out_stats)

print("=" * 70)
