import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

let pathname = "/configuracao";
vi.mock("next/navigation", () => ({ usePathname: () => pathname }));
let perms: string[] = [];
vi.mock("@/lib/plataforma/PlataformaProvider", () => ({
  usePlataforma: () => ({ can: (p: string) => perms.includes(p), canGlobal: (p: string) => perms.includes(p), enabled: () => true }),
}));

import { ModuleTabs } from "./ModuleTabs";

const nomes = () => screen.getAllByRole("link").map((a) => a.textContent);

describe("ModuleTabs", () => {
  it("Trâmite só tem a aba índice", () => {
    perms = [];
    pathname = "/tramite";
    render(<ModuleTabs module="tramite" index={{ href: "/tramite", label: "Processos" }} ariaLabel="Seções do Trâmite" />);
    expect(nomes()).toEqual(["Processos"]);
  });

  it("marca o índice na rota do módulo", () => {
    perms = [];
    pathname = "/tramite";
    render(<ModuleTabs module="tramite" index={{ href: "/tramite", label: "Processos" }} ariaLabel="Seções do Trâmite" />);
    expect(screen.getByRole("link", { name: "Processos" })).toHaveAttribute("aria-current", "page");
  });
});
