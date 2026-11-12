"use client";

import { X } from "lucide-react";
import { type FormEvent, useEffect, useId, useState } from "react";

import { PessoaComAvatar, PessoasComAvatar } from "@/components/plataforma/AvatarPessoa";
import { PessoasDaTarefa } from "@/components/projetos/PessoasDaTarefa";
import { useAction, fmtDateTime } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ModalShell } from "@/components/ui/ModalShell";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";
import type { UploadTicket } from "@/lib/plataforma/types";
import { ESTADO_TAREFA, pessoasDaTarefa, tarefaInclui, type AtualizacaoTarefa, type EstadoTarefa, type PessoaProjeto, type Tarefa } from "@/lib/projetos/projetos";

export interface AnexoTarefa {
  id: string;
  tarefa_id: string;
  nome: string;
  content_type: string;
  tamanho: number;
  autor_nome: string;
  created_at: string;
}

export interface MovimentoTarefa {
  id: string;
  estado_antes: string;
  estado_depois: string;
  responsavel_antes_nome: string;
  responsavel_depois_nome: string;
  autor_nome: string;
  created_at: string;
}

function rotuloEstado(estado: string) {
  return ESTADO_TAREFA[estado as EstadoTarefa]?.label ?? estado;
}

function nomesDe(texto: string) {
  return texto.split("\n").map((nome) => nome.trim()).filter(Boolean);
}

function textoPessoas(antes: string, depois: string) {
  const saiuDe = nomesDe(antes);
  const entrouEm = nomesDe(depois);
  if (saiuDe.join("\n") === entrouEm.join("\n")) return "";
  if (saiuDe.length <= 1 && entrouEm.length <= 1) {
    if (!antes) return `definiu o responsável ${depois}`;
    if (!depois) return "tirou o responsável";
    return `trocou o responsável de ${antes} para ${depois}`;
  }
  const saiu = saiuDe.filter((nome) => !entrouEm.includes(nome));
  const entrou = entrouEm.filter((nome) => !saiuDe.includes(nome));
  const partes: string[] = [];
  if (entrou.length) partes.push(`adicionou ${entrou.join(", ")}`);
  if (saiu.length) partes.push(`tirou ${saiu.join(", ")}`);
  return partes.join(" e ");
}

function textoMovimento(m: MovimentoTarefa) {
  const partes: string[] = [];
  if (m.estado_antes !== m.estado_depois) {
    partes.push(`moveu de ${rotuloEstado(m.estado_antes)} para ${rotuloEstado(m.estado_depois)}`);
  }
  const pessoas = textoPessoas(m.responsavel_antes_nome, m.responsavel_depois_nome);
  if (pessoas) partes.push(pessoas);
  return partes.join(" e ");
}

const TIPOS: Record<string, string> = {
  png: "image/png",
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  webp: "image/webp",
  gif: "image/gif",
  pdf: "application/pdf",
  txt: "text/plain",
  csv: "text/csv",
  md: "text/markdown",
  doc: "application/msword",
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  xls: "application/vnd.ms-excel",
  xlsx: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  ppt: "application/vnd.ms-powerpoint",
  pptx: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  zip: "application/zip",
};

function tipoDoArquivo(file: File): string {
  if (file.type) return file.type;
  const ext = file.name.split(".").pop()?.toLowerCase() ?? "";
  return TIPOS[ext] ?? mimeOf(file);
}

export function TarefaPainel({
  tarefa,
  tarefas = [],
  projetoId,
  mutavel,
  podeComentar = false,
  onExcluir,
  onMover,
  onAbrir,
  onClose,
  onSaved,
}: {
  tarefa: Tarefa | null;
  tarefas?: Tarefa[];
  projetoId: string;
  mutavel: boolean;
  podeComentar?: boolean;
  onExcluir?: (tarefa: Tarefa) => void;
  onMover?: (tarefa: Tarefa, estado: EstadoTarefa) => void;
  onAbrir?: (tarefa: Tarefa) => void;
  onClose: () => void;
  onSaved: (mensagem?: string) => void;
}) {
  const { run, pending } = useAction();
  const [relato, setRelato] = useState("");
  const [resp, setResp] = useState<PessoaProjeto[]>([]);
  const [arquivo, setArquivo] = useState<File | null>(null);
  const [confirmacao, setConfirmacao] = useState("");
  const [nova, setNova] = useState("");
  const [atualizacao, setAtualizacao] = useState("");

  useEffect(() => {
    if (!tarefa) return;
    setRelato(tarefa.relato ?? "");
    setResp(pessoasDaTarefa(tarefa));
    setArquivo(null);
    setConfirmacao("");
    setNova("");
    setAtualizacao("");
  }, [tarefa]);

  const anexos = useApiQuery<AnexoTarefa[]>(
    tarefa ? `v1/projetos/${projetoId}/tarefas/${tarefa.id}/anexos` : null,
  );
  const movimentos = useApiQuery<MovimentoTarefa[]>(
    tarefa ? `v1/projetos/${projetoId}/tarefas/${tarefa.id}/movimentos` : null,
  );
  const atualizacoes = useApiQuery<AtualizacaoTarefa[]>(
    tarefa ? `v1/projetos/${projetoId}/tarefas/${tarefa.id}/atualizacoes` : null,
  );

  async function salvar() {
    if (!tarefa) return;
    const nomes = resp.map((pessoa) => pessoa.nome).filter(Boolean);
    const mensagem = nomes.length
      ? `Responsável da tarefa salvo: ${nomes.join(", ")}.`
      : "Tarefa atualizada.";
    const ok = await run(
      () =>
        apiClient.put(`v1/projetos/${projetoId}/tarefas/${tarefa.id}`, {
          titulo: tarefa.titulo,
          descricao: tarefa.descricao,
          relato,
          pessoas: resp,
          estado: tarefa.estado,
          ordem: tarefa.ordem,
          data_limite: tarefa.data_limite ?? null,
          predecessora_id: tarefa.predecessora_id ?? null,
        }),
      mensagem,
    );
    if (ok) {
      onSaved(mensagem);
      onClose();
    }
  }

  async function criarSubtarefa(e: FormEvent) {
    e.preventDefault();
    if (!tarefa || tarefa.pai_id) return;
    const titulo = nova.trim();
    if (!titulo) return;
    const filhas = tarefas.filter((item) => item.pai_id === tarefa.id);
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${projetoId}/tarefas`, {
          titulo,
          pai_id: tarefa.id,
          estado: "a_fazer",
          ordem: filhas.length + 1,
        }),
      "Subtarefa adicionada",
    );
    if (ok) {
      setNova("");
      onSaved();
    }
  }

  async function publicar(e: FormEvent) {
    e.preventDefault();
    if (!tarefa || !atualizacao.trim()) return;
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${projetoId}/tarefas/${tarefa.id}/atualizacoes`, {
          texto: atualizacao.trim(),
        }),
      "Atualização publicada",
    );
    if (ok) {
      setAtualizacao("");
      void atualizacoes.mutate();
    }
  }

  async function enviar(e: FormEvent) {
    e.preventDefault();
    if (!tarefa || !arquivo) return;
    const file = arquivo;
    const ok = await run(async () => {
      const { data: ticket } = await apiClient.post<UploadTicket>(
        `v1/projetos/${projetoId}/tarefas/${tarefa.id}/anexos/uploads`,
        { nome: file.name, content_type: tipoDoArquivo(file) },
      );
      await putToTicket(ticket, file);
      return apiClient.post(`v1/projetos/${projetoId}/tarefas/${tarefa.id}/anexos`, {
        nome: file.name,
        object_key: ticket.object_key,
      });
    }, "Arquivo anexado");
    if (ok) {
      setArquivo(null);
      void anexos.mutate();
    }
  }

  async function abrirArquivo(anexo: AnexoTarefa) {
    if (!tarefa) return;
    const ok = await run(async () => {
      const { data } = await apiClient.get<{ url: string }>(
        `v1/projetos/${projetoId}/tarefas/${tarefa.id}/anexos/${anexo.id}`,
      );
      if (data?.url) window.open(data.url, "_blank", "noopener,noreferrer");
    }, "");
    if (!ok) return;
  }

  const titleId = useId();
  const descId = useId();

  return (
    <ModalShell
      open={tarefa != null}
      onClose={onClose}
      variant="drawer"
      amplitude="ampla"
      labelledBy={titleId}
      describedBy={descId}
    >
      <div className="flex h-full min-h-0 flex-col">
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-surface-border px-6 py-5">
          <div className="min-w-0">
            <h2 id={titleId} className="truncate text-xl font-semibold">
              {tarefa?.titulo ?? "Tarefa"}
            </h2>
            <p id={descId} className="mt-1 text-sm text-muted">
              Quem faz, como foi feita e os arquivos desta tarefa.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Fechar tarefa"
            className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-muted hover:bg-black/5 hover:text-foreground dark:hover:bg-white/10"
          >
            <X aria-hidden className="h-5 w-5" />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
      {tarefa && (
        <div className="flex flex-col gap-4">
          {mutavel ? (
            <PessoasDaTarefa
              projetoId={projetoId}
              value={resp}
              onChange={setResp}
            />
          ) : (
            <PessoasComAvatar
              pessoas={pessoasDaTarefa(tarefa)}
              vazio="ninguém atribuído"
              className="text-sm text-muted"
            />
          )}
          {tarefa.pai_id && (
            <p className="text-xs text-muted">
              Subtarefa de {tarefas.find((item) => item.id === tarefa.pai_id)?.titulo ?? "outra tarefa"}
            </p>
          )}
          {!tarefa.pai_id && (
            <div className="flex flex-col gap-2">
              <p className="text-sm font-medium text-foreground">Subtarefas</p>
              {tarefas.filter((item) => item.pai_id === tarefa.id).length === 0 && (
                <p className="text-xs text-muted">Nenhuma subtarefa. Use quando esta tarefa crescer.</p>
              )}
              <ul className="flex flex-col gap-2">
                {tarefas
                  .filter((item) => item.pai_id === tarefa.id)
                  .map((filha) => (
                    <li key={filha.id} className="flex flex-wrap items-center gap-2">
                      <button
                        type="button"
                        className="text-left text-sm text-foreground hover:underline"
                        onClick={() => onAbrir?.(filha)}
                      >
                        {filha.titulo}
                      </button>
                      <PessoasComAvatar
                        pessoas={pessoasDaTarefa(filha)}
                        vazio="ninguém"
                        className="text-xs text-muted"
                      />
                      {mutavel && onMover ? (
                        <select
                          aria-label={`Estado da subtarefa ${filha.titulo}`}
                          value={filha.estado}
                          disabled={pending || !tarefaInclui(filha)}
                          onChange={(e) => onMover(filha, e.target.value as EstadoTarefa)}
                          className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground"
                        >
                          {(Object.keys(ESTADO_TAREFA) as EstadoTarefa[]).map((estado) => (
                            <option key={estado} value={estado}>
                              {ESTADO_TAREFA[estado].label}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <span className="text-xs text-muted">{rotuloEstado(filha.estado)}</span>
                      )}
                    </li>
                  ))}
              </ul>
              {mutavel && (
                <form onSubmit={(e) => void criarSubtarefa(e)} className="flex items-center gap-2">
                  <Input
                    id="nova-subtarefa"
                    value={nova}
                    onChange={(e) => setNova(e.target.value)}
                    placeholder="Nova subtarefa"
                    aria-label="Nova subtarefa"
                    maxLength={160}
                    className="h-8"
                  />
                  <Button type="submit" size="sm" variant="secondary" disabled={pending || !nova.trim()}>
                    Adicionar
                  </Button>
                </form>
              )}
            </div>
          )}
          <Textarea
            id="tarefa-relato"
            label="Como foi feita"
            value={relato}
            onChange={(e) => setRelato(e.target.value)}
            placeholder="O que foi feito, o que travou, o que ficou pendente."
            rows={6}
            maxLength={4000}
            readOnly={!mutavel}
          />
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-foreground">Atualizações</p>
            {(atualizacoes.data ?? []).length === 0 && (
              <p className="text-xs text-muted">Nenhuma atualização nesta tarefa.</p>
            )}
            <ul className="flex flex-col gap-3">
              {(atualizacoes.data ?? []).map((item) => (
                <li key={item.id} className="flex flex-col gap-1">
                  <PessoaComAvatar id={item.autor_id} nome={item.autor_nome} className="text-sm" />
                  <p className="whitespace-pre-wrap text-sm text-foreground">{item.texto}</p>
                  <span className="text-[11px] text-muted">{fmtDateTime(item.created_at)}</span>
                </li>
              ))}
            </ul>
            {podeComentar && (
              <form onSubmit={(e) => void publicar(e)} className="flex flex-col gap-2">
                <Textarea
                  id="tarefa-atualizacao"
                  label="Nova atualização"
                  value={atualizacao}
                  onChange={(e) => setAtualizacao(e.target.value)}
                  placeholder="O que mudou, o que foi combinado, o que falta."
                  rows={3}
                  maxLength={2000}
                />
                <Button type="submit" size="sm" variant="secondary" className="self-start" disabled={pending || !atualizacao.trim()}>
                  Publicar
                </Button>
              </form>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-foreground">Arquivos e fotos</p>
            {(anexos.data ?? []).length === 0 && (
              <p className="text-xs text-muted">Nenhum arquivo nesta tarefa.</p>
            )}
            <ul className="flex flex-col gap-1">
              {(anexos.data ?? []).map((anexo) => (
                <li key={anexo.id}>
                  <button
                    type="button"
                    className="text-left text-sm text-primary hover:underline"
                    onClick={() => void abrirArquivo(anexo)}
                  >
                    {anexo.nome}
                  </button>
                  {anexo.autor_nome && (
                    <span className="ml-2 text-[11px] text-muted">{anexo.autor_nome}</span>
                  )}
                </li>
              ))}
            </ul>
            {mutavel && (
              <form onSubmit={enviar} className="flex flex-wrap items-center gap-2">
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif,.pdf,.txt,.csv,.md,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip"
                  onChange={(e) => setArquivo(e.target.files?.[0] ?? null)}
                  aria-label="Arquivo ou foto da tarefa"
                />
                <Button type="submit" size="sm" variant="secondary" disabled={pending || !arquivo}>
                  Enviar
                </Button>
              </form>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-foreground">Registros</p>
            {(movimentos.data ?? []).length === 0 && (
              <p className="text-xs text-muted">Nenhuma movimentação ainda.</p>
            )}
            <ul className="flex flex-col gap-2">
              {(movimentos.data ?? []).map((m) => (
                <li key={m.id} className="text-sm text-foreground">
                  <span className="text-muted">{fmtDateTime(m.created_at)}</span>
                  {m.autor_nome ? ` · ${m.autor_nome}` : ""} {textoMovimento(m)}
                </li>
              ))}
            </ul>
          </div>
          {mutavel && onExcluir && (
            <div className="flex flex-col gap-2 border-t border-surface-border pt-4">
              <p className="text-sm font-medium text-foreground">Apagar tarefa</p>
              <p className="text-xs text-muted">
                Digite o título exato da tarefa para confirmar. Essa ação não tem volta.
                {tarefas.some((item) => item.pai_id === tarefa.id)
                  ? " As subtarefas também são apagadas."
                  : ""}
              </p>
              <Input
                id="tarefa-confirmar-exclusao"
                label="Título para confirmar"
                value={confirmacao}
                onChange={(e) => setConfirmacao(e.target.value)}
                autoComplete="off"
              />
              <Button
                type="button"
                variant="danger"
                size="sm"
                className="self-start"
                disabled={pending || confirmacao.trim() !== tarefa.titulo.trim()}
                onClick={() => onExcluir(tarefa)}
              >
                Apagar tarefa
              </Button>
            </div>
          )}
        </div>
      )}
        </div>
        {tarefa && mutavel && (
          <div className="flex shrink-0 items-center gap-2 border-t border-surface-border px-6 py-4">
            <Button type="button" disabled={pending} onClick={() => void salvar()}>
              Salvar
            </Button>
          </div>
        )}
      </div>
    </ModalShell>
  );
}
