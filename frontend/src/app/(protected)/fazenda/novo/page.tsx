"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { buttonClass } from "@/components/ui/Button";

import { Protocolar } from "../Protocolar";
import { CatalogoServicos, DetalheServico, programaValido, servicoDe } from "../Servicos";

/** Catálogo, detalhe e formulário, na ordem do protocolo de referência. */
export default function NovoProcessoPage() {
  const params = useSearchParams();
  const programa = programaValido(params.get("programa"));
  const detalhe = params.get("etapa") === "detalhe";
  const servico = programa ? servicoDe(programa) : null;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Secretaria Municipal de Fazenda"
        title={servico ? servico.titulo : "Serviços"}
        description={
          servico
            ? servico.descricao
            : "Busque o serviço e solicite. O formulário abre depois da escolha."
        }
        actions={
          <Link
            href={servico ? "/fazenda/novo" : "/fazenda"}
            className={buttonClass("secondary", "sm")}
          >
            Voltar
          </Link>
        }
      />
      {programa && detalhe && <DetalheServico programa={programa} />}
      {programa && !detalhe && <Protocolar programa={programa} />}
      {!programa && <CatalogoServicos />}
    </div>
  );
}
