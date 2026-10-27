"""Liga os módulos desligados, pela API e como administrador (fica na
auditoria), para os povoadores terem onde gravar.

    python3 ligar_modulos.py               # todos, menos o módulo modelo
    python3 ligar_modulos.py blog wiki     # só esses
"""
import sys

from nxlib import *  # noqa: F403

ADMIN = "teste.admin"
FORA = {"example"}  # o módulo modelo não é funcionalidade do produto
pedidos = set(sys.argv[1:])

# Mais de uma volta: um módulo que depende de outro só liga depois dele.
for volta in range(3):
    code, d = call(ADMIN, "GET", "system/modules")
    if code != 200:
        check("listar módulos", False, d)
        break
    # "configured" é o que o administrador pediu; "enabled" ainda depende
    # das dependências estarem ligadas.
    faltam = [m["key"] for m in d["data"] if not m["configured"] and m["key"] not in FORA and (not pedidos or m["key"] in pedidos)]
    if not faltam:
        break
    for key in faltam:
        code, r = call(ADMIN, "PATCH", f"admin/modules/{key}", {"enabled": True})
        print(f"  {'ligado ' if code == 200 else 'falhou '} {key}" + ("" if code == 200 else f" (HTTP {code} {str(r)[:120]})"))
done()
