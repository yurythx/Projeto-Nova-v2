"use client";

import { Folder, FolderOpen, Settings, X } from "lucide-react";
import { type FormEvent, useId, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ModalShell } from "@/components/ui/ModalShell";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const TIPOS = [
  { value: "maquina", label: "VM ou container" },
  { value: "aplicacao", label: "Aplicação" },
  { value: "painel", label: "Painel" },
  { value: "outro", label: "Outro" },
] as const;

type Tipo = (typeof TIPOS)[number]["value"];

interface Ficha {
  titulo: string;
  acesso: string;
  endereco: string;
  usuario: string;
  senha: string;
}

function ficha(tipo: string): Ficha {
  if (tipo === "maquina") {
    return {
      titulo: "VM ou container",
      acesso: "Acesso SSH",
      endereco: "IP",
      usuario: "Usuário SSH",
      senha: "Senha SSH",
    };
  }
  if (tipo === "aplicacao") {
    return {
      titulo: "Aplicação",
      acesso: "Acesso de administrador",
      endereco: "Endereço",
      usuario: "Usuário administrador",
      senha: "Senha de administrador",
    };
  }
  if (tipo === "painel") {
    return {
      titulo: "Painel",
      acesso: "Acesso do painel",
      endereco: "Endereço",
      usuario: "Usuário",
      senha: "Senha",
    };
  }
  return {
    titulo: "Outro",
    acesso: "Acesso",
    endereco: "Endereço",
    usuario: "Usuário",
    senha: "Senha",
  };
}

interface Segredo {
  id: string;
  nome: string;
  usuario: string;
  tipo?: Tipo | string;
  endereco?: string;
  rotaciona?: boolean;
  rotacao_pendente?: boolean;
  administra?: string[];
  retirada?: Retirada;
  pasta_id?: string;
  unidade_id?: string;
  versao: number;
}

interface Membro {
  usuario_id: string;
  pode_revelar: boolean;
}

interface Pasta {
  id: string;
  nome: string;
  unidade_id?: string;
  membros?: Membro[];
}

interface Pessoa {
  id: string;
  nome: string;
}

interface Retirada {
  usuario_id: string;
  nome: string;
  ate: string;
}

interface Revelado extends Segredo {
  senha: string;
  notas: string;
  retirada?: Retirada;
}

const PRAZOS = [
  { value: "15", label: "15 minutos" },
  { value: "30", label: "30 minutos" },
  { value: "60", label: "1 hora" },
  { value: "120", label: "2 horas" },
  { value: "480", label: "8 horas" },
];

function classeNo(ativo: boolean) {
  return `flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-sm ${
    ativo
      ? "bg-primary/10 font-medium text-foreground"
      : "text-foreground hover:bg-black/5 dark:hover:bg-white/10"
  }`;
}

function Contagem({ n }: { n: number }) {
  return (
    <span aria-hidden className="shrink-0 text-xs text-muted">
      {n}
    </span>
  );
}

function nomeDaPasta(pastas: Pasta[] | undefined, id?: string) {
  if (!id) return "";
  const pasta = (pastas ?? []).find((item) => item.id === id);
  return pasta ? ` · pasta ${pasta.nome}` : "";
}

function PastaCard({
  pastas,
  vista,
  gere,
  contagem,
  onVista,
  onSaved,
}: {
  pastas: Pasta[];
  vista: string;
  gere: boolean;
  contagem: { total: number; sem: number; porPasta: Record<string, number> };
  onVista: (vista: string) => void;
  onSaved: () => void;
}) {
  const { run, pending } = useAction();
  const [aberta, setAberta] = useState(false);
  const [confirma, setConfirma] = useState<Pasta | null>(null);
  const [painel, setPainel] = useState<null | "menu" | "membros">(null);
  const pastaAtual = pastas.find((pasta) => pasta.id === vista);

  function escolher(id: string) {
    setPainel(null);
    onVista(id);
  }

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(async () => {
      await apiClient.post("v1/cofre/pastas", {
        nome: String(fd.get("nome") ?? ""),
        unidade_id: String(fd.get("unidade_id") ?? "") || null,
        membros: [],
      });
      form.reset();
      setAberta(false);
      onSaved();
    }, "Pasta guardada");
    if (ok === undefined) return;
  }

  async function apagar(pasta: Pasta) {
    const ok = await run(async () => {
      await apiClient.delete(`v1/cofre/pastas/${pasta.id}`);
      onSaved();
    }, `Pasta ${pasta.nome} apagada`);
    if (ok === undefined) return;
    setConfirma(null);
  }

  return (
    <div className="flex flex-col gap-3">
      <nav aria-label="Pastas">
        <ul className="flex flex-col gap-0.5">
          <li>
            <span className="flex items-center gap-2 px-2 py-1.5 text-sm font-semibold text-foreground">
              <FolderOpen aria-hidden className="h-4 w-4 shrink-0" />
              Cofre
            </span>
            <ul
              aria-label="Pastas do cofre"
              className="ml-3 flex flex-col gap-0.5 border-l border-surface-border pl-2"
            >
              <li>
                <button
                  type="button"
                  aria-current={vista === "todas" ? "true" : undefined}
                  className={classeNo(vista === "todas")}
                  onClick={() => escolher("todas")}
                >
                  <span>Todas</span>
                  <Contagem n={contagem.total} />
                </button>
              </li>
              {pastas.map((pasta) => (
                <li key={pasta.id}>
                  <button
                    type="button"
                    aria-current={vista === pasta.id ? "true" : undefined}
                    className={classeNo(vista === pasta.id)}
                    onClick={() => escolher(pasta.id)}
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <Folder aria-hidden className="h-4 w-4 shrink-0" />
                      <span className="truncate">{pasta.nome}</span>
                    </span>
                    <Contagem n={contagem.porPasta[pasta.id] ?? 0} />
                  </button>
                </li>
              ))}
              <li>
                <button
                  type="button"
                  aria-current={vista === "sem" ? "true" : undefined}
                  className={classeNo(vista === "sem")}
                  onClick={() => escolher("sem")}
                >
                  <span>Sem pasta</span>
                  <Contagem n={contagem.sem} />
                </button>
              </li>
            </ul>
          </li>
        </ul>
        {gere ? (
          <div className="mt-3 flex items-center gap-2">
            {!aberta ? (
              <Button type="button" variant="secondary" onClick={() => setAberta(true)}>
                Nova pasta
              </Button>
            ) : null}
            {pastaAtual ? (
              <button
                type="button"
                aria-label={`Opções da pasta ${pastaAtual.nome}`}
                aria-expanded={painel === "menu"}
                aria-haspopup="menu"
                className={buttonClass("secondary", "md", "h-10 w-10 px-0")}
                onClick={() => setPainel((atual) => (atual === "menu" ? null : "menu"))}
              >
                <Settings aria-hidden className="h-4 w-4" />
              </button>
            ) : null}
          </div>
        ) : null}
        {painel === "menu" && pastaAtual ? (
          <div
            role="menu"
            aria-label={`Opções da pasta ${pastaAtual.nome}`}
            className="mt-2 flex flex-col gap-1"
          >
            <Button
              type="button"
              variant="secondary"
              className="justify-start"
              onClick={() => setPainel("membros")}
            >
              Quem entra em {pastaAtual.nome}
              {(pastaAtual.membros?.length ?? 0) > 0 ? ` (${pastaAtual.membros?.length})` : ""}
            </Button>
            <Button
              type="button"
              variant="danger"
              className="justify-start"
              onClick={() => {
                setPainel(null);
                setConfirma(pastaAtual);
              }}
            >
              Apagar pasta {pastaAtual.nome}
            </Button>
          </div>
        ) : null}
      </nav>
      {gere && aberta ? (
        <form className="grid max-w-xl gap-3 sm:grid-cols-2" onSubmit={(e) => void criar(e)}>
          <Input id="pasta-nome" name="nome" label="Nome da pasta *" required maxLength={160} />
          <UnidadeDonaSelect
            id="pasta-unidade"
            permission="cofre:manage"
            label="Unidade"
            optional
          />
          <div className="sm:col-span-2">
            <Button type="submit" loading={pending}>
              Guardar pasta
            </Button>
          </div>
        </form>
      ) : null}
      <Dialog
        open={painel === "membros" && Boolean(pastaAtual)}
        onClose={() => setPainel(null)}
        title={`Quem entra em ${pastaAtual?.nome ?? ""}`}
      >
        {pastaAtual ? (
          <MembrosDaPasta
            pasta={pastaAtual}
            onSaved={() => {
              onSaved();
              setPainel(null);
            }}
          />
        ) : null}
      </Dialog>
      <Dialog
        open={confirma !== null}
        onClose={() => setConfirma(null)}
        title={
          confirma && (contagem.porPasta[confirma.id] ?? 0) > 0
            ? `A pasta ${confirma.nome} tem senha`
            : `Apagar a pasta ${confirma?.nome ?? ""}?`
        }
        description={
          confirma && (contagem.porPasta[confirma.id] ?? 0) > 0
            ? `Esta pasta tem ${contagem.porPasta[confirma.id]} ${contagem.porPasta[confirma.id] === 1 ? "senha" : "senhas"}. Ela não apaga enquanto houver senha dentro. Mova ou apague as senhas antes. A senha não sai junto com a pasta.`
            : "A pasta sai do cofre. Esta ação não pode ser desfeita."
        }
        footer={
          confirma && (contagem.porPasta[confirma.id] ?? 0) > 0 ? (
            <Button
              type="button"
              variant="secondary"
              className="ml-auto"
              onClick={() => setConfirma(null)}
            >
              Entendi
            </Button>
          ) : (
            <>
              <Button type="button" variant="ghost" onClick={() => setConfirma(null)}>
                Cancelar
              </Button>
              <Button
                type="button"
                variant="danger"
                className="ml-auto"
                loading={pending}
                onClick={() => confirma && void apagar(confirma)}
              >
                Confirmar exclusão da pasta {confirma?.nome}
              </Button>
            </>
          )
        }
      />
    </div>
  );
}

function MembrosDaPasta({ pasta, onSaved }: { pasta: Pasta; onSaved: () => void }) {
  const { run, pending } = useAction();
  const pessoas = useApiQuery<Pessoa[]>(
    `v1/cofre/pessoas${pasta.unidade_id ? `?unidade_id=${pasta.unidade_id}` : ""}`,
  );

  async function gravar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(async () => {
      await apiClient.put(`v1/cofre/pastas/${pasta.id}/membros`, {
        membros: fd.getAll("pessoa").map((id) => ({
          usuario_id: String(id),
          pode_revelar: fd.getAll("revela").map(String).includes(String(id)),
        })),
      });
      onSaved();
    }, `Quem entra em ${pasta.nome} foi gravado`);
    if (ok === undefined) return;
  }

  const lista = pessoas.data ?? [];
  const ja = (pasta.membros ?? []).filter(
    (membro) => !lista.some((pessoa) => pessoa.id === membro.usuario_id),
  );
  const opcoes = [
    ...lista.map((pessoa) => ({
      id: pessoa.id,
      nome: pessoa.nome,
      revela: (pasta.membros ?? []).some(
        (membro) => membro.usuario_id === pessoa.id && membro.pode_revelar,
      ),
      entra: (pasta.membros ?? []).some((membro) => membro.usuario_id === pessoa.id),
    })),
    ...ja.map((membro) => ({
      id: membro.usuario_id,
      nome: "Já incluída",
      revela: membro.pode_revelar,
      entra: true,
    })),
  ];

  return (
    <form className="flex flex-col gap-2" onSubmit={(ev) => void gravar(ev)}>
      <fieldset className="flex flex-col gap-2">
        <legend className="text-sm font-medium">Quem entra em {pasta.nome}</legend>
        {opcoes.map((pessoa) => (
          <div key={pessoa.id} className="flex items-center gap-3 text-sm">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                name="pessoa"
                value={pessoa.id}
                defaultChecked={pessoa.entra}
              />
              {pessoa.nome}
            </label>
            <label className="flex items-center gap-2 text-muted">
              <input
                type="checkbox"
                name="revela"
                value={pessoa.id}
                defaultChecked={pessoa.revela}
              />
              pode revelar {pessoa.nome}
            </label>
          </div>
        ))}
        {opcoes.length === 0 && <p className="text-sm text-muted">Nenhuma pessoa neste recorte.</p>}
      </fieldset>
      <div>
        <Button type="submit" loading={pending}>
          Guardar quem entra em {pasta.nome}
        </Button>
      </div>
    </form>
  );
}

function hora(iso: string) {
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** A lista nunca pede a senha. Revelar é um pedido à parte e fica auditado. */
export default function CofrePage() {
  const { can, me } = usePlataforma();
  const ve = can("cofre:read") || can("cofre:reveal") || can("cofre:manage") || can("cofre:grant");
  const gere = can("cofre:manage");
  const revela = can("cofre:reveal") || can("cofre:manage");
  const concede = can("cofre:grant");
  const lista = useApiQuery<Segredo[]>(ve ? "v1/cofre/segredos" : null);
  const pastas = useApiQuery<Pasta[]>(ve ? "v1/cofre/pastas" : null);
  const { run, pending } = useAction();
  const [abertos, setAbertos] = useState<Record<string, { senha: string; notas: string }>>({});
  const [pedido, setPedido] = useState<null | { id: string; modo: "retirar" }>(null);
  const [mais, setMais] = useState<Record<string, boolean>>({});
  const [vista, setVista] = useState("todas");
  const [selecionado, setSelecionado] = useState<string | null>(null);
  const [criando, setCriando] = useState(false);
  const [tipoNovo, setTipoNovo] = useState<Tipo>("outro");
  const [excluirAlvo, setExcluirAlvo] = useState<Segredo | null>(null);
  const [busca, setBusca] = useState("");
  const tituloPainel = useId();
  const visiveis = (lista.data ?? []).filter((segredo) => {
    if (vista === "sem") return !segredo.pasta_id;
    if (vista !== "todas") return segredo.pasta_id === vista;
    return true;
  });
  const termo = busca.trim().toLowerCase();
  const filtrados = visiveis.filter((segredo) => {
    if (!termo) return true;
    return `${segredo.nome} ${segredo.usuario}`.toLowerCase().includes(termo);
  });
  const escolhido = filtrados.find((segredo) => segredo.id === selecionado) ?? null;
  const pastaAberta = (pastas.data ?? []).find((pasta) => pasta.id === vista);
  const itens = lista.data ?? [];
  const contagem = {
    total: itens.length,
    sem: itens.filter((item) => !item.pasta_id).length,
    porPasta: itens.reduce<Record<string, number>>((acc, item) => {
      if (item.pasta_id) acc[item.pasta_id] = (acc[item.pasta_id] ?? 0) + 1;
      return acc;
    }, {}),
  };
  const tituloLista =
    vista === "todas"
      ? "Todas as senhas"
      : vista === "sem"
        ? "Sem pasta"
        : (pastaAberta?.nome ?? "Pasta");
  const nova = ficha(tipoNovo);

  if (!ve) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Cofre" title="Cofre de senhas" />
        <EmptyState
          title="Seu perfil não abre o cofre"
          description="Peça o acesso à unidade dona do segredo."
        />
      </div>
    );
  }

  async function guardar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const unidade = String(fd.get("unidade_id") ?? "");
    const pasta = vista !== "todas" && vista !== "sem" ? vista : "";
    const ok = await run(async () => {
      await apiClient.post("v1/cofre/segredos", {
        nome: String(fd.get("nome") ?? ""),
        usuario: String(fd.get("usuario") ?? ""),
        senha: String(fd.get("senha") ?? ""),
        notas: String(fd.get("notas") ?? ""),
        tipo: String(fd.get("tipo") ?? "outro") || "outro",
        endereco: String(fd.get("endereco") ?? ""),
        administra: [],
        unidade_id: unidade || null,
        rotaciona: false,
        ...(pasta ? { pasta_id: pasta } : {}),
      });
      form.reset();
      await lista.mutate();
    }, "Senha guardada no cofre");
    if (ok === undefined) return;
  }

  async function revelar(segredo: Segredo) {
    const item = await run(
      () =>
        apiClient
          .post<Revelado>(`v1/cofre/segredos/${segredo.id}/revelar`, {})
          .then((res) => res.data),
      "Senha revelada. A consulta ficou na auditoria.",
    );
    if (!item) return;
    setPedido(null);
    setAbertos((atual) => ({ ...atual, [segredo.id]: { senha: item.senha, notas: item.notas } }));
    await lista.mutate();
  }

  async function retirar(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const item = await run(
      () =>
        apiClient
          .post<Revelado>(`v1/cofre/segredos/${segredo.id}/retirar`, {
            senha_de_acesso: String(fd.get("senha_de_acesso") ?? ""),
            minutos: Number(fd.get("minutos") ?? 60),
          })
          .then((res) => res.data),
      "Segredo retirado. A consulta ficou na auditoria.",
    );
    form.reset();
    if (!item) return;
    setPedido(null);
    setAbertos((atual) => ({ ...atual, [segredo.id]: { senha: item.senha, notas: item.notas } }));
    await lista.mutate();
  }

  async function devolver(segredo: Segredo) {
    await run(
      async () => {
        await apiClient.post(`v1/cofre/segredos/${segredo.id}/devolver`, {});
        setPedido(null);
        setAbertos((atual) => {
          const copia = { ...atual };
          delete copia[segredo.id];
          return copia;
        });
        await lista.mutate();
      },
      segredo.rotaciona
        ? "Retirada encerrada. A senha do cofre mudou: revele e aplique no endereço."
        : "Segredo devolvido",
    );
  }

  async function aplicada(segredo: Segredo) {
    await run(async () => {
      await apiClient.post(`v1/cofre/segredos/${segredo.id}/rotacao/aplicada`, {});
      await lista.mutate();
    }, "Aplicação registrada");
  }

  async function trocar(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(async () => {
      await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
        nome: segredo.nome,
        usuario: segredo.usuario,
        senha: String(fd.get("senha") ?? ""),
      });
      form.reset();
      setAbertos((atual) => {
        const copia = { ...atual };
        delete copia[segredo.id];
        return copia;
      });
      await lista.mutate();
    }, "Senha trocada");
    if (ok === undefined) return;
  }

  async function guardarPasta(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const pasta = String(new FormData(e.currentTarget).get("pasta_id") ?? "");
    await run(
      async () => {
        await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
          nome: segredo.nome,
          usuario: segredo.usuario,
          mover_pasta: true,
          pasta_id: pasta || null,
        });
        await lista.mutate();
      },
      pasta ? "Segredo na pasta" : "Segredo fora da pasta",
    );
  }

  async function mover(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const unidade = String(fd.get("unidade_id") ?? "");
    await run(async () => {
      await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
        nome: segredo.nome,
        usuario: segredo.usuario,
        mover_unidade: true,
        unidade_id: unidade || null,
      });
      await lista.mutate();
    }, "Segredo movido");
  }

  async function ligar(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const escolhidas = new FormData(e.currentTarget).getAll("maquina").map(String);
    await run(async () => {
      await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
        nome: segredo.nome,
        usuario: segredo.usuario,
        tipo: segredo.tipo || "outro",
        endereco: segredo.endereco ?? "",
        administra: escolhidas,
      });
      await lista.mutate();
    }, "Máquinas do painel atualizadas");
  }

  async function excluir(segredo: Segredo) {
    const ok = await run(async () => {
      await apiClient.delete(`v1/cofre/segredos/${segredo.id}`);
      setSelecionado(null);
      await lista.mutate();
    }, `Senha ${segredo.nome} apagada do cofre`);
    if (ok === undefined) return;
    setExcluirAlvo(null);
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Cofre"
        title="Cofre de senhas"
        description="A árvore escolhe a pasta. A lista mostra as senhas desse recorte. O clique abre o acesso ao lado."
      />

      <div className="grid items-start gap-6 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <PastaCard
          pastas={pastas.data ?? []}
          vista={vista}
          gere={gere}
          contagem={contagem}
          onVista={(id) => {
            setVista(id);
            setSelecionado(null);
            setCriando(false);
          }}
          onSaved={() => {
            void pastas.mutate();
            void lista.mutate();
          }}
        />

        <section className="flex flex-col gap-3" aria-label="Logins">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-foreground">{tituloLista}</h2>
              <p className="text-sm text-muted">
                {filtrados.length} {filtrados.length === 1 ? "senha" : "senhas"}
              </p>
            </div>
            {gere && (
              <Button
                type="button"
                onClick={() => {
                  setTipoNovo("outro");
                  setCriando(true);
                  setSelecionado(null);
                }}
              >
                Nova senha
              </Button>
            )}
          </div>
          <Input
            id="cofre-busca"
            label="Buscar"
            value={busca}
            onChange={(ev) => setBusca(ev.target.value)}
            placeholder="Nome ou usuário"
          />
          <DataState
            loading={lista.isLoading}
            error={lista.error}
            onRetry={() => void lista.mutate()}
            empty={false}
          >
            {filtrados.length === 0 ? (
              <p className="text-sm text-muted">Nenhum segredo neste recorte</p>
            ) : (
              <ul className="flex flex-col gap-2">
                {filtrados.map((segredo) => {
                  const acesso = ficha(segredo.tipo ?? "outro");
                  return (
                    <li key={segredo.id}>
                      <button
                        type="button"
                        aria-label={segredo.nome}
                        aria-current={selecionado === segredo.id ? "true" : undefined}
                        className={`flex w-full items-center justify-between gap-3 rounded-xl border px-4 py-3 text-left ${
                          selecionado === segredo.id
                            ? "border-primary bg-primary/5"
                            : "border-surface-border bg-surface"
                        }`}
                        onClick={() => {
                          setSelecionado(segredo.id);
                          setCriando(false);
                        }}
                      >
                        <span className="min-w-0">
                          <span className="block truncate text-base font-semibold text-foreground">
                            {segredo.nome}
                          </span>
                          <span className="mt-0.5 block truncate text-sm text-muted">
                            {segredo.usuario ? `Usuário ${segredo.usuario}` : "Sem usuário"}
                          </span>
                        </span>
                        <Badge>{acesso.titulo}</Badge>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </DataState>
        </section>
      </div>

      <ModalShell
        open={criando || Boolean(escolhido)}
        onClose={() => {
          setCriando(false);
          setSelecionado(null);
        }}
        variant="drawer"
        amplitude="media"
        labelledBy={tituloPainel}
      >
        <div className="flex h-full min-h-0 flex-col">
          <div className="flex shrink-0 items-center justify-between gap-3 border-b border-surface-border px-4 py-3">
            <h2 id={tituloPainel} className="truncate text-base font-semibold">
              {criando
                ? pastaAberta
                  ? `Nova senha em ${pastaAberta.nome}`
                  : "Nova senha"
                : escolhido?.nome}
            </h2>
            <button
              type="button"
              onClick={() => {
                setCriando(false);
                setSelecionado(null);
              }}
              aria-label="Fechar painel"
              className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted hover:bg-black/5 hover:text-foreground dark:hover:bg-white/10"
            >
              <X aria-hidden className="h-4 w-4" />
            </button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            {criando && gere ? (
              <form className="grid gap-3" onSubmit={(e) => void guardar(e)}>
                <Select
                  id="cofre-tipo"
                  name="tipo"
                  label="Tipo"
                  value={tipoNovo}
                  onChange={(ev) => setTipoNovo(ev.target.value as Tipo)}
                  options={TIPOS.map((item) => ({ value: item.value, label: item.label }))}
                />
                <Input id="cofre-nome" name="nome" label="Nome *" required maxLength={160} />
                {tipoNovo !== "outro" ? (
                  <Input
                    id="cofre-endereco"
                    name="endereco"
                    label={nova.endereco}
                    maxLength={300}
                    autoComplete="off"
                  />
                ) : null}
                <Input
                  id="cofre-usuario"
                  name="usuario"
                  label={tipoNovo === "outro" ? "Usuário" : nova.usuario}
                  maxLength={160}
                  autoComplete="off"
                />
                <Input
                  id="cofre-senha"
                  name="senha"
                  type="password"
                  label={tipoNovo === "outro" ? "Senha *" : `${nova.senha} *`}
                  required
                  maxLength={2000}
                  autoComplete="new-password"
                />
                <Textarea id="cofre-notas" name="notas" label="Notas" maxLength={4000} rows={2} />
                <div>
                  <Button type="submit" loading={pending}>
                    Guardar no cofre
                  </Button>
                </div>
              </form>
            ) : null}
            {!criando && escolhido ? (
              <ul className="flex flex-col gap-3">
                {[escolhido].map((segredo) => {
                  const aberto = abertos[segredo.id];
                  const reserva = segredo.retirada;
                  const minha = reserva?.usuario_id === me?.id;
                  const ocupado = Boolean(reserva) && !minha;
                  const abertoMais = Boolean(mais[segredo.id]);
                  const acesso = ficha(segredo.tipo ?? "outro");
                  return (
                    <li key={segredo.id}>
                      <div className="flex flex-col gap-3">
                        <div className="rounded-xl border border-surface-border bg-surface p-4">
                          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
                            {acesso.acesso}
                          </p>
                          <p className="mt-1 text-sm text-muted">
                            {acesso.titulo}
                            {nomeDaPasta(pastas.data, segredo.pasta_id)}
                          </p>
                          {segredo.endereco ? (
                            <p className="mt-3">
                              <span className="block text-sm font-medium">{acesso.endereco}</span>
                              <span className="text-lg font-semibold">{segredo.endereco}</span>
                            </p>
                          ) : null}
                          <p className="mt-3">
                            <span className="block text-sm font-medium">{acesso.usuario}</span>
                            <span className="text-lg font-semibold">{segredo.usuario || "—"}</span>
                          </p>
                        </div>
                        {segredo.rotacao_pendente && (
                          <p className="text-sm">
                            A senha do cofre mudou. Revele e aplique
                            {segredo.endereco ? ` em ${segredo.endereco}` : " no endereço"}. O cofre
                            não entra no destino.
                          </p>
                        )}
                        {reserva && (
                          <p className="text-sm text-muted">
                            {minha
                              ? "Você retirou"
                              : `Retirado por ${reserva.nome || "outra pessoa"}`}{" "}
                            até {hora(reserva.ate)}
                          </p>
                        )}
                        {segredo.tipo === "painel" && (
                          <p className="text-sm text-muted">
                            Administra:{" "}
                            {nomesAdministrados(lista.data ?? [], segredo) || "nenhuma máquina"}
                          </p>
                        )}
                        {aberto && (
                          <div className="flex flex-col gap-2">
                            <Input
                              id={`senha-revelada-${segredo.id}`}
                              label={acesso.senha}
                              readOnly
                              value={aberto.senha}
                              autoComplete="off"
                            />
                            {aberto.notas && <p className="text-sm text-muted">{aberto.notas}</p>}
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                variant="secondary"
                                onClick={() => void navigator.clipboard.writeText(aberto.senha)}
                              >
                                Copiar senha de {segredo.nome}
                              </Button>
                              {segredo.usuario ? (
                                <Button
                                  type="button"
                                  variant="secondary"
                                  onClick={() =>
                                    void navigator.clipboard.writeText(segredo.usuario)
                                  }
                                >
                                  Copiar usuário de {segredo.nome}
                                </Button>
                              ) : null}
                              <Button
                                type="button"
                                variant="secondary"
                                onClick={() =>
                                  setAbertos((atual) => {
                                    const copia = { ...atual };
                                    delete copia[segredo.id];
                                    return copia;
                                  })
                                }
                              >
                                Ocultar {segredo.nome}
                              </Button>
                            </div>
                          </div>
                        )}
                        <div className="flex flex-wrap gap-2">
                          {revela && !aberto && !ocupado && (
                            <Button
                              type="button"
                              variant="secondary"
                              onClick={() => void revelar(segredo)}
                            >
                              Revelar {segredo.nome}
                            </Button>
                          )}
                          {(revela || gere || concede) && (
                            <Button
                              type="button"
                              variant="secondary"
                              aria-expanded={abertoMais}
                              onClick={() =>
                                setMais((atual) => ({ ...atual, [segredo.id]: !atual[segredo.id] }))
                              }
                            >
                              {abertoMais ? `Menos de ${segredo.nome}` : `Mais de ${segredo.nome}`}
                            </Button>
                          )}
                          {(minha || (gere && reserva)) && (
                            <Button
                              type="button"
                              variant="secondary"
                              onClick={() => void devolver(segredo)}
                            >
                              Devolver {segredo.nome}
                            </Button>
                          )}
                          {gere && segredo.rotacao_pendente && (
                            <Button
                              type="button"
                              variant="secondary"
                              onClick={() => void aplicada(segredo)}
                            >
                              Já apliquei {segredo.nome}
                            </Button>
                          )}
                          {abertoMais && gere && (
                            <Button
                              type="button"
                              variant="danger"
                              onClick={() => setExcluirAlvo(segredo)}
                            >
                              Excluir {segredo.nome}
                            </Button>
                          )}
                        </div>
                        {abertoMais &&
                          revela &&
                          !aberto &&
                          !ocupado &&
                          pedido?.id !== segredo.id && (
                            <div>
                              <Button
                                type="button"
                                variant="secondary"
                                onClick={() => setPedido({ id: segredo.id, modo: "retirar" })}
                              >
                                Retirar {segredo.nome}
                              </Button>
                            </div>
                          )}
                        {abertoMais &&
                          revela &&
                          !aberto &&
                          !ocupado &&
                          pedido?.id === segredo.id &&
                          pedido.modo === "retirar" && (
                            <form
                              className="flex flex-col gap-2 sm:flex-row sm:items-end"
                              onSubmit={(e) => void retirar(e, segredo)}
                            >
                              <Select
                                id={`prazo-${segredo.id}`}
                                name="minutos"
                                label="Por quanto tempo"
                                defaultValue="60"
                                options={PRAZOS}
                              />
                              <Input
                                id={`acesso-retirada-${segredo.id}`}
                                name="senha_de_acesso"
                                type="password"
                                label="Sua senha de acesso"
                                required
                                autoComplete="current-password"
                              />
                              <Button type="submit" variant="secondary">
                                Confirmar retirada de {segredo.nome}
                              </Button>
                            </form>
                          )}
                        {abertoMais && gere && (
                          <form
                            className="flex flex-col gap-2 sm:flex-row sm:items-end"
                            onSubmit={(e) => void trocar(e, segredo)}
                          >
                            <Input
                              id={`nova-senha-${segredo.id}`}
                              name="senha"
                              type="password"
                              label={`Nova senha de ${segredo.nome}`}
                              required
                              autoComplete="new-password"
                            />
                            <Button type="submit" variant="secondary">
                              Trocar senha de {segredo.nome}
                            </Button>
                          </form>
                        )}
                        {abertoMais && gere && segredo.tipo === "painel" && (
                          <form
                            className="flex flex-col gap-2"
                            onSubmit={(e) => void ligar(e, segredo)}
                          >
                            <MaquinasDoPainel
                              key={(segredo.administra ?? []).join(",")}
                              maquinas={(lista.data ?? []).filter(
                                (item) => item.tipo === "maquina",
                              )}
                              marcadas={segredo.administra}
                              idPrefix={segredo.id}
                            />
                            <Button type="submit" variant="secondary">
                              Guardar máquinas de {segredo.nome}
                            </Button>
                          </form>
                        )}
                        {abertoMais &&
                          gere &&
                          ((pastas.data ?? []).length > 0 || segredo.pasta_id) && (
                            <form
                              className="flex flex-col gap-2 sm:flex-row sm:items-end"
                              onSubmit={(e) => void guardarPasta(e, segredo)}
                            >
                              <Select
                                key={segredo.pasta_id ?? "sem"}
                                id={`pasta-${segredo.id}`}
                                name="pasta_id"
                                label={`Pasta de ${segredo.nome}`}
                                defaultValue={segredo.pasta_id ?? ""}
                                options={[
                                  { value: "", label: "Sem pasta" },
                                  ...(pastas.data ?? []).map((pasta) => ({
                                    value: pasta.id,
                                    label: pasta.nome,
                                  })),
                                  ...(segredo.pasta_id &&
                                  !(pastas.data ?? []).some(
                                    (pasta) => pasta.id === segredo.pasta_id,
                                  )
                                    ? [{ value: segredo.pasta_id, label: "Pasta atual" }]
                                    : []),
                                ]}
                              />
                              <Button type="submit" variant="secondary">
                                Guardar pasta de {segredo.nome}
                              </Button>
                            </form>
                          )}
                        {abertoMais && concede && (
                          <form
                            className="flex flex-col gap-2 sm:flex-row sm:items-end"
                            onSubmit={(e) => void mover(e, segredo)}
                          >
                            <UnidadeDonaSelect
                              id={`mover-${segredo.id}`}
                              permission="cofre:grant"
                              label={`Unidade de ${segredo.nome}`}
                              defaultValue={segredo.unidade_id}
                            />
                            <Button type="submit" variant="secondary">
                              Mover {segredo.nome}
                            </Button>
                          </form>
                        )}
                      </div>
                    </li>
                  );
                })}
              </ul>
            ) : null}
          </div>
        </div>
      </ModalShell>
      <Dialog
        open={excluirAlvo !== null}
        onClose={() => setExcluirAlvo(null)}
        title={`Apagar a senha ${excluirAlvo?.nome ?? ""}?`}
        description="A senha sai do cofre e não volta. Esta ação não pode ser desfeita."
        footer={
          <>
            <Button type="button" variant="ghost" onClick={() => setExcluirAlvo(null)}>
              Cancelar
            </Button>
            <Button
              type="button"
              variant="danger"
              className="ml-auto"
              loading={pending}
              onClick={() => excluirAlvo && void excluir(excluirAlvo)}
            >
              Confirmar exclusão de {excluirAlvo?.nome}
            </Button>
          </>
        }
      />
    </div>
  );
}

function nomesAdministrados(lista: Segredo[], painel: Segredo) {
  const nomes = new Map(
    lista.map((item) => [item.id, item.endereco ? `${item.nome} · ${item.endereco}` : item.nome]),
  );
  return (painel.administra ?? [])
    .map((id) => nomes.get(id) ?? "máquina fora do alcance")
    .join(", ");
}

function MaquinasDoPainel({
  maquinas,
  marcadas = [],
  idPrefix = "novo",
}: {
  maquinas: Segredo[];
  marcadas?: string[];
  idPrefix?: string;
}) {
  return (
    <fieldset className="flex flex-col gap-2 sm:col-span-2">
      <legend className="text-sm font-medium">Máquinas que este painel administra</legend>
      {maquinas.length === 0 && (
        <p className="text-sm text-muted">Nenhuma máquina no cofre ainda.</p>
      )}
      {maquinas.map((maquina) => (
        <label
          key={maquina.id}
          className="flex items-center gap-2 text-sm"
          htmlFor={`${idPrefix}-${maquina.id}`}
        >
          <input
            id={`${idPrefix}-${maquina.id}`}
            type="checkbox"
            name="maquina"
            value={maquina.id}
            defaultChecked={marcadas.includes(maquina.id)}
            className="h-4 w-4 accent-primary"
          />
          {maquina.nome}
          {maquina.endereco ? ` · ${maquina.endereco}` : ""}
        </label>
      ))}
    </fieldset>
  );
}
