#!/usr/bin/env bash
# ==============================================================================
# Meta de cobertura por módulo (plugin) — roda no CI depois do `go test`.
#
# Lê o perfil gerado por `go test -coverpkg=./internal/...,./pkg/... -coverprofile=...`
# (vários pacotes de teste somados: o mesmo bloco aparece uma vez por binário,
# então vale a MAIOR contagem de cada bloco) e exige 100% de cobertura de
# instruções em cada diretório de internal/modules/<plugin>, em cada pacote de
# internal/platform/<pacote> (menos os dublês dbtest/storagetest) e em
# internal/app (composição) e em pkg/<pacote>. Lista as linhas descobertas quando falha.
#
# Catraca (ADR 064): scripts/coverage-floor.txt lista o piso dos módulos que
# ainda não chegaram à meta. O módulo listado não pode cair abaixo do piso;
# o que não está listado continua exigindo a meta. Quando o módulo sobe, o
# script mostra o piso novo; em 100%, a linha sai do arquivo.
#
# Uso: scripts/coverage-gate.sh backend/coverage.out [meta=100]
# ==============================================================================
set -euo pipefail

PROFILE=${1:?uso: $0 <coverage.out> [meta]}
TARGET=${2:-100}
FLOOR=${COVERAGE_FLOOR:-$(dirname "$0")/coverage-floor.txt}
[ -s "$PROFILE" ] || { echo "perfil de cobertura vazio ou ausente: $PROFILE" >&2; exit 1; }

# O arquivo de pisos é reconhecido pelo nome, não pela posição: vazio ou
# ausente, nenhuma linha do perfil vira piso por engano.
awk -v target="$TARGET" -v floor="$FLOOR" '
BEGIN {
    while ((getline linha < floor) > 0) {
        if (linha ~ /^[[:space:]]*(#|$)/) continue
        split(linha, campo, /[[:space:]]+/)
        piso[campo[1]] = campo[2]
    }
    close(floor)
}
FNR == 1 && /^mode:/ { next }
{
    key = $1; stmts = $2; count = $3
    if (!(key in n) || count > c[key]) c[key] = count
    n[key] = stmts
}
END {
    for (key in n) {
        file = key; sub(/:.*/, "", file)
        if (match(file, /internal\/modules\/[^\/]+\//)) {
            mod = substr(file, RSTART + 17, RLENGTH - 18)
        } else if (match(file, /internal\/platform\/[^\/]+\//)) {
            mod = "platform/" substr(file, RSTART + 18, RLENGTH - 19)
            # dublês de teste (fakes de banco/armazenamento), não produção
            if (file ~ /\/(dbtest|storagetest)\//) continue
        } else if (file ~ /internal\/app\//) {
            mod = "app"
        } else if (match(file, /\/pkg\/[^\/]+\//)) {
            mod = substr(file, RSTART + 1, RLENGTH - 2)
        } else {
            continue
        }
        total[mod] += n[key]
        if (c[key] > 0) {
            covered[mod] += n[key]
        } else if (n[key] > 0) {
            line = key; sub(/^[^:]*:/, "", line); sub(/\..*/, "", line)
            short = file; sub(/.*internal\//, "", short)
            miss[mod] = miss[mod] "\n      " short ":" line
        }
    }
    fail = 0
    printf "%-30s %10s\n", "módulo", "cobertura"
    for (mod in total) {
        pct = 100 * covered[mod] / total[mod]
        meta = (mod in piso) ? piso[mod] : target
        printf "%-30s %9.1f%%\n", mod, pct
        if (pct + 1e-9 < meta) {
            fail = 1
            if (mod in piso) {
                printf "    ✗ caiu abaixo do piso de %s%% (catraca, ADR 064) — linhas descobertas:%s\n", meta, miss[mod]
            } else {
                printf "    ✗ abaixo da meta de %s%% — linhas descobertas:%s\n", target, miss[mod]
            }
        } else if ((mod in piso) && pct + 1e-9 >= target) {
            printf "    ↑ chegou a %s%%: apague a linha de %s em coverage-floor.txt\n", target, mod
        } else if ((mod in piso) && pct - meta >= 0.5) {
            printf "    ↑ suba o piso de %s para %.1f em coverage-floor.txt\n", mod, int(pct * 10) / 10
        }
    }
    if (length(total) == 0) {
        print "nenhum módulo no perfil: o go test rodou com -coverpkg?" > "/dev/stderr"
        exit 1
    }
    if (fail) exit 1
    printf "\n✓ nenhum módulo abaixo da meta de %s%% ou do seu piso\n", target
}' "$PROFILE"
