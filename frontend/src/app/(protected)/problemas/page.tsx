"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { ListaFaixas, NomeClicavel, ordenarColunas, seloDoEstado, useAparencia } from "@/components/plataforma/Faixas";
import { colunasVisiveis, rotuloQuadro, useQuadro } from "@/components/plataforma/useQuadro";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { COLUNAS_PROBLEMA, ProblemasKanban } from "@/components/servicos/ProblemasKanban";
import { ProblemaDetalhe } from "@/app/(protected)/problemas/[id]/page";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  ESTADO_PROBLEMA,
  type EstadoProblema,
  type Problema,
  type Recorrencia,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

type Vista = "lista" | "kanban";

/** Caminho legal a partir de registrado. O id é o estado; o nome da coluna não entra. */
function passosDoProblema(alvo: EstadoProblema): EstadoProblema[] {
  switch (alvo) {
    case "registrado":
      return [];
    case "em_analise":
      return ["em_analise"];
    case "erro_conhecido":
      return ["em_analise", "erro_conhecido"];
    case "resolvido":
      return ["em_analise", "resolvido"];
    case "encerrado":
      return ["em_analise", "resolvido", "encerrado"];
  }
}

export default function ProblemasPage() {
  return (
    <Suspense fallback={null}>
      <ProblemasConteudo />
    </Suspense>
  );
}

/** Problemas e a base de erros conhecidos. Quem gere registra e
 * investiga; quem atende a fila consulta o contorno. Exibe em lista ou em
 * quadro kanban. */
function ProblemasConteudo() {
  const { can } = usePlataforma();
  const router = useRouter();
  const params = useSearchParams();
  const painel = usePainelNaUrl();
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const gere = can("problemas:manage");
  const le = gere || can("problemas:read");
  const [estado, setEstado] = useState("");
  const [busca, setBusca] = useState("");
  const [formAberto, setFormAberto] = useState(false);
  const [quadro, setQuadro] = useState<EstadoProblema>("registrado");
  const look = useAparencia("nova-faixas:problemas");
  const catalogo = useQuadro("problemas", COLUNAS_PROBLEMA);
  const controles = { oficial: catalogo.oficial };
  const kanban = vista === "kanban";

  const problemas = useApiQuery<Problema[]>(
    le
      ? withQuery("v1/problemas", {
          estado: kanban ? undefined : estado,
          q: kanban ? undefined : busca,
        })
      : null,
  );
  const sugestoes = useApiQuery<Recorrencia[]>(gere ? "v1/problemas/sugestoes" : null);
  const { run, pending } = useAction();

  // Estado para modal de transição que exige campos
  const [modalDados, setModalDados] = useState<{
    problema: Problema;
    para: EstadoProblema;
    campo: "contorno" | "causa_raiz";
    titulo: string;
    rotulo: string;
  } | null>(null);
  const [textoModal, setTextoModal] = useState("");

  if (!le) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Assistência" title="Problemas" />
        <EmptyState
          title="Seu perfil não consulta problemas"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  function trocarVista(v: Vista) {
    router.replace(painel.href({ vista: v === "lista" ? "lista" : "quadro" }));
  }

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const alvo = quadro;
    const criado = await run(async () => {
      const res = await apiClient.post<Problema>("v1/problemas", {
        titulo: String(fd.get("titulo")).trim(),
        descricao: String(fd.get("descricao")).trim(),
        causa_raiz: String(fd.get("causa_raiz") ?? "").trim(),
        contorno: String(fd.get("contorno") ?? "").trim(),
        unidade_id: String(fd.get("unidade_id") || "") || null,
      });
      for (const passo of passosDoProblema(alvo)) {
        await apiClient.post(`v1/problemas/${res.data.id}/estado`, { estado: passo });
      }
      return res.data;
    }, "Problema registrado");
    if (criado) {
      setFormAberto(false);
      router.push(`/problemas/${criado.id}`);
    }
  }

  async function renomearProblema(p: Problema, nome: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/problemas/${p.id}`, {
          titulo: nome.slice(0, 160),
          descricao: p.descricao,
          causa_raiz: p.causa_raiz,
          contorno: p.contorno,
          produto_id: p.produto_id ?? null,
          unidade_id: p.unidade_id ?? null,
        }),
      "Nome atualizado",
    );
    if (ok) void problemas.mutate();
  }

  /** Abre o problema da recorrência e já liga os incidentes dela. */
  async function abrirDaSugestao(r: Recorrencia) {
    const criado = await run(async () => {
      const { data } = await apiClient.post<Problema>("v1/problemas", {
        titulo: `Incidentes repetidos: ${r.oferta_nome}`.slice(0, 160),
        unidade_id: r.unidade_id ?? null,
      });
      for (const s of r.solicitacoes)
        await apiClient.put(`v1/problemas/${data.id}/incidentes/${s}`);
      return data;
    }, "Problema aberto com os incidentes");
    if (criado) router.push(`/problemas/${criado.id}`);
  }

  async function mover(p: Problema, para: EstadoProblema) {
    if (para === "erro_conhecido" && !p.contorno) {
      setModalDados({
        problema: p,
        para,
        campo: "contorno",
        titulo: "Publicar Erro Conhecido",
        rotulo: "Solução de contorno *",
      });
      setTextoModal("");
      return;
    }
    if (para === "resolvido" && !p.causa_raiz) {
      setModalDados({
        problema: p,
        para,
        campo: "causa_raiz",
        titulo: "Marcar como Resolvido",
        rotulo: "Causa raiz identificada *",
      });
      setTextoModal("");
      return;
    }
    const ok = await run(
      () => apiClient.post(`v1/problemas/${p.id}/estado`, { estado: para }),
      `Problema #${p.numero}: ${seloDoEstado(catalogo.colunas, para, ESTADO_PROBLEMA[para]).label.toLowerCase()}`,
    );
    if (ok) void problemas.mutate();
  }

  async function salvarCampoETransicionar(e: FormEvent) {
    e.preventDefault();
    if (!modalDados || !textoModal.trim()) return;
    const { problema: p, para, campo } = modalDados;

    const okPut = await run(
      () =>
        apiClient.put(`v1/problemas/${p.id}`, {
          titulo: p.titulo,
          descricao: p.descricao,
          causa_raiz: campo === "causa_raiz" ? textoModal.trim() : p.causa_raiz,
          contorno: campo === "contorno" ? textoModal.trim() : p.contorno,
          produto_id: p.produto_id ?? null,
          unidade_id: p.unidade_id ?? null,
        }),
      "Dados atualizados",
    );
    if (!okPut) return;

    const okPost = await run(
      () => apiClient.post(`v1/problemas/${p.id}/estado`, { estado: para }),
      `Problema #${p.numero}: ${seloDoEstado(catalogo.colunas, para, ESTADO_PROBLEMA[para]).label.toLowerCase()}`,
    );
    setModalDados(null);
    if (okPost) void problemas.mutate();
  }

  const lista = problemas.data ?? [];
  const colunas = colunasVisiveis(catalogo.colunas, lista, (p) => p.estado);
  const quadros = ordenarColunas(catalogo.colunas.filter((coluna) => coluna.ativa !== false)).map((coluna) => ({
    value: coluna.id,
    label: rotuloQuadro(catalogo.oficial, catalogo.colunas, look.gravado, coluna.id),
  }));

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Problemas"
        description="A causa por trás dos incidentes que se repetem. Com a solução de contorno registrada, o problema vira erro conhecido e a equipe de atendimento consulta aqui."
        actions={
          gere ? (
            <>
              <Link href="/problemas/colunas" className={buttonClass("secondary")}>
                Colunas
              </Link>
              <Button
                type="button"
                onClick={() => {
                  setQuadro("registrado");
                  setFormAberto(true);
                }}
              >
                Novo problema
              </Button>
            </>
          ) : undefined
        }
      />

      <SectionTabsInline
        label="Forma de ver os problemas"
        value={vista}
        onChange={trocarVista}
        tabs={[
          { value: "kanban", label: "Quadro" },
          { value: "lista", label: "Lista" },
        ]}
      />

      {!kanban && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Select
            id="problemas-estado"
            label="Estado"
            value={estado}
            onChange={(e) => setEstado(e.target.value)}
            options={[
              { value: "", label: "Todos" },
              ...ordenarColunas(catalogo.colunas.filter((coluna) => coluna.ativa !== false)).map((coluna) => ({
                value: coluna.id,
                label: coluna.titulo,
              })),
            ]}
          />
          <Input
            id="problemas-busca"
            label="Buscar no título ou no contorno"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
          />
        </div>
      )}

      <DataState
        loading={problemas.isLoading}
        error={problemas.error}
        onRetry={() => void problemas.mutate()}
        empty={false}
      >
        {lista.length === 0 && (
          <p className="text-sm text-muted">Nenhum problema neste recorte</p>
        )}
        {kanban ? (
          <ProblemasKanban
            itens={lista}
            onMover={mover}
            pending={pending}
            aoRenomear={gere ? (p, nome) => void renomearProblema(p, nome) : undefined}
            aoAbrir={(p) => painel.abrir(p.id)}
            colunas={colunas}
            controles={controles}
          />
        ) : (
          <ListaFaixas
            rotulo="Lista de problemas"
            memoria="nova-faixas:problemas-lista"
            aparencia="nova-faixas:problemas"
            colunas={colunas}
            {...controles}
            itens={lista}
            colunaDe={(p) => p.estado}
            idDe={(p) => p.id}
            nomeDe={(p) => p.titulo}
            aoRenomear={gere ? (p, nome) => void renomearProblema(p, nome) : undefined}
            aoAbrir={(p) => painel.abrir(p.id)}
            render={(p, vista) => (
              <div className="flex flex-col gap-1">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <NomeClicavel
                    nome={vista.titulo || p.titulo}
                    rotulo={`#${p.numero} · ${vista.titulo || p.titulo}`}
                    href={`/problemas/${p.id}`}
                    onSalvar={vista.salvarNome}
                    className="font-medium text-primary hover:underline"
                  />
                  <Badge tone={seloDoEstado(catalogo.colunas, p.estado, ESTADO_PROBLEMA[p.estado]).tone}>
                    {seloDoEstado(catalogo.colunas, p.estado, ESTADO_PROBLEMA[p.estado]).label}
                  </Badge>
                </div>
                {p.contorno && <p className="text-sm text-muted">Contorno: {p.contorno}</p>}
                <button
                  type="button"
                  className="text-left text-xs font-medium text-primary hover:underline"
                  onClick={() => painel.abrir(p.id)}
                >
                  Editar
                </button>
              </div>
            )}
          />
        )}
      </DataState>

      {gere && (sugestoes.data ?? []).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Incidentes que se repetem</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <ul className="flex flex-col gap-2">
              {(sugestoes.data ?? []).map((r) => (
                <li
                  key={r.oferta_id}
                  className="flex flex-wrap items-center justify-between gap-2 text-sm"
                >
                  <span>
                    <strong>{r.oferta_nome}</strong>: {r.total} incidentes em 30 dias, o último em{" "}
                    {fmtDateTime(r.ultimo_em)}
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => void abrirDaSugestao(r)}
                  >
                    Abrir problema de {r.oferta_nome}
                  </Button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {gere && formAberto && (
        <Dialog
          open
          size="full"
          onClose={() => setFormAberto(false)}
          title="Novo problema"
          description="O problema entra no quadro escolhido. O nome da coluna é só o rótulo; o sistema guarda o estado."
        >
          <form onSubmit={criar} className="grid gap-3">
            <Input
              id="problema-titulo"
              name="titulo"
              label="Título *"
              required
              maxLength={160}
              autoComplete="off"
            />
            <Textarea
              id="problema-descricao"
              name="descricao"
              label="O que se observa"
              maxLength={4000}
              rows={12}
              className="min-h-[42vh]"
            />
            <Select
              id="problema-quadro"
              label="Quadro"
              value={quadro}
              onChange={(e) => setQuadro(e.target.value as EstadoProblema)}
              options={quadros}
            />
            {quadro === "erro_conhecido" && (
              <Textarea
                id="problema-contorno"
                name="contorno"
                label="Solução de contorno *"
                required
                maxLength={4000}
                rows={12}
                className="min-h-[42vh]"
              />
            )}
            {(quadro === "resolvido" || quadro === "encerrado") && (
              <Textarea
                id="problema-causa"
                name="causa_raiz"
                label="Causa raiz *"
                required
                maxLength={4000}
                rows={12}
                className="min-h-[42vh]"
              />
            )}
            <UnidadeDonaSelect
              id="problema-unidade"
              permission="problemas:manage"
              label="Unidade"
            />
            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setFormAberto(false)}>
                Cancelar
              </Button>
              <Button type="submit" loading={pending}>
                Registrar problema
              </Button>
            </div>
          </form>
        </Dialog>
      )}

      {/* Modal para preenchimento de contorno ou causa raiz ao mover */}
      {modalDados && (
        <Dialog
          open={true}
          size="full"
          onClose={() => setModalDados(null)}
          title={modalDados.titulo}
          description={`Para avançar o problema #${modalDados.problema.numero} para ${seloDoEstado(catalogo.colunas, modalDados.para, ESTADO_PROBLEMA[modalDados.para]).label.toLowerCase()}, preencha a informação obrigatória:`}
          footer={
            <>
              <Button type="button" variant="ghost" onClick={() => setModalDados(null)}>
                Cancelar
              </Button>
              <Button
                type="submit"
                form="modal-transicao-form"
                loading={pending}
                disabled={!textoModal.trim()}
              >
                Confirmar passagem
              </Button>
            </>
          }
        >
          <form id="modal-transicao-form" onSubmit={salvarCampoETransicionar} className="mt-2">
            <Textarea
              id="modal-campo-texto"
              label={modalDados.rotulo}
              required
              rows={12}
              className="min-h-[42vh]"
              value={textoModal}
              onChange={(e) => setTextoModal(e.target.value)}
              placeholder="Descreva detalhadamente..."
            />
          </form>
        </Dialog>
      )}
      <PainelDoQuadro
        aberto={painel.id !== null}
        id={painel.id}
        titulo="Detalhe do problema"
        href={painel.id ? `/problemas/${painel.id}` : "/problemas"}
        onClose={painel.fechar}
      >
        {painel.id && <ProblemaDetalhe />}
      </PainelDoQuadro>
    </div>
  );
}
