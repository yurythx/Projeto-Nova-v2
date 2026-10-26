import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { MatrizRiscoCalculadora } from "@/components/servicos/MatrizRiscoCalculadora";

describe("MatrizRiscoCalculadora (Gestão de Riscos do CAB)", () => {
  it("renderiza assistente e calcula score de risco", () => {
    const onAplicar = vi.fn();
    render(<MatrizRiscoCalculadora riscoAtual="medio" onAplicarRisco={onAplicar} />);

    expect(screen.getByTestId("matriz-risco-calculadora")).toBeInTheDocument();
    expect(screen.getByText("Calculadora de Risco ITIL / Matriz do CAB")).toBeInTheDocument();
    expect(screen.getByText(/Risco médio \(Score 10\)/)).toBeInTheDocument();
  });

  it("permite expandir o assistente, alterar parâmetros e aplicar o novo risco", async () => {
    const user = userEvent.setup();
    const onAplicar = vi.fn();
    render(<MatrizRiscoCalculadora riscoAtual="baixo" onAplicarRisco={onAplicar} />);

    // Expande a calculadora
    await user.click(screen.getByRole("button", { name: /Calculadora de Risco ITIL/i }));

    expect(screen.getByText("Matriz de Calor ITIL (Impacto × Complexidade)")).toBeInTheDocument();
    expect(screen.getByText("1. Impacto no Serviço")).toBeInTheDocument();

    // Seleciona Impacto Alto (3), Complexidade Alta (3) e Rollback Crítico (3)
    await user.selectOptions(screen.getByLabelText("1. Impacto no Serviço"), "3");
    await user.selectOptions(screen.getByLabelText("2. Complexidade / Testes"), "3");
    await user.selectOptions(screen.getByLabelText("3. Plano de Reversão"), "3");

    // Score calculado deve ser 3*2 + 3*2 + 3*1 = 15 (Alto)
    expect(screen.getByText(/Risco alto \(Score 15\)/)).toBeInTheDocument();
    expect(screen.getByText(/exige homologação colegiada de todo o CAB/i)).toBeInTheDocument();
    // A mudança estava como baixo: a divergência aparece antes de aplicar.
    expect(screen.getByRole("status")).toHaveTextContent(
      "A mudança está como risco baixo; a matriz sugere risco alto.",
    );

    // Aplica o risco sugerido
    await user.click(
      screen.getByRole("button", { name: /Aplicar risco sugerido \(Risco alto\)/i }),
    );

    expect(onAplicar).toHaveBeenCalledTimes(1);
    expect(onAplicar).toHaveBeenCalledWith(
      "alto",
      expect.stringContaining("[Cálculo de Risco CAB: Score 15/15 - Risco alto]"),
      15,
    );
  });
});
