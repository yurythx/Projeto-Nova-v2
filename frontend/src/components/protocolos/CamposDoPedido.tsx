"use client";

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";

export interface Campo {
  id: string;
  tipo: string;
  rotulo: string;
  obrigatorio: boolean;
  placeholder?: string;
  opcoes?: string[];
  revela?: string[];
}

export function CampoInput({
  campo,
  valor,
  onChange,
  onArquivo,
}: {
  campo: Campo;
  valor: unknown;
  onChange: (valor: unknown) => void;
  onArquivo: (arquivo: File) => void;
}) {
  if (campo.tipo === "page") {
    return <h3 className="pt-2 text-sm font-semibold">{campo.rotulo}</h3>;
  }
  const label = campo.obrigatorio ? `${campo.rotulo} *` : campo.rotulo;
  if (campo.tipo === "textarea") {
    return (
      <Textarea
        id={campo.id}
        label={label}
        required={campo.obrigatorio}
        placeholder={campo.placeholder}
        value={typeof valor === "string" ? valor : ""}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }
  if (campo.tipo === "select" || campo.tipo === "radio") {
    return (
      <Select
        id={campo.id}
        label={label}
        required={campo.obrigatorio}
        placeholder="Selecione…"
        value={typeof valor === "string" ? valor : ""}
        options={(campo.opcoes ?? []).map((opcao) => ({ value: opcao, label: opcao }))}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }
  if (campo.tipo === "checkbox" || campo.tipo === "boolean") {
    return (
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="h-4 w-4 accent-primary"
          checked={valor === true}
          onChange={(e) => onChange(e.target.checked)}
        />
        {label}
      </label>
    );
  }
  if (campo.tipo === "file") {
    return (
      <div className="flex flex-col gap-1">
        <Input
          id={campo.id}
          label={label}
          type="file"
          required={campo.obrigatorio && typeof valor !== "string"}
          onChange={(e) => {
            const arquivo = e.target.files?.[0];
            if (arquivo) onArquivo(arquivo);
          }}
        />
        {typeof valor === "string" && <p className="text-xs text-muted">Arquivo recebido.</p>}
      </div>
    );
  }
  if (campo.tipo === "map" || campo.tipo === "map_area" || campo.tipo === "ambulante") {
    return (
      <CampoMapa
        campo={campo}
        valor={typeof valor === "string" ? valor : ""}
        area={campo.tipo === "map_area"}
        onChange={onChange}
      />
    );
  }
  if (campo.tipo === "checkbox_group") {
    const marcados = Array.isArray(valor) ? valor.map(String) : [];
    return (
      <fieldset className="flex flex-col gap-2">
        <legend className="text-sm font-medium">{label}</legend>
        {(campo.opcoes ?? []).map((opcao) => (
          <label key={opcao} className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="h-4 w-4 accent-primary"
              checked={marcados.includes(opcao)}
              onChange={(e) =>
                onChange(e.target.checked ? [...marcados, opcao] : marcados.filter((item) => item !== opcao))
              }
            />
            {opcao}
          </label>
        ))}
      </fieldset>
    );
  }
  return (
    <Input
      id={campo.id}
      label={label}
      required={campo.obrigatorio}
      placeholder={campo.placeholder}
      type={tipoHtml(campo.tipo)}
      inputMode={campo.tipo === "number" ? "decimal" : undefined}
      value={valor == null ? "" : String(valor)}
      onChange={(e) => onChange(campo.tipo === "number" && e.target.value !== "" ? Number(e.target.value) : e.target.value)}
    />
  );
}

export function respostasVisiveis(campos: Campo[], respostas: Record<string, unknown>): Record<string, unknown> {
  const preenchido: Record<string, unknown> = {};
  for (const campo of campos) {
    if (campo.tipo === "page" || oculto(campos, respostas, campo.id)) continue;
    const valor = respostas[campo.id];
    if (valor !== undefined && valor !== "") preenchido[campo.id] = valor;
  }
  return preenchido;
}

export function oculto(campos: Campo[], respostas: Record<string, unknown>, id: string): boolean {
  return campos.some(
    (campo) => campo.tipo === "boolean" && (campo.revela ?? []).includes(id) && respostas[campo.id] !== true,
  );
}

function CampoMapa({
  campo,
  valor,
  area,
  onChange,
}: {
  campo: Campo;
  valor: string;
  area: boolean;
  onChange: (valor: unknown) => void;
}) {
  const label = campo.obrigatorio ? `${campo.rotulo} *` : campo.rotulo;
  const [primeiro, segundo] = valor.split(";");
  const [lat, lng] = partir(primeiro ?? "");
  const [lat2, lng2] = partir(segundo ?? "");
  const [aviso, setAviso] = useState("");

  function gravar(a: string, b: string, c = lat2, d = lng2) {
    const ponto = juntar(a, b);
    onChange(area ? `${ponto};${juntar(c, d)}` : ponto);
  }

  function localizar() {
    setAviso("");
    if (!navigator.geolocation) {
      setAviso("Este navegador não informa a localização.");
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => gravar(pos.coords.latitude.toFixed(6), pos.coords.longitude.toFixed(6)),
      () => setAviso("Não foi possível ler a localização."),
    );
  }

  const mapa = lat && lng ? `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lng}#map=16/${lat}/${lng}` : "";
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm font-medium">{label}</legend>
      <div className="grid gap-3 sm:grid-cols-2">
        <Input id={`${campo.id}-lat`} label={area ? "Latitude do primeiro ponto" : "Latitude"} inputMode="decimal" value={lat} onChange={(e) => gravar(e.target.value, lng)} />
        <Input id={`${campo.id}-lng`} label={area ? "Longitude do primeiro ponto" : "Longitude"} inputMode="decimal" value={lng} onChange={(e) => gravar(lat, e.target.value)} />
        {area && (
          <>
            <Input id={`${campo.id}-lat2`} label="Latitude do segundo ponto" inputMode="decimal" value={lat2} onChange={(e) => gravar(lat, lng, e.target.value, lng2)} />
            <Input id={`${campo.id}-lng2`} label="Longitude do segundo ponto" inputMode="decimal" value={lng2} onChange={(e) => gravar(lat, lng, lat2, e.target.value)} />
          </>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" variant="secondary" onClick={localizar}>
          Usar minha localização
        </Button>
        {mapa && (
          <a className="text-sm text-primary underline" href={mapa} target="_blank" rel="noreferrer">
            Ver no mapa
          </a>
        )}
      </div>
      {aviso && <p className="text-xs text-danger">{aviso}</p>}
    </fieldset>
  );
}

function partir(valor: string): [string, string] {
  const [lat = "", lng = ""] = valor.split(",");
  return [lat.trim(), lng.trim()];
}

function juntar(lat: string, lng: string): string {
  return `${lat.trim()},${lng.trim()}`;
}

function tipoHtml(tipo: string): string {
  if (tipo === "email" || tipo === "password" || tipo === "url" || tipo === "date" || tipo === "time") return tipo;
  if (tipo === "phone") return "tel";
  return "text";
}
