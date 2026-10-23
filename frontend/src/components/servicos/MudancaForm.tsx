"use client";

import { useState, type FormEvent } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { MatrizRiscoCalculadora } from "@/components/servicos/MatrizRiscoCalculadora";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import {
  RISCO,
  TIPO_MUDANCA,
  deCampoData,
  paraCampoData,
  type Mudanca,
  type Risco,
  type TipoMudanca,
} from "@/lib/servicos/servicos";

export interface MudancaCorpo {
  titulo: string;
  descricao: string;
  justificativa: string;
  tipo: string;
  risco: string;
  plano_reversao: string;
  inicio_planejado: string | null;
  fim_planejado: string | null;
  produto_id: string | null;
  unidade_id: string | null;
}

/** Campos da mudança. A janela e o plano de reversão podem ficar para
 * depois no rascunho; enviar para aprovação os exige. */
export function MudancaForm({
  inicial,
  rotulo,
  pending,
  onSalvar,
}: {
  inicial?: Mudanca;
  rotulo: string;
  pending: boolean;
  onSalvar: (corpo: MudancaCorpo) => void;
}) {
  const [risco, setRisco] = useState<Risco>(inicial?.risco ?? "medio");
  const [justificativaCalculo, setJustificativaCalculo] = useState<string>("");

  function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const texto = (nome: string) => String(fd.get(nome)).trim();

    const justDigitada = texto("justificativa");
    let justFinal = justDigitada;
    if (justificativaCalculo && !justDigitada.includes("[Cálculo de Risco CAB")) {
      justFinal = justDigitada ? `${justDigitada}\n\n${justificativaCalculo}` : justificativaCalculo;
    }

    onSalvar({
      titulo: texto("titulo"),
      descricao: texto("descricao"),
      justificativa: justFinal,
      tipo: texto("tipo"),
      risco: risco,
      plano_reversao: texto("plano_reversao"),
      inicio_planejado: deCampoData(texto("inicio_planejado")),
      fim_planejado: deCampoData(texto("fim_planejado")),
      produto_id: inicial?.produto_id ?? null,
      unidade_id: texto("unidade_id") || null,
    });
  }

  return (
    <form key={inicial?.id ?? "nova"} onSubmit={enviar} className="grid gap-3 sm:grid-cols-2">
      <div className="sm:col-span-2">
        <Input
          id="mudanca-titulo"
          name="titulo"
          label="Título *"
          required
          maxLength={160}
          defaultValue={inicial?.titulo}
          autoComplete="off"
        />
      </div>
      <Select
        id="mudanca-tipo"
        name="tipo"
        label="Tipo"
        defaultValue={inicial?.tipo ?? "normal"}
        options={(Object.keys(TIPO_MUDANCA) as TipoMudanca[]).map((t) => ({
          value: t,
          label: TIPO_MUDANCA[t],
        }))}
      />
      <Select
        id="mudanca-risco"
        name="risco"
        label="Risco"
        value={risco}
        onChange={(e) => setRisco(e.target.value as Risco)}
        options={(Object.keys(RISCO) as Risco[]).map((r) => ({ value: r, label: RISCO[r].label }))}
      />
      <div className="sm:col-span-2">
        <MatrizRiscoCalculadora
          riscoAtual={risco}
          onAplicarRisco={(novoRisco, resumo) => {
            setRisco(novoRisco);
            setJustificativaCalculo(resumo);
          }}
        />
      </div>
      <Input
        id="mudanca-inicio"
        name="inicio_planejado"
        type="datetime-local"
        label="Início planejado"
        defaultValue={paraCampoData(inicial?.inicio_planejado)}
      />
      <Input
        id="mudanca-fim"
        name="fim_planejado"
        type="datetime-local"
        label="Fim planejado"
        defaultValue={paraCampoData(inicial?.fim_planejado)}
      />
      <div className="sm:col-span-2">
        <Textarea
          id="mudanca-descricao"
          name="descricao"
          label="O que muda"
          maxLength={4000}
          rows={2}
          defaultValue={inicial?.descricao}
        />
      </div>
      <div className="sm:col-span-2">
        <Textarea
          id="mudanca-justificativa"
          name="justificativa"
          label="Por que mudar"
          maxLength={4000}
          rows={2}
          defaultValue={inicial?.justificativa}
        />
      </div>
      <div className="sm:col-span-2">
        <Textarea
          id="mudanca-reversao"
          name="plano_reversao"
          label="Plano de reversão (exigido na normal e na emergencial)"
          maxLength={4000}
          rows={2}
          defaultValue={inicial?.plano_reversao}
        />
      </div>
      <UnidadeDonaSelect
        id="mudanca-unidade"
        permission="mudancas:manage"
        label="Unidade"
        defaultValue={inicial?.unidade_id}
      />
      <div className="flex items-end sm:col-span-2">
        <Button type="submit" loading={pending}>
          {rotulo}
        </Button>
      </div>
    </form>
  );
}
