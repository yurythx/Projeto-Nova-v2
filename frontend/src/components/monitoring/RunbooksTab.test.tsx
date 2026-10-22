import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { RunbooksTab } from "./RunbooksTab";

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("RunbooksTab", () => {
  it("copia cada comando operacional e volta ao rótulo depois de 2 s", () => {
    const writeText = vi.fn();
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
    vi.useFakeTimers();
    render(<RunbooksTab />);
    const botoes = screen.getAllByRole("button", { name: "Copiar" });
    expect(botoes.length).toBeGreaterThan(1);
    for (const [i, b] of botoes.entries()) {
      fireEvent.click(b);
      expect(writeText).toHaveBeenCalledTimes(i + 1);
      expect(screen.getByRole("button", { name: "Copiado!" })).toBeInTheDocument();
    }
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining("restore.sh --verify"));
    act(() => vi.advanceTimersByTime(2000));
    expect(screen.queryByRole("button", { name: "Copiado!" })).not.toBeInTheDocument();
  });
});
