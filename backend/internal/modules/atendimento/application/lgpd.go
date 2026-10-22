package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/platform/database"
)

// ExportPersonalData devolve o que o atendimento guarda sobre o titular
// para o pacote de acesso e portabilidade (LGPD art. 18, II e V),
// inclusive as coordenadas das visitas ainda não apagadas. Nada guardado:
// nil.
func (s *Service) ExportPersonalData(ctx context.Context, db database.DBTX, user uuid.UUID) (any, error) {
	d, err := s.repo.DadosDoTitular(ctx, db, user)
	if err != nil {
		return nil, err
	}
	if len(d.Solicitacoes) == 0 && len(d.Comentarios) == 0 && len(d.Visitas) == 0 {
		return nil, nil
	}
	return d, nil
}

// ErasePersonalData atende a eliminação na transação que anonimiza a
// conta: as coordenadas das visitas saem já. Solicitação, comentário e
// visita ficam — são registro da prestação do serviço público — e passam
// a apontar para a conta anonimizada.
func (s *Service) ErasePersonalData(ctx context.Context, tx database.DBTX, user uuid.UUID) error {
	return s.repo.ApagarLocalizacoesDe(ctx, tx, user)
}
