package auth

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"testing"
)

func iamScopeReportSQL(t *testing.T) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("sem caller")
	}
	dir := filepath.Dir(file)
	for range 10 {
		p := filepath.Join(dir, "scripts", "iam-scope-report.sql")
		b, err := os.ReadFile(p)
		if err == nil {
			return b
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("scripts/iam-scope-report.sql não encontrado a partir do teste")
	return nil
}

// O relatório de impacto (scripts/iam-scope-report.sql) repete a lista de
// permissões com escopo: as duas precisam andar juntas.
func TestRelatorioDeImpactoEspelhaScopedPermissions(t *testing.T) {
	sql := iamScopeReportSQL(t)
	var noSQL []string
	for _, m := range regexp.MustCompile(`'([a-z]+:[a-z]+)'`).FindAllStringSubmatch(string(sql), -1) {
		noSQL = append(noSQL, m[1])
	}
	var noGo []string
	for _, p := range ScopedPermissions {
		noGo = append(noGo, string(p))
	}
	slices.Sort(noSQL)
	slices.Sort(noGo)
	if !slices.Equal(noSQL, noGo) {
		t.Fatalf("iam-scope-report.sql %v difere de ScopedPermissions %v", noSQL, noGo)
	}
}
