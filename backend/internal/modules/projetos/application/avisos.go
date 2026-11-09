package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

func (s *Service) entregar(ctx context.Context, enviadas []notificacoes.Enviada) {
	if s.avisos != nil && len(enviadas) > 0 {
		s.avisos.Entregar(ctx, enviadas)
	}
}

// registrar grava um aviso para cada destinatário, menos quem fez a ação.
func (s *Service) registrar(ctx context.Context, db database.DBTX, autor uuid.UUID, destinos []uuid.UUID, n notificacoes.Nova) ([]notificacoes.Enviada, error) {
	return s.enviarAviso(ctx, db, autor, destinos, n, false)
}

// enviarAviso grava o aviso. Na atribuição e na movimentação, quem fez
// a ação também recebe, porque o sino é o registro de que a pessoa ficou
// com o projeto ou a tarefa.
func (s *Service) enviarAviso(ctx context.Context, db database.DBTX, autor uuid.UUID, destinos []uuid.UUID, n notificacoes.Nova, incluirAutor bool) ([]notificacoes.Enviada, error) {
	if s.avisos == nil || len(destinos) == 0 {
		return nil, nil
	}
	visto := map[uuid.UUID]struct{}{}
	limpos := make([]uuid.UUID, 0, len(destinos))
	for _, id := range destinos {
		if id == uuid.Nil || (!incluirAutor && id == autor) {
			continue
		}
		if _, ok := visto[id]; ok {
			continue
		}
		visto[id] = struct{}{}
		limpos = append(limpos, id)
	}
	if len(limpos) == 0 {
		return nil, nil
	}
	return notificacoes.Registrar(ctx, db, limpos, n)
}

func chaveAviso(partes ...string) string {
	chave := "projetos"
	for _, p := range partes {
		chave += ":" + p
	}
	return chave + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func linkProjeto(id uuid.UUID) string {
	return "/projetos/" + id.String()
}

func igualID(a, b *uuid.UUID) bool {
	if a == nil || *a == uuid.Nil {
		return b == nil || *b == uuid.Nil
	}
	return b != nil && *a == *b
}

func idDe(p *uuid.UUID) []uuid.UUID {
	if p == nil || *p == uuid.Nil {
		return nil
	}
	return []uuid.UUID{*p}
}

func rotuloEstadoProjeto(estado string) string {
	switch estado {
	case domain.EstadoPlanejado:
		return "Planejado"
	case domain.EstadoEmAndamento:
		return "Em andamento"
	case domain.EstadoSuspenso:
		return "Suspenso"
	case domain.EstadoConcluido:
		return "Concluído"
	case domain.EstadoCancelado:
		return "Cancelado"
	default:
		return estado
	}
}

func rotuloEstadoTarefa(estado string) string {
	switch estado {
	case domain.EstadoTarefaAFazer:
		return "A fazer"
	case domain.EstadoTarefaEmAndamento:
		return "Em andamento"
	case domain.EstadoTarefaConcluida:
		return "Concluída"
	case domain.EstadoTarefaBloqueada:
		return "Bloqueada"
	default:
		return estado
	}
}

func (s *Service) responsaveisDoProjeto(ctx context.Context, db database.DBTX, p *domain.Projeto) ([]uuid.UUID, error) {
	if p == nil {
		return nil, nil
	}
	ids := idDe(p.GerenteID)
	tarefas, err := s.repo.ListarTarefas(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	for i := range tarefas {
		ids = append(ids, idDe(tarefas[i].ResponsavelID)...)
	}
	return ids, nil
}

func (s *Service) avisarGerenteNovo(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto) ([]notificacoes.Enviada, error) {
	if p == nil || p.GerenteID == nil {
		return nil, nil
	}
	return s.enviarAviso(ctx, tx, autor, []uuid.UUID{*p.GerenteID}, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Você é o responsável pelo projeto #" + strconv.FormatInt(p.Numero, 10),
		Mensagem: cortar(mensagemAtribuida(s.nomeDoAutor(ctx, tx, autor), p.Titulo), 1000),
		Link:     linkProjeto(p.ID),
		Chave:    chaveAviso("projeto", p.ID.String(), "gerente"),
	}, true)
}

func (s *Service) avisarGerenteSaiu(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto, antigo *uuid.UUID) ([]notificacoes.Enviada, error) {
	if p == nil || igualID(antigo, p.GerenteID) {
		return nil, nil
	}
	return s.registrar(ctx, tx, autor, idDe(antigo), notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Você deixou de ser responsável pelo projeto #" + strconv.FormatInt(p.Numero, 10),
		Mensagem: cortar(p.Titulo, 1000),
		Link:     linkProjeto(p.ID),
		Chave:    chaveAviso("projeto", p.ID.String(), "gerente-saiu"),
	})
}

func (s *Service) avisarTrocaDeGerente(ctx context.Context, tx pgx.Tx, autor uuid.UUID, antes, depois *domain.Projeto, titulo, mensagem string) ([]notificacoes.Enviada, error) {
	if depois == nil {
		return nil, nil
	}
	if antes == nil || igualID(antes.GerenteID, depois.GerenteID) {
		return s.avisarProjeto(ctx, tx, autor, depois, titulo, mensagem, "atualizado")
	}
	var enviadas []notificacoes.Enviada
	saiu, err := s.avisarGerenteSaiu(ctx, tx, autor, depois, antes.GerenteID)
	if err != nil {
		return nil, err
	}
	enviadas = append(enviadas, saiu...)
	novo, err := s.avisarGerenteNovo(ctx, tx, autor, depois)
	if err != nil {
		return nil, err
	}
	enviadas = append(enviadas, novo...)
	destinos, err := s.responsaveisDoProjeto(ctx, tx, depois)
	if err != nil {
		return nil, err
	}
	semGerentes := make([]uuid.UUID, 0, len(destinos))
	for _, id := range destinos {
		if depois.GerenteID != nil && id == *depois.GerenteID {
			continue
		}
		if antes.GerenteID != nil && id == *antes.GerenteID {
			continue
		}
		semGerentes = append(semGerentes, id)
	}
	quem := depois.GerenteNome
	if quem == "" {
		quem = "ninguém"
	}
	parte, err := s.registrar(ctx, tx, autor, semGerentes, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Responsável do projeto #" + strconv.FormatInt(depois.Numero, 10),
		Mensagem: cortar("Passou a ser "+quem, 1000),
		Link:     linkProjeto(depois.ID),
		Chave:    chaveAviso("projeto", depois.ID.String(), "gerente-trocou"),
	})
	if err != nil {
		return nil, err
	}
	return append(enviadas, parte...), nil
}

func (s *Service) avisarProjeto(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto, titulo, mensagem, assunto string) ([]notificacoes.Enviada, error) {
	destinos, err := s.responsaveisDoProjeto(ctx, tx, p)
	if err != nil {
		return nil, err
	}
	return s.enviarAviso(ctx, tx, autor, destinos, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   cortar(titulo, 200),
		Mensagem: cortar(mensagem, 1000),
		Link:     linkProjeto(p.ID),
		Chave:    chaveAviso("projeto", p.ID.String(), assunto),
	}, assunto == "estado")
}

func (s *Service) avisarSucessoras(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto, concluida domain.Tarefa) ([]notificacoes.Enviada, error) {
	if p == nil || concluida.Estado != domain.EstadoTarefaConcluida {
		return nil, nil
	}
	tarefas, err := s.repo.ListarTarefas(ctx, tx, p.ID)
	if err != nil {
		return nil, err
	}
	var enviadas []notificacoes.Enviada
	for i := range tarefas {
		seguinte := tarefas[i]
		if seguinte.PredecessoraID == nil || *seguinte.PredecessoraID != concluida.ID || seguinte.ResponsavelID == nil {
			continue
		}
		parte, err := s.enviarAviso(ctx, tx, autor, []uuid.UUID{*seguinte.ResponsavelID}, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "A tarefa anterior terminou",
			Mensagem: cortar(concluida.Titulo+" foi concluída. "+seguinte.Titulo+" já pode ser concluída.", 1000),
			Link:     linkProjeto(p.ID),
			Chave:    chaveAviso("tarefa", seguinte.ID.String(), "anterior"),
		}, true)
		if err != nil {
			return nil, err
		}
		enviadas = append(enviadas, parte...)
	}
	return enviadas, nil
}

func (s *Service) avisarTarefa(ctx context.Context, tx pgx.Tx, autor uuid.UUID, p *domain.Projeto, antes, depois domain.Tarefa) ([]notificacoes.Enviada, error) {
	if p == nil {
		return nil, nil
	}
	var enviadas []notificacoes.Enviada
	ja := map[uuid.UUID]struct{}{}
	marcar := func(ids []uuid.UUID) {
		for _, id := range ids {
			ja[id] = struct{}{}
		}
	}
	if !igualID(antes.ResponsavelID, depois.ResponsavelID) && depois.ResponsavelID != nil {
		mensagem := depois.Titulo + " · projeto #" + strconv.FormatInt(p.Numero, 10) + " " + p.Titulo
		if antes.ID != uuid.Nil && antes.Estado != depois.Estado {
			mensagem = depois.Titulo + " foi para " + rotuloEstadoTarefa(depois.Estado) + " · projeto #" + strconv.FormatInt(p.Numero, 10)
		}
		mensagem = mensagemAtribuida(s.nomeDoAutor(ctx, tx, autor), mensagem)
		parte, err := s.enviarAviso(ctx, tx, autor, []uuid.UUID{*depois.ResponsavelID}, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Você é o responsável pela tarefa",
			Mensagem: cortar(mensagem, 1000),
			Link:     linkProjeto(p.ID),
			Chave:    chaveAviso("tarefa", depois.ID.String(), "responsavel"),
		}, true)
		if err != nil {
			return nil, err
		}
		enviadas = append(enviadas, parte...)
		marcar([]uuid.UUID{*depois.ResponsavelID})
	}
	if !igualID(antes.ResponsavelID, depois.ResponsavelID) {
		parte, err := s.registrar(ctx, tx, autor, idDe(antes.ResponsavelID), notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Você deixou de ser responsável pela tarefa",
			Mensagem: cortar(antes.Titulo+" · projeto #"+strconv.FormatInt(p.Numero, 10)+" "+p.Titulo, 1000),
			Link:     linkProjeto(p.ID),
			Chave:    chaveAviso("tarefa", depois.ID.String(), "responsavel-saiu"),
		})
		if err != nil {
			return nil, err
		}
		enviadas = append(enviadas, parte...)
		marcar(idDe(antes.ResponsavelID))
	}
	if antes.ID == uuid.Nil || !tarefaMudou(antes, depois) {
		if p.GerenteID != nil {
			if _, ok := ja[*p.GerenteID]; !ok {
				titulo := "Responsável da tarefa no projeto #" + strconv.FormatInt(p.Numero, 10)
				mensagem := depois.Titulo + " ficou com " + nomeOuNinguem(depois.ResponsavelNome)
				if antes.ID == uuid.Nil {
					titulo = "Nova tarefa no projeto #" + strconv.FormatInt(p.Numero, 10)
					mensagem = depois.Titulo
				}
				parte, err := s.enviarAviso(ctx, tx, autor, []uuid.UUID{*p.GerenteID}, notificacoes.Nova{
					Modulo:   "projetos",
					Titulo:   titulo,
					Mensagem: cortar(mensagem, 1000),
					Link:     linkProjeto(p.ID),
					Chave:    chaveAviso("tarefa", depois.ID.String(), "gerente"),
				}, antes.ID != uuid.Nil)
				if err != nil {
					return nil, err
				}
				enviadas = append(enviadas, parte...)
			}
		}
		return enviadas, nil
	}
	destinos := append(idDe(p.GerenteID), idDe(depois.ResponsavelID)...)
	filtrados := make([]uuid.UUID, 0, len(destinos))
	for _, id := range destinos {
		if _, ok := ja[id]; ok {
			continue
		}
		filtrados = append(filtrados, id)
	}
	parte, err := s.enviarAviso(ctx, tx, autor, filtrados, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Tarefa no projeto #" + strconv.FormatInt(p.Numero, 10),
		Mensagem: cortar(resumoTarefa(antes, depois), 1000),
		Link:     linkProjeto(p.ID),
		Chave:    chaveAviso("tarefa", depois.ID.String(), "mudou"),
	}, true)
	if err != nil {
		return nil, err
	}
	return append(enviadas, parte...), nil
}

func (s *Service) nomeDoAutor(ctx context.Context, db database.DBTX, autor uuid.UUID) string {
	if s == nil || autor == uuid.Nil {
		return ""
	}
	nome, err := s.repo.NomeDaPessoa(ctx, db, autor, nil)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(nome)
}

func mensagemAtribuida(autor, corpo string) string {
	quem := strings.TrimSpace(autor)
	if quem == "" {
		quem = "alguém"
	}
	texto := "Atribuído por " + quem + "."
	corpo = strings.TrimSpace(corpo)
	if corpo == "" {
		return texto
	}
	return texto + " " + corpo
}

func nomeOuNinguem(nome string) string {
	if nome == "" {
		return "ninguém"
	}
	return nome
}

func tarefaMudou(antes, depois domain.Tarefa) bool {
	return antes.Titulo != depois.Titulo ||
		antes.Descricao != depois.Descricao ||
		antes.Relato != depois.Relato ||
		antes.Estado != depois.Estado ||
		antes.Ordem != depois.Ordem ||
		!igualID(antes.PredecessoraID, depois.PredecessoraID) ||
		!mesmoPrazo(antes.DataLimite, depois.DataLimite)
}

func mesmoPrazo(a, b *time.Time) bool {
	if a == nil || a.IsZero() {
		return b == nil || b.IsZero()
	}
	if b == nil || b.IsZero() {
		return false
	}
	return a.Equal(*b)
}

func resumoTarefa(antes, depois domain.Tarefa) string {
	texto := depois.Titulo
	if !igualID(antes.ResponsavelID, depois.ResponsavelID) {
		texto += " ficou com " + nomeOuNinguem(depois.ResponsavelNome)
	}
	if antes.Estado != depois.Estado {
		texto += " foi para " + rotuloEstadoTarefa(depois.Estado)
	} else if antes.Titulo != depois.Titulo {
		texto = "O título passou a ser " + depois.Titulo
	} else if antes.Relato != depois.Relato {
		texto += " tem um relato novo"
	} else {
		texto += " foi atualizada"
	}
	return texto
}
