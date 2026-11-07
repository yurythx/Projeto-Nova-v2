-- +goose Up
-- O diretório (AD e GLPI) grava o mesmo trecho no nome e no sobrenome.
-- A função tira o bloco repetido; a sincronização passa a gravar o nome já limpo.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION nova_nome_sem_repeticao(s text) RETURNS text
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    w text[];
    n int;
    k int;
    i int;
    ok boolean;
    j int;
BEGIN
    IF s IS NULL OR btrim(s) = '' THEN
        RETURN '';
    END IF;
    w := regexp_split_to_array(btrim(regexp_replace(s, '\s+', ' ', 'g')), ' ');
    n := cardinality(w);
    IF n < 2 THEN
        RETURN w[1];
    END IF;
    k := n / 2;
    WHILE k >= 1 LOOP
        i := 1;
        WHILE i + 2 * k - 1 <= n LOOP
            ok := true;
            j := 0;
            WHILE j < k LOOP
                IF lower(w[i + j]) IS DISTINCT FROM lower(w[i + k + j]) THEN
                    ok := false;
                    EXIT;
                END IF;
                j := j + 1;
            END LOOP;
            IF ok THEN
                RETURN array_to_string(w[1:i + k - 1] || w[i + 2 * k:n], ' ');
            END IF;
            i := i + 1;
        END LOOP;
        k := k - 1;
    END LOOP;
    RETURN array_to_string(w, ' ');
END $$;
-- +goose StatementEnd

UPDATE users
   SET display_name = nova_nome_sem_repeticao(display_name)
 WHERE display_name IS NOT NULL
   AND nova_nome_sem_repeticao(display_name) IS DISTINCT FROM btrim(display_name);

UPDATE projetos p
   SET gerente_nome = u.display_name
  FROM users u
 WHERE p.gerente_id = u.id
   AND p.gerente_nome IS DISTINCT FROM u.display_name;

UPDATE projetos_tarefas t
   SET responsavel_nome = u.display_name
  FROM users u
 WHERE t.responsavel_id = u.id
   AND t.responsavel_nome IS DISTINCT FROM u.display_name;

UPDATE projetos_tarefa_movimentos m
   SET responsavel_antes_nome = u.display_name
  FROM users u
 WHERE m.responsavel_antes_id = u.id
   AND m.responsavel_antes_nome IS DISTINCT FROM u.display_name;

UPDATE projetos_tarefa_movimentos m
   SET responsavel_depois_nome = u.display_name
  FROM users u
 WHERE m.responsavel_depois_id = u.id
   AND m.responsavel_depois_nome IS DISTINCT FROM u.display_name;

UPDATE projetos_tarefa_movimentos m
   SET autor_nome = u.display_name
  FROM users u
 WHERE m.autor_id = u.id
   AND m.autor_nome IS DISTINCT FROM u.display_name;

UPDATE projetos_tarefa_atualizacoes a
   SET autor_nome = u.display_name
  FROM users u
 WHERE a.autor_id = u.id
   AND a.autor_nome IS DISTINCT FROM u.display_name;

-- +goose Down
DROP FUNCTION IF EXISTS nova_nome_sem_repeticao(text);
