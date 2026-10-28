import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ResumoExpediente } from "./ResumoExpediente";

describe("Resumo de prazos e equipes da carga", () => {
  it("mostra a política, as metas, os feriados e as equipes da simulação", () => {
    render(
      <ResumoExpediente
        gravou={false}
        expediente={{
          politica: "criada",
          calendario: "Horário Comercial",
          metas: ["alta: resposta 30 min, solução 240 min"],
          feriados: 12,
          equipes: 3,
          membros: 9,
          membros_sem_par: 2,
          grupos_sem_unidade: ["Grupo da OUTRA"],
          grupos_sem_chamado: 381,
        }}
      />,
    );
    const resumo = screen.getByRole("region", { name: "Prazos e equipes" });
    expect(within(resumo).getByText(/simulação: nada foi gravado/)).toBeInTheDocument();
    expect(resumo).toHaveTextContent('criada a partir do calendário "Horário Comercial"');
    expect(resumo).toHaveTextContent("alta: resposta 30 min, solução 240 min");
    expect(resumo).toHaveTextContent("12 feriado(s) novo(s)");
    expect(resumo).toHaveTextContent(
      "3 equipe(s) dos grupos do GLPI, com 9 pessoa(s); 2 sem conta aqui",
    );
    expect(resumo).toHaveTextContent("Grupos sem unidade na tabela: Grupo da OUTRA.");
    expect(resumo).toHaveTextContent("381 grupo(s) do GLPI que nunca atenderam");
  });

  it("na carga, sem calendário e sem pendências", () => {
    render(
      <ResumoExpediente
        gravou
        expediente={{
          politica: "existente",
          feriados: 0,
          equipes: 1,
          membros: 1,
          membros_sem_par: 0,
        }}
      />,
    );
    const resumo = screen.getByRole("region", { name: "Prazos e equipes" });
    expect(resumo).not.toHaveTextContent("simulação");
    expect(resumo).toHaveTextContent("já configurada fica como está.");
    expect(resumo).not.toHaveTextContent("Metas");
    expect(resumo).not.toHaveTextContent("sem conta");
    expect(resumo).not.toHaveTextContent("Grupos sem unidade");
    expect(resumo).not.toHaveTextContent("nunca atenderam");
  });
});
