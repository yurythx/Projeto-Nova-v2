import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";

import {
  IconeDoCatalogo,
  inferirIconePorTexto,
} from "./IconeDoCatalogo";

describe("IconeDoCatalogo — Inferência Inteligente e Resolução", () => {
  it("infere corretamente ícones baseados no nome da categoria ou oferta", () => {
    expect(inferirIconePorTexto("Equipamentos e Computadores")).toBe("monitor");
    expect(inferirIconePorTexto("Reparo de Notebook")).toBe("laptop");
    expect(inferirIconePorTexto("Impressoras e Scanners")).toBe("printer");
    expect(inferirIconePorTexto("Rede e Conectividade")).toBe("network");
    expect(inferirIconePorTexto("Acesso à Internet Wi-Fi")).toBe("wifi");
    expect(inferirIconePorTexto("Servidores e Hospedagem")).toBe("server");
    expect(inferirIconePorTexto("Sistemas e Aplicativos")).toBe("app-window");
    expect(inferirIconePorTexto("Acessos, Senhas e Permissões")).toBe("key-round");
    expect(inferirIconePorTexto("E-mail Institucional")).toBe("mail");
    expect(inferirIconePorTexto("Telefonia e Ramais")).toBe("phone");
    expect(inferirIconePorTexto("Manutenção Geral")).toBe("wrench");
    expect(inferirIconePorTexto("Infraestrutura Predial e Salas")).toBe("building-2");
    expect(inferirIconePorTexto("Dúvidas e Orientações")).toBe("circle-help");
    expect(inferirIconePorTexto("Qualquer coisa aleatória")).toBeNull();
  });

  it("renderiza o ícone inferido quando nome está vazio", () => {
    const { container } = render(
      <IconeDoCatalogo nome="" contextoNome="Impressoras da Secretaria" data-testid="icone-cat" />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("prioriza o nome explícito cadastrado sobre a inferência", () => {
    const { container } = render(
      <IconeDoCatalogo nome="wifi" contextoNome="Impressora de Rede" />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("usa padrão de suporte ITSM headset quando não há nome nem inferência", () => {
    const { container } = render(
      <IconeDoCatalogo nome="" contextoNome="Categoria Incomum X" padrao="headset" />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });
});
