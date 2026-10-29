"use client";

import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";

/**
 * Sincronizar com o GLPI até a virada (ADR 081): ligada, o servidor pede
 * a extração a cada 15 minutos e carrega logo depois; os chamados que
 * vieram de lá ficam só para leitura aqui. Na virada, desliga-se.
 */
export function Sincronizacao({
  ligada,
  extraidoEm,
  ocupado,
  onMudar,
}: {
  ligada: boolean;
  extraidoEm?: string;
  ocupado: boolean;
  onMudar: (ligar: boolean) => Promise<unknown> | void;
}) {
  return (
    <section aria-labelledby="sincronizacao-titulo" className="grid gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <h3 id="sincronizacao-titulo" className="font-semibold">
          Sincronizar até a virada
        </h3>
        <Badge tone={ligada ? "success" : "neutral"}>{ligada ? "Ligada" : "Desligada"}</Badge>
      </div>
      <p className="text-sm text-muted">
        Enquanto o GLPI segue em uso, a cada 15 minutos o conector extrai o que mudou e a carga traz
        os chamados novos e as mudanças dos já importados (estado, técnico, solução, respostas e
        avaliação). Ligada, os chamados que vieram do GLPI ficam só para consulta aqui. Na virada,
        desligue: eles passam a ser atendidos aqui.
      </p>
      {extraidoEm && (
        <p className="text-sm text-muted">
          Última extração: {new Date(extraidoEm).toLocaleString("pt-BR")}
        </p>
      )}
      <div>
        {ligada ? (
          <ConfirmButton
            variant="danger"
            size="md"
            title="Desligar a sincronização?"
            description="Faça isso na virada: os chamados que vieram do GLPI deixam de mudar por lá e passam a ser atendidos aqui."
            confirmLabel="Desligar"
            onConfirm={() => onMudar(false)}
          >
            Desligar a sincronização
          </ConfirmButton>
        ) : (
          <Button type="button" disabled={ocupado} onClick={() => void onMudar(true)}>
            Ligar a sincronização
          </Button>
        )}
      </div>
    </section>
  );
}
