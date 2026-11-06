"use client";

import { type FormEvent, useEffect, useState } from "react";

import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { ResponsavelDaTarefa } from "@/components/projetos/ResponsavelDaTarefa";
import { useAction, fmtDateTime } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";
import type { UploadTicket } from "@/lib/plataforma/types";
import { ESTADO_TAREFA, type EstadoTarefa, type Tarefa } from "@/lib/projetos/projetos";

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

function textoMovimento(m: MovimentoTarefa) {
  const partes: string[] = [];
  if (m.estado_antes !== m.estado_depois) {
    partes.push(`moveu de ${rotuloEstado(m.estado_antes)} para ${rotuloEstado(m.estado_depois)}`);
  }
  if (m.responsavel_antes_nome !== m.responsavel_depois_nome) {
    if (!m.responsavel_antes_nome) partes.push(`definiu o responsável ${m.responsavel_depois_nome}`);
    else if (!m.responsavel_depois_nome) partes.push("tirou o responsável");
    else partes.push(`trocou o responsável de ${m.responsavel_antes_nome} para ${m.responsavel_depois_nome}`);
  }
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
  onExcluir?: (tarefa: Tarefa) => void;
  onMover?: (tarefa: Tarefa, estado: EstadoTarefa) => void;
  onAbrir?: (tarefa: Tarefa) => void;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { run, pending } = useAction();
  const [relato, setRelato] = useState("");
  const [resp, setResp] = useState({ id: "", nome: "" });
  const [arquivo, setArquivo] = useState<File | null>(null);
  const [confirmacao, setConfirmacao] = useState("");
  const [nova, setNova] = useState("");

  useEffect(() => {
    if (!tarefa) return;
    setRelato(tarefa.relato ?? "");
    setResp({ id: tarefa.responsavel_id ?? "", nome: tarefa.responsavel_nome ?? "" });
    setArquivo(null);
    setConfirmacao("");
    setNova("");
  }, [tarefa]);

  const anexos = useApiQuery<AnexoTarefa[]>(
    tarefa ? `v1/projetos/${projetoId}/tarefas/${tarefa.id}/anexos` : null,
  );
  const movimentos = useApiQuery<MovimentoTarefa[]>(
    tarefa ? `v1/projetos/${projetoId}/tarefas/${tarefa.id}/movimentos` : null,
  );

  async function salvar() {
    if (!tarefa) return;
    const ok = await run(
      () =>
        apiClient.put(`v1/projetos/${projetoId}/tarefas/${tarefa.id}`, {
          titulo: tarefa.titulo,
          descricao: tarefa.descricao,
          relato,
          responsavel_id: resp.id || undefined,
          responsavel_nome: resp.id ? resp.nome : "",
          estado: tarefa.estado,
          ordem: tarefa.ordem,
          data_limite: tarefa.data_limite ?? null,
          predecessora_id: tarefa.predecessora_id ?? null,
        }),
      "Tarefa atualizada",
    );
    if (ok) {
      onSaved();
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

  return (
    <Dialog
      open={tarefa != null}
      size="lg"
      onClose={onClose}
      title={tarefa?.titulo ?? "Tarefa"}
      description="Quem faz, como foi feita e os arquivos desta tarefa."
      footer={
        mutavel ? (
          <Button type="button" disabled={pending} onClick={() => void salvar()}>
            Salvar
          </Button>
        ) : null
      }
    >
      {tarefa && (
        <div className="mt-3 flex flex-col gap-4">
          {mutavel ? (
            <ResponsavelDaTarefa
              projetoId={projetoId}
              value={resp}
              onChange={setResp}
            />
          ) : (
            <PessoaComAvatar
              id={tarefa.responsavel_id}
              nome={tarefa.responsavel_nome}
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
                      <PessoaComAvatar
                        id={filha.responsavel_id}
                        nome={filha.responsavel_nome}
                        vazio="ninguém"
                        className="text-xs text-muted"
                      />
                      {mutavel && onMover ? (
                        <select
                          aria-label={`Estado da subtarefa ${filha.titulo}`}
                          value={filha.estado}
                          disabled={pending || !filha.responsavel_id}
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
    </Dialog>
  );
}
