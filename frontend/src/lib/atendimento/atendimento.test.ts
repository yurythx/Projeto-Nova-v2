import { describe, expect, it } from "vitest";

import { rotuloAcao, rotuloEstado, rotuloTipo, tomEstado } from "./atendimento";

describe("vocabulário do atendimento", () => {
  it("a mesma passagem tem nome de equipe e de quem abriu", () => {
    expect(rotuloAcao("encerrado", "resolvido", true)).toBe("Confirmar e encerrar");
    expect(rotuloAcao("em_atendimento", "atendido", true)).toBe("Não resolveu, reabrir");
    expect(rotuloAcao("em_atendimento", "resolvido", false)).toBe("Reabrir");
    expect(rotuloAcao("em_atendimento", "aguardando_terceiro", false)).toBe("Retomar atendimento");
    expect(rotuloAcao("aguardando_servidor", "em_atendimento", false)).toBe(
      "Pedir informação ao servidor",
    );
    expect(rotuloAcao("aguardando_terceiro", "em_atendimento", false)).toBe("Aguardar terceiro");
    expect(rotuloAcao("resolvido", "em_atendimento", false)).toBe("Marcar como resolvida");
    expect(rotuloAcao("atendido", "em_atendimento", false)).toBe("Marcar como atendida");
    expect(rotuloAcao("encerrado", "atendido", false)).toBe("Encerrar");
    expect(rotuloAcao("cancelado", "aberto", true)).toBe("Cancelar solicitação");
    expect(rotuloAcao("aberto", "em_atendimento", false)).toBe("Aberta");
  });

  it("estado ou tipo que a tela ainda não conhece aparece pelo nome técnico", () => {
    expect(rotuloEstado("aguardando_servidor")).toBe("Aguardando resposta do servidor");
    expect(rotuloEstado("em_revisao")).toBe("em_revisao");
    expect(tomEstado("cancelado")).toBe("danger");
    expect(tomEstado("em_revisao")).toBe("neutral");
    expect(rotuloTipo("requisicao")).toBe("Requisição");
    expect(rotuloTipo("problema")).toBe("problema");
  });
});
