package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

func (r *Repository) RoteiroWhatsApp(ctx context.Context, db database.DBTX, usuario uuid.UUID) (domain.RoteiroWhatsApp, error) {
	var rot domain.RoteiroWhatsApp
	var raw []byte
	err := db.QueryRow(ctx, `
		SELECT usuario_id, etapa, oferta_id, titulo, descricao, respostas, campo_indice, pagina, updated_at
		FROM atendimento_whatsapp_roteiro WHERE usuario_id = $1`, usuario).Scan(
		&rot.UsuarioID, &rot.Etapa, &rot.OfertaID, &rot.Titulo, &rot.Descricao, &raw, &rot.CampoIndice, &rot.Pagina, &rot.UpdatedAt)
	if err != nil {
		return domain.RoteiroWhatsApp{}, mapDB(err)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &rot.Respostas); err != nil {
			return domain.RoteiroWhatsApp{}, mapDB(err)
		}
	}
	if rot.Respostas == nil {
		rot.Respostas = map[string]string{}
	}
	return rot, nil
}

func (r *Repository) SalvarRoteiroWhatsApp(ctx context.Context, db database.DBTX, rot domain.RoteiroWhatsApp) error {
	if rot.Respostas == nil {
		rot.Respostas = map[string]string{}
	}
	raw, err := json.Marshal(rot.Respostas)
	if err != nil {
		return mapDB(err)
	}
	if rot.UpdatedAt.IsZero() {
		rot.UpdatedAt = time.Now().UTC()
	}
	_, err = db.Exec(ctx, `
		INSERT INTO atendimento_whatsapp_roteiro
			(usuario_id, etapa, oferta_id, titulo, descricao, respostas, campo_indice, pagina, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (usuario_id) DO UPDATE SET
			etapa = EXCLUDED.etapa,
			oferta_id = EXCLUDED.oferta_id,
			titulo = EXCLUDED.titulo,
			descricao = EXCLUDED.descricao,
			respostas = EXCLUDED.respostas,
			campo_indice = EXCLUDED.campo_indice,
			pagina = EXCLUDED.pagina,
			updated_at = EXCLUDED.updated_at`,
		rot.UsuarioID, rot.Etapa, rot.OfertaID, rot.Titulo, rot.Descricao, raw, rot.CampoIndice, rot.Pagina, rot.UpdatedAt)
	return mapDB(err)
}

func (r *Repository) ApagarRoteiroWhatsApp(ctx context.Context, db database.DBTX, usuario uuid.UUID) error {
	_, err := db.Exec(ctx, `DELETE FROM atendimento_whatsapp_roteiro WHERE usuario_id = $1`, usuario)
	return mapDB(err)
}

func (r *Repository) ReservarMensagemWhatsApp(ctx context.Context, db database.DBTX, messageID string) (bool, error) {
	tag, err := db.Exec(ctx, `INSERT INTO atendimento_whatsapp_mensagens (message_id) VALUES ($1) ON CONFLICT DO NOTHING`, messageID)
	if err != nil {
		return false, mapDB(err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) MensagemWhatsAppJaEntrou(ctx context.Context, db database.DBTX, messageID string) (bool, error) {
	var ja bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM atendimento_whatsapp_mensagens WHERE message_id = $1)`, messageID).Scan(&ja)
	return ja, mapDB(err)
}

func (r *Repository) LiberarMensagemWhatsApp(ctx context.Context, db database.DBTX, messageID string) error {
	_, err := db.Exec(ctx, `DELETE FROM atendimento_whatsapp_mensagens WHERE message_id = $1`, messageID)
	return mapDB(err)
}
