package application

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

func idsDoGrupo(t domain.Tarefa) []uuid.UUID {
	if len(t.Pessoas) > 0 {
		ids := make([]uuid.UUID, 0, len(t.Pessoas))
		for _, pessoa := range t.Pessoas {
			if pessoa.ID != uuid.Nil {
				ids = append(ids, pessoa.ID)
			}
		}
		return ids
	}
	return idDe(t.ResponsavelID)
}

func nomesDoGrupo(t domain.Tarefa) string {
	if len(t.Pessoas) == 0 {
		return t.ResponsavelNome
	}
	nomes := make([]string, 0, len(t.Pessoas))
	for _, pessoa := range t.Pessoas {
		nome := strings.TrimSpace(pessoa.Nome)
		if nome != "" {
			nomes = append(nomes, nome)
		}
	}
	return strings.Join(nomes, "\n")
}

func rotuloGrupo(t domain.Tarefa) string {
	return strings.ReplaceAll(nomesDoGrupo(t), "\n", ", ")
}

func mesmoGrupo(a, b domain.Tarefa) bool {
	aa, bb := idsDoGrupo(a), idsDoGrupo(b)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func diferenca(atuais, anteriores []uuid.UUID) []uuid.UUID {
	tem := map[uuid.UUID]struct{}{}
	for _, id := range anteriores {
		tem[id] = struct{}{}
	}
	var fora []uuid.UUID
	visto := map[uuid.UUID]struct{}{}
	for _, id := range atuais {
		if _, ok := tem[id]; ok {
			continue
		}
		if _, ok := visto[id]; ok {
			continue
		}
		visto[id] = struct{}{}
		fora = append(fora, id)
	}
	return fora
}

func primeiroDoGrupo(grupo []domain.Pessoa) (*uuid.UUID, string) {
	if len(grupo) == 0 {
		return nil, ""
	}
	id := grupo[0].ID
	return &id, grupo[0].Nome
}

func (s *Service) garantirParticipante(ctx context.Context, db database.DBTX, projetoID uuid.UUID, id *uuid.UUID, nome string) error {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	return s.repo.IncluirParticipante(ctx, db, projetoID, domain.Pessoa{ID: *id, Nome: nome})
}

// fixarGrupo aceita a lista enviada. Sem lista, o responsável único antigo
// continua valendo. Cada pessoa precisa participar do projeto, ou ser o gerente.
func (s *Service) fixarGrupo(ctx context.Context, db database.DBTX, projeto *domain.Projeto, atual *domain.Tarefa, responsavelID *uuid.UUID, responsavel string, pessoas *[]domain.Pessoa) ([]domain.Pessoa, error) {
	var brutas []domain.Pessoa
	switch {
	case pessoas != nil:
		brutas = *pessoas
	case responsavelID != nil && *responsavelID != uuid.Nil:
		brutas = []domain.Pessoa{{ID: *responsavelID, Nome: responsavel}}
	case strings.TrimSpace(responsavel) == "":
		brutas = nil
	case atual != nil && strings.TrimSpace(responsavel) == strings.TrimSpace(atual.ResponsavelNome):
		if len(atual.Pessoas) > 0 {
			brutas = atual.Pessoas
		} else if atual.ResponsavelID != nil {
			brutas = []domain.Pessoa{{ID: *atual.ResponsavelID, Nome: atual.ResponsavelNome}}
		}
	default:
		return nil, domain.ErrPessoaInvalida
	}
	if len(brutas) > domain.PessoasPorTarefaMax {
		return nil, domain.ErrPessoasDemais
	}
	visto := map[uuid.UUID]struct{}{}
	var grupo []domain.Pessoa
	for _, pessoa := range brutas {
		if pessoa.ID == uuid.Nil {
			if strings.TrimSpace(pessoa.Nome) != "" {
				return nil, domain.ErrPessoaInvalida
			}
			continue
		}
		if _, ok := visto[pessoa.ID]; ok {
			continue
		}
		visto[pessoa.ID] = struct{}{}
		if len(grupo) >= domain.PessoasPorTarefaMax {
			return nil, domain.ErrPessoasDemais
		}
		nome, err := s.nomeNoProjeto(ctx, db, projeto, pessoa.ID)
		if err != nil {
			return nil, err
		}
		grupo = append(grupo, domain.Pessoa{ID: pessoa.ID, Nome: nome})
	}
	return grupo, nil
}

func (s *Service) nomeNoProjeto(ctx context.Context, db database.DBTX, projeto *domain.Projeto, id uuid.UUID) (string, error) {
	if projeto.GerenteID != nil && *projeto.GerenteID == id {
		if nome := strings.TrimSpace(projeto.GerenteNome); nome != "" {
			return nome, nil
		}
	}
	ok, err := s.repo.EhParticipante(ctx, db, projeto.ID, id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", domain.ErrForaDoProjeto
	}
	return s.repo.NomeDeResponsavel(ctx, db, id, projeto.UnidadeID)
}

func (s *Service) ListarMembros(ctx context.Context, idt auth.Identity, projetoID uuid.UUID) ([]domain.Pessoa, error) {
	projeto, err := s.repo.Obter(ctx, s.pool, projetoID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	lista, err := s.repo.ListarParticipantes(ctx, s.pool, projetoID)
	if err != nil {
		return nil, err
	}
	if projeto.GerenteID == nil || *projeto.GerenteID == uuid.Nil {
		return lista, nil
	}
	for i := range lista {
		if lista[i].ID == *projeto.GerenteID {
			lista[i].Papel = domain.PapelGerente
			return lista, nil
		}
	}
	return append([]domain.Pessoa{{
		ID: *projeto.GerenteID, Nome: projeto.GerenteNome, Papel: domain.PapelGerente,
	}}, lista...), nil
}

func (s *Service) IncluirParticipante(ctx context.Context, idt auth.Identity, projetoID uuid.UUID, id uuid.UUID) ([]domain.Pessoa, error) {
	if id == uuid.Nil {
		return nil, domain.ErrPessoaInvalida
	}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		projeto, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		nome, err := s.repo.NomeDeResponsavel(ctx, tx, id, projeto.UnidadeID)
		if err != nil {
			return err
		}
		if err := s.repo.IncluirParticipante(ctx, tx, projetoID, domain.Pessoa{ID: id, Nome: nome}); err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.participante.added", "projeto", projetoID.String(), nil, id))
	})
	if err != nil {
		return nil, err
	}
	return s.ListarMembros(ctx, idt, projetoID)
}

func (s *Service) ExcluirParticipante(ctx context.Context, idt auth.Identity, projetoID, pessoa uuid.UUID) ([]domain.Pessoa, error) {
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		projeto, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		if projeto.GerenteID != nil && *projeto.GerenteID == pessoa {
			return domain.ErrGerentePermanece
		}
		ok, err := s.repo.EhParticipante(ctx, tx, projetoID, pessoa)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrParticipanteAusente
		}
		antes, err := s.repo.ListarTarefas(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if err := s.repo.ExcluirParticipante(ctx, tx, projetoID, pessoa); err != nil {
			return err
		}
		if err := s.repo.RetirarPessoaDasTarefas(ctx, tx, projetoID, pessoa); err != nil {
			return err
		}
		depois, err := s.repo.ListarTarefas(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		porID := map[uuid.UUID]domain.Tarefa{}
		for i := range depois {
			porID[depois[i].ID] = depois[i]
		}
		for i := range antes {
			if !pessoaEsta(antes[i], pessoa) {
				continue
			}
			atual, ok := porID[antes[i].ID]
			if !ok {
				continue
			}
			if err := s.anotarMovimento(ctx, tx, idt, antes[i], atual); err != nil {
				return err
			}
		}
		if err := audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.participante.removed", "projeto", projetoID.String(), pessoa, nil)); err != nil {
			return err
		}
		parte, err := s.registrar(ctx, tx, idt.UserID, []uuid.UUID{pessoa}, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Você saiu do projeto #" + strconv.FormatInt(projeto.Numero, 10),
			Mensagem: cortar(projeto.Titulo+". Você saiu das tarefas desse projeto.", 1000),
			Link:     linkProjeto(projeto.ID),
			Chave:    chaveAviso("projeto", projeto.ID.String(), "saiu", pessoa.String()),
		})
		if err != nil {
			return err
		}
		enviadas = append(enviadas, parte...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.entregar(ctx, enviadas)
	return s.ListarMembros(ctx, idt, projetoID)
}

func pessoaEsta(t domain.Tarefa, id uuid.UUID) bool {
	for _, pessoa := range idsDoGrupo(t) {
		if pessoa == id {
			return true
		}
	}
	return false
}
