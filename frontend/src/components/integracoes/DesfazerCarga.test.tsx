import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DesfazerCarga, ResumoDesfazer } from "./DesfazerCarga";

const base = {
  gravou: false,
  chamados: 14998,
  ofertas: 51,
  categorias: 4,
  equipes: 2,
  politicas: 1,
  feriados: 0,
  chamados_mantidos: 2,
  exemplos_mantidos: [15002, 15010],
  ofertas_mantidas: 1,
  cadastros_mantidos: 1,
};

describe("Desfazer a carga", () => {
  it("o resumo da simulação conta o que sairia e o que fica", () => {
    render(<ResumoDesfazer relatorio={base} />);
    const resumo = screen.getByRole("region", { name: "Desfazer a carga" });
    expect(within(resumo).getByText(/simulação: nada foi removido/)).toBeInTheDocument();
    expect(resumo).toHaveTextContent("14998 chamado(s) seriam removido(s)");
    expect(resumo).toHaveTextContent("51 oferta(s), 4 categoria(s), 2 equipe(s), 1 política(s)");
    expect(resumo).toHaveTextContent(
      "2 chamado(s) ficam porque alguém mexeu neles aqui depois da carga (nº 15002, 15010…)",
    );
    expect(resumo).toHaveTextContent("1 oferta(s) e 1 outro(s) cadastro(s) ficam");
  });

  it("o resumo do feito, sem nada que fique", () => {
    render(
      <ResumoDesfazer
        relatorio={{
          ...base,
          gravou: true,
          chamados_mantidos: 0,
          exemplos_mantidos: [],
          ofertas_mantidas: 0,
          cadastros_mantidos: 0,
        }}
      />,
    );
    const resumo = screen.getByRole("region", { name: "Desfazer a carga" });
    expect(resumo).not.toHaveTextContent("simulação");
    expect(resumo).toHaveTextContent("14998 chamado(s) removido(s)");
    expect(resumo).not.toHaveTextContent("ficam");
  });

  it("simula sem confirmar e só remove com o nome da conexão", async () => {
    const onPedir = vi.fn().mockResolvedValue(true);
    render(<DesfazerCarga nome="GLPI da Prefeitura" ocupado={false} onPedir={onPedir} />);
    fireEvent.click(screen.getByRole("button", { name: "Simular desfazer" }));
    expect(onPedir).toHaveBeenCalledWith("desfazer_simulacao");

    const remover = screen.getByRole("button", { name: "Desfazer carga" });
    const campo = screen.getByRole("textbox", { name: /digite o nome da conexão/ });
    expect(remover).toBeDisabled();
    fireEvent.change(campo, { target: { value: "GLPI" } });
    expect(remover).toBeDisabled();
    fireEvent.change(campo, { target: { value: " GLPI da Prefeitura " } });
    expect(remover).toBeEnabled();
    fireEvent.click(remover);
    expect(onPedir).toHaveBeenLastCalledWith("desfazer", "GLPI da Prefeitura");
    await vi.waitFor(() => expect(campo).toHaveValue(""));
  });

  it("com uma tarefa em andamento, nada se pede", () => {
    render(<DesfazerCarga nome="X" ocupado onPedir={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Simular desfazer" })).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "X" } });
    expect(screen.getByRole("button", { name: "Desfazer carga" })).toBeDisabled();
  });
});
