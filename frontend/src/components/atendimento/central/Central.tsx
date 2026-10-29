"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { BookOpen, ChevronRight, ClipboardList, SlidersHorizontal } from "lucide-react";
import { type FormEvent, type ReactNode, useCallback, useMemo, useState, useSyncExternalStore } from "react";
import { getActiveEntitySnapshot, subscribeActiveEntity } from "@/lib/plataforma/activeEntityStore";

import { DeflexaoConhecimento } from "@/components/atendimento/DeflexaoConhecimento";
import { PedidoDeContato } from "@/components/canais/ContatoParaAvisos";
import { IconeDoCatalogo } from "@/components/atendimento/IconeDoCatalogo";
import { DataState } from "@/components/plataforma/DataState";
import { Pagination } from "@/components/plataforma/Pagination";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { useFrames } from "@/components/realtime/RealtimeProvider";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree } from "@/lib/plataforma/types";
import {
  FRAME_SOLICITACAO,
  proximoPasso,
  rotuloEstado,
  tomEstado,
  type Categoria,
  type Oferta,
  type Solicitacao,
  type TipoSolicitacao,
} from "@/lib/atendimento/atendimento";

/** As duas portas do catálogo, como na central do GLPI. */
export const PORTAS = {
  pedido: {
    tipo: "requisicao" as TipoSolicitacao,
    titulo: "Abrir um chamado",
    descricao: "Acesso, equipamento, instalação, cadastro: algo que você precisa.",
    icone: "headset",
  },
  problema: {
    tipo: "incidente" as TipoSolicitacao,
    titulo: "Relatar um problema",
    descricao: "Algo parou de funcionar ou não funciona como deveria.",
    icone: "triangle-alert",
  },
};
export type Porta = keyof typeof PORTAS;

/** Categoria das ofertas sem categoria. */
export const OUTROS = "outros";

const BASE = "/atendimento";

/** Bloco clicável da central. */
export function Bloco({
  href,
  titulo,
  descricao,
  icone,
  selo,
}: {
  href: string;
  titulo: string;
  descricao?: string;
  icone: ReactNode;
  selo?: ReactNode;
}) {
  return (
    <li>
      <Link
        href={href}
        className="group flex h-full items-start gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm transition-colors hover:border-primary/50 hover:bg-surface-hover focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary"
      >
        <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          {icone}
        </span>
        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex items-center gap-2 font-semibold group-hover:text-primary">
            {titulo}
            {selo}
          </span>
          {descricao ? <span className="text-sm text-muted">{descricao}</span> : null}
        </span>
        <ChevronRight size={18} aria-hidden="true" className="mt-1 shrink-0 text-muted" />
      </Link>
    </li>
  );
}

function Blocos({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <ul aria-label={rotulo} className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {children}
    </ul>
  );
}

/** Trilha de volta: Central › Abrir um chamado › Acesso. */
function Trilha({ passos }: { passos: { rotulo: string; href?: string }[] }) {
  return (
    <nav aria-label="Você está em" className="text-sm text-muted">
      <ol className="flex flex-wrap items-center gap-1">
        {passos.map((p, i) => (
          <li key={i} className="flex items-center gap-1">
            {i > 0 && <ChevronRight size={14} aria-hidden="true" />}
            {p.href ? (
              <Link href={p.href} className="hover:text-primary hover:underline">
                {p.rotulo}
              </Link>
            ) : (
              <span aria-current="page" className="font-medium text-foreground">
                {p.rotulo}
              </span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}

function useMinhas(situacao: "abertas" | "todas", pagina: number, tamanho = 20) {
  const minhas = useApiPage<Solicitacao>(
    withQuery("v1/atendimento/minhas", { situacao, page: pagina, page_size: tamanho }),
  );
  // A mudança feita pela equipe chega no tópico pessoal de quem abriu.
  useFrames((f) => {
    if (f.type === FRAME_SOLICITACAO && f.topic?.startsWith("user:")) void minhas.mutate();
  });
  return minhas;
}

function ListaDeChamados({ itens }: { itens: Solicitacao[] }) {
  return (
    <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
      {itens.map((s) => (
        <li
          key={s.id}
          className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="min-w-0">
            <Link
              href={`${BASE}/solicitacoes/${s.id}`}
              className="font-medium text-primary hover:underline"
            >
              #{s.numero} · {s.titulo}
            </Link>
            <p className="text-sm text-muted">
              {s.oferta_nome} · atualizada em {fmtDateTime(s.updated_at)}
            </p>
            <p className="text-sm">{proximoPasso(s.estado)}</p>
          </div>
          <Badge tone={tomEstado(s.estado)}>{rotuloEstado(s.estado)}</Badge>
        </li>
      ))}
    </ul>
  );
}

/** Início da central: as quatro portas e o que está em andamento. */
export function CentralInicio() {
  const minhas = useMinhas("abertas", 1, 5);
  const abertas = minhas.data?.meta?.total_items ?? minhas.data?.items.length ?? 0;
  return (
    <div className="flex flex-col gap-8">
      <PedidoDeContato />
      <Blocos rotulo="O que você precisa">
        {(Object.keys(PORTAS) as Porta[]).map((p) => (
          <Bloco
            key={p}
            href={`${BASE}?ver=${p}`}
            titulo={PORTAS[p].titulo}
            descricao={PORTAS[p].descricao}
            icone={<IconeDoCatalogo nome={PORTAS[p].icone} size={24} />}
          />
        ))}
        <Bloco
          href={`${BASE}?ver=chamados`}
          titulo="Meus chamados"
          descricao="Acompanhe, responda e avalie o que você abriu."
          icone={<ClipboardList size={24} aria-hidden="true" />}
          selo={abertas > 0 ? <Badge tone="info">{abertas} em andamento</Badge> : undefined}
        />
        <Bloco
          href="/wiki"
          titulo="Base de conhecimento"
          descricao="Tutoriais e respostas prontas, para resolver na hora."
          icone={<BookOpen size={24} aria-hidden="true" />}
        />
      </Blocos>

      <section aria-labelledby="andamento-titulo" className="flex flex-col gap-3">
        <h2 id="andamento-titulo" className="text-lg font-semibold">
          Em andamento
        </h2>
        <DataState
          loading={minhas.isLoading}
          error={minhas.error}
          onRetry={() => void minhas.mutate()}
          empty={(minhas.data?.items ?? []).length === 0}
          emptyTitle="Nada em andamento"
          emptyDescription="Quando você abrir um chamado, ele aparece aqui."
        >
          <ListaDeChamados itens={minhas.data?.items ?? []} />
          {abertas > 5 && (
            <Link href={`${BASE}?ver=chamados`} className="text-sm text-primary hover:underline">
              Ver todos os chamados
            </Link>
          )}
        </DataState>
      </section>
    </div>
  );
}

function ordenar<T extends { ordem?: number; nome: string }>(lista: T[]): T[] {
  return [...lista].sort(
    (a, b) => (a.ordem ?? 0) - (b.ordem ?? 0) || a.nome.localeCompare(b.nome, "pt-BR"),
  );
}

/** Catálogo de uma porta: as categorias em blocos e, dentro de uma, as ofertas. */
export function CatalogoEmBlocos({ porta, categoria }: { porta: Porta; categoria?: string }) {
  const { me } = usePlataforma();
  const p = PORTAS[porta];
  const catalogo = useApiQuery<Oferta[]>("v1/atendimento/catalogo");
  const categorias = useApiQuery<Categoria[]>("v1/atendimento/catalogo/categorias");
  const tree = useApiQuery<OrgTree[]>("v1/iam/org-tree");
  const [busca, setBusca] = useState("");
  const [secretariaFiltro, setSecretariaFiltro] = useState<string>("");
  const activeEntity = useSyncExternalStore(subscribeActiveEntity, getActiveEntitySnapshot, () => "");

  const unidadeParaEntidade = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const ent of tree.data ?? []) {
      mapa.set(ent.id, ent.id);
      for (const u of ent.unidades) {
        mapa.set(u.id, ent.id);
      }
    }
    return mapa;
  }, [tree.data]);

  const secretarias = useMemo(() => {
    const mapa = new Map<string, { id: string; nome: string; sigla: string }>();
    const unidadesNaArvore = new Map<string, { id: string; nome: string; sigla: string }>();
    for (const ent of tree.data ?? []) {
      for (const u of ent.unidades) {
        unidadesNaArvore.set(u.id, {
          id: u.id,
          nome: u.nome,
          sigla: u.sigla || u.nome,
        });
      }
    }
    for (const o of catalogo.data ?? []) {
      if (o.tipo === p.tipo) {
        if (o.secretaria_id) {
          mapa.set(o.secretaria_id, {
            id: o.secretaria_id,
            nome: o.secretaria_nome || o.secretaria_sigla || "Unidade",
            sigla: o.secretaria_sigla || o.secretaria_nome || "Unidade",
          });
        } else if (o.unidade_id && unidadesNaArvore.has(o.unidade_id)) {
          const u = unidadesNaArvore.get(o.unidade_id)!;
          mapa.set(u.id, u);
        }
      }
    }
    return [...mapa.values()].sort((a, b) => a.sigla.localeCompare(b.sigla, "pt-BR"));
  }, [catalogo.data, p.tipo, tree.data]);

  const secretariaPadrao = useMemo(() => {
    if (secretarias.length <= 1) return "todas";
    for (const escopo of me?.scopes ?? []) {
      if (escopo.unidade_id) {
        if (secretarias.some((s) => s.id === escopo.unidade_id)) {
          return escopo.unidade_id;
        }
        for (const ent of tree.data ?? []) {
          const matchU = ent.unidades.find((u) => u.id === escopo.unidade_id);
          if (matchU?.parent_id && secretarias.some((s) => s.id === matchU.parent_id)) {
            return matchU.parent_id;
          }
        }
      }
      if (escopo.entidade_id) {
        const ent = (tree.data ?? []).find((e) => e.id === escopo.entidade_id);
        if (ent) {
          const achou = secretarias.find((s) => ent.unidades.some((u) => u.id === s.id));
          if (achou) return achou.id;
        }
      }
    }
    return secretarias[0]?.id ?? "todas";
  }, [me?.scopes, secretarias, tree.data]);

  const secretariaAtiva =
    secretariaFiltro ||
    (activeEntity ? activeEntity : secretariaPadrao);

  const ofertas = useMemo(() => {
    const raw = (catalogo.data ?? []).filter((o) => {
      if (o.tipo !== p.tipo) return false;
      if (secretarias.length <= 1 || secretariaAtiva === "todas") return true;
      const secId = o.secretaria_id || o.unidade_id;
      const entId = unidadeParaEntidade.get(secId || "");
      return secId === secretariaAtiva || entId === secretariaAtiva;
    });

    const vistos = new Set<string>();
    const unicos: Oferta[] = [];
    for (const o of raw) {
      const chave = `${o.secretaria_id || o.unidade_id || "global"}:${o.nome.trim().toLowerCase()}`;
      if (!vistos.has(chave)) {
        vistos.add(chave);
        unicos.push(o);
      }
    }
    return ordenar(unicos);
  }, [catalogo.data, p.tipo, secretarias.length, secretariaAtiva, unidadeParaEntidade]);

  const conhecidas = useMemo(
    () => new Set((categorias.data ?? []).map((c) => c.id)),
    [categorias.data],
  );
  const daCategoria = useCallback(
    (o: Oferta) => (o.categoria_id && conhecidas.has(o.categoria_id) ? o.categoria_id : OUTROS),
    [conhecidas],
  );

  const grupos = useMemo(() => {
    const lista = ordenar(categorias.data ?? []).filter((c) =>
      ofertas.some((o) => daCategoria(o) === c.id),
    );
    const vistas = new Set<string>();
    const deduplicadas: Categoria[] = [];
    for (const cat of lista) {
      const chave = cat.nome.trim().toLowerCase();
      if (!vistas.has(chave)) {
        vistas.add(chave);
        deduplicadas.push(cat);
      }
    }
    if (ofertas.some((o) => daCategoria(o) === OUTROS)) {
      deduplicadas.push({
        id: OUTROS,
        nome: "Outros",
        descricao: "Demais atendimentos.",
        icone: "",
        ordem: 999,
        ativa: true,
      });
    }
    return deduplicadas;
  }, [categorias.data, daCategoria, ofertas]);

  const termo = busca.trim().toLocaleLowerCase("pt-BR");
  const escolhida = grupos.find((g) => g.id === categoria);
  const visiveis = termo
    ? ofertas.filter((o) => `${o.nome} ${o.descricao}`.toLocaleLowerCase("pt-BR").includes(termo))
    : escolhida
      ? ofertas.filter((o) => daCategoria(o) === escolhida.id)
      : [];

  return (
    <div className="flex flex-col gap-5">
      <Trilha
        passos={[
          { rotulo: "Central", href: BASE },
          escolhida ? { rotulo: p.titulo, href: `${BASE}?ver=${porta}` } : { rotulo: p.titulo },
          ...(escolhida ? [{ rotulo: escolhida.nome }] : []),
        ]}
      />
      {secretarias.length > 1 && (
        <SectionTabsInline
          label="Entidade provedora do atendimento"
          value={secretariaAtiva}
          onChange={setSecretariaFiltro}
          tabs={[
            { value: "todas", label: "Todas as centrais" },
            ...secretarias.map((s) => ({
              value: s.id,
              label: s.sigla && s.sigla !== s.nome ? `${s.sigla} · ${s.nome}` : s.nome,
            })),
          ]}
        />
      )}
      <div className="relative">
        <Input
          id="central-busca"
          type="search"
          label="Procurar no catálogo"
          placeholder="Ex.: senha, impressora, internet"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
          autoComplete="off"
        />
      </div>
      <DataState
        loading={catalogo.isLoading || categorias.isLoading}
        error={catalogo.error ?? categorias.error}
        onRetry={() => {
          void catalogo.mutate();
          void categorias.mutate();
        }}
        empty={ofertas.length === 0}
        emptyTitle="Nenhum atendimento disponível aqui"
        emptyDescription="A gestão ainda não cadastrou o que pode ser pedido por esta porta."
        rows={2}
      >
        {termo || escolhida ? (
          visiveis.length === 0 ? (
            <p className="text-sm text-muted">Nada encontrado com “{busca.trim()}”.</p>
          ) : (
            <Blocos rotulo="Atendimentos">
              {visiveis.map((o) => {
                const rotuloSec = o.secretaria_sigla || o.secretaria_nome;
                return (
                  <Bloco
                    key={o.id}
                    href={`${BASE}?oferta=${o.id}`}
                    titulo={o.nome}
                    descricao={o.descricao}
                    icone={<IconeDoCatalogo nome={o.icone} contextoNome={o.nome} padrao={p.icone} size={24} />}
                    selo={
                      secretarias.length > 1 && rotuloSec ? (
                        <Badge tone="neutral">{rotuloSec}</Badge>
                      ) : undefined
                    }
                  />
                );
              })}
            </Blocos>
          )
        ) : (
          <Blocos rotulo="Categorias">
            {grupos.map((g) => (
              <Bloco
                key={g.id}
                href={`${BASE}?ver=${porta}&categoria=${g.id}`}
                titulo={g.nome}
                descricao={g.descricao}
                icone={<IconeDoCatalogo nome={g.icone} contextoNome={g.nome} padrao={p.icone} size={24} />}
              />
            ))}
          </Blocos>
        )}
      </DataState>
    </div>
  );
}

/** Formulário do pedido de uma oferta. Enviado, abre o chamado criado. */
export function NovoPedido({ ofertaId }: { ofertaId: string }) {
  const router = useRouter();
  const catalogo = useApiQuery<Oferta[]>("v1/atendimento/catalogo");
  const oferta = catalogo.data?.find((o) => o.id === ofertaId);
  const porta: Porta = oferta?.tipo === "incidente" ? "problema" : "pedido";
  const [titulo, setTitulo] = useState("");
  const { run, pending } = useAction();

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const localizacao = String(fd.get("localizacao") || "").trim();
    const patrimonio = String(fd.get("patrimonio") || "").trim();
    const impacto = String(fd.get("impacto") || "").trim();
    const detalhes = String(fd.get("descricao") || "").trim();
    const complementos: string[] = [];
    if (localizacao) complementos.push(`• Local / Ramal: ${localizacao}`);
    if (patrimonio) complementos.push(`• Patrimônio / Ativo: ${patrimonio}`);
    if (impacto && impacto !== "individual") {
      const rotulos: Record<string, string> = {
        setorial: "Afeta todo o setor / equipe",
        institucional: "Afeta múltiplos setores / institucional",
      };
      complementos.push(`• Impacto: ${rotulos[impacto] || impacto}`);
    }
    let descricao = detalhes;
    if (complementos.length > 0) {
      const cabecalho = `[Informações Complementares]\n${complementos.join("\n")}`;
      descricao = detalhes ? `${cabecalho}\n\n${detalhes}` : cabecalho;
    }
    const respostas: Record<string, string> = {};
    for (const campo of oferta?.campos ?? []) {
      const valor = String(fd.get(`campo_${campo.chave}`) ?? "").trim();
      if (valor) respostas[campo.chave] = valor;
    }
    const body: Record<string, unknown> = {
      oferta_id: ofertaId,
      titulo: String(fd.get("titulo")).trim(),
      descricao,
    };
    if (Object.keys(respostas).length > 0) body.respostas = respostas;
    const criada = await run(
      () => apiClient.post<Solicitacao>("v1/atendimento/solicitacoes", body),
      "Chamado aberto. Acompanhe por aqui.",
    );
    if (criada) router.push(`${BASE}/solicitacoes/${criada.data.id}`);
  }

  return (
    <div className="flex flex-col gap-5">
      <Trilha
        passos={[
          { rotulo: "Central", href: BASE },
          { rotulo: PORTAS[porta].titulo, href: `${BASE}?ver=${porta}` },
          { rotulo: oferta?.nome ?? "Atendimento" },
        ]}
      />
      <DataState
        loading={catalogo.isLoading}
        error={catalogo.error}
        onRetry={() => void catalogo.mutate()}
        empty={!oferta}
        emptyTitle="Este atendimento não está disponível"
        emptyDescription="Ele pode ter sido desativado. Volte ao catálogo e escolha outro."
        rows={3}
      >
        {oferta && (
          <Card>
            <CardContent className="py-5">
              <div className="mb-4 flex items-start gap-4">
                <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <IconeDoCatalogo nome={oferta.icone} padrao={PORTAS[porta].icone} size={24} />
                </span>
                <div>
                  <h2 className="text-lg font-semibold">{oferta.nome}</h2>
                  {oferta.descricao ? (
                    <p className="text-sm text-muted">{oferta.descricao}</p>
                  ) : null}
                </div>
              </div>
              <form onSubmit={enviar} className="grid gap-3">
                {(oferta.campos ?? []).map((campo) =>
                  campo.tipo === "lista" ? (
                    <Select
                      key={campo.chave}
                      id={`campo-${campo.chave}`}
                      name={`campo_${campo.chave}`}
                      label={`${campo.rotulo}${campo.obrigatorio ? " *" : ""}`}
                      required={campo.obrigatorio}
                      placeholder="Escolha"
                      options={(campo.opcoes ?? []).map((o) => ({ value: o, label: o }))}
                    />
                  ) : (
                    <Input
                      key={campo.chave}
                      id={`campo-${campo.chave}`}
                      name={`campo_${campo.chave}`}
                      label={`${campo.rotulo}${campo.obrigatorio ? " *" : ""}`}
                      required={campo.obrigatorio}
                      inputMode={campo.tipo === "numero" ? "numeric" : undefined}
                      autoComplete="off"
                    />
                  ),
                )}
                <Input
                  id="solicitacao-titulo"
                  name="titulo"
                  label="Resumo *"
                  required
                  maxLength={160}
                  autoComplete="off"
                  placeholder="Ex.: a senha do e-mail expirou"
                  value={titulo}
                  onChange={(e) => setTitulo(e.target.value)}
                />
                <DeflexaoConhecimento
                  termo={`${oferta.nome} ${titulo}`}
                  onResolvido={() => router.push(BASE)}
                />
                <details className="group rounded-lg border border-surface-border bg-black/[0.02] p-3 text-sm dark:bg-white/[0.02]">
                  <summary className="flex cursor-pointer select-none items-center justify-between font-medium text-muted hover:text-foreground">
                    <span className="flex items-center gap-1.5">
                      <SlidersHorizontal className="h-4 w-4 text-primary" aria-hidden="true" />
                      Informações complementares (localização, equipamento, impacto)
                    </span>
                    <span className="text-xs font-normal text-muted">(Opcional)</span>
                  </summary>
                  <div className="mt-3 grid gap-3 border-t border-surface-border pt-2 sm:grid-cols-2">
                    <Input
                      id="solicitacao-localizacao"
                      name="localizacao"
                      label="Localização / Ramal"
                      placeholder="Ex.: Bloco B - Sala 204 (Ramal 3450)"
                      autoComplete="off"
                    />
                    <Input
                      id="solicitacao-patrimonio"
                      name="patrimonio"
                      label="Patrimônio / Equipamento"
                      placeholder="Ex.: PAT-2024-8841 ou Dell OptiPlex"
                      autoComplete="off"
                    />
                    <div className="sm:col-span-2">
                      <Select
                        id="solicitacao-impacto"
                        name="impacto"
                        label="Impacto percebido"
                        placeholder="Individual (apenas meu posto de trabalho)"
                        options={[
                          {
                            value: "individual",
                            label: "Individual (apenas meu posto de trabalho)",
                          },
                          {
                            value: "setorial",
                            label: "Setorial (afeta toda a minha equipe / setor)",
                          },
                          {
                            value: "institucional",
                            label: "Institucional (afeta múltiplos setores ou o prédio)",
                          },
                        ]}
                      />
                    </div>
                  </div>
                </details>
                <Textarea
                  id="solicitacao-descricao"
                  name="descricao"
                  label="Detalhes"
                  maxLength={4000}
                  rows={4}
                  placeholder="Desde quando, onde aparece o erro, um telefone para contato…"
                />
                <div>
                  <Button type="submit" loading={pending}>
                    Abrir chamado
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}
      </DataState>
    </div>
  );
}

/** Meus chamados: em andamento ou todos, com o próximo passo de cada um. */
export function MeusChamados() {
  const [situacao, setSituacao] = useState<"abertas" | "todas">("abertas");
  const [pagina, setPagina] = useState(1);
  const minhas = useMinhas(situacao, pagina);
  return (
    <div className="flex flex-col gap-4">
      <Trilha passos={[{ rotulo: "Central", href: BASE }, { rotulo: "Meus chamados" }]} />
      <SectionTabsInline
        label="Situação dos chamados"
        value={situacao}
        onChange={(v) => {
          setSituacao(v);
          setPagina(1);
        }}
        tabs={[
          { value: "abertas", label: "Em andamento" },
          { value: "todas", label: "Todos" },
        ]}
      />
      <DataState
        loading={minhas.isLoading}
        error={minhas.error}
        onRetry={() => void minhas.mutate()}
        empty={(minhas.data?.items ?? []).length === 0}
        emptyTitle={situacao === "abertas" ? "Nada em andamento" : "Você ainda não abriu chamados"}
      >
        <ListaDeChamados itens={minhas.data?.items ?? []} />
        <Pagination meta={minhas.data?.meta} onPage={setPagina} />
      </DataState>
    </div>
  );
}
