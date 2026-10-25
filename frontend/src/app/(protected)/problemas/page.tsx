"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { ListaFaixas, NomeClicavel } from "@/components/plataforma/Faixas";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { COLUNAS_PROBLEMA, ProblemasKanban } from "@/components/servicos/ProblemasKanban";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
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
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const gere = can("problemas:manage");
  const le = gere || can("problemas:read");
  const [estado, setEstado] = useState("");
  const [busca, setBusca] = useState("");
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
    router.replace(v === "lista" ? "/problemas?vista=lista" : "/problemas?vista=quadro");
  }

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const criado = await run(
      () =>
        apiClient.post<Problema>("v1/problemas", {
          titulo: String(fd.get("titulo")).trim(),
          descricao: String(fd.get("descricao")).trim(),
          unidade_id: String(fd.get("unidade_id")) || null,
        }),
      "Problema registrado",
    );
    if (criado) router.push(`/problemas/${criado.data.id}`);
  }

  async function criarCard(nome: string) {
    const ok = await run(
      () =>
        apiClient.post<Problema>("v1/problemas", {
          titulo: nome.slice(0, 160),
          descricao: "",
          unidade_id: null,
        }),
      "Problema registrado",
    );
    if (ok) void problemas.mutate();
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
      `Problema #${p.numero}: ${ESTADO_PROBLEMA[para].label.toLowerCase()}`,
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
      `Problema #${p.numero}: ${ESTADO_PROBLEMA[para].label.toLowerCase()}`,
    );
    setModalDados(null);
    if (okPost) void problemas.mutate();
  }

  const lista = problemas.data ?? [];

  function novoProblema(colunaId: string) {
    if (colunaId !== "registrado" || !gere) return null;
    return (
      <Button
        type="button"
        size="sm"
        variant="secondary"
        onClick={() => document.getElementById("problema-titulo")?.focus()}
      >
        Novo problema
      </Button>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Problemas"
        description="A causa por trás dos incidentes que se repetem. Com a solução de contorno registrada, o problema vira erro conhecido e a equipe de atendimento consulta aqui."
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
              ...(Object.keys(ESTADO_PROBLEMA) as EstadoProblema[]).map((e) => ({
                value: e,
                label: ESTADO_PROBLEMA[e].label,
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
            extra={novoProblema}
            aoCriar={gere ? (_coluna, nome) => void criarCard(nome) : undefined}
            aoRenomear={gere ? (p, nome) => void renomearProblema(p, nome) : undefined}
          />
        ) : (
          <ListaFaixas
            rotulo="Lista de problemas"
            memoria="nova-faixas:problemas-lista"
            aparencia="nova-faixas:problemas"
            colunas={COLUNAS_PROBLEMA}
            itens={lista}
            colunaDe={(p) => p.estado}
            idDe={(p) => p.id}
            nomeDe={(p) => p.titulo}
            extra={novoProblema}
            aoCriar={gere ? (_coluna, nome) => void criarCard(nome) : undefined}
            aoRenomear={gere ? (p, nome) => void renomearProblema(p, nome) : undefined}
            aoAbrir={(p) => router.push(`/problemas/${p.id}`)}
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
                  <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>
                    {ESTADO_PROBLEMA[p.estado].label}
                  </Badge>
                </div>
                {p.contorno && <p className="text-sm text-muted">Contorno: {p.contorno}</p>}
                <Link href={`/problemas/${p.id}`} className="text-xs font-medium text-primary hover:underline">
                  Editar
                </Link>
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

      {gere && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo problema</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={criar} className="mt-3 grid gap-3">
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
                rows={3}
              />
              <UnidadeDonaSelect
                id="problema-unidade"
                permission="problemas:manage"
                label="Unidade"
              />
              <div>
                <Button type="submit" loading={pending}>
                  Registrar problema
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Modal para preenchimento de contorno ou causa raiz ao mover */}
      {modalDados && (
        <Dialog
          open={true}
          onClose={() => setModalDados(null)}
          title={modalDados.titulo}
          description={`Para avançar o problema #${modalDados.problema.numero} para ${ESTADO_PROBLEMA[modalDados.para].label.toLowerCase()}, preencha a informação obrigatória:`}
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
              rows={3}
              value={textoModal}
              onChange={(e) => setTextoModal(e.target.value)}
              placeholder="Descreva detalhadamente..."
            />
          </form>
        </Dialog>
      )}
    </div>
  );
}
