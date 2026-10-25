package application

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

const retencaoSerie = 366

// Serie devolve os retratos diários do escopo de quem atende, no máximo
// um ano. Quem cobre tudo vê a linha institucional. Quem cobre unidades
// vê a reunião delas, e o intervalo entre falhas fica vazio quando há
// mais de uma unidade no mesmo dia.
func (s *Service) Serie(ctx context.Context, identity auth.Identity, dias int) ([]domain.Retrato, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return []domain.Retrato{}, nil
	}
	if dias < 1 {
		dias = 90
	}
	if dias > retencaoSerie {
		dias = retencaoSerie
	}
	var ancora *uuid.UUID
	if !cob.All && len(cob.Unidades) == 1 {
		ancora = &cob.Unidades[0]
	}
	hoje := s.diaCivil(ctx, ancora)
	desde := hoje.AddDate(0, 0, -(dias - 1))
	pontos, err := s.repo.ListarRetratos(ctx, s.pool, desde, !cob.All, cob.Unidades)
	if err != nil {
		return nil, MapError(err)
	}
	return domain.AgregarSerie(pontos), nil
}

// RegistrarSerie grava o dia de hoje. O mesmo dia é regravado; os
// anteriores permanecem como foram medidos.
func (s *Service) RegistrarSerie(ctx context.Context) error {
	unidades, err := s.repo.UnidadesComSolicitacao(ctx, s.pool)
	if err != nil {
		return err
	}
	for _, unidade := range unidades {
		if err := s.gravarEscopo(ctx, s.diaCivil(ctx, &unidade), &unidade); err != nil {
			return err
		}
	}
	return s.gravarEscopo(ctx, s.diaCivil(ctx, nil), nil)
}

// diaCivil segue o fuso da política da unidade. Sem política, ou com
// fuso ilegível, vale Brasília. A linha institucional usa a política
// institucional. Quem cobre várias unidades lê a janela nesse fuso.
func (s *Service) diaCivil(ctx context.Context, unidade *uuid.UUID) time.Time {
	fuso := domain.FusoPadrao
	if pol, err := s.repo.ResolverPolitica(ctx, s.pool, unidade); err == nil {
		if nome := strings.TrimSpace(pol.Fuso); nome != "" {
			fuso = nome
		}
	}
	return domain.DiaNoFuso(time.Now(), fuso)
}

func (s *Service) gravarEscopo(ctx context.Context, dia time.Time, unidade *uuid.UUID) error {
	restrito := unidade != nil
	var unidades []uuid.UUID
	if unidade != nil {
		unidades = []uuid.UUID{*unidade}
	}
	lista, err := s.repo.ListarParaGovernanca(ctx, s.pool, restrito, unidades, TetoGovernanca+1)
	if err != nil {
		return err
	}
	truncado := len(lista) > TetoGovernanca
	if truncado {
		lista = lista[:TetoGovernanca]
	}
	if err := s.medir(ctx, lista); err != nil {
		return err
	}
	passagens, err := s.passagensConcluidas(ctx, lista)
	if err != nil {
		return err
	}
	ind := domain.Resumir(lista, passagens)
	resumo, err := s.repo.ResumoAvaliacoes(ctx, s.pool, restrito, unidades, 0)
	if err != nil {
		return err
	}
	var csat *float64
	if resumo.Total > 0 {
		media := math.Round(float64(resumo.Soma)/float64(resumo.Total)*10) / 10
		csat = &media
	}
	p := domain.Retrato{
		Dia: dia, UnidadeID: unidade,
		MTTRHoras: ind.MTTRHoras, MTBFHoras: ind.MTBFHoras,
		SLAPercentual: ind.SLAPercentual, FCRPercentual: ind.FCRPercentual, CSATMedia: csat,
		SomaMTTRMin: ind.SomaMTTRMin, MTTRN: ind.Amostra.IncidentesResolvidos,
		SLACumprido: ind.SLACumprido, SLAN: ind.Amostra.ComPrazoConcluido,
		FCRN: ind.Amostra.FCR, Concluidas: ind.Amostra.Concluidas, Incidentes: ind.Amostra.Incidentes,
		CSATSoma: resumo.Soma, CSATN: resumo.Total, Truncado: truncado,
	}
	return s.repo.GravarRetrato(ctx, s.pool, p)
}

// RunSerie mede o dia ao subir e de hora em hora.
func (s *Service) RunSerie(ctx context.Context) error {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := s.RegistrarSerie(ctx); err != nil && ctx.Err() == nil && s.logger != nil {
			s.logger.WarnContext(ctx, "atendimento: série de indicadores falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
