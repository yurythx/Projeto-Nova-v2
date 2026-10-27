#!/usr/bin/env bash
# Testes do scripts/coverage-gate.sh (roda no CI, job infra).
#
# Um gate que aprova sem conferir é pior do que nenhum: a primeira versão
# da catraca lia um arquivo de pisos vazio como se fosse o perfil e
# aprovava tudo. Estes casos garantem que isso não volta.
set -euo pipefail

GATE="$(cd "$(dirname "$0")" && pwd)/coverage-gate.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
falhas=0

# Perfil com o módulo "alfa" em 50% (1 de 2 blocos) e "beta" em 100%.
cat >"$tmp/perfil.out" <<'EOF'
mode: set
github.com/x/backend/internal/modules/alfa/a.go:1.1,2.1 1 1
github.com/x/backend/internal/modules/alfa/a.go:3.1,4.1 1 0
github.com/x/backend/internal/modules/beta/b.go:1.1,2.1 1 1
EOF

espera() { # espera <0|1> <descrição> <pisos ou "-"> [perfil]
  local quer=$1 desc=$2 pisos=$3 perfil=${4:-$tmp/perfil.out} obtido=0
  if [ "$pisos" = "-" ]; then
    COVERAGE_FLOOR="$tmp/nao-existe" bash "$GATE" "$perfil" >"$tmp/saida" 2>&1 || obtido=1
  else
    printf '%b' "$pisos" >"$tmp/pisos.txt"
    COVERAGE_FLOOR="$tmp/pisos.txt" bash "$GATE" "$perfil" >"$tmp/saida" 2>&1 || obtido=1
  fi
  if [ "$obtido" = "$quer" ]; then
    echo "ok    $desc"
  else
    echo "FALHA $desc (esperava saída $quer, veio $obtido)"; sed 's/^/      /' "$tmp/saida"
    falhas=$((falhas + 1))
  fi
}

espera 1 "módulo abaixo da meta, sem arquivo de pisos, reprova" "-"
espera 1 "arquivo de pisos vazio não aprova nada" ""
espera 1 "arquivo de pisos só com comentário não aprova nada" "# nada\n"
espera 0 "piso igual à cobertura aprova" "alfa 50\n"
espera 0 "piso abaixo da cobertura aprova" "alfa 40\n"
espera 1 "cobertura abaixo do piso reprova" "alfa 60\n"
espera 1 "piso de outro módulo não alivia este" "gama 10\n"
espera 1 "perfil ausente reprova" "alfa 50\n" "$tmp/nao-existe.out"
printf 'mode: set\n' >"$tmp/vazio.out"
espera 1 "perfil sem nenhum módulo reprova" "alfa 50\n" "$tmp/vazio.out"

printf 'alfa 40\n' >"$tmp/pisos.txt"
if COVERAGE_FLOOR="$tmp/pisos.txt" bash "$GATE" "$tmp/perfil.out" | grep -q "suba o piso de alfa para 50.0"; then
  echo "ok    sugere subir o piso quando a cobertura melhora"
else
  echo "FALHA não sugeriu subir o piso"; falhas=$((falhas + 1))
fi

[ "$falhas" = 0 ] && echo "todos os casos passaram" || { echo "$falhas caso(s) falharam"; exit 1; }
