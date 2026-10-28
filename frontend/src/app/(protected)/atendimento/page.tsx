"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import {
  CatalogoEmBlocos,
  CentralInicio,
  MeusChamados,
  NovoPedido,
  PORTAS,
  type Porta,
} from "@/components/atendimento/central/Central";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { buttonClass } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/**
 * Central de atendimento (ADR 078), no modelo da central do GLPI: blocos
 * para pedir, relatar um problema, acompanhar e ler a base de
 * conhecimento; o catálogo também em blocos. A visão vem da URL, para o
 * voltar do navegador funcionar. A palavra "incidente" não aparece aqui.
 */
export default function AtendimentoPortalPage() {
  const { can, me } = usePlataforma();
  const params = useSearchParams();

  if (!can("atendimento:abrir")) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Central de atendimento" />
        <EmptyState
          title="Seu perfil não abre chamados"
          description="Peça à gestão da sua unidade o acesso para abrir e acompanhar pedidos de atendimento."
          action={
            can("atendimento:atender") ? (
              <Link href="/atendimento/fila" className={buttonClass("secondary")}>
                Ir para a fila da equipe
              </Link>
            ) : undefined
          }
        />
      </div>
    );
  }

  const ver = params.get("ver");
  const oferta = params.get("oferta");
  const primeiroNome = me?.name?.split(" ")[0];
  let conteudo;
  if (oferta) conteudo = <NovoPedido ofertaId={oferta} />;
  else if (ver === "chamados") conteudo = <MeusChamados />;
  else if (ver && ver in PORTAS)
    conteudo = (
      <CatalogoEmBlocos porta={ver as Porta} categoria={params.get("categoria") ?? undefined} />
    );
  else conteudo = <CentralInicio />;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Central de atendimento"
        description={
          primeiroNome
            ? `Olá, ${primeiroNome}. Escolha o que precisa: a equipe responsável responde por aqui, e você recebe o aviso no sino.`
            : "Escolha o que precisa: a equipe responsável responde por aqui, e você recebe o aviso no sino."
        }
      />
      {conteudo}
    </div>
  );
}
