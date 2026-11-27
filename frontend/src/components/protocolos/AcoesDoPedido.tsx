"use client";

import { useState, type FormEvent } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Dialog } from "@/components/ui/Dialog";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree } from "@/lib/plataforma/types";

import { CampoAnexos, type Anexo } from "./Anexos";
import { FINAIS } from "./estilo";
import { BOTAO_PERIGO, BOTAO_PRIMARIO, BOTAO_SECUNDARIO, CAMPO, ROTULO } from "./ui";

export type TipoAcao =
  | "complemento"
  | "resposta"
  | "nota"
  | "concluir"
  | "cancelar"
  | "aceitar"
  | "transferir"
  | "encaminhar"
  | "recusar_encaminhamento"
  | "devolver"
  | "editar"
  | "avancar";

export interface AcaoAberta {
  tipo: TipoAcao;
  pedidoId: string;
  numero: string;
  /** Unidade atual: o encaminhamento não oferece a própria. */
  unidadeId?: string;
  /** Valores atuais para a edição rápida. */
  prioridade?: string;
  prazo?: string;
  /** Processo SEI atual, para a edição e a conclusão. */
  processoSei?: string;
}

/** O que a mesa precisa saber do pedido para escolher as ações. */
export interface EstadoDoPedido {
  status: string;
  decisao?: string;
  responsavel_id?: string;
  exige_aceite?: boolean;
  aprovador_id?: string;
  encaminhamento?: { unidade_id: string; unidade_nome?: string } | null;
}

/** Aprova a conclusão o aprovador indicado no serviço; sem indicação, quem gere a unidade. */
export function podeAprovar(p: EstadoDoPedido, meuId: string | undefined, gere: boolean): boolean {
  return p.aprovador_id ? Boolean(meuId) && p.aprovador_id === meuId : gere;
}

/** Situação do pedido na mesa: encerrado, encaminhado, aguardando aprovação ou travado sem dono. */
export function situacaoDoPedido(p: EstadoDoPedido) {
  const semDono = !p.responsavel_id;
  return {
    final: FINAIS.has(p.status),
    encaminhado: Boolean(p.encaminhamento),
    aguardando: p.status === "pending_approval" && Boolean(p.decisao),
    semDono,
    /** O serviço exige aceite e ninguém aceitou: só aceitar, editar, encaminhar e cancelar. */
    travado: Boolean(p.exige_aceite) && semDono,
  };
}

export const MOTIVO_TRAVADO = "Aceite o protocolo para liberar esta ação";

const TITULO: Record<TipoAcao, string> = {
  complemento: "Solicitar Complementação",
  resposta: "Registrar Resposta da Complementação",
  nota: "Adicionar Nota Interna",
  concluir: "Concluir Protocolo",
  cancelar: "Cancelar Protocolo",
  aceitar: "Aceitar Responsabilidade",
  transferir: "Transferir Responsabilidade",
  encaminhar: "Encaminhar para Outra Unidade",
  recusar_encaminhamento: "Recusar Encaminhamento",
  devolver: "Devolver a Conclusão",
  editar: "Editar Protocolo",
  avancar: "Encaminhar para a Próxima Etapa",
};

const CONFIRMAR: Partial<Record<TipoAcao, string>> = {
  cancelar: "Cancelar protocolo",
  aceitar: "Aceitar responsabilidade",
  transferir: "Transferir",
  encaminhar: "Encaminhar",
  recusar_encaminhamento: "Recusar encaminhamento",
  devolver: "Devolver",
  editar: "Salvar",
  avancar: "Avançar",
};

const ROTULO_TEXTO: Partial<Record<TipoAcao, string>> = {
  concluir: "Justificativa *",
  nota: "Nota (não aparece para o cidadão) *",
  resposta: "Resposta recebida *",
  complemento: "O que o cidadão precisa enviar *",
  encaminhar: "Justificativa *",
  recusar_encaminhamento: "Motivo da recusa *",
  devolver: "O que precisa ser revisto *",
  editar: "Justificativa da edição *",
};

interface Pessoa {
  id: string;
  nome: string;
}

/** Modal da ação escolhida no menu do pedido. Chama onFeito depois de gravar. */
export function ModalDeAcao({
  acao,
  onFechar,
  onFeito,
}: {
  acao: AcaoAberta | null;
  onFechar: () => void;
  onFeito: () => void;
}) {
  const { can } = usePlataforma();
  const { run, pending } = useAction();
  const [texto, setTexto] = useState("");
  const [decisao, setDecisao] = useState("");
  const [anexos, setAnexos] = useState<Anexo[]>([]);
  const [enviandoAnexo, setEnviandoAnexo] = useState(false);
  const tipo = acao?.tipo;
  const levaAnexos =
    tipo === "complemento" || tipo === "nota" || tipo === "concluir" || tipo === "avancar";
  const levaSei = tipo === "concluir" || tipo === "editar";
  const escolhePessoa = tipo === "transferir" || (tipo === "aceitar" && can("protocolos:gerir"));
  const pessoas = useApiQuery<Pessoa[]>(
    acao && escolhePessoa ? `v1/protocolos/pedidos/${acao.pedidoId}/pessoas` : null,
  );
  const arvore = useApiQuery<OrgTree[]>(tipo === "encaminhar" ? "v1/iam/org-tree" : null);
  const unidades = (arvore.data ?? [])
    .flatMap((entidade) => entidade.unidades)
    .filter((u) => u.ativo && u.id !== acao?.unidadeId)
    .map((u) => ({ id: u.id, nome: u.sigla ? `${u.sigla} · ${u.nome}` : u.nome }))
    .sort((a, b) => a.nome.localeCompare(b.nome, "pt-BR"));

  function fechar() {
    setTexto("");
    setDecisao("");
    setAnexos([]);
    onFechar();
  }

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!acao) return;
    const dados = new FormData(e.currentTarget);
    const campo = (nome: string) => String(dados.get(nome) ?? "");
    const base = `v1/protocolos/pedidos/${acao.pedidoId}`;
    // Só vai o que mudou: sem anexo, sem prazo e com o mesmo SEI, o corpo fica o de sempre.
    const extras: Record<string, unknown> = {};
    if (levaAnexos && anexos.length > 0) extras.anexos = anexos.map((a) => a.chave);
    if (levaSei && campo("processo_sei").trim() !== (acao.processoSei ?? "")) {
      extras.processo_sei = campo("processo_sei").trim();
    }
    if (tipo === "complemento" && Number(campo("prazo_dias") || 0) > 0) {
      extras.prazo_dias = Number(campo("prazo_dias"));
    }
    const pedidos: Record<TipoAcao, () => [string, unknown, string]> = {
      complemento: () => [
        `${base}/complemento`,
        { mensagem: texto, ...extras },
        "Complementação solicitada",
      ],
      resposta: () => [`${base}/resposta`, { mensagem: texto }, "Resposta registrada"],
      nota: () => [`${base}/nota`, { mensagem: texto, ...extras }, "Nota adicionada"],
      concluir: () => [
        `${base}/concluir`,
        { decisao, justificativa: texto, ...extras },
        "Protocolo concluído",
      ],
      cancelar: () => [`${base}/cancelar`, {}, "Protocolo cancelado"],
      aceitar: () => [
        `${base}/aceitar`,
        campo("responsavel") ? { responsavel_id: campo("responsavel") } : {},
        "Responsabilidade aceita",
      ],
      transferir: () => [
        `${base}/transferir`,
        { responsavel_id: campo("responsavel") },
        "Responsabilidade transferida",
      ],
      encaminhar: () => [
        `${base}/encaminhar`,
        {
          unidade_id: campo("unidade"),
          justificativa: texto,
          prazo_dias: Number(campo("prazo_dias") || 0),
        },
        "Encaminhamento enviado: aguarda o aceite da unidade de destino",
      ],
      recusar_encaminhamento: () => [
        `${base}/encaminhamento/recusar`,
        { mensagem: texto },
        "Encaminhamento recusado",
      ],
      devolver: () => [`${base}/devolver`, { mensagem: texto }, "Conclusão devolvida"],
      editar: () => [
        `${base}/editar`,
        {
          justificativa: texto,
          prioridade: campo("prioridade"),
          prazo: campo("prazo"),
          ...extras,
        },
        "Protocolo atualizado",
      ],
      avancar: () => [`${base}/avancar`, { ...extras }, "Protocolo avançou de etapa"],
    };
    const [caminho, corpo, aviso] = pedidos[acao.tipo]();
    const ok = await run(() => apiClient.post(caminho, corpo), aviso);
    if (ok) {
      fechar();
      onFeito();
    }
  }

  return (
    <Dialog
      open={acao !== null}
      onClose={fechar}
      title={tipo ? TITULO[tipo] : ""}
      description={acao ? `Protocolo #${acao.numero}` : undefined}
    >
      <form onSubmit={(e) => void enviar(e)} className="mt-4 flex flex-col gap-4">
        {tipo === "cancelar" && (
          <p className="text-sm text-gray-600 dark:text-slate-300">
            O protocolo é encerrado como cancelado e o cidadão vê a mudança na consulta. Esta ação
            não pode ser desfeita.
          </p>
        )}
        {tipo === "aceitar" && !escolhePessoa && (
          <p className="text-sm text-gray-600 dark:text-slate-300">
            Você assumirá diretamente a responsabilidade por este protocolo. O cidadão é avisado de
            que o pedido está em análise.
          </p>
        )}
        {escolhePessoa && (
          <div>
            <label htmlFor="acao-responsavel" className={ROTULO}>
              {tipo === "aceitar" ? "Quem assume" : "Novo responsável *"}
            </label>
            <select
              id="acao-responsavel"
              name="responsavel"
              required={tipo === "transferir"}
              defaultValue=""
              className={CAMPO}
            >
              <option value="">{tipo === "aceitar" ? "Eu mesmo" : "Selecione…"}</option>
              {(pessoas.data ?? []).map((pessoa) => (
                <option key={pessoa.id} value={pessoa.id}>
                  {pessoa.nome}
                </option>
              ))}
            </select>
            <p className="mt-1 text-xs text-gray-500 dark:text-slate-400">
              Pessoas com acesso à unidade do protocolo.
            </p>
          </div>
        )}
        {tipo === "encaminhar" && (
          <>
            <p className="text-sm text-gray-600 dark:text-slate-300">
              O protocolo fica parado até a unidade de destino aceitar. Aceito, ele muda de unidade
              e espera um novo responsável.
            </p>
            <div>
              <label htmlFor="acao-unidade" className={ROTULO}>
                Unidade de destino *
              </label>
              <select id="acao-unidade" name="unidade" required defaultValue="" className={CAMPO}>
                <option value="">Selecione…</option>
                {unidades.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.nome}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="acao-prazo-dias" className={ROTULO}>
                Prazo no destino (dias úteis)
              </label>
              <input
                id="acao-prazo-dias"
                name="prazo_dias"
                type="number"
                min={0}
                max={365}
                defaultValue={0}
                className={CAMPO}
              />
            </div>
          </>
        )}
        {tipo === "editar" && (
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="acao-prioridade" className={ROTULO}>
                Prioridade
              </label>
              <select
                id="acao-prioridade"
                name="prioridade"
                defaultValue={acao?.prioridade ?? "normal"}
                className={CAMPO}
              >
                <option value="baixa">Baixa</option>
                <option value="normal">Normal</option>
                <option value="alta">Alta</option>
                <option value="urgente">Urgente</option>
              </select>
            </div>
            <div>
              <label htmlFor="acao-prazo" className={ROTULO}>
                Prazo
              </label>
              <input
                id="acao-prazo"
                name="prazo"
                type="date"
                defaultValue={acao?.prazo ?? ""}
                className={CAMPO}
              />
            </div>
          </div>
        )}
        {tipo === "avancar" && (
          <p className="text-sm text-gray-600 dark:text-slate-300">
            O protocolo segue para a próxima etapa do fluxo. Junte, se quiser, o parecer ou o
            documento desta etapa.
          </p>
        )}
        {tipo === "complemento" && (
          <div>
            <label htmlFor="acao-prazo-complemento" className={ROTULO}>
              Prazo para o cidadão responder (dias úteis)
            </label>
            <input
              id="acao-prazo-complemento"
              name="prazo_dias"
              type="number"
              min={0}
              max={90}
              defaultValue={0}
              className={CAMPO}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-slate-400">
              Zero deixa sem prazo. Vencido sem resposta, o protocolo é cancelado e o cidadão é
              lembrado na véspera.
            </p>
          </div>
        )}
        {levaSei && (
          <div>
            <label htmlFor="acao-sei" className={ROTULO}>
              Processo SEI
            </label>
            <input
              id="acao-sei"
              name="processo_sei"
              defaultValue={acao?.processoSei ?? ""}
              placeholder="00000.000000/2026-00"
              maxLength={40}
              className={CAMPO}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-slate-400">
              {tipo === "concluir"
                ? "Se a decisão saiu no SEI, informe o processo e cite o documento na justificativa."
                : "O processo que corre em paralelo no SEI. Vazio desfaz o vínculo."}
            </p>
          </div>
        )}
        {tipo === "concluir" && (
          <div>
            <label htmlFor="acao-decisao" className={ROTULO}>
              Decisão *
            </label>
            <select
              id="acao-decisao"
              required
              value={decisao}
              onChange={(e) => setDecisao(e.target.value)}
              className={CAMPO}
            >
              <option value="">Selecione…</option>
              <option value="favoravel">Favorável (deferido)</option>
              <option value="desfavoravel">Desfavorável (indeferido)</option>
            </select>
          </div>
        )}
        {tipo && ROTULO_TEXTO[tipo] && (
          <div>
            <label htmlFor="acao-texto" className={ROTULO}>
              {ROTULO_TEXTO[tipo]}
            </label>
            <textarea
              id="acao-texto"
              required
              rows={4}
              maxLength={tipo === "concluir" || tipo === "editar" ? 4000 : 2000}
              value={texto}
              onChange={(e) => setTexto(e.target.value)}
              className={`${CAMPO} min-h-[100px] resize-y`}
            />
          </div>
        )}
        {acao && levaAnexos && (
          <CampoAnexos
            pedidoId={acao.pedidoId}
            valor={anexos}
            onChange={setAnexos}
            onEnviando={setEnviandoAnexo}
          />
        )}
        <div className="flex justify-end gap-3">
          <button type="button" onClick={fechar} className={BOTAO_SECUNDARIO}>
            Cancelar
          </button>
          <button
            type="submit"
            disabled={pending || enviandoAnexo}
            className={
              tipo === "cancelar" || tipo === "recusar_encaminhamento"
                ? BOTAO_PERIGO
                : BOTAO_PRIMARIO
            }
          >
            {(tipo && CONFIRMAR[tipo]) || "Confirmar"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

/** Data AAAA-MM-DD no fuso de Cuiabá, para o campo de prazo. */
export function diaDoPrazo(iso?: string): string {
  if (!iso) return "";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleDateString("en-CA", { timeZone: "America/Cuiaba" });
}
