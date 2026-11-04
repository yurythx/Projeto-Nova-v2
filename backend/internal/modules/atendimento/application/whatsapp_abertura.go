package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

const (
	textoMenu       = "Vamos abrir um chamado por aqui. O mesmo pedido pode ser feito na Central de Atendimento, com o seu login.\n\nEscolha o serviço."
	textoSemServico = "Não há serviço disponível para abrir por aqui. Use a Central de Atendimento, com o seu login."
	textoCancelado  = "A abertura foi cancelada. Quando quiser, escreva de novo. A Central de Atendimento continua disponível, com login."
	dicaCancelar    = "Para desistir, responda cancelar."
	idVerMais       = "mais"
)

func soTexto(s string) domain.RespostaWhatsApp {
	return domain.RespostaWhatsApp{Texto: s}
}

// abrirPeloWhatsApp conduz o roteiro quando a mensagem não é de um chamado
// que já existe. A Central, com login, abre o mesmo chamado por Abrir.
func (s *Service) abrirPeloWhatsApp(ctx context.Context, usuario uuid.UUID, texto, chave string) (domain.RespostaWhatsApp, bool, error) {
	rot, err := s.repo.RoteiroWhatsApp(ctx, s.pool, usuario)
	if err != nil && !errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.RespostaWhatsApp{}, false, err
	}
	tem := err == nil
	if tem && (rot.Etapa == domain.RoteiroFeito || time.Since(rot.UpdatedAt) > domain.ValidadeRoteiroWhatsApp) {
		if err := s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario); err != nil {
			return domain.RespostaWhatsApp{}, false, err
		}
		tem = false
	}
	if tem && !domain.ComandoNovo(texto) {
		r, err := s.comReservaWhatsApp(ctx, chave, func() (domain.RespostaWhatsApp, error) {
			return s.continuarAbertura(ctx, usuario, rot, texto)
		})
		return r, true, err
	}
	if domain.ComandoNovo(texto) {
		r, err := s.comReservaWhatsApp(ctx, chave, func() (domain.RespostaWhatsApp, error) {
			return s.iniciarAbertura(ctx, usuario, 0)
		})
		return r, true, err
	}
	abertos, _, err := s.repo.ListSolicitacoes(ctx, s.pool, domain.FiltroSolicitacao{Solicitante: &usuario, Limit: 2})
	if err != nil {
		return domain.RespostaWhatsApp{}, false, err
	}
	switch {
	case len(abertos) == 0:
		r, err := s.comReservaWhatsApp(ctx, chave, func() (domain.RespostaWhatsApp, error) {
			return s.iniciarAbertura(ctx, usuario, 0)
		})
		return r, true, err
	case len(abertos) > 1:
		return soTexto(respostaVarios + " Para abrir outro, responda novo."), true, nil
	default:
		return domain.RespostaWhatsApp{}, false, nil
	}
}

func (s *Service) comReservaWhatsApp(ctx context.Context, chave string, fn func() (domain.RespostaWhatsApp, error)) (domain.RespostaWhatsApp, error) {
	ok, err := s.repo.ReservarMensagemWhatsApp(ctx, s.pool, chave)
	if err != nil || !ok {
		return domain.RespostaWhatsApp{}, err
	}
	r, err := fn()
	if err != nil {
		_ = s.repo.LiberarMensagemWhatsApp(ctx, s.pool, chave)
	}
	return r, err
}

func (s *Service) iniciarAbertura(ctx context.Context, usuario uuid.UUID, pagina int) (domain.RespostaWhatsApp, error) {
	ofertas, err := s.Catalogo(ctx)
	if err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	if len(ofertas) == 0 {
		_ = s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario)
		return soTexto(textoSemServico), nil
	}
	visiveis, temMais, usada := domain.PaginaDeOfertas(ofertas, pagina)
	rot := domain.RoteiroWhatsApp{
		UsuarioID: usuario, Etapa: domain.RoteiroOferta, Pagina: usada, Respostas: map[string]string{},
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot); err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	return menuDeOfertas(visiveis, temMais), nil
}

func menuDeOfertas(visiveis []domain.Oferta, temMais bool) domain.RespostaWhatsApp {
	opcoes := make([]domain.OpcaoWhatsApp, 0, len(visiveis)+1)
	for _, o := range visiveis {
		titulo := o.Nome
		if o.SecretariaSigla != "" {
			titulo += " · " + o.SecretariaSigla
		}
		opcoes = append(opcoes, domain.OpcaoWhatsApp{ID: o.ID.String(), Titulo: titulo})
	}
	if temMais {
		opcoes = append(opcoes, domain.OpcaoWhatsApp{ID: idVerMais, Titulo: "Ver mais"})
	}
	return domain.RespostaWhatsApp{Texto: textoMenu, Opcoes: opcoes}
}

func (s *Service) continuarAbertura(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp, texto string) (domain.RespostaWhatsApp, error) {
	if domain.ComandoCancelar(texto) {
		if err := s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario); err != nil {
			return domain.RespostaWhatsApp{}, err
		}
		return soTexto(textoCancelado), nil
	}
	switch rot.Etapa {
	case domain.RoteiroOferta:
		return s.etapaOferta(ctx, usuario, rot, texto)
	case domain.RoteiroTitulo:
		return s.etapaTitulo(ctx, rot, texto)
	case domain.RoteiroDescricao:
		return s.etapaDescricao(ctx, usuario, rot, texto)
	case domain.RoteiroCampo:
		return s.etapaCampo(ctx, usuario, rot, texto)
	default:
		_ = s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario)
		return s.iniciarAbertura(ctx, usuario, 0)
	}
}

func (s *Service) etapaOferta(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp, texto string) (domain.RespostaWhatsApp, error) {
	ofertas, err := s.Catalogo(ctx)
	if err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	visiveis, temMais, usada := domain.PaginaDeOfertas(ofertas, rot.Pagina)
	rot.Pagina = usada
	menu := menuDeOfertas(visiveis, temMais)
	escolha, ok := domain.EscolherOpcao(texto, menu.Opcoes)
	if !ok {
		menu.Texto = "Não encontrei essa opção.\n\n" + textoMenu
		return menu, nil
	}
	if escolha.ID == idVerMais {
		return s.iniciarAbertura(ctx, usuario, rot.Pagina+1)
	}
	id, err := uuid.Parse(escolha.ID)
	if err != nil {
		menu.Texto = "Não encontrei essa opção.\n\n" + textoMenu
		return menu, nil
	}
	o, err := s.repo.GetOferta(ctx, s.pool, id)
	if err != nil || !o.Ativa {
		if err != nil && !errors.Is(err, domain.ErrNaoEncontrado) {
			return domain.RespostaWhatsApp{}, err
		}
		return s.iniciarAbertura(ctx, usuario, 0)
	}
	rot.Etapa = domain.RoteiroTitulo
	rot.OfertaID = &o.ID
	rot.UpdatedAt = time.Now().UTC()
	if err := s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot); err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	return soTexto(fmt.Sprintf("Serviço: %s.\nEscreva um título curto para o chamado (até 160 caracteres). %s", o.Nome, dicaCancelar)), nil
}

func (s *Service) etapaTitulo(ctx context.Context, rot domain.RoteiroWhatsApp, texto string) (domain.RespostaWhatsApp, error) {
	titulo := strings.TrimSpace(texto)
	if titulo == "" || utf8.RuneCountInString(titulo) > domain.NomeMax {
		return soTexto("O título precisa ter até 160 caracteres.\nEscreva um título curto para o chamado. " + dicaCancelar), nil
	}
	rot.Etapa = domain.RoteiroDescricao
	rot.Titulo = titulo
	rot.UpdatedAt = time.Now().UTC()
	if err := s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot); err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	return soTexto("Título: " + titulo + ".\nAgora descreva o que aconteceu. " + dicaCancelar), nil
}

func (s *Service) etapaDescricao(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp, texto string) (domain.RespostaWhatsApp, error) {
	descricao := strings.TrimSpace(texto)
	if descricao == "" || utf8.RuneCountInString(descricao) > domain.TextoMax {
		return soTexto("A descrição precisa ter até 4000 caracteres. Conte o que aconteceu. " + dicaCancelar), nil
	}
	rot.Descricao = descricao
	o, err := s.ofertaDoRoteiro(ctx, usuario, rot)
	if err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	if o.ID == uuid.Nil {
		return s.iniciarAbertura(ctx, usuario, 0)
	}
	if len(domain.CamposObrigatorios(o.Campos)) == 0 {
		return s.concluirAbertura(ctx, usuario, rot, o)
	}
	rot.Etapa = domain.RoteiroCampo
	rot.CampoIndice = 0
	rot.Respostas = map[string]string{}
	rot.UpdatedAt = time.Now().UTC()
	if err := s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot); err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	return perguntaDeCampo(domain.CamposObrigatorios(o.Campos)[0]), nil
}

func (s *Service) etapaCampo(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp, texto string) (domain.RespostaWhatsApp, error) {
	o, err := s.ofertaDoRoteiro(ctx, usuario, rot)
	if err != nil {
		return domain.RespostaWhatsApp{}, err
	}
	if o.ID == uuid.Nil {
		return s.iniciarAbertura(ctx, usuario, 0)
	}
	campos := domain.CamposObrigatorios(o.Campos)
	if rot.CampoIndice < 0 || rot.CampoIndice >= len(campos) {
		return s.concluirAbertura(ctx, usuario, rot, o)
	}
	c := campos[rot.CampoIndice]
	valor, ok := domain.RespostaDeCampo(c, texto)
	if !ok {
		r := perguntaDeCampo(c)
		r.Texto = "Não entendi a resposta.\n" + r.Texto
		return r, nil
	}
	if rot.Respostas == nil {
		rot.Respostas = map[string]string{}
	}
	rot.Respostas[c.Chave] = valor
	rot.CampoIndice++
	if rot.CampoIndice < len(campos) {
		rot.UpdatedAt = time.Now().UTC()
		if err := s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot); err != nil {
			return domain.RespostaWhatsApp{}, err
		}
		return perguntaDeCampo(campos[rot.CampoIndice]), nil
	}
	return s.concluirAbertura(ctx, usuario, rot, o)
}

func perguntaDeCampo(c domain.Campo) domain.RespostaWhatsApp {
	texto := c.Rotulo + ". " + dicaCancelar
	if c.Tipo == "numero" {
		texto = c.Rotulo + " (só números). " + dicaCancelar
	}
	if c.Tipo != "lista" || len(c.Opcoes) == 0 || len(c.Opcoes) > 10 {
		return soTexto(texto)
	}
	opcoes := make([]domain.OpcaoWhatsApp, len(c.Opcoes))
	for i, op := range c.Opcoes {
		opcoes[i] = domain.OpcaoWhatsApp{ID: op, Titulo: op}
	}
	return domain.RespostaWhatsApp{Texto: texto, Opcoes: opcoes}
}

func (s *Service) ofertaDoRoteiro(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp) (domain.Oferta, error) {
	if rot.OfertaID == nil {
		return domain.Oferta{}, nil
	}
	o, err := s.repo.GetOferta(ctx, s.pool, *rot.OfertaID)
	if errors.Is(err, domain.ErrNaoEncontrado) || (err == nil && !o.Ativa) {
		_ = s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario)
		return domain.Oferta{}, nil
	}
	return o, err
}

func (s *Service) concluirAbertura(ctx context.Context, usuario uuid.UUID, rot domain.RoteiroWhatsApp, o domain.Oferta) (domain.RespostaWhatsApp, error) {
	if _, err := domain.ValidarRespostas(o.Campos, rot.Respostas); err != nil {
		_ = s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario)
		return soTexto("Não consegui guardar as respostas deste serviço. Vamos começar de novo: escreva aqui, ou use a Central de Atendimento."), nil
	}
	sol, err := s.Abrir(ctx, auth.Identity{UserID: usuario}, AberturaEntrada{
		OfertaID: o.ID, Titulo: rot.Titulo, Descricao: rot.Descricao, Respostas: rot.Respostas,
	})
	if err != nil {
		if errors.Is(err, domain.ErrOfertaInativa) || errors.Is(err, domain.ErrOfertaInexistente) {
			return s.iniciarAbertura(ctx, usuario, 0)
		}
		if errors.Is(err, domain.ErrSolicitacaoInvalida) || errors.Is(err, domain.ErrEntradaInvalida) {
			rot.Etapa = domain.RoteiroTitulo
			rot.UpdatedAt = time.Now().UTC()
			_ = s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot)
			return soTexto("Não consegui abrir com esses dados. Escreva o título de novo. " + dicaCancelar), nil
		}
		return domain.RespostaWhatsApp{}, err
	}
	if err := s.repo.ApagarRoteiroWhatsApp(ctx, s.pool, usuario); err != nil {
		rot.Etapa = domain.RoteiroFeito
		rot.UpdatedAt = time.Now().UTC()
		_ = s.repo.SalvarRoteiroWhatsApp(ctx, s.pool, rot)
	}
	texto := fmt.Sprintf("Abrimos o chamado #%d (%s). A equipe responde por aqui e na Central de Atendimento.", sol.Numero, sol.Titulo)
	if len(o.Campos) > len(domain.CamposObrigatorios(o.Campos)) {
		texto += " Os dados opcionais você completa na Central, com o seu login."
	}
	texto += " Para abrir outro, responda novo."
	return soTexto(texto), nil
}
