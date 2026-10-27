import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { Experiencia, MedidaXLA } from "@/lib/atendimento/atendimento";
import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { ExperienciaCard } from "./ExperienciaCard";

const medida = (m: Partial<MedidaXLA>): MedidaXLA => ({
  valor: null,
  meta: null,
  atingida: null,
  amostra: 0,
  maior_melhor: false,
  ...m,
});

const experiencia: Experiencia = {
  csat: medida({ valor: 4.6, meta: 4.5, atingida: true, amostra: 10, maior_melhor: true }),
  facilidade: medida({ valor: 3.2, meta: 4, atingida: false, amostra: 1, maior_melhor: true }),
  reabertura: medida({ valor: 12.5, amostra: 8 }),
  idas_e_voltas: medida({ meta: 1, amostra: 0 }),
  espera_horas: medida({ valor: 30, meta: 48, atingida: true, amostra: 8 }),
};

// Gestão de uma unidade: vê, mas não define as metas da instituição.
const daUnidade = {
  permissions: ["atendimento:manage"],
  scopes: [
    {
      perfil: "p",
      origem: "manual" as const,
      unidade_id: "u1",
      permissions: ["atendimento:manage"],
    },
  ],
};

describe("ExperienciaCard", () => {
  it("mostra cada medida com a amostra e a situação da meta", async () => {
    mockBackend(identityRoutes(daUnidade));
    renderApp(<ExperienciaCard experiencia={experiencia} />);
    expect(screen.getByText("4,6")).toBeInTheDocument();
    expect(screen.getByText("Meta atingida (≥ 4,5)")).toBeInTheDocument();
    expect(screen.getByText("Abaixo da meta (≥ 4)")).toBeInTheDocument();
    expect(screen.getByText("12,5%")).toBeInTheDocument();
    expect(screen.getByText("sem meta")).toBeInTheDocument();
    // Com meta, mas sem amostra: não se afirma nada.
    expect(screen.getByText("meta ≤ 1")).toBeInTheDocument();
    expect(screen.getByText("sem amostra")).toBeInTheDocument();
    expect(screen.getByText("Meta atingida (≤ 48 h)")).toBeInTheDocument();
    expect(screen.getAllByText(/1 registro$/)).toHaveLength(1);
    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "Definir metas" })).not.toBeInTheDocument(),
    );
  });

  it("a gestão global define as metas; em branco fica sem meta", async () => {
    const user = userEvent.setup();
    const api = mockBackend({ ...identityRoutes(), "PUT v1/atendimento/xla": { data: {} } });
    const onMetasSalvas = vi.fn();
    renderApp(<ExperienciaCard experiencia={experiencia} onMetasSalvas={onMetasSalvas} />);
    await user.click(await screen.findByRole("button", { name: "Definir metas" }));
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: "Definir metas" }));
    expect(screen.getByLabelText("Nota: mínimo")).toHaveValue("4.5");
    await user.clear(screen.getByLabelText("Facilidade: mínimo"));
    await user.type(screen.getByLabelText("Reabertura: máximo"), "10,5");
    await user.click(screen.getByRole("button", { name: "Salvar metas" }));
    await waitFor(() => expect(onMetasSalvas).toHaveBeenCalled());
    expect(api.to("PUT v1/atendimento/xla")[0]!.body).toEqual({
      csat_min: 4.5,
      facilidade_min: null,
      reabertura_max: 10.5,
      idas_max: 1,
      espera_max_horas: 48,
    });
  });
});
