import json
import urllib.request

creds = [
    ("carlos.diretoria", "Nova@Rondonopolis2026!"),
    ("mariana.mkt", "Nova@Rondonopolis2026!"),
    ("lucas.tech", "Nova@Rondonopolis2026!"),
    ("marcos.ops", "Nova@Rondonopolis2026!"),
    ("ana.protocolo", "Nova@Rondonopolis2026!"),
    ("rodrigo.agro", "Nova@Rondonopolis2026!"),
]

for username, password in creds:
    body = json.dumps({"username": username, "password": password}).encode()
    req = urllib.request.Request("http://127.0.0.1:8010/api/v1/auth/login", data=body, headers={"Content-Type": "application/json"})
    try:
        res = urllib.request.urlopen(req)
        data = json.loads(res.read().decode())
        token = data["data"]["access_token"]
        req_me = urllib.request.Request("http://127.0.0.1:8010/api/v1/me", headers={"Authorization": "Bearer " + token})
        res_me = urllib.request.urlopen(req_me)
        me = json.loads(res_me.read().decode())["data"]
        perms_count = len(me.get("permissions", []))
        scopes_count = len(me.get("scopes", []))

        req_proc = urllib.request.Request("http://127.0.0.1:8010/api/v1/tramite/processos", headers={"Authorization": "Bearer " + token})
        res_proc = urllib.request.urlopen(req_proc)
        procs = json.loads(res_proc.read().decode())["data"]
        procs_count = len(procs) if isinstance(procs, list) else 0

        req_ev = urllib.request.Request("http://127.0.0.1:8010/api/v1/calendar/events?from=2026-10-01T00:00:00Z&to=2026-10-31T23:59:59Z", headers={"Authorization": "Bearer " + token})
        res_ev = urllib.request.urlopen(req_ev)
        events = json.loads(res_ev.read().decode())["data"]
        events_count = len(events) if isinstance(events, list) else 0

        req_wiki = urllib.request.Request("http://127.0.0.1:8010/api/v1/wiki/tree", headers={"Authorization": "Bearer " + token})
        res_wiki = urllib.request.urlopen(req_wiki)
        wiki_tree = json.loads(res_wiki.read().decode())["data"]

        def count_nodes(nodes):
            total = 0
            if not isinstance(nodes, list):
                return total
            for n in nodes:
                total += 1
                if "children" in n and n["children"]:
                    total += count_nodes(n["children"])
            return total

        wiki_count = count_nodes(wiki_tree)

        req_rooms = urllib.request.Request("http://127.0.0.1:8010/api/v1/mercurio/rooms", headers={"Authorization": "Bearer " + token})
        res_rooms = urllib.request.urlopen(req_rooms)
        rooms = json.loads(res_rooms.read().decode())["data"]
        rooms_count = len(rooms) if isinstance(rooms, list) else 0

        req_files = urllib.request.Request("http://127.0.0.1:8010/api/v1/files/browse", headers={"Authorization": "Bearer " + token})
        res_files = urllib.request.urlopen(req_files)
        files_data = json.loads(res_files.read().decode())["data"]
        folders_count = len(files_data.get("folders", []))

        print(f"[OK] {username}: {me['name']} | escopos={scopes_count} | permissoes={perms_count} | processos={procs_count} | eventos_agenda={events_count} | paginas_wiki={wiki_count} | salas_chat={rooms_count} | pastas_arquivos={folders_count}")
    except Exception as e:
        print(f"[FAIL] {username}: {e}")




