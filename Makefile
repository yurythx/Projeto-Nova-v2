.PHONY: dev up down logs build test lint format \
	deploy prod-seed-admin prod-ps prod-logs prod-limpeza prod-down iam-scope-report ia-modelo estrutura-gerar estrutura-aplicar \
	demo-keycloak demo-generate demo-seed demo-test demo-popular \
	migrate-up migrate-down migrate-status migrate-redo seed-admin \
	backend-shell frontend-shell rabbitmq-status clean \
	backend-build backend-test backend-lint backend-sec backend-format test-db-reset \
	frontend-build frontend-test frontend-lint frontend-format \
	infra-init infra-plan infra-apply infra-destroy infra-deploy \
	infra-segredos infra-segredos-migrar

ifneq (,$(wildcard .env))
include .env
export
endif

COMPOSE      := docker compose -f docker-compose.yml -f docker-compose.dev.yml
# Sem -f: respeita COMPOSE_FILE do .env (Keycloak de teste); padrão = docker-compose.yml.
COMPOSE_PROD := docker compose
GOOSE_DIR    := backend/migrations
DB_USER      ?= nova
DB_PASSWORD  ?= nova_pass
DB_NAME      ?= nova
HOST_DB_PORT ?= 5433
DB_DSN       ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(HOST_DB_PORT)/$(DB_NAME)?sslmode=disable

## --- Orquestração local ---

dev: ## Sobe todos os serviços em modo desenvolvimento (com overrides de dev)
	$(COMPOSE) up --build

up: ## Sobe todos os serviços em modo produção
	$(COMPOSE_PROD) up --build -d

## --- Servidor (produção, só docker-compose.yml — ver docs/DEPLOY.md) ---

deploy: ## git pull + gera .env/chave na 1ª vez + build + sobe e espera healthy
	./scripts/deploy.sh

iam-administradores: ## Mapeia o grupo do AD de administradores da plataforma (acesso total)
	$(COMPOSE_PROD) exec -T postgres sh -c 'psql -q -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < deploy/iam/administradores-ad.sql

iam-protocolo: ## Mapeia os grupos do AD do Protocolo ao perfil Protocolo
	$(COMPOSE_PROD) exec -T postgres sh -c 'psql -q -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < deploy/iam/protocolo-ad.sql

prod-seed-admin: ## Cria/reseta o admin local dentro do container (sem Go no host)
	$(COMPOSE_PROD) run --rm --no-deps --entrypoint ./seedadmin backend-api

prod-ps: ## Estado dos serviços de produção
	$(COMPOSE_PROD) ps

prod-limpeza: ## Libera disco: cache de build sem uso há 3 dias e imagens órfãs (nunca volumes)
	docker builder prune -a -f --filter until=72h
	docker image prune -f
	df -h /

prod-logs: ## Logs dos serviços de produção
	$(COMPOSE_PROD) logs -f --tail=200

prod-down: ## Para os serviços de produção (mantém os volumes)
	$(COMPOSE_PROD) down

iam-scope-report: ## Impacto da permissão com escopo (ADR 013) nas lotações e grupos atuais
	$(COMPOSE_PROD) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) < scripts/iam-scope-report.sql

estrutura-gerar: ## Regera deploy/estrutura a partir do export do AD (AD=docs/AD)
	node scripts/estrutura/ad-para-estrutura.mjs $${AD:-docs/AD}

estrutura-aplicar: ## Aplica (idempotente) a estrutura organizacional real no Postgres
	$(COMPOSE_PROD) exec -T postgres psql -q -v ON_ERROR_STOP=1 -U $(DB_USER) -d $(DB_NAME) < deploy/estrutura/estrutura.sql

## --- Ambiente de teste: Keycloak + dados fictícios (ver docs/DEPLOY.md) ---

demo-keycloak: ## Liga o Keycloak de teste, sobe a stack e aplica os dados fictícios
	./scripts/demo-keycloak.sh

demo-generate: ## Regera realm do Keycloak + SQL + CSV a partir de scripts/demo-data
	node scripts/demo-data/generate.mjs

demo-seed: ## Aplica (idempotente) a estrutura organizacional e os usuários fictícios no Postgres
	$(COMPOSE_PROD) exec -T postgres psql -q -v ON_ERROR_STOP=1 -U $(DB_USER) -d $(DB_NAME) < deploy/demo/seed-demo.sql

demo-test: ## Cenários ponta a ponta com os usuários fictícios (login real pelo Keycloak)
	python3 scripts/demo-data/cenarios/tramite.py
	python3 scripts/demo-data/cenarios/modulos.py
	python3 scripts/demo-data/cenarios/conteudo.py
	python3 scripts/demo-data/cenarios/colaboracao.py
	python3 scripts/demo-data/cenarios/tramite_documentos.py
	python3 scripts/demo-data/cenarios/integracoes.py
	python3 scripts/demo-data/cenarios/plataforma.py

demo-popular: ## Povoa todos os apps com dados fictícios realistas (uma vez)
	cd scripts/demo-data/cenarios && python3 popular.py

down: ## Para e remove todos os serviços
	$(COMPOSE) down

logs: ## Acompanha os logs de todos os serviços
	$(COMPOSE) logs -f

ia-modelo: ## Baixa (ou atualiza) o modelo IA_LOCAL_MODELO no serviço ia-local (overlay docker-compose.ia.yml)
	$(COMPOSE_PROD) run --rm ia-local-modelo

clean: ## Para os serviços e remove os volumes (DESTRÓI os dados locais)
	$(COMPOSE) down -v

## --- Build ---

build: backend-build frontend-build ## Compila os binários/artefatos do backend e do frontend

backend-build:
	cd backend && go build ./...

frontend-build:
	cd frontend && npm run build

## --- Testes ---

test: backend-test frontend-test ## Roda as suítes de teste do backend e do frontend

TEST_PG_CONTAINER ?= projeto-nova-test-pg
TEST_DB_NAME ?= nova_test
TEST_DB_USER ?= nova
TEST_DB_PASSWORD ?= nova_ci_password

test-db-reset: ## Recria o Postgres de teste vazio (template0) e aplica as migrations
	./scripts/test-db-reset.sh

backend-test: 
	cd backend && go test ./... -p 1

frontend-test:
	cd frontend && npm test

## --- Lint / format ---

lint: backend-lint frontend-lint ## Roda lint no backend e no frontend

backend-lint:
	cd backend && go vet ./...

backend-sec: ## SAST do backend (govulncheck + staticcheck + gosec) — mesmo conjunto do CI
	cd backend && govulncheck ./...
	cd backend && staticcheck ./...
	cd backend && gosec -quiet -exclude=G104 ./...

frontend-lint:
	cd frontend && npm run lint

format: backend-format frontend-format ## Formata o backend e o frontend

backend-format:
	cd backend && go fmt ./...

frontend-format:
	cd frontend && npm run format

## --- Migrations (Goose) ---

migrate-up: ## Aplica todas as migrations pendentes
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" up

migrate-down: ## Reverte a última migration
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" down

migrate-status: ## Mostra o status das migrations
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" status

migrate-redo: ## Exercita a reversibilidade: up -> down-to 0 -> up (toda migration precisa de Down válido)
	cd $(GOOSE_DIR) && goose postgres "$(DB_DSN)" up \
		&& goose postgres "$(DB_DSN)" down-to 0 \
		&& goose postgres "$(DB_DSN)" up \
		&& goose postgres "$(DB_DSN)" status

seed-admin: ## Cria/reseta o usuário admin local com senha ALEATÓRIA
	cd backend && DB_HOST=localhost DB_PORT=$(HOST_DB_PORT) DB_NAME=$(DB_NAME) DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) \
		go run ./cmd/seedadmin

new-module: ## Gera o esqueleto Clean Architecture de um novo módulo (Uso: make new-module NAME=contratos)
	@./scripts/create-module.sh $(NAME)

## --- Shells ---

backend-shell: ## Abre um shell no container backend-api em execução
	$(COMPOSE) exec backend-api sh

frontend-shell: ## Abre um shell no container frontend em execução
	$(COMPOSE) exec frontend sh

## --- RabbitMQ ---

rabbitmq-status: ## Mostra o status do nó/filas do RabbitMQ
	$(COMPOSE) exec rabbitmq rabbitmqctl status

## --- Infraestrutura Proxmox (IaC via Terraform & Ansible) ---

infra-init: ## Inicializa o Terraform na pasta deploy/infra/terraform
	cd deploy/infra/terraform && terraform init

infra-plan: ## Executa o planejamento da infraestrutura Proxmox
	cd deploy/infra/terraform && terraform plan

infra-apply: ## Provisiona a VM KVM no Proxmox (192.168.1.13) e gera o inventário
	cd deploy/infra/terraform && terraform apply

infra-destroy: ## Destroi a infraestrutura no Proxmox
	cd deploy/infra/terraform && terraform destroy

# Segredos do Ansible: secrets.vault.yml (cifrado, preferível) e, para quem
# ainda não migrou, secrets.yml e cloudflare.secrets.yml em texto. Nenhum
# entra no git: o repositório é público. A senha do cofre fica fora do
# repositório, em $(VAULT_PASS) (ou digite: VAULT_PASS= make infra-deploy).
VAULT_PASS ?= $(HOME)/.config/projeto-nova/vault-pass
VAULT_ARGS = $(if $(wildcard $(VAULT_PASS)),--vault-password-file $(VAULT_PASS),--ask-vault-pass)

infra-deploy: ## Docker + stack + backup + monitoramento na VM (Ansible; no Windows, pelo WSL)
	cd deploy/infra/ansible && ansible-galaxy collection install -r requirements.yml --force && \
		ansible-playbook site.yml \
			$$(for f in secrets.yml cloudflare.secrets.yml; do [ -f $$f ] && printf -- '-e @%s ' $$f; done) \
			$$([ -f secrets.vault.yml ] && printf -- '-e @secrets.vault.yml $(VAULT_ARGS)')

infra-segredos: ## Abre (ou cria) o cofre cifrado deploy/infra/ansible/secrets.vault.yml
	cd deploy/infra/ansible && if [ -f secrets.vault.yml ]; then \
		ansible-vault edit $(VAULT_ARGS) secrets.vault.yml; else \
		ansible-vault create $(VAULT_ARGS) secrets.vault.yml; fi

infra-segredos-migrar: ## Junta secrets.yml e cloudflare.secrets.yml no cofre cifrado e apaga os abertos
	cd deploy/infra/ansible && [ ! -f secrets.vault.yml ] && \
		cat $$(ls secrets.yml cloudflare.secrets.yml 2>/dev/null) > secrets.vault.yml && \
		ansible-vault encrypt $(VAULT_ARGS) secrets.vault.yml && \
		shred -u $$(ls secrets.yml cloudflare.secrets.yml 2>/dev/null) && \
		echo "Cofre criado: secrets.vault.yml (os arquivos abertos foram apagados)"
