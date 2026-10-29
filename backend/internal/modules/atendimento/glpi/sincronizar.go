package glpi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// A sincronização até a virada (ADR 081): enquanto o GLPI segue em uso, o
// chamado já importado que mudou lá (versão = date_mod) é trazido de novo
// — estado, técnico, solução, fechamento, acompanhamentos novos e a
// avaliação. Aqui ele fica só para leitura, então não há o que mesclar.

// registro é o chamado já importado: o destino e a versão do GLPI que
// entrou por último. Versão vazia: importado antes do ADR 081, a primeira
// sincronização o traz de novo.
type registro struct {
	destino uuid.UUID
	versao  string
}

func (im *Importador) registrados(ctx context.Context) (map[string]registro, error) {
	rows, err := im.Pool.Query(ctx, `SELECT id_origem, destino, COALESCE(versao, '') FROM atendimento_importacao
		WHERE origem = $1 AND tipo = 'solicitacao'`, Origem)
	if err != nil {
		return nil, fmt.Errorf("importados: %w", err)
	}
	defer rows.Close()
	out := map[string]registro{}
	for rows.Next() {
		var id string
		var r registro
		if err := rows.Scan(&id, &r.destino, &r.versao); err != nil {
			return nil, err
		}
		out[id] = r
	}
	return out, rows.Err()
}

// leitura é o que o chamado traz além dele mesmo.
type leitura struct {
	solicitante, tecnico *pessoaCasada
	acomps               []Acompanhamento
	autores              []*pessoaCasada
	sat                  *Satisfacao
}

// ler busca na fonte os participantes, os acompanhamentos (com os autores
// casados) e a avaliação do chamado.
func (im *Importador) ler(ctx context.Context, ch Chamado) (leitura, error) {
	var l leitura
	partes, err := im.Fonte.Participantes(ctx, ch.ID)
	if err != nil {
		return l, err
	}
	for _, p := range partes {
		switch {
		case p.Papel == 1 && l.solicitante == nil:
			l.solicitante, err = im.pessoa(ctx, p.PessoaID)
		case p.Papel == 2 && l.tecnico == nil:
			l.tecnico, err = im.pessoa(ctx, p.PessoaID)
		}
		if err != nil {
			return l, err
		}
	}
	if l.acomps, err = im.Fonte.Acompanhamentos(ctx, ch.ID); err != nil {
		return l, err
	}
	l.autores = make([]*pessoaCasada, len(l.acomps))
	for i, a := range l.acomps {
		if l.autores[i], err = im.pessoa(ctx, a.AutorID); err != nil {
			return l, err
		}
	}
	l.sat, err = im.Fonte.Satisfacao(ctx, ch.ID)
	return l, err
}

// marcos são a solução e o fechamento do estado, pelas datas do GLPI; fim
// é o último dos dois. Fora da conclusão, nenhum.
func (im *Importador) marcos(estado string, ch Chamado, abertura time.Time) (resolvida, encerrada, fim *time.Time) {
	switch estado {
	case domain.EstadoResolvido, domain.EstadoAtendido, domain.EstadoEncerrado:
		r := im.data(ch.Solucao, im.data(ch.Fechamento, abertura))
		resolvida, fim = &r, &r
		if estado == domain.EstadoEncerrado {
			e := im.data(ch.Fechamento, r)
			encerrada, fim = &e, &e
		}
	}
	return resolvida, encerrada, fim
}

// gravarAvaliacao grava a pesquisa respondida, se ainda não há avaliação.
func (im *Importador) gravarAvaliacao(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, sat *Satisfacao) error {
	if sat == nil || sat.Nota == nil || !domain.PodeAvaliar(sol) {
		return nil
	}
	if _, err := im.Repo.GetAvaliacao(ctx, tx, sol.ID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrNaoEncontrado) {
		return err
	}
	fim := sol.CreatedAt
	if sol.ResolvidaEm != nil {
		fim = *sol.ResolvidaEm
	}
	_, err := im.Repo.InsertAvaliacao(ctx, tx, domain.Avaliacao{
		ID: uuid.New(), SolicitacaoID: sol.ID, AutorID: sol.SolicitanteID, Nota: Nota(*sat.Nota),
		Comentario: Texto(sat.Comentario, domain.TextoMax), CreatedAt: im.data(sat.Em, fim),
	})
	return err
}

// atualizar traz de novo o chamado que mudou no GLPI. A passagem de
// estado leva a data da mudança lá; o acompanhamento que já entrou (pelo
// registro ou, nos importados antes do ADR 081, pela mesma hora) não se
// repete.
func (im *Importador) atualizar(ctx context.Context, ch Chamado, id uuid.UUID) error {
	l, err := im.ler(ctx, ch)
	if err != nil || !im.Gravar {
		return err
	}
	return database.WithTx(ctx, im.Pool, func(ctx context.Context, tx pgx.Tx) error {
		sol, err := im.Repo.GetSolicitacao(ctx, tx, id)
		if err != nil {
			return err
		}
		tipoGLPI := tipoIncidente
		if sol.Tipo == domain.TipoRequisicao {
			tipoGLPI = tipoRequisicao
		}
		mudou := im.data(ch.Modificado, time.Now().UTC())
		next := sol
		next.Estado = Estado(ch.Status, tipoGLPI)
		next.ResponsavelID = nil
		if l.tecnico != nil {
			next.ResponsavelID = l.tecnico.id
		}
		next.ResolvidaEm, next.EncerradaEm, _ = im.marcos(next.Estado, ch, sol.CreatedAt)
		if err := im.Repo.UpdateSolicitacao(ctx, tx, next); err != nil {
			return err
		}
		if next.Estado != sol.Estado {
			autor := ContaImportacao
			if next.ResponsavelID != nil {
				autor = *next.ResponsavelID
			}
			if _, err := im.Repo.InsertPassagem(ctx, tx, domain.Passagem{SolicitacaoID: id, De: &sol.Estado, Para: next.Estado,
				AutorID: &autor, Em: mudou}); err != nil {
				return err
			}
		}
		for i, a := range l.acomps {
			var ja bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM atendimento_importacao
					WHERE origem = $1 AND tipo = 'acompanhamento' AND id_origem = $2)
				OR EXISTS (SELECT 1 FROM atendimento_comentarios WHERE solicitacao_id = $3 AND created_at = $4)`,
				Origem, strconv.Itoa(a.ID), id, im.data(a.Em, sol.CreatedAt)).Scan(&ja); err != nil {
				return err
			}
			if ja {
				continue
			}
			if err := im.gravarAcompanhamento(ctx, tx, sol, a, l.autores[i]); err != nil {
				return err
			}
		}
		if err := im.gravarAvaliacao(ctx, tx, next, l.sat); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE atendimento_solicitacoes SET updated_at = $2 WHERE id = $1`, id, mudou); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE atendimento_importacao SET versao = $3, sincronizado_em = now()
			WHERE origem = $1 AND tipo = 'solicitacao' AND id_origem = $2`, Origem, strconv.Itoa(ch.ID), ch.Modificado)
		return err
	})
}
