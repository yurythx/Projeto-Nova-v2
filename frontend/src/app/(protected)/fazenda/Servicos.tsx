"use client";

import Link from "next/link";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

import { NOMES, pecasLegais } from "./pecas";

/** Os dois serviços dos decretos. A lista é legal: a tela não cria um terceiro. */
export const SERVICOS = [
  {
    id: "regulariza",
    titulo: "Regulariza Rondonópolis",
    categoria: "Tributos",
    descricao: "Remissão, parcelamento e ITBI com alíquota reduzida.",
    prazo: "Adesão até 31/12/2026, 23h59, horário de Cuiabá",
    fila: "Fila de análise de 10 dias úteis, em ordem de protocolo. Rito prioritário a partir de 01/12/2026.",
    detalhe:
      "Remissão dos juros e anistia das multas de IPTU e taxas de 2021 a 2026. Parcelamento em até 12 parcelas, com mínimo de R$ 100. O estado permanece aguardando pagamento até a primeira baixa.",
  },
  {
    id: "avanca",
    titulo: "Avança Rondonópolis",
    categoria: "Tributos",
    descricao: "Reconhecimento do investimento, ITBI a 1% no imóvel matriz e isenção de IPTU.",
    prazo: "Requerimento até 31/12/2030, 23h59, horário de Cuiabá",
    fila: "Triagem em 5 dias úteis, parecer em 30 dias e decisão em 10 dias. A exigência aberta suspende esses prazos.",
    detalhe:
      "ITBI de 1% só no imóvel matriz, com as três condições do decreto. A isenção de IPTU vale por três exercícios, a partir do seguinte ao despacho.",
  },
] as const;

export type Programa = (typeof SERVICOS)[number]["id"];

export function programaValido(valor: string | null): Programa | null {
  return SERVICOS.some((s) => s.id === valor) ? (valor as Programa) : null;
}

export function servicoDe(programa: Programa) {
  return SERVICOS.find((item) => item.id === programa)!;
}

/** Cartão no desenho do protocolo de referência, com botão e selo deste sistema. */
export function CartaoServico({ servico }: { servico: (typeof SERVICOS)[number] }) {
  return (
    <li>
      <article className="flex h-full flex-col gap-3 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
        <div className="flex items-start justify-between gap-3">
          <h3 className="text-base font-semibold">{servico.titulo}</h3>
          <Badge tone="info">{servico.categoria}</Badge>
        </div>
        <p className="text-sm text-muted">{servico.descricao}</p>
        <p className="text-xs text-muted">{servico.prazo}</p>
        <div className="mt-auto flex flex-wrap gap-2">
          <Link
            href={`/fazenda/novo?programa=${servico.id}&etapa=detalhe`}
            className={buttonClass("secondary", "sm")}
            aria-label={`Ver detalhes de ${servico.titulo}`}
          >
            Ver detalhes
          </Link>
          <Link
            href={`/fazenda/novo?programa=${servico.id}`}
            className={buttonClass("primary", "sm")}
            aria-label={`Solicitar ${servico.titulo}`}
          >
            Solicitar
          </Link>
        </div>
      </article>
    </li>
  );
}

export function GradeServicos({ itens }: { itens: readonly (typeof SERVICOS)[number][] }) {
  if (itens.length === 0) {
    return <p className="text-sm text-muted">Nenhum serviço com esse nome.</p>;
  }
  return (
    <ul className="grid gap-4 md:grid-cols-2">
      {itens.map((servico) => (
        <CartaoServico key={servico.id} servico={servico} />
      ))}
    </ul>
  );
}

/** Catálogo: busca pelo nome e os cartões. A secretaria é uma só, a Fazenda. */
export function CatalogoServicos() {
  const [busca, setBusca] = useState("");
  const termo = busca.trim().toLowerCase();
  const itens = SERVICOS.filter(
    (servico) => !termo || servico.titulo.toLowerCase().includes(termo),
  );
  return (
    <div className="flex flex-col gap-4">
      <Input
        id="busca-servico"
        label="Buscar serviço"
        placeholder="Nome do serviço"
        value={busca}
        onChange={(e) => setBusca(e.target.value)}
        autoComplete="off"
      />
      <GradeServicos itens={itens} />
    </div>
  );
}

/** O que o serviço pede, antes do formulário. */
export function DetalheServico({ programa }: { programa: Programa }) {
  const servico = servicoDe(programa);
  const pecas = pecasLegais({
    programa,
    entrada: "govbr",
    itbi: false,
    moduloC: false,
    temCessao: false,
    temContrato: false,
  });
  return (
    <article className="flex max-w-3xl flex-col gap-4 rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
      <div className="flex flex-wrap items-center gap-2">
        <Badge tone="info">{servico.categoria}</Badge>
        <span className="text-sm text-muted">Secretaria Municipal de Fazenda</span>
      </div>
      <h2 className="text-xl font-semibold">{servico.titulo}</h2>
      <p className="text-sm">{servico.detalhe}</p>
      <p className="text-sm text-muted">{servico.prazo}</p>
      <p className="text-sm">{servico.fila}</p>
      <div>
        <h3 className="text-sm font-semibold">Peças de quem entra pelo gov.br, sem contrato</h3>
        <ul className="mt-2 list-disc pl-5 text-sm">
          {pecas.map((tipo) => (
            <li key={tipo}>{NOMES[tipo]}</li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-muted">
          Procuração, módulo C, cessão, contrato e ITBI alteram essa lista no formulário.
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        {programa === "regulariza" && (
          <Link href="/regulariza" className={buttonClass("secondary")}>
            Simular
          </Link>
        )}
        <Link href={`/fazenda/novo?programa=${programa}`} className={buttonClass("primary")}>
          Solicitar
        </Link>
      </div>
    </article>
  );
}
