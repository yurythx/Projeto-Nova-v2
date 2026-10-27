"""Povoa a gestão de serviços (atendimento, ativos, problemas, mudanças,
projetos, contratos, cofre, produtos e ocorrências) com volume de teste,
criado pelos usuários fictícios pela API, como o popular.py.

A Central de Serviços de TI fica na sede: as equipes são os técnicos de
TI da Prefeitura e os pedidos vêm de servidores de todas as unidades.

    python3 popular_gestao.py            # só roda uma vez (marca em .demo-gestao)
    python3 popular_gestao.py --force    # roda de novo (cria duplicatas)
    CHAMADOS=800 python3 popular_gestao.py
"""
import random
import secrets
import sys
from datetime import date, datetime, timedelta, timezone

from nxlib import *  # noqa: F403

MARK = os.path.join(ROOT, ".demo-gestao")
if os.path.exists(MARK) and "--force" not in sys.argv:
    print(f"Já povoado em {open(MARK).read().strip()} — use --force para repetir.")
    sys.exit(0)

rnd = random.Random(20261027)
ADMIN = "teste.admin"
TECS = [u for u in users_in("PREF-SEDE-TI") if not u.startswith("teste.")]
DIRETOR_TI = TECS[0]
SOLICITANTES = rnd.sample([u["usuario"] for u in USERS if not u["usuario"].startswith("teste.") and "PREF-SEDE-TI" not in u["grupos"].split()], 45)
N_CHAMADOS = int(os.environ.get("CHAMADOS", "450"))
now = datetime.now(timezone.utc).replace(microsecond=0)
iso = lambda dt: dt.strftime("%Y-%m-%dT%H:%M:%SZ")  # noqa: E731
_ids = {}


def ok(label, code, d, want=(200, 201, 204)):
    if code not in want:
        print(f"  ! {label}: HTTP {code} {str(d)[:200]}")
        FAILS.append(label)
        return None
    return d.get("data", d) if isinstance(d, dict) else (d or {})


def obter_ou_criar(label, user, rota, corpo, campo="nome", lista=None):
    """Reaproveita pelo nome o que uma execução anterior já criou: rodar
    de novo não duplica o cadastro, só os registros do dia a dia."""
    code, d = call(user, "GET", f"{lista or rota}?page_size=200")
    itens = d.get("data") if code == 200 and isinstance(d, dict) else None
    achado = next((i for i in itens or [] if i.get(campo) == corpo[campo]), None)
    return achado or ok(label, *call(user, "POST", rota, corpo))


def uid(u):
    if u not in _ids:
        _ids[u] = call(u, "GET", "me")[1]["data"]["id"]
    return _ids[u]


unidades = {u["nome"]: u["id"] for u in call(ADMIN, "GET", "iam/unidades")[1]["data"]}
SEDE = unidades["Sede Administrativa da Prefeitura"]

print("== Perfil dos técnicos de TI (Central de Serviços)")
perfil = obter_ou_criar("perfil técnico", ADMIN, "iam/perfis", campo="slug", corpo={
    "slug": "tecnico-ti-demo", "nome": "Técnico de TI (teste)",
    "descricao": "Atende a Central de Serviços e cuida de ativos, problemas, mudanças, projetos e contratos.",
    "permissoes": ["atendimento:atender", "atendimento:abrir", "ativos:manage", "problemas:read", "problemas:manage",
                   "mudancas:manage", "projetos:read", "projetos:manage", "contratos:manage", "cofre:read", "cofre:reveal"]})
if perfil:
    ok("mapeamento PREF-SEDE-TI", *call(ADMIN, "POST", "iam/ad-mappings", {
        "ad_group": "PREF-SEDE-TI", "perfil_id": perfil["id"], "descricao": "Técnicos de TI (teste)"}), want=(200, 201, 409))

print("== Produtos digitais")
PRODUTOS = {}
for nome, estagio, util in [
    ("E-mail institucional", "operacao", "Correio e agenda de todos os servidores"),
    ("Rede corporativa", "operacao", "Cabeamento, Wi-Fi e links das unidades"),
    ("Sistema de Protocolo", "operacao", "Processos administrativos e trâmite"),
    ("Folha de pagamento", "operacao", "Cálculo e pagamento dos servidores"),
    ("Prontuário eletrônico", "suporte", "Atendimento nas unidades de saúde"),
    ("Portal do cidadão", "operacao", "Serviços online para a população"),
    ("Sistema escolar", "transicao", "Matrículas, notas e frequência"),
    ("Telefonia IP", "construcao", "Ramais e central telefônica"),
    ("Backup corporativo", "operacao", "Cópias dos servidores e bancos"),
    ("Assinatura digital", "desenho", "Documentos com validade jurídica"),
]:
    d = obter_ou_criar(f"produto {nome}", ADMIN, "produtos", {"nome": nome, "estagio": estagio, "utilidade": util,
                                                             "garantia": "Disponível 99% no horário de expediente", "unidade_id": SEDE})
    if d:
        PRODUTOS[nome] = d["id"]

print("== Feriados")
for dia, nome, tipo in [("2026-11-02", "Finados", "nacional"), ("2026-11-15", "Proclamação da República", "nacional"),
                        ("2026-11-20", "Consciência Negra", "nacional"), ("2026-12-10", "Aniversário do município", "municipal"),
                        ("2026-12-24", "Véspera de Natal", "ponto_facultativo"), ("2026-12-25", "Natal", "nacional"),
                        ("2026-12-31", "Véspera de Ano Novo", "ponto_facultativo")]:
    ok(f"feriado {nome}", *call(ADMIN, "POST", "atendimento/feriados", {"data": dia, "nome": nome, "tipo": tipo}), want=(200, 201, 409))

print("== Equipes da Central de Serviços")
EQUIPES = {}
divisao = {"N1 - Service Desk": (TECS[1:6], 8), "Redes e Telefonia": (TECS[3:5] + TECS[6:7], 24),
           "Sistemas": (TECS[1:2] + TECS[4:5] + TECS[7:8], 48), "Infraestrutura": (TECS[3:4] + TECS[6:9], 24),
           "Suporte de Campo": (TECS[2:3] + TECS[5:6] + TECS[8:9], 16)}
for nome, (membros, ola) in divisao.items():
    d = obter_ou_criar(f"equipe {nome}", ADMIN, "atendimento/equipes", {"nome": nome, "unidade_id": SEDE, "ola_horas": ola,
                                                                        "membros": [uid(m) for m in membros]})
    if d:
        EQUIPES[nome] = (d["id"], membros)

print("== Ofertas (catálogo da Central)")
OFERTAS = []
CATALOGO = [
    ("Computador não liga", "incidente", "E-mail institucional", ["O computador da recepção não liga", "Estação não dá sinal de vídeo", "PC desliga sozinho"]),
    ("Sem acesso à internet", "incidente", "Rede corporativa", ["Unidade sem internet desde cedo", "Internet muito lenta", "Wi-Fi caindo toda hora"]),
    ("E-mail não envia ou não recebe", "incidente", "E-mail institucional", ["Não recebo e-mails externos", "Caixa de e-mail cheia", "E-mail volta com erro"]),
    ("Impressora com problema", "incidente", "Rede corporativa", ["Impressora atolando papel", "Impressora não aparece na rede", "Impressão saindo manchada"]),
    ("Erro no Sistema de Protocolo", "incidente", "Sistema de Protocolo", ["Erro ao tramitar processo", "Não consigo anexar documento", "Processo sumiu da caixa"]),
    ("Erro na folha de pagamento", "incidente", "Folha de pagamento", ["Contracheque não abre", "Erro ao calcular horas extras", "Relatório da folha travando"]),
    ("Prontuário eletrônico fora do ar", "incidente", "Prontuário eletrônico", ["Prontuário lento na UPA", "Não consigo salvar atendimento", "Tela branca no prontuário"]),
    ("Telefone sem linha", "incidente", "Telefonia IP", ["Ramal mudo", "Telefone não faz ligação externa", "Chiado na linha"]),
    ("Sistema escolar com erro", "incidente", "Sistema escolar", ["Não consigo lançar notas", "Matrícula duplicada", "Diário de classe não abre"]),
    ("Criar conta de rede e e-mail", "requisicao", "E-mail institucional", ["Conta para novo estagiário", "E-mail para a nova coordenadora", "Acesso para servidor transferido"]),
    ("Instalar programa", "requisicao", "Rede corporativa", ["Instalar leitor de PDF", "Instalar programa de planilhas", "Atualizar navegador"]),
    ("Acesso a pasta compartilhada", "requisicao", "Rede corporativa", ["Acesso à pasta do RH", "Liberar pasta da contabilidade", "Pasta da direção"]),
    ("Novo ponto de rede", "requisicao", "Rede corporativa", ["Ponto de rede na sala 3", "Cabo para a nova recepção", "Ponto para impressora"]),
    ("Novo ramal telefônico", "requisicao", "Telefonia IP", ["Ramal para a sala de vacina", "Transferir ramal de sala", "Ramal para o almoxarifado"]),
    ("Liberar site bloqueado", "requisicao", "Rede corporativa", ["Liberar site do fornecedor", "Acesso ao portal do governo", "Liberar videoconferência"]),
    ("Restaurar arquivo apagado", "requisicao", "Backup corporativo", ["Recuperar planilha de ontem", "Arquivo apagado da pasta comum", "Versão anterior do documento"]),
    ("Empréstimo de projetor", "requisicao", "Rede corporativa", ["Projetor para reunião", "Projetor para capacitação", "Notebook e projetor para evento"]),
]
for nome, tipo, produto, _ in CATALOGO:
    d = obter_ou_criar(f"oferta {nome}", ADMIN, "atendimento/ofertas", {
        "nome": nome, "tipo": tipo, "unidade_id": SEDE, "produto_id": PRODUTOS.get(produto), "ativa": True,
        "descricao": f"Central de Serviços de TI: {nome.lower()}.",
        "campos": [{"chave": "patrimonio", "rotulo": "Patrimônio do equipamento", "tipo": "texto", "obrigatorio": False}] if tipo == "incidente" else []})
    if d:
        OFERTAS.append((d["id"], tipo, nome))

# Admissão e desligamento: aprovação do diretor de TI e tarefas por equipe.
FLUXOS = []
for nome, tarefas in [("Admissão de servidor", [("Criar conta de rede e e-mail", "Sistemas"), ("Preparar computador", "Suporte de Campo"),
                                                 ("Liberar acesso aos sistemas", "Sistemas"), ("Configurar ramal", "Redes e Telefonia")]),
                      ("Desligamento de servidor", [("Bloquear contas", "Sistemas"), ("Recolher equipamento", "Suporte de Campo"),
                                                    ("Arquivar caixa de e-mail", "Infraestrutura")])]:
    d = obter_ou_criar(f"oferta {nome}", ADMIN, "atendimento/ofertas", {"nome": nome, "tipo": "requisicao", "unidade_id": SEDE, "ativa": True,
                                                                        "descricao": f"{nome}: aprovação da TI e tarefas por equipe."})
    if not d:
        continue
    ok(f"etapas {nome}", *call(ADMIN, "PUT", f"atendimento/ofertas/{d['id']}/etapas", {"etapas": [{"nome": "Diretoria de TI", "aprovadores": [uid(DIRETOR_TI)]}]}))
    ok(f"tarefas {nome}", *call(ADMIN, "PUT", f"atendimento/ofertas/{d['id']}/tarefas",
                                {"tarefas": [{"titulo": t, "equipe_id": EQUIPES[e][0]} for t, e in tarefas if e in EQUIPES]}))
    FLUXOS.append((d["id"], nome))

print("== Contratos, fornecedores e licenças")
FORNECEDORES = {}
for nome in ["Conecta Telecom Ltda", "InfoServ Manutenção", "Sistemas Públicos S/A", "Papel & Toner Distribuidora",
             "DataCenter Cuiabá", "Segurança Digital ME", "Rede Norte Cabeamento", "SoftGov Licenças"]:
    d = obter_ou_criar(f"fornecedor {nome}", ADMIN, "contratos/fornecedores", {"nome": nome, "email": "contato@" + nome.split()[0].lower() + ".test",
                                                                                "telefone": "(66) 3411-%04d" % rnd.randint(0, 9999), "unidade_id": SEDE})
    if d:
        FORNECEDORES[nome] = d["id"]
CONTRATOS = []
for i, (titulo, forn, meses, valor, prazo) in enumerate([
    ("Link de internet das unidades", "Conecta Telecom Ltda", 12, 48_000_00, 8), ("Manutenção de computadores", "InfoServ Manutenção", 24, 96_000_00, 48),
    ("Sistema de Protocolo", "Sistemas Públicos S/A", 36, 210_000_00, 24), ("Outsourcing de impressão", "Papel & Toner Distribuidora", 12, 60_000_00, 24),
    ("Hospedagem do portal", "DataCenter Cuiabá", 12, 36_000_00, 4), ("Antivírus corporativo", "Segurança Digital ME", 12, 18_000_00, 24),
    ("Cabeamento estruturado", "Rede Norte Cabeamento", 6, 75_000_00, 72), ("Folha de pagamento", "Sistemas Públicos S/A", 24, 150_000_00, 24),
    ("Telefonia IP", "Conecta Telecom Ltda", 24, 54_000_00, 8), ("Licenças de escritório", "SoftGov Licenças", 12, 40_000_00, 48),
    ("Prontuário eletrônico", "Sistemas Públicos S/A", 36, 320_000_00, 4), ("Backup em nuvem", "DataCenter Cuiabá", 12, 24_000_00, 8)]):
    inicio = date(2026, 1, 1) + timedelta(days=30 * (i % 9))
    if forn not in FORNECEDORES:
        continue
    d = ok(f"contrato {titulo}", *call(ADMIN, "POST", "contratos/contratos", {"titulo": titulo, "fornecedor_id": FORNECEDORES[forn], "numero": f"{i + 1:03d}/2026",
                                                                              "inicio": inicio.isoformat(), "fim": (inicio + timedelta(days=30 * meses)).isoformat(),
                                                                              "valor_centavos": valor, "unidade_id": SEDE}))
    if d:
        CONTRATOS.append(d["id"])
        ok(f"prazo {titulo}", *call(ADMIN, "PUT", f"contratos/contratos/{d['id']}/prazo", {"prazo_horas": prazo}))
for nome, programa, qtd in [("Escritório", "Suíte de escritório", 400), ("Antivírus", "Antivírus corporativo", 650), ("Projeto", "CAD", 10),
                            ("Banco de dados", "SGBD", 4), ("Backup", "Agente de backup", 30), ("Videoconferência", "Salas virtuais", 50),
                            ("Leitor de PDF Pro", "PDF", 120), ("Edição de imagens", "Imagens", 8), ("Sistema operacional", "Servidores", 20), ("Gestão de ativos", "Inventário", 1)]:
    forn = rnd.choice(list(FORNECEDORES.values())) if FORNECEDORES else None
    if forn:
        ok(f"licença {nome}", *call(ADMIN, "POST", "contratos/licencas", {"nome": nome, "programa": programa, "quantidade": qtd, "fornecedor_id": forn,
                                                                          "fim": date(2027, rnd.randint(1, 12), 1).isoformat(), "unidade_id": SEDE}))

print("== Ativos (inventário das unidades)")
ATIVOS = []
for unome, uid_ in unidades.items():
    sigla = "".join(w[0] for w in unome.split() if w[0].isupper())[:5]
    sw = ok(f"switch {unome}", *call(TECS[3], "POST", "ativos", {"nome": f"Switch {sigla}", "tipo": "rede", "estado": "em_uso", "unidade_id": uid_,
                                                                  "identificador": f"SW-{sigla}-01", "produto_id": PRODUTOS.get("Rede corporativa")}))
    for n in range(rnd.randint(5, 9)):
        tipo, nome = rnd.choice([("hardware", "Desktop"), ("hardware", "Desktop"), ("hardware", "Notebook"), ("hardware", "Impressora"), ("hardware", "Monitor")])
        estado = rnd.choices(["em_uso", "manutencao", "planejado", "desativado"], [80, 8, 6, 6])[0]
        d = ok(f"ativo {nome}", *call(rnd.choice(TECS), "POST", "ativos", {"nome": f"{nome} {sigla}-{n + 1:02d}", "tipo": tipo, "estado": estado, "unidade_id": uid_,
                                                                          "identificador": f"PAT-{rnd.randint(10000, 99999)}", "descricao": f"{nome} da unidade {unome}"}))
        if d:
            ATIVOS.append(d["id"])
            if sw and nome in ("Desktop", "Impressora") and rnd.random() < 0.7:
                ok("relação", *call(TECS[3], "POST", f"ativos/{d['id']}/relacoes", {"destino_id": sw["id"], "tipo": "conectado"}))
for nome, tipo in [("Servidor de arquivos", "hardware"), ("Servidor de e-mail", "hardware"), ("Firewall", "rede"), ("Storage de backup", "hardware"),
                   ("Banco de dados da folha", "software"), ("Portal do cidadão", "servico"), ("Central telefônica", "rede")]:
    d = ok(f"ativo {nome}", *call(TECS[6], "POST", "ativos", {"nome": nome, "tipo": tipo, "estado": "em_uso", "unidade_id": SEDE,
                                                              "identificador": f"DC-{rnd.randint(100, 999)}", "descricao": "Datacenter da sede"}))
    if d:
        ATIVOS.append(d["id"])

print(f"== Chamados ({N_CHAMADOS}) — abertura, atendimento, conversa e conclusão")
INCIDENTES, chamados = [], 0
titulos = {nome: exemplos for nome, _, _, exemplos in CATALOGO}
for i in range(N_CHAMADOS):
    oferta_id, tipo, nome = rnd.choices(OFERTAS, [3 if t == "incidente" else 2 for _, t, _ in OFERTAS])[0]
    quem = rnd.choice(SOLICITANTES)
    titulo = rnd.choice(titulos[nome])
    sol = ok("abrir", *call(quem, "POST", "atendimento/solicitacoes", {"oferta_id": oferta_id, "titulo": titulo,
                                                                        "descricao": f"{titulo}. Preciso de ajuda, por favor. Ramal {rnd.randint(2000, 2800)}."}))
    if not sol:
        continue
    chamados += 1
    sid, fim = sol["id"], ("resolvido" if tipo == "incidente" else "atendido")
    if tipo == "incidente":
        INCIDENTES.append(sid)
    destino = rnd.choices(["aberto", "em_atendimento", "aguardando_servidor", "aguardando_terceiro", fim, "encerrado", "cancelado"],
                          [12, 18, 8, 5, 25, 28, 4])[0]
    if destino == "aberto":
        continue
    tec = rnd.choice(TECS[1:])
    if destino == "cancelado":
        ok("cancelar", *call(quem, "POST", f"atendimento/solicitacoes/{sid}/estado", {"estado": "cancelado"}))
        continue
    if not ok("assumir", *call(tec, "PUT", f"atendimento/solicitacoes/{sid}/responsavel", {"responsavel_id": uid(tec)})):
        continue
    if tipo == "incidente":
        ok("classificar", *call(tec, "PUT", f"atendimento/solicitacoes/{sid}/classificacao", {"impacto": rnd.randint(1, 3), "urgencia": rnd.randint(1, 3)}))
    ok("resposta", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/comentarios",
                         {"texto": rnd.choice(["Recebido, já estou verificando.", "Pode me informar o patrimônio do equipamento?",
                                               "Vou acessar remotamente em alguns minutos.", "Encaminhei para a equipe responsável."]), "interno": False}))
    if rnd.random() < 0.4:
        ok("nota interna", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/comentarios",
                                 {"texto": rnd.choice(["Reincidente nesta unidade.", "Verificar garantia do equipamento.", "Possível problema no switch."]), "interno": True}))
    if destino in ("aguardando_servidor", "aguardando_terceiro"):
        ok("aguardar", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/estado", {"estado": destino}))
        continue
    if destino == "em_atendimento":
        continue
    ok("apontamento", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/apontamentos",
                            {"horas": rnd.choice([0.25, 0.5, 0.5, 1, 1.5, 2, 3]), "descricao": "Diagnóstico e solução", "data_apontamento": date.today().isoformat()}))
    if not ok("concluir", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/estado", {"estado": fim})):
        continue
    if rnd.random() < 0.75:
        ok("avaliar", *call(quem, "POST", f"atendimento/solicitacoes/{sid}/avaliacao",
                            {"nota": rnd.choices([5, 4, 3, 2, 1], [50, 28, 12, 6, 4])[0], "facilidade": rnd.choices([5, 4, 3, 2, 1], [40, 30, 15, 10, 5])[0],
                             "comentario": rnd.choice(["", "Muito rápido, obrigado!", "Resolvido, mas demorou.", "Atendimento excelente."])}))
    if destino == "encerrado":
        ok("encerrar", *call(quem, "POST", f"atendimento/solicitacoes/{sid}/estado", {"estado": "encerrado"}))
    if (i + 1) % 50 == 0:
        print(f"  {i + 1} chamados processados")

print("== Admissões e desligamentos (aprovação e tarefas)")
for oferta_id, nome in FLUXOS:
    for _ in range(8):
        quem = rnd.choice(SOLICITANTES)
        sol = ok(nome, *call(quem, "POST", "atendimento/solicitacoes", {"oferta_id": oferta_id, "titulo": f"{nome}: {rnd.choice(USERS)['nome']}",
                                                                         "descricao": "Início/término do vínculo na próxima segunda-feira."}))
        if not sol:
            continue
        sid = sol["id"]
        if rnd.random() < 0.2:
            continue
        aprovar = rnd.random() > 0.15
        ok("aprovação", *call(DIRETOR_TI, "POST", f"atendimento/solicitacoes/{sid}/aprovacao", {"aprovar": aprovar, "motivo": "" if aprovar else "Falta o ato de nomeação."}))
        if not aprovar or rnd.random() < 0.3:
            continue
        tec = TECS[1]
        ok("assumir", *call(tec, "PUT", f"atendimento/solicitacoes/{sid}/responsavel", {"responsavel_id": uid(tec)}))
        det = call(tec, "GET", f"atendimento/solicitacoes/{sid}")[1].get("data", {})
        for t in det.get("tarefas", []):
            membros = next((m for e, (eid, m) in EQUIPES.items() if eid == t.get("equipe_id")), [tec])
            ok("tarefa", *call(membros[0], "POST", f"atendimento/solicitacoes/{sid}/tarefas/{t['id']}", {"concluir": True, "observacao": "Feito."}))
        ok("concluir", *call(tec, "POST", f"atendimento/solicitacoes/{sid}/estado", {"estado": "atendido"}))

print("== Problemas")
for titulo, causa in [("Quedas recorrentes de internet nas PSFs", "Link de rádio saturado"), ("Impressoras perdendo a rede", "DHCP com concessão curta"),
                      ("Lentidão no prontuário às segundas", "Rotina de backup no horário de pico"), ("Contas bloqueando sozinhas", "Senha antiga salva no celular"),
                      ("E-mails caindo no spam", "Registro SPF incompleto"), ("Computadores antigos travando", "Disco mecânico no fim da vida"),
                      ("Erro de certificado no portal", "Certificado intermediário ausente"), ("Folha travando no fechamento", "Índice faltando no banco"),
                      ("Ramais mudos após chuva", "Cabo externo sem proteção"), ("Matrículas duplicadas", "Duplo clique no botão salvar"),
                      ("Wi-Fi fraco nas escolas", "Pontos de acesso mal posicionados"), ("Backup falhando aos domingos", "Janela concorrente com antivírus")]:
    tec = rnd.choice(TECS[1:])
    d = ok(f"problema {titulo}", *call(tec, "POST", "problemas", {"titulo": titulo, "descricao": f"Vários chamados com o mesmo sintoma: {titulo.lower()}.",
                                                                   "unidade_id": SEDE, "contorno": "Reiniciar o equipamento resolve por algumas horas."}))
    if not d:
        continue
    for sid in rnd.sample(INCIDENTES, min(len(INCIDENTES), rnd.randint(2, 6))):
        ok("vínculo incidente", *call(tec, "PUT", f"problemas/{d['id']}/incidentes/{sid}"))
    for aid in rnd.sample(ATIVOS, min(len(ATIVOS), rnd.randint(1, 3))):
        ok("vínculo ativo", *call(tec, "PUT", f"problemas/{d['id']}/ativos/{aid}"))
    for estado in ["em_analise", "erro_conhecido", "resolvido", "encerrado"][:rnd.randint(0, 4)]:
        if estado == "erro_conhecido":
            call(tec, "PUT", f"problemas/{d['id']}", {"titulo": titulo, "causa_raiz": causa, "contorno": "Reiniciar o equipamento resolve por algumas horas.", "unidade_id": SEDE})
        if not ok(f"problema → {estado}", *call(tec, "POST", f"problemas/{d['id']}/estado", {"estado": estado})):
            break

print("== Mudanças")
ok("congelamento", *call(ADMIN, "POST", "mudancas/congelamentos", {"inicio": "2026-12-20T00:00:00Z", "fim": "2027-01-05T00:00:00Z",
                                                                   "motivo": "Fechamento do exercício e recesso"}))
for i, (titulo, tipo, risco) in enumerate([
    ("Atualizar firmware dos switches", "normal", "medio"), ("Trocar firewall da sede", "normal", "alto"), ("Migrar e-mail para nova versão", "normal", "alto"),
    ("Aplicar atualizações mensais", "padrao", "baixo"), ("Ampliar link da UPA", "normal", "medio"), ("Corrigir falha crítica no portal", "emergencial", "alto"),
    ("Instalar Wi-Fi nas escolas", "normal", "medio"), ("Novo servidor de arquivos", "normal", "medio"), ("Trocar nobreak do datacenter", "normal", "alto"),
    ("Atualizar sistema de protocolo", "normal", "medio"), ("Reiniciar serviços do prontuário", "padrao", "baixo"), ("Renovar certificado do portal", "padrao", "baixo"),
    ("Mover ramais para a nova central", "normal", "alto"), ("Ajustar regras de backup", "padrao", "baixo"), ("Bloquear portas antigas no firewall", "emergencial", "medio"),
    ("Implantar segundo link na sede", "normal", "medio"), ("Atualizar antivírus", "padrao", "baixo"), ("Trocar computadores da SEMED", "normal", "baixo"),
    ("Habilitar autenticação em dois fatores", "normal", "alto"), ("Desligar servidor antigo", "normal", "medio")]):
    tec = rnd.choice(TECS[1:])
    inicio = now + timedelta(days=rnd.randint(-40, 40), hours=rnd.randint(18, 22))
    d = ok(f"mudança {titulo}", *call(tec, "POST", "mudancas", {"titulo": titulo, "tipo": tipo, "risco": risco, "unidade_id": SEDE,
                                                                 "descricao": f"{titulo} conforme plano da equipe.", "justificativa": "Estabilidade e segurança.",
                                                                 "plano_reversao": "Restaurar a configuração anterior a partir do backup.",
                                                                 "inicio_planejado": iso(inicio), "fim_planejado": iso(inicio + timedelta(hours=3))}))
    if not d:
        continue
    caminho = rnd.choice([[], ["em_avaliacao"], ["em_avaliacao", "aprovada"], ["em_avaliacao", "aprovada", "em_execucao"],
                          ["em_avaliacao", "aprovada", "em_execucao", "concluida"], ["em_avaliacao", "aprovada", "em_execucao", "concluida"],
                          ["em_avaliacao", "rejeitada"], ["em_avaliacao", "aprovada", "em_execucao", "falhou"]])
    for estado in caminho:
        quem = ADMIN if estado in ("aprovada", "rejeitada") else tec
        if not ok(f"mudança → {estado}", *call(quem, "POST", f"mudancas/{d['id']}/estado", {"estado": estado, "texto": "Registrado pela equipe."})):
            break

print("== Projetos")
for titulo, prioridade, orcamento, estado in [
    ("Modernização da rede das escolas", "alta", 350000, "em_andamento"), ("Prontuário em todas as PSFs", "critica", 500000, "em_andamento"),
    ("Portal do cidadão 2.0", "media", 180000, "planejado"), ("Digitalização do arquivo municipal", "media", 120000, "em_andamento"),
    ("Telefonia IP na Prefeitura", "alta", 90000, "concluido"), ("Datacenter de contingência", "alta", 650000, "planejado"),
    ("Capacitação em segurança da informação", "baixa", 15000, "em_andamento"), ("Inventário de software", "media", 8000, "suspenso"),
    ("Wi-Fi público nas praças", "media", 220000, "planejado"), ("Assinatura digital nos processos", "alta", 60000, "em_andamento")]:
    gerente = rnd.choice(TECS[:4])
    ini = date.today() - timedelta(days=rnd.randint(10, 200))
    d = ok(f"projeto {titulo}", *call(gerente, "POST", "projetos", {"titulo": titulo, "prioridade": prioridade, "orcamento": orcamento, "unidade_id": SEDE,
                                                                     "objetivo": f"Entregar {titulo.lower()}.", "descricao": "Projeto do plano diretor de TI.",
                                                                     "gerente_id": uid(gerente), "data_inicio_planejada": ini.isoformat(),
                                                                     "data_fim_planejada": (ini + timedelta(days=rnd.randint(90, 360))).isoformat()}))
    if not d:
        continue
    pid = d["id"]
    for ordem, t in enumerate(["Levantamento", "Termo de referência", "Licitação", "Instalação", "Testes", "Treinamento", "Encerramento"]):
        est = rnd.choice(["a_fazer", "em_andamento", "concluida"]) if estado != "concluido" else "concluida"
        tk = ok("tarefa", *call(gerente, "POST", f"projetos/{pid}/tarefas", {"titulo": t, "ordem": ordem, "estado": est, "responsavel_nome": rnd.choice(USERS)["nome"]}))
        if tk and est != "a_fazer":
            ok("apontamento", *call(gerente, "POST", f"projetos/{pid}/apontamentos", {"tarefa_id": tk["id"], "horas": rnd.choice([2, 4, 6, 8]), "custo_hora": 85,
                                                                                     "descricao": f"{t} — trabalho da semana", "data_apontamento": date.today().isoformat()}))
    caminhos = {"planejado": [], "em_andamento": ["em_andamento"], "suspenso": ["em_andamento", "suspenso"], "concluido": ["em_andamento", "concluido"]}
    for est in caminhos[estado]:
        if not ok(f"projeto → {est}", *call(gerente, "POST", f"projetos/{pid}/estado", {"estado": est})):
            break

print("== Cofre de senhas")
for pasta, membros in [("Infraestrutura", TECS[3:9]), ("Sistemas", TECS[1:3] + TECS[7:8]), ("Telefonia", TECS[3:5])]:
    p = ok(f"pasta {pasta}", *call(DIRETOR_TI, "POST", "cofre/pastas", {"nome": pasta, "unidade_id": SEDE,
                                                                       "membros": [{"usuario_id": uid(m), "pode_revelar": True} for m in membros]}))
    for n in range(rnd.randint(4, 7)):
        tipo = rnd.choice(["maquina", "aplicacao", "painel", "outro"])
        ok("segredo", *call(DIRETOR_TI, "POST", "cofre/segredos", {"nome": f"{pasta} — {tipo} {n + 1}", "tipo": tipo, "usuario": rnd.choice(["admin", "root", "suporte"]),
                                                                  "senha": secrets.token_urlsafe(18), "endereco": f"10.0.{rnd.randint(0, 9)}.{rnd.randint(2, 250)}",
                                                                  "pasta_id": p["id"] if p else None, "unidade_id": SEDE, "notas": "Senha de teste (fictícia)."}))

print("== Incidentes graves (encerrados, com revisão)")
for titulo, impacto, servicos in [("Sem internet em todas as unidades", "fora", ["Rede corporativa", "Portal do cidadão"]),
                                  ("Prontuário eletrônico lento", "degradado", ["Prontuário eletrônico"])]:
    d = ok(f"ocorrência {titulo}", *call(ADMIN, "POST", "ocorrencias", {"titulo": titulo, "impacto": impacto, "servicos": servicos,
                                                                         "texto": "A equipe está investigando a causa."}))
    if not d:
        continue
    for estado, texto in [("identificado", "Causa identificada no equipamento do provedor."), ("monitorando", "Serviço restabelecido, em observação."),
                          ("resolvido", "Normalizado. Obrigado pela paciência.")]:
        ok(f"ocorrência → {estado}", *call(ADMIN, "POST", f"ocorrencias/{d['id']}/atualizacoes", {"estado": estado, "impacto": impacto, "texto": texto}))
    ok("revisão", *call(ADMIN, "PUT", f"ocorrencias/{d['id']}/revisao", {"texto": "Linha do tempo, causa raiz e ações: redundância de link e alerta de saturação."}))

open(MARK, "w").write(now.isoformat())
print(f"\nGestão de serviços povoada: {chamados} chamados abertos, {len(ATIVOS)} ativos, {len(CONTRATOS)} contratos.")
done()
