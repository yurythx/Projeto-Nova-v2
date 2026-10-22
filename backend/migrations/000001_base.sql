-- Esquema base do Projeto-nova-v2.
--
-- Consolida o histórico de migrations do produto anterior (que tinha os
-- módulos Atlas e Acervo) no esquema atual. A aplicação nasce com banco
-- novo, então não há instalação a migrar (ADR 046). Gerado a partir de um
-- banco migrado (pg_dump --schema-only) e revisado; as regras que as
-- CHECKs e os gatilhos garantem estão em docs/REGRAS_DE_NEGOCIO.md.
--
-- Novas mudanças de esquema entram em migrations seguintes; esta não se
-- edita.

-- +goose Up
-- +goose StatementBegin
-- As funções SQL vêm antes das tabelas que leem (ordem do pg_dump): o
-- corpo só é conferido no uso.
SET LOCAL check_function_bodies = false;

CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;

CREATE FUNCTION public.audit_logs_chain() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_pos  BIGINT;
    v_hash TEXT;
BEGIN
    -- Chave fixa do lock da cadeia de auditoria do Nexus.
    PERFORM pg_advisory_xact_lock(7300101);
    SELECT chain_pos, hash INTO v_pos, v_hash
      FROM audit_logs ORDER BY chain_pos DESC LIMIT 1;

    NEW.chain_pos      := COALESCE(v_pos, 0) + 1;
    NEW.prev_hash      := COALESCE(v_hash, repeat('0', 64));
    -- Carimbo de tempo do SERVIDOR de banco, nunca o do cliente.
    NEW.created_at     := clock_timestamp();
    NEW.actor_roles    := COALESCE(NEW.actor_roles, '{}');
    NEW.entity_context := COALESCE(NEW.entity_context, '{}'::jsonb);
    NEW.metadata       := COALESCE(NEW.metadata, '{}'::jsonb);
    NEW.hash           := audit_row_digest(NEW);
    RETURN NEW;
END;
$$;

CREATE TABLE public.audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    actor_id uuid,
    action text NOT NULL,
    resource_type text,
    resource_id text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    correlation_id uuid,
    ip_address inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    actor_subject text,
    actor_roles text[] DEFAULT '{}'::text[] NOT NULL,
    user_agent text,
    entity_context jsonb DEFAULT '{}'::jsonb NOT NULL,
    diff_before jsonb,
    diff_after jsonb,
    chain_pos bigint NOT NULL,
    prev_hash text NOT NULL,
    hash text NOT NULL
);

CREATE FUNCTION public.audit_row_digest(p public.audit_logs) RETURNS text
    LANGUAGE sql STABLE
    AS $$
    SELECT encode(digest(jsonb_build_array(
        p.chain_pos,
        p.id,
        p.actor_id,
        p.actor_subject,
        to_jsonb(p.actor_roles),
        host(p.ip_address),
        p.user_agent,
        p.entity_context,
        p.action,
        p.resource_type,
        p.resource_id,
        p.diff_before,
        p.diff_after,
        p.metadata,
        p.correlation_id,
        to_char(p.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
        p.prev_hash
    )::text, 'sha256'), 'hex')
$$;

CREATE FUNCTION public.audit_verify_chain(p_from bigint DEFAULT 1, p_limit bigint DEFAULT NULL::bigint) RETURNS TABLE(checked bigint, valid boolean, first_invalid_pos bigint, reason text)
    LANGUAGE plpgsql STABLE
    AS $$
DECLARE
    r       audit_logs;
    v_prev  TEXT;
    v_count BIGINT := 0;
BEGIN
    IF p_from <= 1 THEN
        v_prev := repeat('0', 64);
    ELSE
        SELECT hash INTO v_prev FROM audit_logs WHERE chain_pos = p_from - 1;
        IF v_prev IS NULL THEN
            RETURN QUERY SELECT 0::BIGINT, false, p_from - 1, 'elo anterior ausente'::TEXT;
            RETURN;
        END IF;
    END IF;

    FOR r IN
        SELECT * FROM audit_logs WHERE chain_pos >= GREATEST(p_from, 1)
        ORDER BY chain_pos
        LIMIT p_limit
    LOOP
        v_count := v_count + 1;
        IF r.chain_pos <> GREATEST(p_from, 1) + v_count - 1 THEN
            RETURN QUERY SELECT v_count, false, r.chain_pos, 'lacuna na sequência da cadeia'::TEXT;
            RETURN;
        END IF;
        IF r.prev_hash <> v_prev THEN
            RETURN QUERY SELECT v_count, false, r.chain_pos, 'prev_hash não corresponde ao hash anterior'::TEXT;
            RETURN;
        END IF;
        IF r.hash <> audit_row_digest(r) THEN
            RETURN QUERY SELECT v_count, false, r.chain_pos, 'conteúdo alterado (hash divergente)'::TEXT;
            RETURN;
        END IF;
        v_prev := r.hash;
    END LOOP;

    RETURN QUERY SELECT v_count, true, NULL::BIGINT, NULL::TEXT;
END;
$$;

CREATE FUNCTION public.log_anonymous_consent_audit() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata, ip_address, user_agent)
    VALUES (NULL, 'lgpd_consent_given_anon', 'anonymous_consents', NEW.device_hash,
            jsonb_build_object('term_version', NEW.term_version),
            NEW.ip_address, NEW.user_agent);
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.log_data_subject_request_audit() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata, ip_address)
    VALUES (NEW.user_id, 'lgpd.data_subject_request.' || NEW.kind, 'data_subject_request', NEW.id::text,
            jsonb_build_object('kind', NEW.kind, 'status', NEW.status),
            NEW.requested_ip);
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.log_lgpd_consent_audit() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO audit_logs (actor_id, action, resource_type, resource_id, metadata, ip_address, user_agent)
    VALUES (NEW.user_id, 'lgpd_consent_given', 'user_consents', NEW.id::text,
            jsonb_build_object('term_version', NEW.term_version),
            NEW.ip_address, NEW.user_agent);
    RETURN NEW;
END;
$$;

CREATE FUNCTION public.nexus_scope_active(e uuid, u uuid, d uuid) RETURNS boolean
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$
    SELECT (e IS NULL OR EXISTS (SELECT 1 FROM entidades WHERE id = e AND ativo))
       AND (u IS NULL OR EXISTS (SELECT 1 FROM unidades WHERE id = u AND ativo))
       AND (d IS NULL OR EXISTS (SELECT 1 FROM departamentos WHERE id = d AND ativo))
$$;

CREATE FUNCTION public.nexus_search_tsquery(cfg regconfig, q text) RETURNS tsquery
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$
    SELECT websearch_to_tsquery(cfg, nexus_unaccent(q)) || COALESCE((
        SELECT to_tsquery(cfg, string_agg(quote_literal(t) || ':*', ' & '))
          FROM regexp_split_to_table(lower(nexus_unaccent(q)), '[^[:alnum:]]+') AS t
         WHERE length(t) >= 2
           AND q !~ '["-]'
           AND q !~* '\mor\M'
    ), ''::tsquery)
$$;

CREATE FUNCTION public.nexus_unaccent(txt text) RETURNS text
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
    SELECT public.unaccent('public.unaccent'::regdictionary, txt)
$$;

CREATE FUNCTION public.nexus_unidades_acima(u uuid) RETURNS uuid[]
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$
    SELECT CASE WHEN u IS NULL THEN '{}'::UUID[] ELSE (
        WITH RECURSIVE acima(id, parent_id, nivel) AS (
            SELECT id, parent_id, 0 FROM unidades WHERE id = u
            UNION ALL
            SELECT x.id, x.parent_id, a.nivel + 1 FROM unidades x JOIN acima a ON x.id = a.parent_id WHERE a.nivel < 64
        )
        SELECT COALESCE(array_agg(id), '{}'::UUID[]) FROM acima)
    END
$$;

CREATE FUNCTION public.nexus_unidades_cobertas(e uuid, u uuid, d uuid) RETURNS uuid[]
    LANGUAGE sql STABLE PARALLEL SAFE
    AS $$
    SELECT CASE
        WHEN d IS NOT NULL THEN '{}'::UUID[]
        WHEN u IS NOT NULL THEN (
            WITH RECURSIVE arvore(id) AS (
                SELECT u
                UNION
                SELECT x.id FROM unidades x JOIN arvore a ON x.parent_id = a.id
            )
            SELECT array_agg(id) FROM arvore)
        WHEN e IS NOT NULL THEN COALESCE((SELECT array_agg(id) FROM unidades WHERE entidade_id = e), '{}'::UUID[])
        ELSE '{}'::UUID[]
    END
$$;

CREATE FUNCTION public.nexus_valid_permissions(perms text[]) RETURNS boolean
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $_$
    SELECT COALESCE(bool_and(p ~ '^(\*|[a-z_]+:(\*|[a-z_]+))$'), true) FROM unnest(perms) AS p
$_$;

CREATE FUNCTION public.notify_system_modules_changed() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    PERFORM pg_notify('nexus_modules_channel', COALESCE(NEW.key, OLD.key));
    RETURN NULL;
END;
$$;

CREATE FUNCTION public.prevent_audit_logs_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: % is not allowed (attempted on id=%)',
        TG_OP,
        COALESCE(OLD.id, NEW.id);
END;
$$;

CREATE FUNCTION public.prevent_audit_logs_truncate() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: TRUNCATE is not allowed';
END;
$$;

CREATE FUNCTION public.prevent_audit_worm_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'audit_worm_exports is append-only: % não é permitido', TG_OP;
END;
$$;

CREATE FUNCTION public.prevent_tramite_movimentos_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- A exclusão em cascata do processo (ON DELETE CASCADE) é a única
    -- remoção admitida; o Nexus nunca apaga processo pela API.
    IF TG_OP = 'DELETE' AND NOT EXISTS (SELECT 1 FROM tramite_processos WHERE id = OLD.processo_id) THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'tramite_movimentos é append-only: % não é permitido', TG_OP;
END;
$$;

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE TABLE public.ad_group_mappings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    ad_group text NOT NULL,
    perfil_id uuid NOT NULL,
    entidade_id uuid,
    unidade_id uuid,
    departamento_id uuid,
    descricao text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL
);

CREATE TABLE public.anonymous_consents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    device_hash text NOT NULL,
    term_version text NOT NULL,
    ip_address inet,
    user_agent text,
    accepted_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.atendimento_comentarios (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    solicitacao_id uuid NOT NULL,
    autor_id uuid,
    texto text NOT NULL,
    interno boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT atendimento_comentarios_texto_chk CHECK (((char_length(btrim(texto)) >= 1) AND (char_length(btrim(texto)) <= 4000)))
);

CREATE TABLE public.atendimento_equipe_membros (
    equipe_id uuid NOT NULL,
    user_id uuid NOT NULL
);

CREATE TABLE public.atendimento_equipes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    unidade_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT atendimento_equipes_nome_chk CHECK (((char_length(btrim(nome)) >= 1) AND (char_length(btrim(nome)) <= 160)))
);

CREATE TABLE public.atendimento_evidencias (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    visita_id uuid NOT NULL,
    tipo text NOT NULL,
    object_key text NOT NULL,
    content_type text NOT NULL,
    tamanho bigint NOT NULL,
    autor_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT atendimento_evidencias_tipo_chk CHECK ((tipo = ANY (ARRAY['foto'::text, 'assinatura'::text])))
);

CREATE TABLE public.atendimento_feriados (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    unidade_id uuid,
    data date NOT NULL,
    nome text NOT NULL,
    tipo text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT atendimento_feriados_nome_chk CHECK (((char_length(btrim(nome)) >= 1) AND (char_length(btrim(nome)) <= 160))),
    CONSTRAINT atendimento_feriados_tipo_chk CHECK ((tipo = ANY (ARRAY['nacional'::text, 'estadual'::text, 'municipal'::text, 'ponto_facultativo'::text])))
);

CREATE TABLE public.atendimento_ofertas (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    produto_id uuid,
    unidade_id uuid,
    ativa boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tipo text DEFAULT 'requisicao'::text NOT NULL,
    CONSTRAINT atendimento_ofertas_nome_chk CHECK (((char_length(btrim(nome)) >= 1) AND (char_length(btrim(nome)) <= 160))),
    CONSTRAINT atendimento_ofertas_tipo_chk CHECK ((tipo = ANY (ARRAY['incidente'::text, 'requisicao'::text])))
);

CREATE TABLE public.atendimento_passagens (
    id bigint NOT NULL,
    solicitacao_id uuid NOT NULL,
    de text,
    para text NOT NULL,
    autor_id uuid,
    em timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE public.atendimento_passagens ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.atendimento_passagens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

CREATE TABLE public.atendimento_politicas (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    unidade_id uuid,
    fuso text DEFAULT 'America/Sao_Paulo'::text NOT NULL,
    turnos jsonb DEFAULT '[]'::jsonb NOT NULL,
    matriz jsonb DEFAULT '[]'::jsonb NOT NULL,
    metas jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.atendimento_solicitacoes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    numero bigint NOT NULL,
    tipo text NOT NULL,
    estado text NOT NULL,
    titulo text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    oferta_id uuid NOT NULL,
    produto_id uuid,
    unidade_id uuid,
    solicitante_id uuid NOT NULL,
    responsavel_id uuid,
    resolvida_em timestamp with time zone,
    encerrada_em timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    impacto smallint DEFAULT 2 NOT NULL,
    urgencia smallint DEFAULT 2 NOT NULL,
    prioridade text,
    meta_resposta_min integer,
    meta_solucao_min integer,
    CONSTRAINT atendimento_solicitacoes_descricao_chk CHECK ((char_length(descricao) <= 4000)),
    CONSTRAINT atendimento_solicitacoes_impacto_chk CHECK (((impacto >= 1) AND (impacto <= 3))),
    CONSTRAINT atendimento_solicitacoes_metas_chk CHECK ((((prioridade IS NULL) AND (meta_resposta_min IS NULL) AND (meta_solucao_min IS NULL)) OR ((prioridade IS NOT NULL) AND (meta_resposta_min > 0) AND (meta_solucao_min > 0)))),
    CONSTRAINT atendimento_solicitacoes_prioridade_chk CHECK ((prioridade = ANY (ARRAY['critica'::text, 'alta'::text, 'media'::text, 'baixa'::text]))),
    CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK ((((tipo = 'incidente'::text) AND (estado = ANY (ARRAY['aberto'::text, 'em_atendimento'::text, 'aguardando_servidor'::text, 'aguardando_terceiro'::text, 'resolvido'::text, 'encerrado'::text, 'cancelado'::text]))) OR ((tipo = 'requisicao'::text) AND (estado = ANY (ARRAY['aberto'::text, 'em_atendimento'::text, 'aguardando_servidor'::text, 'aguardando_terceiro'::text, 'atendido'::text, 'encerrado'::text, 'cancelado'::text]))))),
    CONSTRAINT atendimento_solicitacoes_titulo_chk CHECK (((char_length(btrim(titulo)) >= 1) AND (char_length(btrim(titulo)) <= 160))),
    CONSTRAINT atendimento_solicitacoes_urgencia_chk CHECK (((urgencia >= 1) AND (urgencia <= 3)))
);

ALTER TABLE public.atendimento_solicitacoes ALTER COLUMN numero ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.atendimento_solicitacoes_numero_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

CREATE TABLE public.atendimento_visitas (
    id uuid NOT NULL,
    solicitacao_id uuid NOT NULL,
    tecnico_id uuid NOT NULL,
    estado text DEFAULT 'em_andamento'::text NOT NULL,
    local text DEFAULT ''::text NOT NULL,
    base_legal text NOT NULL,
    destino_lat double precision,
    destino_lng double precision,
    destino_raio_m integer,
    inicio_em timestamp with time zone NOT NULL,
    inicio_lat double precision,
    inicio_lng double precision,
    inicio_precisao_m double precision,
    distancia_m double precision,
    fora_da_area boolean DEFAULT false NOT NULL,
    fim_em timestamp with time zone,
    fim_lat double precision,
    fim_lng double precision,
    fim_precisao_m double precision,
    assinante_nome text DEFAULT ''::text NOT NULL,
    observacao text DEFAULT ''::text NOT NULL,
    localizacao_expira_em timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT atendimento_visitas_assinante_chk CHECK ((char_length(assinante_nome) <= 160)),
    CONSTRAINT atendimento_visitas_destino_chk CHECK ((((destino_lat IS NULL) AND (destino_lng IS NULL) AND (destino_raio_m IS NULL)) OR (((destino_lat >= ('-90'::integer)::double precision) AND (destino_lat <= (90)::double precision)) AND ((destino_lng >= ('-180'::integer)::double precision) AND (destino_lng <= (180)::double precision)) AND ((destino_raio_m >= 10) AND (destino_raio_m <= 50000))))),
    CONSTRAINT atendimento_visitas_estado_chk CHECK ((estado = ANY (ARRAY['em_andamento'::text, 'concluida'::text, 'cancelada'::text]))),
    CONSTRAINT atendimento_visitas_fim_chk CHECK (((estado = 'em_andamento'::text) = (fim_em IS NULL))),
    CONSTRAINT atendimento_visitas_local_chk CHECK ((char_length(local) <= 300)),
    CONSTRAINT atendimento_visitas_obs_chk CHECK ((char_length(observacao) <= 4000))
);

CREATE TABLE public.ativos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    tipo text NOT NULL,
    estado text NOT NULL,
    identificador text DEFAULT ''::text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    produto_id uuid,
    unidade_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ativos_descricao_chk CHECK ((char_length(descricao) <= 4000)),
    CONSTRAINT ativos_estado_chk CHECK ((estado = ANY (ARRAY['planejado'::text, 'em_uso'::text, 'manutencao'::text, 'desativado'::text]))),
    CONSTRAINT ativos_identificador_chk CHECK ((char_length(identificador) <= 80)),
    CONSTRAINT ativos_nome_chk CHECK (((char_length(btrim(nome)) >= 1) AND (char_length(btrim(nome)) <= 160))),
    CONSTRAINT ativos_tipo_chk CHECK ((tipo = ANY (ARRAY['hardware'::text, 'software'::text, 'servico'::text, 'rede'::text, 'outro'::text])))
);

CREATE TABLE public.audit_worm_exports (
    day date NOT NULL,
    row_count integer NOT NULL,
    sha256 text NOT NULL,
    prev_sha256 text DEFAULT ''::text NOT NULL,
    object_key text NOT NULL,
    exported_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.blog_post_publico (
    post_id uuid NOT NULL,
    entidade_id uuid,
    unidade_id uuid,
    CONSTRAINT blog_post_publico_um_alvo CHECK ((num_nonnulls(entidade_id, unidade_id) = 1))
);

CREATE TABLE public.blog_posts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug text NOT NULL,
    title text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    body text DEFAULT ''::text NOT NULL,
    cover_object_key text DEFAULT ''::text NOT NULL,
    kind text DEFAULT 'noticia'::text NOT NULL,
    status text DEFAULT 'draft'::text NOT NULL,
    pinned boolean DEFAULT false NOT NULL,
    author_id uuid,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    search tsvector GENERATED ALWAYS AS (((setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(title)), 'A'::"char") || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(summary)), 'B'::"char")) || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(body)), 'C'::"char"))) STORED,
    unidade_id uuid,
    CONSTRAINT blog_posts_kind_check CHECK ((kind = ANY (ARRAY['noticia'::text, 'comunicado'::text]))),
    CONSTRAINT blog_posts_published_has_date CHECK (((status <> 'published'::text) OR (published_at IS NOT NULL))),
    CONSTRAINT blog_posts_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text])))
);

CREATE TABLE public.branding_settings (
    id text DEFAULT 'default'::text NOT NULL,
    app_name text DEFAULT 'Projeto-nova-v2'::text NOT NULL,
    app_description text DEFAULT 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.'::text NOT NULL,
    org_name text DEFAULT 'Projeto-nova-v2'::text NOT NULL,
    logo_url text DEFAULT ''::text NOT NULL,
    favicon_url text DEFAULT ''::text NOT NULL,
    support_email text DEFAULT ''::text NOT NULL,
    support_phone text DEFAULT ''::text NOT NULL,
    support_hours text DEFAULT ''::text NOT NULL,
    tokens jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT branding_settings_singleton CHECK ((id = 'default'::text))
);

CREATE TABLE public.calendar_event_publico (
    event_id uuid NOT NULL,
    entidade_id uuid,
    unidade_id uuid,
    CONSTRAINT calendar_event_publico_um_alvo CHECK ((num_nonnulls(entidade_id, unidade_id) = 1))
);

CREATE TABLE public.calendar_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    location text DEFAULT ''::text NOT NULL,
    room_id uuid,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone NOT NULL,
    all_day boolean DEFAULT false NOT NULL,
    visibility text DEFAULT 'internal'::text NOT NULL,
    status text DEFAULT 'confirmed'::text NOT NULL,
    organizer_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT calendar_events_range_check CHECK ((ends_at > starts_at)),
    CONSTRAINT calendar_events_status_check CHECK ((status = ANY (ARRAY['confirmed'::text, 'cancelled'::text]))),
    CONSTRAINT calendar_events_visibility_check CHECK ((visibility = ANY (ARRAY['public'::text, 'internal'::text, 'private'::text])))
);

CREATE TABLE public.calendar_rooms (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    location text DEFAULT ''::text NOT NULL,
    capacity integer DEFAULT 0 NOT NULL,
    resources text[] DEFAULT '{}'::text[] NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unidade_id uuid,
    CONSTRAINT calendar_rooms_capacity_check CHECK ((capacity >= 0))
);

CREATE TABLE public.catalog_services (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug text NOT NULL,
    title text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    category text DEFAULT 'Geral'::text NOT NULL,
    audience text DEFAULT ''::text NOT NULL,
    requirements text[] DEFAULT '{}'::text[] NOT NULL,
    steps text[] DEFAULT '{}'::text[] NOT NULL,
    channels jsonb DEFAULT '[]'::jsonb NOT NULL,
    sla text DEFAULT ''::text NOT NULL,
    cost text DEFAULT 'Gratuito'::text NOT NULL,
    icon text DEFAULT 'file-text'::text NOT NULL,
    responsible_unidade_id uuid,
    status text DEFAULT 'draft'::text NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    search tsvector GENERATED ALWAYS AS (((setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(title)), 'A'::"char") || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(((category || ' '::text) || summary))), 'B'::"char")) || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(description)), 'C'::"char"))) STORED,
    CONSTRAINT catalog_services_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'archived'::text])))
);

CREATE TABLE public.contact_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    protocol text NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    subject text NOT NULL,
    category text DEFAULT 'duvida'::text NOT NULL,
    message text NOT NULL,
    consent_at timestamp with time zone NOT NULL,
    ip_address inet,
    user_agent text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'new'::text NOT NULL,
    assigned_to uuid,
    notes text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unidade_id uuid,
    CONSTRAINT contact_messages_category_check CHECK ((category = ANY (ARRAY['duvida'::text, 'sugestao'::text, 'reclamacao'::text, 'elogio'::text, 'outro'::text]))),
    CONSTRAINT contact_messages_status_check CHECK ((status = ANY (ARRAY['new'::text, 'in_progress'::text, 'answered'::text, 'archived'::text])))
);

CREATE SEQUENCE public.contact_protocol_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE public.data_subject_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    kind text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    result_object_key text,
    detail text DEFAULT ''::text NOT NULL,
    requested_ip inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    CONSTRAINT data_subject_requests_kind_check CHECK ((kind = ANY (ARRAY['export'::text, 'erasure'::text]))),
    CONSTRAINT data_subject_requests_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'completed'::text, 'failed'::text, 'rejected'::text])))
);

CREATE TABLE public.departamentos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    unidade_id uuid NOT NULL,
    nome text NOT NULL,
    sigla text DEFAULT ''::text NOT NULL,
    slug text NOT NULL,
    ad_group text DEFAULT ''::text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    telefone text DEFAULT ''::text NOT NULL,
    ativo boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.directory_profiles (
    user_id uuid NOT NULL,
    job_title text DEFAULT ''::text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    extension text DEFAULT ''::text NOT NULL,
    unidade_id uuid,
    departamento_id uuid,
    bio text DEFAULT ''::text NOT NULL,
    visible boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.egress_deliveries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    target_id uuid NOT NULL,
    event_id uuid NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    last_status_code integer,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    delivered_at timestamp with time zone,
    CONSTRAINT egress_deliveries_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'delivered'::text, 'failed'::text, 'dead'::text])))
);

CREATE TABLE public.egress_targets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    kind text DEFAULT 'webhook'::text NOT NULL,
    url text NOT NULL,
    secret_encrypted text DEFAULT ''::text NOT NULL,
    event_patterns text[] DEFAULT ARRAY['#'::text] NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_by text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT egress_targets_kind_check CHECK ((kind = ANY (ARRAY['webhook'::text, 'n8n'::text, 'zabbix'::text, 'grafana'::text])))
);

CREATE TABLE public.entidades (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    sigla text DEFAULT ''::text NOT NULL,
    slug text NOT NULL,
    documento text DEFAULT ''::text NOT NULL,
    ativo boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.example_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'ACTIVE'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.feature_flags (
    key text NOT NULL,
    enabled boolean NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text
);

CREATE TABLE public.files_folder_acl (
    folder_id uuid NOT NULL,
    subject_type text NOT NULL,
    subject text NOT NULL,
    can_write boolean DEFAULT false NOT NULL,
    granted_by text DEFAULT ''::text NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT files_folder_acl_subject_type_check CHECK ((subject_type = ANY (ARRAY['everyone'::text, 'user'::text, 'perfil'::text, 'ad_group'::text, 'unidade'::text, 'departamento'::text])))
);

CREATE TABLE public.files_folders (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    owner_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unidade_id uuid,
    CONSTRAINT files_folders_name_check CHECK ((((char_length(name) >= 1) AND (char_length(name) <= 200)) AND (name !~ '[/\\]'::text))),
    CONSTRAINT files_folders_parent_not_self CHECK (((parent_id IS NULL) OR (parent_id <> id)))
);

CREATE TABLE public.files_objects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    folder_id uuid NOT NULL,
    name text NOT NULL,
    object_key text NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    content_type text DEFAULT 'application/octet-stream'::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    owner_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    search tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, public.nexus_unaccent(name))) STORED,
    CONSTRAINT files_objects_name_check CHECK ((((char_length(name) >= 1) AND (char_length(name) <= 255)) AND (name !~ '[/\\]'::text))),
    CONSTRAINT files_objects_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'ready'::text])))
);

CREATE TABLE public.ia_conexoes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    provedor text NOT NULL,
    endpoint text NOT NULL,
    modelo text NOT NULL,
    chave_cifrada text DEFAULT ''::text NOT NULL,
    externo boolean NOT NULL,
    timeout_segundos integer DEFAULT 30 NOT NULL,
    teste_ok boolean,
    teste_em timestamp with time zone,
    teste_latencia_ms integer,
    teste_erro text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT ia_conexoes_check CHECK ((((teste_ok IS NULL) = (teste_em IS NULL)) AND ((teste_ok IS NULL) = (teste_latencia_ms IS NULL)))),
    CONSTRAINT ia_conexoes_endpoint_check CHECK ((endpoint ~ '^https?://'::text)),
    CONSTRAINT ia_conexoes_modelo_check CHECK (((char_length(modelo) >= 1) AND (char_length(modelo) <= 120))),
    CONSTRAINT ia_conexoes_nome_check CHECK (((char_length(nome) >= 1) AND (char_length(nome) <= 80))),
    CONSTRAINT ia_conexoes_provedor_check CHECK ((provedor = ANY (ARRAY['ollama'::text, 'openai'::text, 'azure'::text, 'gemini'::text, 'anthropic'::text, 'groq'::text, 'openrouter'::text, 'mistral'::text, 'compativel'::text]))),
    CONSTRAINT ia_conexoes_timeout_segundos_check CHECK (((timeout_segundos >= 1) AND (timeout_segundos <= 300)))
);

CREATE TABLE public.ia_uso (
    funcao text NOT NULL,
    principal_id uuid,
    reserva_id uuid,
    mascarar_dados_pessoais boolean DEFAULT true NOT NULL,
    externo_autorizado_por text DEFAULT ''::text NOT NULL,
    externo_autorizado_em timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT ia_uso_check CHECK (((reserva_id IS NULL) OR ((principal_id IS NOT NULL) AND (reserva_id <> principal_id)))),
    CONSTRAINT ia_uso_funcao_check CHECK ((funcao = 'plataforma.ia'::text))
);

CREATE TABLE public.idempotency_keys (
    key text NOT NULL,
    request_hash text NOT NULL,
    status text DEFAULT 'processing'::text NOT NULL,
    response_status integer,
    response_body bytea,
    content_type text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT idempotency_keys_status_check CHECK ((status = ANY (ARRAY['processing'::text, 'completed'::text, 'failed'::text])))
);

CREATE TABLE public.integrations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    status text DEFAULT 'unknown'::text NOT NULL,
    last_check_at timestamp with time zone,
    last_success_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT integrations_status_check CHECK ((status = ANY (ARRAY['unknown'::text, 'online'::text, 'offline'::text, 'degraded'::text, 'disabled'::text])))
);

CREATE TABLE public.jobs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type text NOT NULL,
    status text DEFAULT 'queued'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    result jsonb,
    error text,
    correlation_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    CONSTRAINT jobs_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'processing'::text, 'completed'::text, 'failed'::text, 'dead_letter'::text])))
);

CREATE TABLE public.keycloak_settings (
    id text DEFAULT 'default'::text NOT NULL,
    issuer_url text DEFAULT ''::text NOT NULL,
    realm text DEFAULT ''::text NOT NULL,
    client_id text DEFAULT ''::text NOT NULL,
    client_secret_encrypted text DEFAULT ''::text NOT NULL,
    audience text DEFAULT ''::text NOT NULL,
    frontend_client_id text DEFAULT ''::text NOT NULL,
    frontend_client_secret_encrypted text DEFAULT ''::text NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT ''::text NOT NULL,
    CONSTRAINT keycloak_settings_singleton CHECK ((id = 'default'::text))
);

CREATE TABLE public.mercurio_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    room_id uuid NOT NULL,
    author_id uuid NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    edited_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT mercurio_messages_body_len CHECK (((char_length(body) >= 1) AND (char_length(body) <= 4000)))
);

CREATE TABLE public.mercurio_reads (
    room_id uuid NOT NULL,
    user_id uuid NOT NULL,
    last_read_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.mercurio_room_members (
    room_id uuid NOT NULL,
    user_id uuid NOT NULL,
    joined_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.mercurio_rooms (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    kind text NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    ad_group text DEFAULT ''::text NOT NULL,
    departamento_id uuid,
    dm_key text,
    archived boolean DEFAULT false NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mercurio_rooms_department_scope CHECK (((kind <> 'department'::text) OR (ad_group <> ''::text) OR (departamento_id IS NOT NULL))),
    CONSTRAINT mercurio_rooms_direct_has_key CHECK (((kind <> 'direct'::text) OR (dm_key IS NOT NULL))),
    CONSTRAINT mercurio_rooms_kind_check CHECK ((kind = ANY (ARRAY['global'::text, 'department'::text, 'direct'::text])))
);

CREATE TABLE public.mudancas (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    numero bigint NOT NULL,
    titulo text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    justificativa text DEFAULT ''::text NOT NULL,
    tipo text NOT NULL,
    risco text NOT NULL,
    estado text DEFAULT 'rascunho'::text NOT NULL,
    plano_reversao text DEFAULT ''::text NOT NULL,
    inicio_planejado timestamp with time zone,
    fim_planejado timestamp with time zone,
    resultado text DEFAULT ''::text NOT NULL,
    executada_inicio timestamp with time zone,
    executada_fim timestamp with time zone,
    produto_id uuid,
    unidade_id uuid,
    solicitante_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mudancas_estado_chk CHECK ((estado = ANY (ARRAY['rascunho'::text, 'em_avaliacao'::text, 'aprovada'::text, 'rejeitada'::text, 'em_execucao'::text, 'concluida'::text, 'falhou'::text, 'cancelada'::text]))),
    CONSTRAINT mudancas_janela_chk CHECK (((inicio_planejado IS NULL) OR (fim_planejado IS NULL) OR (fim_planejado > inicio_planejado))),
    CONSTRAINT mudancas_risco_chk CHECK ((risco = ANY (ARRAY['baixo'::text, 'medio'::text, 'alto'::text]))),
    CONSTRAINT mudancas_textos_chk CHECK (((char_length(descricao) <= 4000) AND (char_length(justificativa) <= 4000) AND (char_length(plano_reversao) <= 4000) AND (char_length(resultado) <= 4000))),
    CONSTRAINT mudancas_tipo_chk CHECK ((tipo = ANY (ARRAY['padrao'::text, 'normal'::text, 'emergencial'::text]))),
    CONSTRAINT mudancas_titulo_chk CHECK (((char_length(btrim(titulo)) >= 1) AND (char_length(btrim(titulo)) <= 160)))
);

CREATE TABLE public.mudancas_aprovacoes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mudanca_id uuid NOT NULL,
    aprovador_id uuid,
    decisao text NOT NULL,
    parecer text DEFAULT ''::text NOT NULL,
    em timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mudancas_aprovacoes_decisao_chk CHECK ((decisao = ANY (ARRAY['aprovada'::text, 'rejeitada'::text, 'automatica'::text]))),
    CONSTRAINT mudancas_aprovacoes_parecer_chk CHECK ((char_length(parecer) <= 4000))
);

ALTER TABLE public.mudancas ALTER COLUMN numero ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.mudancas_numero_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

CREATE TABLE public.mudancas_vinculos (
    mudanca_id uuid NOT NULL,
    tipo text NOT NULL,
    alvo_id uuid NOT NULL,
    rotulo text NOT NULL,
    vinculado_por uuid,
    vinculado_em timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mudancas_vinculos_tipo_chk CHECK ((tipo = ANY (ARRAY['problema'::text, 'incidente'::text, 'ativo'::text])))
);

CREATE TABLE public.notificacoes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    modulo text NOT NULL,
    titulo text NOT NULL,
    mensagem text DEFAULT ''::text NOT NULL,
    link text DEFAULT ''::text NOT NULL,
    chave text NOT NULL,
    lida_em timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notificacoes_chave_check CHECK (((char_length(chave) >= 1) AND (char_length(chave) <= 200))),
    CONSTRAINT notificacoes_link_check CHECK (((link = ''::text) OR (link ~ '^/[^/\\]'::text))),
    CONSTRAINT notificacoes_mensagem_check CHECK ((char_length(mensagem) <= 1000)),
    CONSTRAINT notificacoes_modulo_check CHECK (((char_length(modulo) >= 1) AND (char_length(modulo) <= 40))),
    CONSTRAINT notificacoes_titulo_check CHECK (((char_length(titulo) >= 1) AND (char_length(titulo) <= 200)))
);

CREATE TABLE public.notificacoes_preferencias (
    user_id uuid NOT NULL,
    modulo text NOT NULL,
    ativo boolean NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.outbox_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    payload jsonb NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    last_error text,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT outbox_events_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'published'::text, 'failed'::text])))
);

CREATE TABLE public.perfis (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug text NOT NULL,
    nome text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    permissoes text[] DEFAULT '{}'::text[] NOT NULL,
    sistema boolean DEFAULT false NOT NULL,
    ativo boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_perfis_permissoes_formato CHECK (public.nexus_valid_permissions(permissoes))
);

CREATE TABLE public.problemas (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    numero bigint NOT NULL,
    titulo text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    estado text DEFAULT 'registrado'::text NOT NULL,
    causa_raiz text DEFAULT ''::text NOT NULL,
    contorno text DEFAULT ''::text NOT NULL,
    produto_id uuid,
    unidade_id uuid,
    criado_por uuid,
    resolvido_em timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT problemas_erro_conhecido_chk CHECK (((estado <> 'erro_conhecido'::text) OR (btrim(contorno) <> ''::text))),
    CONSTRAINT problemas_estado_chk CHECK ((estado = ANY (ARRAY['registrado'::text, 'em_analise'::text, 'erro_conhecido'::text, 'resolvido'::text, 'encerrado'::text]))),
    CONSTRAINT problemas_resolvido_chk CHECK (((estado <> ALL (ARRAY['resolvido'::text, 'encerrado'::text])) OR (btrim(causa_raiz) <> ''::text))),
    CONSTRAINT problemas_textos_chk CHECK (((char_length(descricao) <= 4000) AND (char_length(causa_raiz) <= 4000) AND (char_length(contorno) <= 4000))),
    CONSTRAINT problemas_titulo_chk CHECK (((char_length(btrim(titulo)) >= 1) AND (char_length(btrim(titulo)) <= 160)))
);

CREATE TABLE public.problemas_ativos (
    problema_id uuid NOT NULL,
    ativo_id uuid NOT NULL,
    vinculado_por uuid,
    vinculado_em timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.problemas_incidentes (
    problema_id uuid NOT NULL,
    solicitacao_id uuid NOT NULL,
    numero bigint NOT NULL,
    titulo text NOT NULL,
    vinculado_por uuid,
    vinculado_em timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE public.problemas ALTER COLUMN numero ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.problemas_numero_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);

CREATE TABLE public.processed_events (
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.produtos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    nome text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    estagio text NOT NULL,
    utilidade text DEFAULT ''::text NOT NULL,
    garantia text DEFAULT ''::text NOT NULL,
    unidade_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT produtos_estagio_chk CHECK ((estagio = ANY (ARRAY['descoberta'::text, 'desenho'::text, 'aquisicao'::text, 'construcao'::text, 'transicao'::text, 'operacao'::text, 'entrega'::text, 'suporte'::text]))),
    CONSTRAINT produtos_nome_chk CHECK (((char_length(btrim(nome)) >= 1) AND (char_length(btrim(nome)) <= 160)))
);

CREATE TABLE public.rate_limit_buckets (
    bucket text NOT NULL,
    key text NOT NULL,
    window_start timestamp with time zone NOT NULL,
    count integer DEFAULT 0 NOT NULL
);

CREATE TABLE public.signum_challenges (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    envelope_id uuid NOT NULL,
    user_id uuid NOT NULL,
    nonce_hash character(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.signum_envelopes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    document_sha256 character(64) NOT NULL,
    source_module text DEFAULT ''::text NOT NULL,
    source_ref text DEFAULT ''::text NOT NULL,
    sequential boolean DEFAULT false NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT signum_envelopes_hash_format CHECK ((document_sha256 ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT signum_envelopes_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'completed'::text, 'refused'::text, 'cancelled'::text])))
);

CREATE TABLE public.signum_signers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    envelope_id uuid NOT NULL,
    user_id uuid NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    signed_at timestamp with time zone,
    signature_hash character(64),
    method text DEFAULT ''::text NOT NULL,
    ip_address inet,
    user_agent text DEFAULT ''::text NOT NULL,
    reason text DEFAULT ''::text NOT NULL,
    CONSTRAINT signum_signers_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'signed'::text, 'refused'::text])))
);

CREATE TABLE public.system_modules (
    key text NOT NULL,
    enabled boolean NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_by text DEFAULT 'system'::text NOT NULL
);

CREATE TABLE public.tramite_acessos (
    processo_id uuid NOT NULL,
    user_id uuid NOT NULL,
    granted_by uuid NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.tramite_documentos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    processo_id uuid NOT NULL,
    tipo text DEFAULT 'despacho'::text NOT NULL,
    titulo text NOT NULL,
    origem text NOT NULL,
    conteudo text DEFAULT ''::text NOT NULL,
    object_key text DEFAULT ''::text NOT NULL,
    content_type text DEFAULT ''::text NOT NULL,
    size_bytes bigint DEFAULT 0 NOT NULL,
    sha256 character(64),
    status text DEFAULT 'rascunho'::text NOT NULL,
    envelope_id uuid,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tramite_documentos_origem_check CHECK ((origem = ANY (ARRAY['redigido'::text, 'anexo'::text]))),
    CONSTRAINT tramite_documentos_status_check CHECK ((status = ANY (ARRAY['rascunho'::text, 'aguardando_assinatura'::text, 'assinado'::text, 'cancelado'::text])))
);

CREATE TABLE public.tramite_movimentos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    processo_id uuid NOT NULL,
    acao text NOT NULL,
    de_unidade_id uuid,
    para_unidade_id uuid,
    despacho text DEFAULT ''::text NOT NULL,
    actor_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tramite_movimentos_acao_check CHECK ((acao = ANY (ARRAY['abertura'::text, 'tramitacao'::text, 'conclusao'::text, 'arquivamento'::text, 'reabertura'::text, 'documento'::text, 'assinatura_solicitada'::text, 'assinatura_concluida'::text, 'acesso_concedido'::text, 'acesso_revogado'::text])))
);

CREATE TABLE public.tramite_numeracao (
    ano integer NOT NULL,
    ultimo integer DEFAULT 0 NOT NULL
);

CREATE TABLE public.tramite_processos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    numero text NOT NULL,
    tipo_id uuid NOT NULL,
    assunto text NOT NULL,
    interessado text DEFAULT ''::text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    sigilo text DEFAULT 'publico'::text NOT NULL,
    status text DEFAULT 'aberto'::text NOT NULL,
    unidade_origem_id uuid NOT NULL,
    unidade_atual_id uuid NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    concluido_at timestamp with time zone,
    search tsvector GENERATED ALWAYS AS (((setweight(to_tsvector('simple'::regconfig, numero), 'A'::"char") || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(assunto)), 'A'::"char")) || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(((interessado || ' '::text) || descricao))), 'B'::"char"))) STORED,
    CONSTRAINT tramite_processos_sigilo_check CHECK ((sigilo = ANY (ARRAY['publico'::text, 'restrito'::text, 'sigiloso'::text]))),
    CONSTRAINT tramite_processos_status_check CHECK ((status = ANY (ARRAY['aberto'::text, 'em_tramitacao'::text, 'concluido'::text, 'arquivado'::text])))
);

CREATE TABLE public.tramite_tipos (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    slug text NOT NULL,
    nome text NOT NULL,
    descricao text DEFAULT ''::text NOT NULL,
    ativo boolean DEFAULT true NOT NULL
);

CREATE TABLE public.unidades (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    entidade_id uuid NOT NULL,
    parent_id uuid,
    nome text NOT NULL,
    sigla text DEFAULT ''::text NOT NULL,
    slug text NOT NULL,
    ad_group text DEFAULT ''::text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    telefone text DEFAULT ''::text NOT NULL,
    endereco text DEFAULT ''::text NOT NULL,
    ativo boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ck_unidades_parent_not_self CHECK (((parent_id IS NULL) OR (parent_id <> id)))
);

CREATE TABLE public.user_consents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    term_version text NOT NULL,
    ip_address inet,
    user_agent text,
    accepted_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE public.user_scopes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    perfil_id uuid NOT NULL,
    entidade_id uuid,
    unidade_id uuid,
    departamento_id uuid,
    principal boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by text DEFAULT ''::text NOT NULL
);

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    keycloak_subject text,
    username text NOT NULL,
    email text NOT NULL,
    display_name text DEFAULT ''::text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    password_hash text,
    roles text[] DEFAULT '{}'::text[] NOT NULL,
    failed_login_attempts integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone,
    groups text[] DEFAULT '{}'::text[] NOT NULL,
    ad_synced_at timestamp with time zone,
    CONSTRAINT users_has_auth_method CHECK (((keycloak_subject IS NOT NULL) OR (password_hash IS NOT NULL)))
);

CREATE TABLE public.wiki_page_publico (
    page_id uuid NOT NULL,
    entidade_id uuid,
    unidade_id uuid,
    CONSTRAINT wiki_page_publico_um_alvo CHECK ((num_nonnulls(entidade_id, unidade_id) = 1))
);

CREATE TABLE public.wiki_pages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    parent_id uuid,
    slug text NOT NULL,
    title text NOT NULL,
    body text DEFAULT ''::text NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    search tsvector GENERATED ALWAYS AS ((setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(title)), 'A'::"char") || setweight(to_tsvector('portuguese'::regconfig, public.nexus_unaccent(body)), 'C'::"char"))) STORED,
    unidade_id uuid,
    CONSTRAINT wiki_pages_parent_not_self CHECK (((parent_id IS NULL) OR (parent_id <> id)))
);

CREATE TABLE public.wiki_revisions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    page_id uuid NOT NULL,
    version integer NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    edited_by uuid NOT NULL,
    edited_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY public.ad_group_mappings
    ADD CONSTRAINT ad_group_mappings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.anonymous_consents
    ADD CONSTRAINT anonymous_consents_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.anonymous_consents
    ADD CONSTRAINT anonymous_consents_unique UNIQUE (device_hash, term_version);

ALTER TABLE ONLY public.atendimento_comentarios
    ADD CONSTRAINT atendimento_comentarios_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_equipe_membros
    ADD CONSTRAINT atendimento_equipe_membros_pkey PRIMARY KEY (equipe_id, user_id);

ALTER TABLE ONLY public.atendimento_equipes
    ADD CONSTRAINT atendimento_equipes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_evidencias
    ADD CONSTRAINT atendimento_evidencias_object_key_key UNIQUE (object_key);

ALTER TABLE ONLY public.atendimento_evidencias
    ADD CONSTRAINT atendimento_evidencias_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_feriados
    ADD CONSTRAINT atendimento_feriados_dia_uq UNIQUE NULLS NOT DISTINCT (unidade_id, data);

ALTER TABLE ONLY public.atendimento_feriados
    ADD CONSTRAINT atendimento_feriados_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_ofertas
    ADD CONSTRAINT atendimento_ofertas_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_passagens
    ADD CONSTRAINT atendimento_passagens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_politicas
    ADD CONSTRAINT atendimento_politicas_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_politicas
    ADD CONSTRAINT atendimento_politicas_unidade_uq UNIQUE NULLS NOT DISTINCT (unidade_id);

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_numero_key UNIQUE (numero);

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.atendimento_visitas
    ADD CONSTRAINT atendimento_visitas_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.ativos
    ADD CONSTRAINT ativos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.audit_worm_exports
    ADD CONSTRAINT audit_worm_exports_pkey PRIMARY KEY (day);

ALTER TABLE ONLY public.blog_post_publico
    ADD CONSTRAINT blog_post_publico_unico UNIQUE NULLS NOT DISTINCT (post_id, entidade_id, unidade_id);

ALTER TABLE ONLY public.blog_posts
    ADD CONSTRAINT blog_posts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.blog_posts
    ADD CONSTRAINT blog_posts_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.branding_settings
    ADD CONSTRAINT branding_settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.calendar_event_publico
    ADD CONSTRAINT calendar_event_publico_unico UNIQUE NULLS NOT DISTINCT (event_id, entidade_id, unidade_id);

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_no_room_overlap EXCLUDE USING gist (room_id WITH =, tstzrange(starts_at, ends_at, '[)'::text) WITH &&) WHERE (((room_id IS NOT NULL) AND (status = 'confirmed'::text)));

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.calendar_rooms
    ADD CONSTRAINT calendar_rooms_name_key UNIQUE (name);

ALTER TABLE ONLY public.calendar_rooms
    ADD CONSTRAINT calendar_rooms_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.catalog_services
    ADD CONSTRAINT catalog_services_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.catalog_services
    ADD CONSTRAINT catalog_services_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.contact_messages
    ADD CONSTRAINT contact_messages_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.contact_messages
    ADD CONSTRAINT contact_messages_protocol_key UNIQUE (protocol);

ALTER TABLE ONLY public.data_subject_requests
    ADD CONSTRAINT data_subject_requests_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.departamentos
    ADD CONSTRAINT departamentos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.directory_profiles
    ADD CONSTRAINT directory_profiles_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY public.egress_deliveries
    ADD CONSTRAINT egress_deliveries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.egress_targets
    ADD CONSTRAINT egress_targets_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.entidades
    ADD CONSTRAINT entidades_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.entidades
    ADD CONSTRAINT entidades_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.example_items
    ADD CONSTRAINT example_items_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (key);

ALTER TABLE ONLY public.files_folder_acl
    ADD CONSTRAINT files_folder_acl_pkey PRIMARY KEY (folder_id, subject_type, subject);

ALTER TABLE ONLY public.files_folders
    ADD CONSTRAINT files_folders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.files_objects
    ADD CONSTRAINT files_objects_object_key_key UNIQUE (object_key);

ALTER TABLE ONLY public.files_objects
    ADD CONSTRAINT files_objects_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.ia_conexoes
    ADD CONSTRAINT ia_conexoes_nome_key UNIQUE (nome);

ALTER TABLE ONLY public.ia_conexoes
    ADD CONSTRAINT ia_conexoes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.ia_uso
    ADD CONSTRAINT ia_uso_pkey PRIMARY KEY (funcao);

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (key);

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.keycloak_settings
    ADD CONSTRAINT keycloak_settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mercurio_messages
    ADD CONSTRAINT mercurio_messages_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mercurio_reads
    ADD CONSTRAINT mercurio_reads_pkey PRIMARY KEY (room_id, user_id);

ALTER TABLE ONLY public.mercurio_room_members
    ADD CONSTRAINT mercurio_room_members_pkey PRIMARY KEY (room_id, user_id);

ALTER TABLE ONLY public.mercurio_rooms
    ADD CONSTRAINT mercurio_rooms_dm_key_key UNIQUE (dm_key);

ALTER TABLE ONLY public.mercurio_rooms
    ADD CONSTRAINT mercurio_rooms_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mudancas_aprovacoes
    ADD CONSTRAINT mudancas_aprovacoes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mudancas
    ADD CONSTRAINT mudancas_numero_key UNIQUE (numero);

ALTER TABLE ONLY public.mudancas
    ADD CONSTRAINT mudancas_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.mudancas_vinculos
    ADD CONSTRAINT mudancas_vinculos_pkey PRIMARY KEY (mudanca_id, tipo, alvo_id);

ALTER TABLE ONLY public.notificacoes
    ADD CONSTRAINT notificacoes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.notificacoes_preferencias
    ADD CONSTRAINT notificacoes_preferencias_pkey PRIMARY KEY (user_id, modulo);

ALTER TABLE ONLY public.outbox_events
    ADD CONSTRAINT outbox_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.perfis
    ADD CONSTRAINT perfis_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.perfis
    ADD CONSTRAINT perfis_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.problemas_ativos
    ADD CONSTRAINT problemas_ativos_pkey PRIMARY KEY (problema_id, ativo_id);

ALTER TABLE ONLY public.problemas_incidentes
    ADD CONSTRAINT problemas_incidentes_pkey PRIMARY KEY (problema_id, solicitacao_id);

ALTER TABLE ONLY public.problemas
    ADD CONSTRAINT problemas_numero_key UNIQUE (numero);

ALTER TABLE ONLY public.problemas
    ADD CONSTRAINT problemas_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.processed_events
    ADD CONSTRAINT processed_events_pkey PRIMARY KEY (consumer, event_id);

ALTER TABLE ONLY public.produtos
    ADD CONSTRAINT produtos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.rate_limit_buckets
    ADD CONSTRAINT rate_limit_buckets_pkey PRIMARY KEY (bucket, key);

ALTER TABLE ONLY public.signum_challenges
    ADD CONSTRAINT signum_challenges_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.signum_envelopes
    ADD CONSTRAINT signum_envelopes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.signum_signers
    ADD CONSTRAINT signum_signers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.system_modules
    ADD CONSTRAINT system_modules_pkey PRIMARY KEY (key);

ALTER TABLE ONLY public.tramite_acessos
    ADD CONSTRAINT tramite_acessos_pkey PRIMARY KEY (processo_id, user_id);

ALTER TABLE ONLY public.tramite_documentos
    ADD CONSTRAINT tramite_documentos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.tramite_movimentos
    ADD CONSTRAINT tramite_movimentos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.tramite_numeracao
    ADD CONSTRAINT tramite_numeracao_pkey PRIMARY KEY (ano);

ALTER TABLE ONLY public.tramite_processos
    ADD CONSTRAINT tramite_processos_numero_key UNIQUE (numero);

ALTER TABLE ONLY public.tramite_processos
    ADD CONSTRAINT tramite_processos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.tramite_tipos
    ADD CONSTRAINT tramite_tipos_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.tramite_tipos
    ADD CONSTRAINT tramite_tipos_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.unidades
    ADD CONSTRAINT unidades_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.departamentos
    ADD CONSTRAINT uq_departamentos_unidade_slug UNIQUE (unidade_id, slug);

ALTER TABLE ONLY public.egress_deliveries
    ADD CONSTRAINT uq_egress_deliveries_target_event UNIQUE (target_id, event_id);

ALTER TABLE ONLY public.notificacoes
    ADD CONSTRAINT uq_notificacoes_chave UNIQUE (user_id, chave);

ALTER TABLE ONLY public.signum_signers
    ADD CONSTRAINT uq_signum_signers_envelope_user UNIQUE (envelope_id, user_id);

ALTER TABLE ONLY public.unidades
    ADD CONSTRAINT uq_unidades_entidade_slug UNIQUE (entidade_id, slug);

ALTER TABLE ONLY public.wiki_revisions
    ADD CONSTRAINT uq_wiki_revisions_page_version UNIQUE (page_id, version);

ALTER TABLE ONLY public.user_consents
    ADD CONSTRAINT user_consents_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.user_consents
    ADD CONSTRAINT user_consents_version_unique UNIQUE (user_id, term_version);

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.wiki_page_publico
    ADD CONSTRAINT wiki_page_publico_unico UNIQUE NULLS NOT DISTINCT (page_id, entidade_id, unidade_id);

ALTER TABLE ONLY public.wiki_pages
    ADD CONSTRAINT wiki_pages_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.wiki_pages
    ADD CONSTRAINT wiki_pages_slug_key UNIQUE (slug);

ALTER TABLE ONLY public.wiki_revisions
    ADD CONSTRAINT wiki_revisions_pkey PRIMARY KEY (id);

CREATE INDEX idx_ad_group_mappings_group ON public.ad_group_mappings USING btree (lower(ad_group));

CREATE INDEX idx_anonymous_consents_accepted_at ON public.anonymous_consents USING btree (accepted_at DESC);

CREATE INDEX idx_atendimento_comentarios_solicitacao ON public.atendimento_comentarios USING btree (solicitacao_id, created_at);

CREATE INDEX idx_atendimento_equipes_unidade ON public.atendimento_equipes USING btree (unidade_id);

CREATE INDEX idx_atendimento_evidencias_visita ON public.atendimento_evidencias USING btree (visita_id, created_at);

CREATE INDEX idx_atendimento_feriados_data ON public.atendimento_feriados USING btree (data);

CREATE INDEX idx_atendimento_ofertas_unidade ON public.atendimento_ofertas USING btree (unidade_id);

CREATE INDEX idx_atendimento_passagens_solicitacao ON public.atendimento_passagens USING btree (solicitacao_id, em);

CREATE INDEX idx_atendimento_solicitacoes_fila ON public.atendimento_solicitacoes USING btree (unidade_id, estado, created_at);

CREATE INDEX idx_atendimento_solicitacoes_prazo ON public.atendimento_solicitacoes USING btree (estado) WHERE ((prioridade IS NOT NULL) AND (estado = ANY (ARRAY['aberto'::text, 'em_atendimento'::text])));

CREATE INDEX idx_atendimento_solicitacoes_responsavel ON public.atendimento_solicitacoes USING btree (responsavel_id, estado) WHERE (responsavel_id IS NOT NULL);

CREATE INDEX idx_atendimento_solicitacoes_solicitante ON public.atendimento_solicitacoes USING btree (solicitante_id, created_at DESC);

CREATE INDEX idx_atendimento_visitas_andamento ON public.atendimento_visitas USING btree (estado) WHERE (estado = 'em_andamento'::text);

CREATE INDEX idx_atendimento_visitas_expira ON public.atendimento_visitas USING btree (localizacao_expira_em) WHERE ((inicio_lat IS NOT NULL) OR (fim_lat IS NOT NULL) OR (destino_lat IS NOT NULL));

CREATE INDEX idx_atendimento_visitas_solicitacao ON public.atendimento_visitas USING btree (solicitacao_id, inicio_em);

CREATE INDEX idx_ativos_produto ON public.ativos USING btree (produto_id) WHERE (produto_id IS NOT NULL);

CREATE INDEX idx_ativos_unidade ON public.ativos USING btree (unidade_id, nome);

CREATE INDEX idx_audit_logs_action ON public.audit_logs USING btree (action);

CREATE INDEX idx_audit_logs_actor_id ON public.audit_logs USING btree (actor_id);

CREATE UNIQUE INDEX idx_audit_logs_chain_pos ON public.audit_logs USING btree (chain_pos);

CREATE INDEX idx_audit_logs_correlation_id ON public.audit_logs USING btree (correlation_id);

CREATE INDEX idx_audit_logs_created_at ON public.audit_logs USING btree (created_at DESC);

CREATE INDEX idx_audit_logs_resource ON public.audit_logs USING btree (resource_type, resource_id);

CREATE INDEX idx_audit_worm_exports_exported_at ON public.audit_worm_exports USING btree (exported_at DESC);

CREATE INDEX idx_blog_posts_search ON public.blog_posts USING gin (search);

CREATE INDEX idx_blog_posts_status_published ON public.blog_posts USING btree (status, published_at DESC);

CREATE INDEX idx_blog_posts_unidade ON public.blog_posts USING btree (unidade_id) WHERE (unidade_id IS NOT NULL);

CREATE INDEX idx_calendar_events_organizer ON public.calendar_events USING btree (organizer_id);

CREATE INDEX idx_calendar_events_range ON public.calendar_events USING gist (tstzrange(starts_at, ends_at, '[)'::text));

CREATE INDEX idx_calendar_rooms_unidade ON public.calendar_rooms USING btree (unidade_id) WHERE (unidade_id IS NOT NULL);

CREATE INDEX idx_catalog_services_search ON public.catalog_services USING gin (search);

CREATE INDEX idx_catalog_services_status ON public.catalog_services USING btree (status, category, "position");

CREATE INDEX idx_contact_messages_status_created ON public.contact_messages USING btree (status, created_at DESC);

CREATE INDEX idx_contact_messages_unidade ON public.contact_messages USING btree (unidade_id, created_at DESC) WHERE (unidade_id IS NOT NULL);

CREATE INDEX idx_data_subject_requests_created_at ON public.data_subject_requests USING btree (created_at DESC);

CREATE INDEX idx_data_subject_requests_status ON public.data_subject_requests USING btree (status) WHERE (status = ANY (ARRAY['pending'::text, 'processing'::text]));

CREATE INDEX idx_data_subject_requests_user_id ON public.data_subject_requests USING btree (user_id);

CREATE INDEX idx_departamentos_unidade ON public.departamentos USING btree (unidade_id);

CREATE INDEX idx_directory_profiles_departamento ON public.directory_profiles USING btree (departamento_id);

CREATE INDEX idx_directory_profiles_unidade ON public.directory_profiles USING btree (unidade_id);

CREATE INDEX idx_egress_deliveries_due ON public.egress_deliveries USING btree (next_attempt_at) WHERE (status = ANY (ARRAY['pending'::text, 'failed'::text]));

CREATE INDEX idx_egress_deliveries_target_created ON public.egress_deliveries USING btree (target_id, created_at DESC);

CREATE INDEX idx_example_items_created_at ON public.example_items USING btree (created_at DESC);

CREATE INDEX idx_files_folders_unidade ON public.files_folders USING btree (unidade_id) WHERE (unidade_id IS NOT NULL);

CREATE INDEX idx_files_objects_folder ON public.files_objects USING btree (folder_id, name);

CREATE INDEX idx_files_objects_pending ON public.files_objects USING btree (created_at) WHERE (status = 'pending'::text);

CREATE INDEX idx_files_objects_search ON public.files_objects USING gin (search);

CREATE INDEX idx_idempotency_keys_updated_at ON public.idempotency_keys USING btree (updated_at);

CREATE UNIQUE INDEX idx_integrations_key ON public.integrations USING btree (key);

CREATE INDEX idx_integrations_type ON public.integrations USING btree (type);

CREATE INDEX idx_jobs_correlation_id ON public.jobs USING btree (correlation_id);

CREATE INDEX idx_jobs_created_at ON public.jobs USING btree (created_at DESC);

CREATE INDEX idx_jobs_status ON public.jobs USING btree (status);

CREATE INDEX idx_jobs_type ON public.jobs USING btree (type);

CREATE INDEX idx_jobs_type_status ON public.jobs USING btree (type, status);

CREATE INDEX idx_mercurio_messages_room_created ON public.mercurio_messages USING btree (room_id, created_at DESC);

CREATE INDEX idx_mercurio_room_members_user ON public.mercurio_room_members USING btree (user_id);

CREATE INDEX idx_mercurio_rooms_ad_group ON public.mercurio_rooms USING btree (lower(ad_group)) WHERE (kind = 'department'::text);

CREATE INDEX idx_mercurio_rooms_kind ON public.mercurio_rooms USING btree (kind) WHERE (NOT archived);

CREATE INDEX idx_mudancas_aprovacoes_mudanca ON public.mudancas_aprovacoes USING btree (mudanca_id, em);

CREATE INDEX idx_mudancas_janela ON public.mudancas USING btree (inicio_planejado) WHERE (estado = ANY (ARRAY['aprovada'::text, 'em_execucao'::text]));

CREATE INDEX idx_mudancas_unidade ON public.mudancas USING btree (unidade_id, estado, created_at);

CREATE INDEX idx_mudancas_vinculos_alvo ON public.mudancas_vinculos USING btree (tipo, alvo_id);

CREATE INDEX idx_notificacoes_caixa ON public.notificacoes USING btree (user_id, created_at DESC);

CREATE INDEX idx_notificacoes_nao_lidas ON public.notificacoes USING btree (user_id) WHERE (lida_em IS NULL);

CREATE INDEX idx_outbox_events_aggregate ON public.outbox_events USING btree (aggregate_type, aggregate_id);

CREATE INDEX idx_outbox_events_correlation_id ON public.outbox_events USING btree (((payload ->> 'correlation_id'::text)));

CREATE INDEX idx_outbox_events_pending ON public.outbox_events USING btree (next_attempt_at, created_at) WHERE (status = 'pending'::text);

CREATE INDEX idx_problemas_erro_conhecido ON public.problemas USING btree (unidade_id) WHERE (estado = 'erro_conhecido'::text);

CREATE INDEX idx_problemas_incidentes_solicitacao ON public.problemas_incidentes USING btree (solicitacao_id);

CREATE INDEX idx_problemas_unidade ON public.problemas USING btree (unidade_id, estado, created_at);

CREATE INDEX idx_processed_events_processed_at ON public.processed_events USING btree (processed_at);

CREATE INDEX idx_produtos_nome ON public.produtos USING btree (nome);

CREATE INDEX idx_produtos_unidade ON public.produtos USING btree (unidade_id);

CREATE INDEX idx_rate_limit_buckets_window_start ON public.rate_limit_buckets USING btree (window_start);

CREATE INDEX idx_signum_challenges_lookup ON public.signum_challenges USING btree (envelope_id, user_id) WHERE (used_at IS NULL);

CREATE INDEX idx_signum_envelopes_source ON public.signum_envelopes USING btree (source_module, source_ref);

CREATE INDEX idx_signum_signers_user_status ON public.signum_signers USING btree (user_id, status);

CREATE INDEX idx_tramite_documentos_processo ON public.tramite_documentos USING btree (processo_id, created_at);

CREATE INDEX idx_tramite_movimentos_processo ON public.tramite_movimentos USING btree (processo_id, created_at);

CREATE INDEX idx_tramite_processos_created ON public.tramite_processos USING btree (created_at DESC);

CREATE INDEX idx_tramite_processos_search ON public.tramite_processos USING gin (search);

CREATE INDEX idx_tramite_processos_unidade_atual ON public.tramite_processos USING btree (unidade_atual_id, status);

CREATE INDEX idx_unidades_entidade ON public.unidades USING btree (entidade_id);

CREATE INDEX idx_unidades_parent ON public.unidades USING btree (parent_id);

CREATE INDEX idx_user_consents_accepted_at ON public.user_consents USING btree (accepted_at DESC);

CREATE INDEX idx_user_consents_user_id ON public.user_consents USING btree (user_id);

CREATE INDEX idx_user_scopes_user ON public.user_scopes USING btree (user_id);

CREATE INDEX idx_users_active ON public.users USING btree (active);

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

CREATE INDEX idx_users_groups ON public.users USING gin (groups);

CREATE UNIQUE INDEX idx_users_keycloak_subject ON public.users USING btree (keycloak_subject);

CREATE INDEX idx_users_search ON public.users USING gin (public.nexus_unaccent(lower(((((display_name || ' '::text) || username) || ' '::text) || email))) public.gin_trgm_ops);

CREATE UNIQUE INDEX idx_users_username ON public.users USING btree (username);

CREATE INDEX idx_wiki_pages_parent ON public.wiki_pages USING btree (parent_id, "position", title);

CREATE INDEX idx_wiki_pages_search ON public.wiki_pages USING gin (search);

CREATE INDEX idx_wiki_pages_unidade ON public.wiki_pages USING btree (unidade_id) WHERE (unidade_id IS NOT NULL);

CREATE UNIQUE INDEX uq_ad_group_mappings ON public.ad_group_mappings USING btree (lower(ad_group), perfil_id, COALESCE(entidade_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(unidade_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(departamento_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE UNIQUE INDEX uq_atendimento_visitas_aberta ON public.atendimento_visitas USING btree (solicitacao_id, tecnico_id) WHERE (estado = 'em_andamento'::text);

CREATE UNIQUE INDEX uq_ativos_identificador ON public.ativos USING btree (identificador) WHERE (identificador <> ''::text);

CREATE UNIQUE INDEX uq_files_folders_name ON public.files_folders USING btree (COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name));

CREATE UNIQUE INDEX uq_tramite_documentos_envelope ON public.tramite_documentos USING btree (envelope_id) WHERE (envelope_id IS NOT NULL);

CREATE UNIQUE INDEX uq_user_scopes ON public.user_scopes USING btree (user_id, perfil_id, COALESCE(entidade_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(unidade_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(departamento_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE TRIGGER trg_anonymous_consents_audit AFTER INSERT ON public.anonymous_consents FOR EACH ROW EXECUTE FUNCTION public.log_anonymous_consent_audit();

CREATE TRIGGER trg_atendimento_equipes_updated BEFORE UPDATE ON public.atendimento_equipes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_atendimento_ofertas_updated BEFORE UPDATE ON public.atendimento_ofertas FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_atendimento_politicas_updated BEFORE UPDATE ON public.atendimento_politicas FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_atendimento_solicitacoes_updated BEFORE UPDATE ON public.atendimento_solicitacoes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_atendimento_visitas_updated BEFORE UPDATE ON public.atendimento_visitas FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_ativos_updated BEFORE UPDATE ON public.ativos FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_audit_logs_chain BEFORE INSERT ON public.audit_logs FOR EACH ROW EXECUTE FUNCTION public.audit_logs_chain();

CREATE TRIGGER trg_audit_logs_immutable BEFORE DELETE OR UPDATE ON public.audit_logs FOR EACH ROW EXECUTE FUNCTION public.prevent_audit_logs_mutation();

CREATE TRIGGER trg_audit_logs_immutable_truncate BEFORE TRUNCATE ON public.audit_logs FOR EACH STATEMENT EXECUTE FUNCTION public.prevent_audit_logs_truncate();

CREATE TRIGGER trg_audit_worm_exports_immutable BEFORE DELETE OR UPDATE ON public.audit_worm_exports FOR EACH ROW EXECUTE FUNCTION public.prevent_audit_worm_mutation();

CREATE TRIGGER trg_blog_posts_set_updated_at BEFORE UPDATE ON public.blog_posts FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_calendar_events_set_updated_at BEFORE UPDATE ON public.calendar_events FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_calendar_rooms_set_updated_at BEFORE UPDATE ON public.calendar_rooms FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_catalog_services_set_updated_at BEFORE UPDATE ON public.catalog_services FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_contact_messages_set_updated_at BEFORE UPDATE ON public.contact_messages FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_data_subject_requests_audit AFTER INSERT ON public.data_subject_requests FOR EACH ROW EXECUTE FUNCTION public.log_data_subject_request_audit();

CREATE TRIGGER trg_data_subject_requests_set_updated_at BEFORE UPDATE ON public.data_subject_requests FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_departamentos_set_updated_at BEFORE UPDATE ON public.departamentos FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_directory_profiles_set_updated_at BEFORE UPDATE ON public.directory_profiles FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_egress_targets_set_updated_at BEFORE UPDATE ON public.egress_targets FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_entidades_set_updated_at BEFORE UPDATE ON public.entidades FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_files_folders_set_updated_at BEFORE UPDATE ON public.files_folders FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_files_objects_set_updated_at BEFORE UPDATE ON public.files_objects FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_integrations_set_updated_at BEFORE UPDATE ON public.integrations FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_mercurio_rooms_set_updated_at BEFORE UPDATE ON public.mercurio_rooms FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_mudancas_updated BEFORE UPDATE ON public.mudancas FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_perfis_set_updated_at BEFORE UPDATE ON public.perfis FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_problemas_updated BEFORE UPDATE ON public.problemas FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_produtos_updated BEFORE UPDATE ON public.produtos FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_system_modules_notify AFTER INSERT OR DELETE OR UPDATE ON public.system_modules FOR EACH ROW EXECUTE FUNCTION public.notify_system_modules_changed();

CREATE TRIGGER trg_tramite_documentos_set_updated_at BEFORE UPDATE ON public.tramite_documentos FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_tramite_movimentos_immutable BEFORE DELETE OR UPDATE ON public.tramite_movimentos FOR EACH ROW EXECUTE FUNCTION public.prevent_tramite_movimentos_mutation();

CREATE TRIGGER trg_tramite_processos_set_updated_at BEFORE UPDATE ON public.tramite_processos FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_unidades_set_updated_at BEFORE UPDATE ON public.unidades FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TRIGGER trg_user_consents_audit AFTER INSERT ON public.user_consents FOR EACH ROW EXECUTE FUNCTION public.log_lgpd_consent_audit();

CREATE TRIGGER trg_users_set_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

ALTER TABLE ONLY public.ad_group_mappings
    ADD CONSTRAINT ad_group_mappings_departamento_id_fkey FOREIGN KEY (departamento_id) REFERENCES public.departamentos(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.ad_group_mappings
    ADD CONSTRAINT ad_group_mappings_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.ad_group_mappings
    ADD CONSTRAINT ad_group_mappings_perfil_id_fkey FOREIGN KEY (perfil_id) REFERENCES public.perfis(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.ad_group_mappings
    ADD CONSTRAINT ad_group_mappings_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.atendimento_comentarios
    ADD CONSTRAINT atendimento_comentarios_autor_id_fkey FOREIGN KEY (autor_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_comentarios
    ADD CONSTRAINT atendimento_comentarios_solicitacao_id_fkey FOREIGN KEY (solicitacao_id) REFERENCES public.atendimento_solicitacoes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_equipe_membros
    ADD CONSTRAINT atendimento_equipe_membros_equipe_id_fkey FOREIGN KEY (equipe_id) REFERENCES public.atendimento_equipes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_equipe_membros
    ADD CONSTRAINT atendimento_equipe_membros_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_equipes
    ADD CONSTRAINT atendimento_equipes_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_evidencias
    ADD CONSTRAINT atendimento_evidencias_autor_id_fkey FOREIGN KEY (autor_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_evidencias
    ADD CONSTRAINT atendimento_evidencias_visita_id_fkey FOREIGN KEY (visita_id) REFERENCES public.atendimento_visitas(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_feriados
    ADD CONSTRAINT atendimento_feriados_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_ofertas
    ADD CONSTRAINT atendimento_ofertas_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_passagens
    ADD CONSTRAINT atendimento_passagens_autor_id_fkey FOREIGN KEY (autor_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_passagens
    ADD CONSTRAINT atendimento_passagens_solicitacao_id_fkey FOREIGN KEY (solicitacao_id) REFERENCES public.atendimento_solicitacoes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_politicas
    ADD CONSTRAINT atendimento_politicas_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_oferta_id_fkey FOREIGN KEY (oferta_id) REFERENCES public.atendimento_ofertas(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_responsavel_id_fkey FOREIGN KEY (responsavel_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_solicitante_id_fkey FOREIGN KEY (solicitante_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.atendimento_solicitacoes
    ADD CONSTRAINT atendimento_solicitacoes_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.atendimento_visitas
    ADD CONSTRAINT atendimento_visitas_solicitacao_id_fkey FOREIGN KEY (solicitacao_id) REFERENCES public.atendimento_solicitacoes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.atendimento_visitas
    ADD CONSTRAINT atendimento_visitas_tecnico_id_fkey FOREIGN KEY (tecnico_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.ativos
    ADD CONSTRAINT ativos_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.blog_post_publico
    ADD CONSTRAINT blog_post_publico_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.blog_post_publico
    ADD CONSTRAINT blog_post_publico_post_id_fkey FOREIGN KEY (post_id) REFERENCES public.blog_posts(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.blog_post_publico
    ADD CONSTRAINT blog_post_publico_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.blog_posts
    ADD CONSTRAINT blog_posts_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.calendar_event_publico
    ADD CONSTRAINT calendar_event_publico_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.calendar_event_publico
    ADD CONSTRAINT calendar_event_publico_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.calendar_events(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.calendar_event_publico
    ADD CONSTRAINT calendar_event_publico_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.calendar_rooms(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.calendar_rooms
    ADD CONSTRAINT calendar_rooms_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.catalog_services
    ADD CONSTRAINT catalog_services_responsible_unidade_id_fkey FOREIGN KEY (responsible_unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.contact_messages
    ADD CONSTRAINT contact_messages_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.data_subject_requests
    ADD CONSTRAINT data_subject_requests_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.departamentos
    ADD CONSTRAINT departamentos_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.directory_profiles
    ADD CONSTRAINT directory_profiles_departamento_id_fkey FOREIGN KEY (departamento_id) REFERENCES public.departamentos(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.directory_profiles
    ADD CONSTRAINT directory_profiles_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.directory_profiles
    ADD CONSTRAINT directory_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.egress_deliveries
    ADD CONSTRAINT egress_deliveries_target_id_fkey FOREIGN KEY (target_id) REFERENCES public.egress_targets(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.files_folder_acl
    ADD CONSTRAINT files_folder_acl_folder_id_fkey FOREIGN KEY (folder_id) REFERENCES public.files_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.files_folders
    ADD CONSTRAINT files_folders_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.files_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.files_folders
    ADD CONSTRAINT files_folders_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.files_objects
    ADD CONSTRAINT files_objects_folder_id_fkey FOREIGN KEY (folder_id) REFERENCES public.files_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.ia_uso
    ADD CONSTRAINT ia_uso_principal_id_fkey FOREIGN KEY (principal_id) REFERENCES public.ia_conexoes(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.ia_uso
    ADD CONSTRAINT ia_uso_reserva_id_fkey FOREIGN KEY (reserva_id) REFERENCES public.ia_conexoes(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.mercurio_messages
    ADD CONSTRAINT mercurio_messages_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.mercurio_messages
    ADD CONSTRAINT mercurio_messages_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.mercurio_rooms(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mercurio_reads
    ADD CONSTRAINT mercurio_reads_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.mercurio_rooms(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mercurio_reads
    ADD CONSTRAINT mercurio_reads_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mercurio_room_members
    ADD CONSTRAINT mercurio_room_members_room_id_fkey FOREIGN KEY (room_id) REFERENCES public.mercurio_rooms(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mercurio_room_members
    ADD CONSTRAINT mercurio_room_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mercurio_rooms
    ADD CONSTRAINT mercurio_rooms_departamento_id_fkey FOREIGN KEY (departamento_id) REFERENCES public.departamentos(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.mudancas_aprovacoes
    ADD CONSTRAINT mudancas_aprovacoes_aprovador_id_fkey FOREIGN KEY (aprovador_id) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.mudancas_aprovacoes
    ADD CONSTRAINT mudancas_aprovacoes_mudanca_id_fkey FOREIGN KEY (mudanca_id) REFERENCES public.mudancas(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mudancas
    ADD CONSTRAINT mudancas_solicitante_id_fkey FOREIGN KEY (solicitante_id) REFERENCES public.users(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.mudancas
    ADD CONSTRAINT mudancas_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.mudancas_vinculos
    ADD CONSTRAINT mudancas_vinculos_mudanca_id_fkey FOREIGN KEY (mudanca_id) REFERENCES public.mudancas(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.mudancas_vinculos
    ADD CONSTRAINT mudancas_vinculos_vinculado_por_fkey FOREIGN KEY (vinculado_por) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.notificacoes_preferencias
    ADD CONSTRAINT notificacoes_preferencias_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.notificacoes
    ADD CONSTRAINT notificacoes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.problemas_ativos
    ADD CONSTRAINT problemas_ativos_problema_id_fkey FOREIGN KEY (problema_id) REFERENCES public.problemas(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.problemas_ativos
    ADD CONSTRAINT problemas_ativos_vinculado_por_fkey FOREIGN KEY (vinculado_por) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.problemas
    ADD CONSTRAINT problemas_criado_por_fkey FOREIGN KEY (criado_por) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.problemas_incidentes
    ADD CONSTRAINT problemas_incidentes_problema_id_fkey FOREIGN KEY (problema_id) REFERENCES public.problemas(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.problemas_incidentes
    ADD CONSTRAINT problemas_incidentes_vinculado_por_fkey FOREIGN KEY (vinculado_por) REFERENCES public.users(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.problemas
    ADD CONSTRAINT problemas_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.produtos
    ADD CONSTRAINT produtos_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.signum_challenges
    ADD CONSTRAINT signum_challenges_envelope_id_fkey FOREIGN KEY (envelope_id) REFERENCES public.signum_envelopes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.signum_signers
    ADD CONSTRAINT signum_signers_envelope_id_fkey FOREIGN KEY (envelope_id) REFERENCES public.signum_envelopes(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.signum_signers
    ADD CONSTRAINT signum_signers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);

ALTER TABLE ONLY public.tramite_acessos
    ADD CONSTRAINT tramite_acessos_processo_id_fkey FOREIGN KEY (processo_id) REFERENCES public.tramite_processos(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.tramite_acessos
    ADD CONSTRAINT tramite_acessos_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.tramite_documentos
    ADD CONSTRAINT tramite_documentos_processo_id_fkey FOREIGN KEY (processo_id) REFERENCES public.tramite_processos(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.tramite_movimentos
    ADD CONSTRAINT tramite_movimentos_de_unidade_id_fkey FOREIGN KEY (de_unidade_id) REFERENCES public.unidades(id);

ALTER TABLE ONLY public.tramite_movimentos
    ADD CONSTRAINT tramite_movimentos_para_unidade_id_fkey FOREIGN KEY (para_unidade_id) REFERENCES public.unidades(id);

ALTER TABLE ONLY public.tramite_movimentos
    ADD CONSTRAINT tramite_movimentos_processo_id_fkey FOREIGN KEY (processo_id) REFERENCES public.tramite_processos(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.tramite_processos
    ADD CONSTRAINT tramite_processos_tipo_id_fkey FOREIGN KEY (tipo_id) REFERENCES public.tramite_tipos(id);

ALTER TABLE ONLY public.tramite_processos
    ADD CONSTRAINT tramite_processos_unidade_atual_id_fkey FOREIGN KEY (unidade_atual_id) REFERENCES public.unidades(id);

ALTER TABLE ONLY public.tramite_processos
    ADD CONSTRAINT tramite_processos_unidade_origem_id_fkey FOREIGN KEY (unidade_origem_id) REFERENCES public.unidades(id);

ALTER TABLE ONLY public.unidades
    ADD CONSTRAINT unidades_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.unidades
    ADD CONSTRAINT unidades_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_consents
    ADD CONSTRAINT user_consents_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_departamento_id_fkey FOREIGN KEY (departamento_id) REFERENCES public.departamentos(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_perfil_id_fkey FOREIGN KEY (perfil_id) REFERENCES public.perfis(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.user_scopes
    ADD CONSTRAINT user_scopes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.wiki_page_publico
    ADD CONSTRAINT wiki_page_publico_entidade_id_fkey FOREIGN KEY (entidade_id) REFERENCES public.entidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.wiki_page_publico
    ADD CONSTRAINT wiki_page_publico_page_id_fkey FOREIGN KEY (page_id) REFERENCES public.wiki_pages(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.wiki_page_publico
    ADD CONSTRAINT wiki_page_publico_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.wiki_pages
    ADD CONSTRAINT wiki_pages_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.wiki_pages(id) ON DELETE RESTRICT;

ALTER TABLE ONLY public.wiki_pages
    ADD CONSTRAINT wiki_pages_unidade_id_fkey FOREIGN KEY (unidade_id) REFERENCES public.unidades(id) ON DELETE SET NULL;

ALTER TABLE ONLY public.wiki_revisions
    ADD CONSTRAINT wiki_revisions_page_id_fkey FOREIGN KEY (page_id) REFERENCES public.wiki_pages(id) ON DELETE CASCADE;

-- Dados de partida: aparência, perfis de sistema, sala geral do chat,
-- tipos de processo e a flag do Módulo Modelo.
INSERT INTO public.branding_settings (id) VALUES ('default');

INSERT INTO public.feature_flags (key, enabled, description) VALUES
    ('module_exemplos_enabled', true, 'Habilita a exibição e uso do Módulo Modelo (Exemplo Blueprint).');

INSERT INTO public.perfis (slug, nome, descricao, permissoes, sistema) VALUES
    ('administrador', 'Administrador da Plataforma', 'Acesso total: módulos, identidade, estrutura organizacional e todos os plugins.', '{*}', true),
    ('auditor', 'Auditor', 'Leitura da trilha de auditoria, verificação da cadeia de integridade e consulta de usuários.', '{audit:read,audit:verify,users:read,monitoring:read}', true),
    ('gestor-iam', 'Gestor de Identidade', 'Administra usuários, lotações, perfis e o mapeamento de grupos do AD.', '{users:read,users:manage,iam:manage}', true),
    ('gestor-conteudo', 'Gestor de Conteúdo', 'Publica comunicados, mantém o catálogo de serviços e modera a wiki.', '{blog:manage,catalog:manage,wiki:manage,contact:read,contact:manage}', true),
    ('protocolo', 'Protocolo', 'Abre e tramita processos administrativos.', '{tramite:create,tramite:route}', true),
    ('servidor', 'Servidor', 'Perfil básico de colaborador autenticado (sem permissões administrativas).', '{}', true);

INSERT INTO public.mercurio_rooms (kind, name, description) VALUES
    ('global', 'Geral', 'Canal aberto a todos os colaboradores autenticados.');

INSERT INTO public.tramite_tipos (slug, nome, descricao) VALUES
    ('requerimento', 'Requerimento', 'Pedido formal de um interessado.'),
    ('memorando', 'Memorando', 'Comunicação interna entre unidades.'),
    ('oficio', 'Ofício', 'Comunicação oficial com órgãos externos.'),
    ('contratacao', 'Contratação', 'Instrução de processo de contratação.'),
    ('outros', 'Outros', 'Demais processos administrativos.');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Esvazia o esquema (menos a tabela do goose): tabelas, funções próprias,
-- tipos, sequências soltas e as extensões.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'goose_db_version' LOOP
        EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', r.tablename);
    END LOOP;
    FOR r IN SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
              WHERE n.nspname = 'public' AND c.relkind = 'S'
                AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.objid = c.oid AND d.deptype IN ('a', 'i', 'e'))
                AND c.relname NOT LIKE 'goose_db_version%' LOOP
        EXECUTE format('DROP SEQUENCE IF EXISTS public.%I CASCADE', r.relname);
    END LOOP;
    FOR r IN SELECT p.oid::regprocedure AS f FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
              WHERE n.nspname = 'public'
                AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.objid = p.oid AND d.deptype = 'e') LOOP
        EXECUTE format('DROP FUNCTION IF EXISTS %s CASCADE', r.f);
    END LOOP;
    FOR r IN SELECT t.typname FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
              WHERE n.nspname = 'public' AND t.typtype IN ('e', 'd')
                AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.objid = t.oid AND d.deptype = 'e') LOOP
        EXECUTE format('DROP TYPE IF EXISTS public.%I CASCADE', r.typname);
    END LOOP;
END $$;
DROP EXTENSION IF EXISTS unaccent;
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS btree_gist;
-- +goose StatementEnd
