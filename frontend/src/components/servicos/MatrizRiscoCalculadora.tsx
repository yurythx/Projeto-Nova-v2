"use client";

import { AlertTriangle, Calculator, Check, ShieldAlert, Sparkles } from "lucide-react";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { RISCO, type Risco } from "@/lib/servicos/servicos";

interface MatrizRiscoCalculadoraProps {
  riscoAtual: Risco;
  onAplicarRisco: (novoRisco: Risco, resumoCalculo: string, score: number) => void;
}

export function MatrizRiscoCalculadora({
  riscoAtual,
  onAplicarRisco,
}: MatrizRiscoCalculadoraProps) {
  const [impacto, setImpacto] = useState<1 | 2 | 3>(2);
  const [complexidade, setComplexidade] = useState<1 | 2 | 3>(2);
  const [rollback, setRollback] = useState<1 | 2 | 3>(2);
  const [aberto, setAberto] = useState(false);

  // Score ITIL: Impacto (peso 2) + Complexidade (peso 2) + Rollback (peso 1)
  // Mínimo = 5, Máximo = 15
  const score = useMemo(() => {
    return impacto * 2 + complexidade * 2 + rollback * 1;
  }, [impacto, complexidade, rollback]);

  const riscoCalculado: Risco = useMemo(() => {
    if (score <= 7) return "baixo";
    if (score <= 11) return "medio";
    return "alto";
  }, [score]);

  const recomendacaoCab = useMemo(() => {
    switch (riscoCalculado) {
      case "baixo":
        return "Risco Baixo: elegível para aprovação simplificada pelo líder técnico. Janela operacional padrão.";
      case "medio":
        return "Risco Médio: exige análise do comitê CAB e plano de contingência detalhado. Agendar em janela de menor tráfego.";
      case "alto":
        return "Risco Alto: exige homologação colegiada de todo o CAB, notificação prévia aos usuários e prontidão da equipe de sustentação.";
    }
  }, [riscoCalculado]);

  function aplicar() {
    const rotuloImpacto = ["", "Baixo (usuário/equipe)", "Médio (setor)", "Alto (institucional)"][impacto];
    const rotuloComplexidade = ["", "Baixa (procedimento testado)", "Média (múltiplos passos)", "Alta (nova arquitetura)"][complexidade];
    const rotuloRollback = ["", "Imediato (< 15 min)", "Moderado (15-60 min)", "Complexo (> 1h)"][rollback];

    const resumo = `[Cálculo de Risco CAB: Score ${score}/15 - ${RISCO[riscoCalculado].label}]\n• Impacto: ${rotuloImpacto}\n• Complexidade: ${rotuloComplexidade}\n• Reversão: ${rotuloRollback}`;
    onAplicarRisco(riscoCalculado, resumo, score);
    setAberto(false);
  }

  return (
    <div
      className="rounded-lg border border-surface-border bg-black/[0.02] p-3 text-sm dark:bg-white/[0.02]"
      data-testid="matriz-risco-calculadora"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button
          type="button"
          onClick={() => setAberto(!aberto)}
          className="flex items-center gap-2 font-medium text-foreground hover:text-primary transition-colors text-xs text-left"
        >
          <Calculator className="h-4 w-4 text-primary" />
          <span>Calculadora de Risco ITIL / Matriz do CAB</span>
          <span className="text-muted font-normal">
            ({aberto ? "Ocultar assistente" : "Calcular pelo impacto e complexidade"})
          </span>
        </button>

        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted">Risco sugerido:</span>
          <Badge tone={RISCO[riscoCalculado].tone}>
            {RISCO[riscoCalculado].label} (Score {score})
          </Badge>
        </div>
      </div>

      {aberto && (
        <div className="mt-3 flex flex-col gap-4 border-t border-surface-border pt-3">
          <div className="grid gap-3 sm:grid-cols-3">
            {/* Impacto */}
            <div className="flex flex-col gap-1.5">
              <label htmlFor="calculadora-impacto" className="text-xs font-semibold text-foreground">
                1. Impacto no Serviço
              </label>
              <select
                id="calculadora-impacto"
                value={impacto}
                onChange={(e) => setImpacto(Number(e.target.value) as 1 | 2 | 3)}
                aria-label="1. Impacto no Serviço"
                className="rounded-md border border-surface-border bg-surface px-2.5 py-1.5 text-xs text-foreground focus:border-primary focus:outline-hidden"
              >
                <option value={1}>1 - Baixo (Usuário ou equipe restrita)</option>
                <option value={2}>2 - Médio (Setor ou sistema intermediário)</option>
                <option value={3}>3 - Alto (Múltiplos setores ou sistema crítico)</option>
              </select>
            </div>

            {/* Complexidade / Probabilidade */}
            <div className="flex flex-col gap-1.5">
              <label htmlFor="calculadora-complexidade" className="text-xs font-semibold text-foreground">
                2. Complexidade / Testes
              </label>
              <select
                id="calculadora-complexidade"
                value={complexidade}
                onChange={(e) => setComplexidade(Number(e.target.value) as 1 | 2 | 3)}
                aria-label="2. Complexidade / Testes"
                className="rounded-md border border-surface-border bg-surface px-2.5 py-1.5 text-xs text-foreground focus:border-primary focus:outline-hidden"
              >
                <option value={1}>1 - Baixa (Procedimento padrão e testado)</option>
                <option value={2}>2 - Média (Múltiplas etapas ou dependências)</option>
                <option value={3}>3 - Alta (Inovação ou sem teste prévio)</option>
              </select>
            </div>

            {/* Plano de Rollback */}
            <div className="flex flex-col gap-1.5">
              <label htmlFor="calculadora-rollback" className="text-xs font-semibold text-foreground">
                3. Plano de Reversão
              </label>
              <select
                id="calculadora-rollback"
                value={rollback}
                onChange={(e) => setRollback(Number(e.target.value) as 1 | 2 | 3)}
                aria-label="3. Plano de Reversão"
                className="rounded-md border border-surface-border bg-surface px-2.5 py-1.5 text-xs text-foreground focus:border-primary focus:outline-hidden"
              >
                <option value={1}>1 - Imediato (&lt; 15 min, sem impacto)</option>
                <option value={2}>2 - Moderado (15 a 60 min)</option>
                <option value={3}>3 - Crítico / Sem rollback simples (&gt; 1h)</option>
              </select>
            </div>
          </div>

          {/* Mini Matriz Visual 3x3 */}
          <div className="flex flex-col gap-2 rounded-lg border border-surface-border bg-surface p-3">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">
              Matriz de Calor ITIL (Impacto × Complexidade)
            </span>
            <div className="grid grid-cols-3 gap-1.5 text-center text-[10px] font-medium">
              {/* Linha Impacto 3 (Alto) */}
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 3 && complexidade === 1
                    ? "ring-2 ring-warning bg-warning/20 font-bold"
                    : "bg-warning/10 text-warning"
                }`}
              >
                Médio (Imp 3 / Cpl 1)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 3 && complexidade === 2
                    ? "ring-2 ring-danger bg-danger/20 font-bold"
                    : "bg-danger/10 text-danger"
                }`}
              >
                Alto (Imp 3 / Cpl 2)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 3 && complexidade === 3
                    ? "ring-2 ring-danger bg-danger/30 font-bold"
                    : "bg-danger/20 text-danger"
                }`}
              >
                Crítico (Imp 3 / Cpl 3)
              </div>

              {/* Linha Impacto 2 (Médio) */}
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 2 && complexidade === 1
                    ? "ring-2 ring-info bg-info/20 font-bold"
                    : "bg-info/10 text-info"
                }`}
              >
                Baixo (Imp 2 / Cpl 1)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 2 && complexidade === 2
                    ? "ring-2 ring-warning bg-warning/20 font-bold"
                    : "bg-warning/10 text-warning"
                }`}
              >
                Médio (Imp 2 / Cpl 2)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 2 && complexidade === 3
                    ? "ring-2 ring-danger bg-danger/20 font-bold"
                    : "bg-danger/10 text-danger"
                }`}
              >
                Alto (Imp 2 / Cpl 3)
              </div>

              {/* Linha Impacto 1 (Baixo) */}
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 1 && complexidade === 1
                    ? "ring-2 ring-success bg-success/20 font-bold"
                    : "bg-success/10 text-success"
                }`}
              >
                Baixo (Imp 1 / Cpl 1)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 1 && complexidade === 2
                    ? "ring-2 ring-info bg-info/20 font-bold"
                    : "bg-info/10 text-info"
                }`}
              >
                Baixo (Imp 1 / Cpl 2)
              </div>
              <div
                className={`rounded p-1.5 transition-all ${
                  impacto === 1 && complexidade === 3
                    ? "ring-2 ring-warning bg-warning/20 font-bold"
                    : "bg-warning/10 text-warning"
                }`}
              >
                Médio (Imp 1 / Cpl 3)
              </div>
            </div>

            <p className="text-xs text-muted mt-1 leading-relaxed">
              <strong className="text-foreground">Parecer da Matriz:</strong> {recomendacaoCab}
            </p>
          </div>

          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={aplicar}
              className="inline-flex items-center gap-1.5"
            >
              <Check className="h-3.5 w-3.5 text-success" />
              Aplicar risco sugerido ({RISCO[riscoCalculado].label})
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
