/** O que a carga faz com o calendário e os grupos do GLPI (ADR 078). */
export interface Expediente {
  politica: "criada" | "existente" | "sem calendário";
  calendario?: string;
  metas?: string[];
  feriados: number;
  equipes: number;
  membros: number;
  membros_sem_par: number;
  grupos_sem_unidade?: string[];
  /** Grupos que nunca atenderam (do AD, em geral): não viram equipe. */
  grupos_sem_chamado?: number;
}

const POLITICA: Record<Expediente["politica"], string> = {
  criada: "Política de prazos institucional criada a partir do calendário",
  existente: "A política de prazos já configurada fica como está",
  "sem calendário": "Sem calendário nos SLAs do GLPI: nenhuma política criada",
};

/** Resumo legível da parte de prazos e equipes do relatório da carga. */
export function ResumoExpediente({
  expediente: e,
  gravou,
}: {
  expediente: Expediente;
  gravou: boolean;
}) {
  const verbo = gravou ? "" : " (simulação: nada foi gravado)";
  return (
    <section
      aria-label="Prazos e equipes"
      className="rounded-lg border border-surface-border p-3 text-sm"
    >
      <h4 className="font-semibold">Prazos e equipes{verbo}</h4>
      <ul className="mt-2 flex list-disc flex-col gap-1 pl-5">
        <li>
          {POLITICA[e.politica]}
          {e.calendario ? ` "${e.calendario}"` : ""}.
        </li>
        {(e.metas ?? []).length > 0 && (
          <li>
            Metas: {e.metas!.join("; ")}. A crítica, que o GLPI não tem, fica com metade da alta.
          </li>
        )}
        <li>{e.feriados} feriado(s) novo(s) do calendário.</li>
        <li>
          {e.equipes} equipe(s) dos grupos do GLPI, com {e.membros} pessoa(s)
          {e.membros_sem_par > 0 ? `; ${e.membros_sem_par} sem conta aqui ficaram de fora` : ""}.
        </li>
        {(e.grupos_sem_chamado ?? 0) > 0 && (
          <li>
            {e.grupos_sem_chamado} grupo(s) do GLPI que nunca atenderam um chamado ficaram de fora.
          </li>
        )}
        {(e.grupos_sem_unidade ?? []).length > 0 && (
          <li>Grupos sem unidade na tabela: {e.grupos_sem_unidade!.join(", ")}.</li>
        )}
      </ul>
    </section>
  );
}
