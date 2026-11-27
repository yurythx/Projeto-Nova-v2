"use client";

import { useState } from "react";

import { BOTAO_SECUNDARIO, CAMPO, Rotulado, ROTULO } from "./ui";

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
    return (
      <h3 className="border-b border-gray-200 pb-2 pt-3 text-base font-semibold text-gray-800 dark:border-slate-700 dark:text-slate-100">
        {campo.rotulo}
      </h3>
    );
  }
  const label = campo.obrigatorio ? `${campo.rotulo} *` : campo.rotulo;
  if (campo.tipo === "textarea") {
    return (
      <Rotulado id={campo.id} rotulo={label}>
        <textarea
          id={campo.id}
          rows={4}
          required={campo.obrigatorio}
          placeholder={campo.placeholder}
          value={typeof valor === "string" ? valor : ""}
          onChange={(e) => onChange(e.target.value)}
          className={`${CAMPO} min-h-[100px] resize-y`}
        />
      </Rotulado>
    );
  }
  if (campo.tipo === "select" || campo.tipo === "radio") {
    return (
      <Rotulado id={campo.id} rotulo={label}>
        <select
          id={campo.id}
          required={campo.obrigatorio}
          value={typeof valor === "string" ? valor : ""}
          onChange={(e) => onChange(e.target.value)}
          className={CAMPO}
        >
          <option value="">Selecione…</option>
          {(campo.opcoes ?? []).map((opcao) => (
            <option key={opcao} value={opcao}>
              {opcao}
            </option>
          ))}
        </select>
      </Rotulado>
    );
  }
  if (campo.tipo === "checkbox" || campo.tipo === "boolean") {
    return (
      <label className="flex cursor-pointer items-center gap-2.5 rounded-lg border border-gray-200 px-3 py-2.5 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-700/50">
        <input
          type="checkbox"
          className="h-4 w-4 accent-blue-500"
          checked={valor === true}
          onChange={(e) => onChange(e.target.checked)}
        />
        {label}
      </label>
    );
  }
  if (campo.tipo === "file") {
    return (
      <Rotulado id={campo.id} rotulo={label}>
        <input
          id={campo.id}
          type="file"
          required={campo.obrigatorio && typeof valor !== "string"}
          onChange={(e) => {
            const arquivo = e.target.files?.[0];
            if (arquivo) onArquivo(arquivo);
          }}
          className={`${CAMPO} file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 dark:file:bg-sky-400/15 dark:file:text-sky-300`}
        />
        {typeof valor === "string" && (
          <p className="mt-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
            Arquivo recebido.
          </p>
        )}
      </Rotulado>
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
        <legend className={ROTULO}>{label}</legend>
        {(campo.opcoes ?? []).map((opcao) => (
          <label
            key={opcao}
            className="flex items-center gap-2 text-sm text-gray-700 dark:text-slate-200"
          >
            <input
              type="checkbox"
              className="h-4 w-4 accent-blue-500"
              checked={marcados.includes(opcao)}
              onChange={(e) =>
                onChange(
                  e.target.checked
                    ? [...marcados, opcao]
                    : marcados.filter((item) => item !== opcao),
                )
              }
            />
            {opcao}
          </label>
        ))}
      </fieldset>
    );
  }
  return (
    <Rotulado id={campo.id} rotulo={label}>
      <input
        id={campo.id}
        required={campo.obrigatorio}
        placeholder={campo.placeholder}
        type={tipoHtml(campo.tipo)}
        inputMode={campo.tipo === "number" ? "decimal" : undefined}
        value={valor == null ? "" : String(valor)}
        onChange={(e) =>
          onChange(
            campo.tipo === "number" && e.target.value !== ""
              ? Number(e.target.value)
              : e.target.value,
          )
        }
        className={CAMPO}
      />
    </Rotulado>
  );
}

export function respostasVisiveis(
  campos: Campo[],
  respostas: Record<string, unknown>,
): Record<string, unknown> {
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
    (campo) =>
      campo.tipo === "boolean" && (campo.revela ?? []).includes(id) && respostas[campo.id] !== true,
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

  const mapa =
    lat && lng ? `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lng}#map=16/${lat}/${lng}` : "";
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className={ROTULO}>{label}</legend>
      <div className="grid gap-3 sm:grid-cols-2">
        <Rotulado id={`${campo.id}-lat`} rotulo={area ? "Latitude do primeiro ponto" : "Latitude"}>
          <input
            id={`${campo.id}-lat`}
            inputMode="decimal"
            value={lat}
            onChange={(e) => gravar(e.target.value, lng)}
            className={CAMPO}
          />
        </Rotulado>
        <Rotulado
          id={`${campo.id}-lng`}
          rotulo={area ? "Longitude do primeiro ponto" : "Longitude"}
        >
          <input
            id={`${campo.id}-lng`}
            inputMode="decimal"
            value={lng}
            onChange={(e) => gravar(lat, e.target.value)}
            className={CAMPO}
          />
        </Rotulado>
        {area && (
          <>
            <Rotulado id={`${campo.id}-lat2`} rotulo="Latitude do segundo ponto">
              <input
                id={`${campo.id}-lat2`}
                inputMode="decimal"
                value={lat2}
                onChange={(e) => gravar(lat, lng, e.target.value, lng2)}
                className={CAMPO}
              />
            </Rotulado>
            <Rotulado id={`${campo.id}-lng2`} rotulo="Longitude do segundo ponto">
              <input
                id={`${campo.id}-lng2`}
                inputMode="decimal"
                value={lng2}
                onChange={(e) => gravar(lat, lng, lat2, e.target.value)}
                className={CAMPO}
              />
            </Rotulado>
          </>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <button type="button" className={BOTAO_SECUNDARIO} onClick={localizar}>
          Usar minha localização
        </button>
        {mapa && (
          <a
            className="text-sm font-medium text-blue-600 underline dark:text-sky-400"
            href={mapa}
            target="_blank"
            rel="noreferrer"
          >
            Ver no mapa
          </a>
        )}
      </div>
      {aviso && <p className="text-xs text-red-600 dark:text-red-400">{aviso}</p>}
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
  if (
    tipo === "email" ||
    tipo === "password" ||
    tipo === "url" ||
    tipo === "date" ||
    tipo === "time"
  )
    return tipo;
  if (tipo === "phone") return "tel";
  return "text";
}
