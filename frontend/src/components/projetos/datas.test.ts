import { describe, expect, it } from "vitest";

import { diaISO, faixaDaSemana, faixaDoPrazo, prazoDoDia } from "./datas";

describe("datas do projeto", () => {
  const sexta = new Date("2026-11-06T15:00:00-03:00");

  it("guarda o dia civil sem deslocar o fuso", () => {
    expect(diaISO("2027-02-27")).toBe("2027-02-27");
    expect(diaISO("2027-02-27T18:00:00-03:00")).toBe("2027-02-27");
    expect(prazoDoDia("2027-02-27")).toBe("2027-02-27T18:00:00-03:00");
    expect(prazoDoDia("")).toBeNull();
  });

  it("separa atraso, esta semana e o que vem depois", () => {
    const semana = faixaDaSemana(sexta);
    expect(semana.hoje).toBe("2026-11-06");
    expect(semana.inicio).toBe("2026-11-02");
    expect(semana.fim).toBe("2026-11-08");
    expect(faixaDoPrazo("2026-11-05T18:00:00-03:00", sexta)).toBe("atrasada");
    expect(faixaDoPrazo("2026-11-06", sexta)).toBe("semana");
    expect(faixaDoPrazo("2026-11-08T18:00:00-03:00", sexta)).toBe("semana");
    expect(faixaDoPrazo("2026-11-09", sexta)).toBe("depois");
    expect(faixaDoPrazo(null, sexta)).toBe("sem");
  });
});
