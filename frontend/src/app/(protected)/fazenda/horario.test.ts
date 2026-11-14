import { describe, expect, it } from "vitest";

import { emCuiaba } from "./horario";

describe("emCuiaba", () => {
  it("mostra o instante no horário de Cuiabá e omite data zero", () => {
    expect(emCuiaba("2026-08-01T03:59:00.000Z")).toBe("31/07/2026, 23:59");
    expect(emCuiaba("0001-01-01T00:00:00Z")).toBe("");
    expect(emCuiaba("")).toBe("");
    expect(emCuiaba(null)).toBe("");
  });
});
