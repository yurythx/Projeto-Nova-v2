import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import RiscosPage from "./page";

const RISCO_MOCK = {
  id: "r1",
  codigo: "RSK-001",
  titulo: "Falha de Link Primário de Internet",
  descricao: "Rompimento físico do link de fibra da Prefeitura.",
  categoria: "infraestrutura",
  probabilidade: 4,
  impacto: 4,
  severidade: 16,
  estrategia: "mitigar",
  plano_mitigacao: "Contratação de link redundante e BGP automático.",
  controle_iso27001: "A.17.1 - Continuidade da Segurança da Informação",
  status: "em_mitigacao",
  responsavel_nome: "Equipe de Redes",
  unidade_id: "u1",
  created_at: "2026-10-31T20:00:00Z",
  updated_at: "2026-10-31T20:00:00Z",
};

const HEATMAP_MOCK = {
  matriz: [{ probabilidade: 4, impacto: 4, severidade: 16, quantidade: 1 }],
  total_riscos: 1,
  criticos: 0,
  altos: 1,
  medios: 0,
  baixos: 0,
};

beforeEach(() => {
  resetNavigation({}, "/riscos");
});

describe("Matriz de Riscos (ISO/IEC 27001)", () => {
  it("renderiza o cabeçalho, KPIs e tabela de riscos", async () => {
    mockBackend({
      ...identityRoutes(),
      "GET v1/riscos/heatmap": { data: HEATMAP_MOCK },
      "GET v1/riscos": { data: [RISCO_MOCK] },
    });

    renderApp(<RiscosPage />);

    expect(await screen.findByText("Matriz de Riscos (ISO/IEC 27001)")).toBeInTheDocument();
    expect(screen.getByText("RSK-001")).toBeInTheDocument();
    expect(screen.getByText("Falha de Link Primário de Internet")).toBeInTheDocument();
    expect(screen.getByText("A.17.1 - Continuidade da Segurança da Informação")).toBeInTheDocument();
    expect(screen.getAllByText("Em mitigação").length).toBeGreaterThan(0);
    expect(screen.getByText("16 • ALTO")).toBeInTheDocument();
  });

  it("permite abrir o modal de cadastro e cadastrar novo risco", async () => {
    let payloadCriado: any = null;

    mockBackend({
      ...identityRoutes(),
      "GET v1/riscos/heatmap": {
        data: { matriz: [], total_riscos: 0, criticos: 0, altos: 0, medios: 0, baixos: 0 },
      },
      "GET v1/riscos": { data: [] },
      "POST v1/riscos": (req: any) => {
        payloadCriado = req.body;
        return {
          status: 201,
          data: {
            ...RISCO_MOCK,
            id: "r2",
            codigo: "RSK-002",
            titulo: req.body?.titulo,
          },
        };
      },
    });

    const user = userEvent.setup();
    renderApp(<RiscosPage />);

    const btnNovo = await screen.findByRole("button", { name: /Novo Risco/i });
    await user.click(btnNovo);

    expect(screen.getByText("Cadastrar Novo Risco (ISO 27001)")).toBeInTheDocument();

    const inputTitulo = screen.getByPlaceholderText(/Falha Crítica no Link de Fibra Óptica/i);
    await user.type(inputTitulo, "Vazamento de Credenciais de Acesso");

    const inputDesc = screen.getByPlaceholderText(/Descreva a causa raiz/i);
    await user.type(inputDesc, "Phishing direcionado contra operadores do sistema.");

    const inputPlano = screen.getByPlaceholderText(/Descreva as medidas corretivas/i);
    await user.type(inputPlano, "Implementar MFA obrigatório via chave FIDO2 e treinamento anti-phishing.");

    const btnSalvar = screen.getByRole("button", { name: /Registrar Risco/i });
    await user.click(btnSalvar);

    await waitFor(() => {
      expect(payloadCriado).not.toBeNull();
      expect(payloadCriado.titulo).toBe("Vazamento de Credenciais de Acesso");
    });
  });
});
