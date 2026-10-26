"use client";

import { Clock, Plus, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";

import { useToast } from "@/components/notifications/ToastProvider";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { ResumoApontamentosAtendimento } from "@/lib/atendimento/atendimento";

export function ApontamentosCard({
  solicitacaoId,
  tecnico,
  finalizada,
}: {
  solicitacaoId: string;
  tecnico: boolean;
  finalizada: boolean;
}) {
  const { showToast } = useToast();
  const [modalAberto, setModalAberto] = useState(false);
  const [salvando, setSalvando] = useState(false);

  const hoje = new Date().toISOString().split("T")[0];
  const [dataApontamento, setDataApontamento] = useState(hoje);
  const [horas, setHoras] = useState("");
  const [descricao, setDescricao] = useState("");

  const { data, mutate, isLoading } = useApiQuery<ResumoApontamentosAtendimento>(
    solicitacaoId ? `v1/atendimento/solicitacoes/${solicitacaoId}/apontamentos` : null,
  );

  const totalHoras = data?.total_horas ?? 0;
  const apontamentos = data?.apontamentos ?? [];

  async function handleSalvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!horas || !descricao) return;
    setSalvando(true);
    const numHoras = parseFloat(horas);

    try {
      await apiClient.post(`v1/atendimento/solicitacoes/${solicitacaoId}/apontamentos`, {
        data_apontamento: dataApontamento || hoje,
        horas: numHoras,
        descricao: descricao.trim(),
      });
      showToast({
        title: "Apontamento registrado",
        description: `${numHoras}h registradas com sucesso.`,
        tone: "success",
      });
      setModalAberto(false);
      setHoras("");
      setDescricao("");
      void mutate();
    } catch (err) {
      showToast({
        title: "Erro ao registrar horas",
        description: (err as Error).message,
        tone: "danger",
      });
    } finally {
      setSalvando(false);
    }
  }

  async function handleExcluir(id: string) {
    try {
      await apiClient.delete(`v1/atendimento/solicitacoes/${solicitacaoId}/apontamentos/${id}`);
      showToast({
        title: "Apontamento removido",
        tone: "info",
      });
      void mutate();
    } catch (err) {
      showToast({
        title: "Erro ao remover apontamento",
        description: (err as Error).message,
        tone: "danger",
      });
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div className="flex items-center gap-2">
          <Clock className="h-5 w-5 text-primary" />
          <CardTitle as="h2">Timesheet & Horas Trabalhadas</CardTitle>
          <Badge tone="info" className="ml-2 font-mono">
            {totalHoras.toFixed(1)}h total
          </Badge>
        </div>
        {tecnico && !finalizada && (
          <Button
            type="button"
            size="sm"
            onClick={() => setModalAberto(true)}
            className="flex items-center gap-1.5"
          >
            <Plus className="h-4 w-4" />
            Apontar horas
          </Button>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        {isLoading && !data && <p className="text-sm text-muted">Carregando apontamentos...</p>}

        {!isLoading && apontamentos.length === 0 && (
          <p className="text-sm text-muted">
            Nenhum apontamento de esforço registrado para esta solicitação.
          </p>
        )}

        {apontamentos.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-surface-border text-xs uppercase tracking-wide text-muted">
                <tr>
                  <th className="pb-2">Data</th>
                  <th className="pb-2">Técnico</th>
                  <th className="pb-2">Duração</th>
                  <th className="pb-2">Atividade Realizada</th>
                  {tecnico && !finalizada && <th className="pb-2 text-right">Ação</th>}
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-border">
                {apontamentos.map((ap) => (
                  <tr key={ap.id} className="group">
                    <td className="py-2.5 font-medium whitespace-nowrap">{ap.data_apontamento}</td>
                    <td className="py-2.5 text-muted whitespace-nowrap">{ap.usuario_nome}</td>
                    <td className="py-2.5 font-semibold text-primary whitespace-nowrap">
                      {ap.horas.toFixed(1)}h
                    </td>
                    <td className="py-2.5 whitespace-pre-wrap">{ap.descricao}</td>
                    {tecnico && !finalizada && (
                      <td className="py-2.5 text-right whitespace-nowrap">
                        <ConfirmButton
                          variant="ghost"
                          size="sm"
                          title="Remover apontamento de horas?"
                          description="Este registro de esforço será excluído definitivamente."
                          confirmLabel="Remover"
                          onConfirm={() => handleExcluir(ap.id)}
                        >
                          <Trash2 className="h-4 w-4 text-danger/80 hover:text-danger" />
                        </ConfirmButton>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {modalAberto && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-xs">
            <div className="w-full max-w-md rounded-xl border border-surface-border bg-surface p-6 shadow-2xl">
              <h3 className="text-lg font-bold">Lançar Apontamento de Horas</h3>
              <p className="mt-1 text-sm text-muted">
                Registre o esforço técnico dedicado à resolução desta demanda.
              </p>
              <form onSubmit={handleSalvar} className="mt-4 flex flex-col gap-4">
                <Input
                  id="data_apontamento"
                  name="data_apontamento"
                  label="Data do Atendimento"
                  type="date"
                  required
                  value={dataApontamento}
                  onChange={(e) => setDataApontamento(e.target.value)}
                />
                <Input
                  id="horas"
                  name="horas"
                  label="Horas Trabalhadas (ex: 1.5 para 1h 30m)"
                  type="number"
                  step="0.25"
                  min="0.25"
                  max="24"
                  required
                  placeholder="1.0"
                  value={horas}
                  onChange={(e) => setHoras(e.target.value)}
                />
                <Textarea
                  id="descricao"
                  name="descricao"
                  label="Descrição das Atividades"
                  required
                  rows={3}
                  placeholder="Ex: Diagnóstico remoto de falha de conexão e testes..."
                  value={descricao}
                  onChange={(e) => setDescricao(e.target.value)}
                />
                <div className="mt-2 flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => setModalAberto(false)}
                    disabled={salvando}
                  >
                    Cancelar
                  </Button>
                  <Button type="submit" loading={salvando}>
                    Salvar Horas
                  </Button>
                </div>
              </form>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
