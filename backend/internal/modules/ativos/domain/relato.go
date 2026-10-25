package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Relato é o que a própria máquina informa. Não descreve a rede.
type Relato struct {
	Identificador string
	Nome          string
	Tipo          string
	Descricao     string
	UnidadeID     uuid.UUID
	Inventario    *Inventario
}

// AplicarRelato cria ou atualiza um ativo de agente. Cadastro manual não
// é reescrito. Desativado e em manutenção conservam o estado decidido
// por uma pessoa.
func AplicarRelato(atual *Ativo, relato Relato) (Ativo, bool, error) {
	relato.Identificador = strings.TrimSpace(relato.Identificador)
	relato.Nome = strings.TrimSpace(relato.Nome)
	relato.Descricao = strings.TrimSpace(relato.Descricao)
	if relato.Identificador == "" || relato.UnidadeID == uuid.Nil {
		return Ativo{}, false, ErrEntradaInvalida
	}
	if relato.Nome == "" {
		relato.Nome = relato.Identificador
	}
	if relato.Tipo == "" {
		relato.Tipo = "hardware"
	}
	inventario, err := ValidarInventario(relato.Inventario)
	if err != nil {
		return Ativo{}, false, err
	}
	if inventario != nil {
		relato.Descricao = inventario.Resumo()
	}
	if atual == nil {
		criado, err := Novo(Entrada{
			Nome: relato.Nome, Tipo: relato.Tipo, Estado: "em_uso",
			Identificador: relato.Identificador, Descricao: relato.Descricao, UnidadeID: &relato.UnidadeID,
		})
		if err != nil {
			return Ativo{}, false, err
		}
		criado.Origem = OrigemAgente
		criado.Inventario = inventario
		return criado, true, nil
	}
	if atual.Origem != OrigemAgente {
		return Ativo{}, false, ErrCadastroManual
	}
	if !MesmaUnidade(atual.UnidadeID, &relato.UnidadeID) {
		return Ativo{}, false, ErrForaDoEscopo
	}
	estado := atual.Estado
	if estado != "desativado" && estado != "manutencao" {
		estado = "em_uso"
	}
	proximo, err := atual.Aplicar(Entrada{
		Nome: relato.Nome, Tipo: relato.Tipo, Estado: estado,
		Identificador: atual.Identificador, Descricao: relato.Descricao,
		ProdutoID: atual.ProdutoID, UnidadeID: atual.UnidadeID,
	})
	if err != nil {
		return Ativo{}, false, err
	}
	if inventario != nil {
		proximo.Inventario = inventario
	}
	proximo.UpdatedAt = time.Now().UTC()
	return proximo, !atual.Igual(proximo), nil
}
