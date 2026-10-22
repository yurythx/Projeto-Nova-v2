-- ==============================================================================
-- PROJETO NEXUS — DESTAQUE EDITORIAL DO BLOG: PROXMOX + TERRAFORM + ANSIBLE
-- ==============================================================================
-- Configura 'artigo-proxmox' como o destaque editorial (pinned = true)
-- com conteúdo aprofundado sobre IaC, vantagens vs VMware e casos reais.
-- ==============================================================================

BEGIN;

-- 1. Despinar artigos anteriores para garantir que apenas o destaque editorial esteja pinado
UPDATE blog_posts SET pinned = false WHERE slug <> 'artigo-proxmox';

-- 2. Atualizar artigo-proxmox com o novo conteúdo editorial e fixar como destaque
UPDATE blog_posts SET
    title = 'Proxmox VE com Terraform e Ansible: Como Aplicar, Vantagens e Casos Reais em Grandes Empresas',
    summary = 'Guia completo sobre a tríade Proxmox VE, Terraform e Ansible: como estruturar o provisionamento declarativo (IaC), automatizar configurações contínuas, reduzir até 80% dos custos em relação ao VMware e quais gigantes mundiais adotam essa arquitetura.',
    body = $BODY$## Visão Geral: A Transição do "ClickOps" para a Infraestrutura como Código Soberana

A virtualização corporativa passou por uma transformação radical nos últimos anos. Historicamente, a criação de máquinas virtuais e containers em ambientes on-premise dependia de processos manuais ou ferramentas proprietárias de alto custo, frequentemente conhecidas pela prática de "ClickOps" — administradores clicando em assistentes gráficos para criar servidores, definir redes e alocar memória, sem reprodutibilidade nem histórico confiável de alterações.

Com a reestruturação comercial global do ecossistema VMware após a aquisição pela Broadcom — que resultou em aumentos expressivos de licenciamento (entre 300% e 1000% para contratos corporativos) e no fim das licenças perpétuas —, o **Proxmox Virtual Environment (VE)** consolidou-se como o hipervisor de código aberto definitivo para datacenters corporativos e órgãos públicos.

No entanto, a maturidade de uma infraestrutura moderna não se limita ao hipervisor: ela exige **Infraestrutura como Código (IaC)** e **Gerência de Configurações Contínua**. É aqui que a união de **Proxmox VE, Terraform e Ansible** cria a arquitetura mais eficiente, soberana e escalável do mercado.

---

## A Arquitetura da Tríade: Papéis e Responsabilidades

Cada tecnologia desempenha um papel cirúrgico e complementar na cadeia de entrega contínua:

| Camada | Tecnologia | Responsabilidade Principal | Mecanismo de Ação |
| :--- | :--- | :--- | :--- |
| **Computação & Storage** | **Proxmox VE** | Hipervisor KVM e containers LXC com clustering e alta disponibilidade (HA) | API REST nativa, Corosync, ZFS e Ceph |
| **Provisionamento (IaC)** | **Terraform** | Declaração do estado desejado da infraestrutura física e virtual | Provedor Proxmox (`bpg/proxmox`), HCL e Cloud-Init |
| **Configuração & Estado** | **Ansible** | Padronização de sistemas operacionais, hardening e deploy de serviços | Agentless via SSH, inventário dinâmico e playbooks |

```
                 +-------------------------------------------------+
                 |              Repositório Git (GitOps)           |
                 |      Manifestos HCL + Playbooks YAML + CI/CD    |
                 +-----------------------+-------------------------+
                                         |
                                         v
                         +-------------------------------+
                         |      TERRAFORM (IaC)          |
                         |   Planeja e provisiona no     |
                         |       Proxmox VE via API      |
                         +---------------+---------------+
                                         |
                                         v
                 +-------------------------------------------------+
                 |            CLUSTER PROXMOX VE                   |
                 |  [VM 101: Web]    [VM 102: App]   [VM 103: DB]  |
                 |    (Cloud-Init)     (Cloud-Init)    (Cloud-Init)|
                 +-----------------------+-------------------------+
                                         |
                                         v
                         +-------------------------------+
                         |      ANSIBLE (Config)         |
                         |   Inventário Dinâmico + SSH   |
                         |  Hardening, Pacotes, Serviços |
                         +-------------------------------+
```

---

## Como Aplicar na Prática: Guia de Implementação Passo a Passo

A implementação da automação segue um fluxo padronizado em quatro fases essenciais:

### 1. Preparação da Imagem Dourada (Golden Template com Cloud-Init)

O segredo do provisionamento rápido é o uso de imagens de nuvem pré-construídas (Cloud Images) do Debian ou Ubuntu integradas com o **Cloud-Init** e o **QEMU Guest Agent**:

```bash
# Download da imagem oficial da distribuição
wget https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img

# Criação da máquina virtual de template (ID 9000)
qm create 9000 --name "ubuntu-2404-template" --memory 2048 --cores 2 --net0 virtio,bridge=vmbr0
qm importdisk 9000 noble-server-cloudimg-amd64.img local-zfs
qm set 9000 --scsihw virtio-scsi-pci --scsi0 local-zfs:vm-9000-disk-0
qm set 9000 --boot c --bootdisk scsi0
qm set 9000 --ide2 local-zfs:cloudinit
qm set 9000 --agent enabled=1
qm set 9000 --serial0 socket --vga serial0
qm template 9000
```

### 2. Provisionamento Declarativo com Terraform

Com o template pronto e um token de API gerado no Proxmox (`admin@pve!terraform`), declaramos as máquinas virtuais em HCL com o provedor moderno `bpg/proxmox`:

```hcl
# main.tf
terraform {
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = ">= 0.60.0"
    }
  }
}

provider "proxmox" {
  endpoint  = "https://pve.nexus.corp:8006/"
  api_token = "admin@pve!terraform=seu-token-secreto"
  insecure  = false
}

resource "proxmox_virtual_environment_vm" "app_nodes" {
  count     = 3
  name      = "srv-app-0${count.index + 1}"
  node_name = "pve-node-01"
  vm_id     = 200 + count.index

  clone {
    vm_id = 9000 # Clona o template Golden
    full  = true
  }

  cpu {
    cores = 4
    type  = "host"
  }

  memory {
    dedicated = 8192
  }

  disk {
    datastore_id = "local-zfs"
    size         = 50
    interface    = "scsi0"
  }

  initialization {
    ip_config {
      ipv4 {
        address = "192.168.10.${50 + count.index}/24"
        gateway = "192.168.10.1"
      }
    }
    user_account {
      username = "devops"
      keys     = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI..."]
    }
  }

  tags = ["env_producao", "role_app", "managed_by_terraform"]
}
```

Executando o ciclo de vida:
```bash
terraform init
terraform plan    # Pré-visualização atômica de todas as alterações
terraform apply   # Criação paralela das 3 máquinas em menos de 60 segundos
```

### 3. Orquestração e Hardening de Configurações com Ansible

Após a criação das instâncias, o Ansible utiliza o **plugin de inventário dinâmico do Proxmox** (`community.general.proxmox`) para descobrir automaticamente todos os servidores que possuem a tag `role_app`:

```yaml
# inventory/proxmox.yml
plugin: community.general.proxmox
url: "https://pve.nexus.corp:8006"
user: "admin@pve!terraform"
token_secret: "seu-token-secreto"
validate_certs: false
keyed_groups:
  - key: tags
    separator: "_"
```

O playbook de configuração garante o estado homogêneo e seguro em todas as instâncias:

```yaml
# playbooks/site.yml
- name: Hardening e Configuração de Baseline Corporativo
  hosts: tags_role_app
  become: true
  tasks:
    - name: Atualizar repositórios e pacotes do sistema
      ansible.builtin.apt:
        update_cache: yes
        upgrade: dist

    - name: Configurar fuso horário corporativo
      community.general.timezone:
        name: "America/Cuiaba"

    - name: Hardening do servidor SSH
      ansible.builtin.lineinfile:
        path: /etc/ssh/sshd_config
        regexp: "{{ item.regexp }}"
        line: "{{ item.line }}"
      loop:
        - { regexp: '^PasswordAuthentication', line: 'PasswordAuthentication no' }
        - { regexp: '^PermitRootLogin', line: 'PermitRootLogin no' }
      notify: Reiniciar SSH

    - name: Configurar regras de firewall UFW
      community.general.ufw:
        rule: allow
        port: "{{ item }}"
        proto: tcp
      loop:
        - 22   # SSH Bastion
        - 80   # HTTP
        - 443  # HTTPS

    - name: Instalar Docker e utilitários de monitoramento
      ansible.builtin.apt:
        name:
          - docker.io
          - docker-compose-v2
          - zabbix-agent2
        state: present

  handlers:
    - name: Reiniciar SSH
      ansible.builtin.service:
        name: sshd
        state: restarted
```

### 4. Integração Contínua com Pipelines GitOps

Toda a infraestrutura é versionada em repositórios Git corporativos. Quando um engenheiro submete um Pull Request alterando parâmetros de hardware ou adicionando uma role do Ansible:
1. O pipeline de CI executa `terraform validate`, `tflint` e `ansible-lint`.
2. Um `terraform plan` é gerado automaticamente no comentário do PR.
3. Após a revisão por pares, o merge na branch `main` dispara o `terraform apply` seguido pelo `ansible-playbook`.

---

## Principais Vantagens em Adotar a Stack Proxmox + Terraform + Ansible

1. **Soberania e Redução Drástica de Custos (vs VMware & Nuvem Pública):**
   - Elimina contratos leoninos de licenciamento por core de CPU da VMware/Broadcom, reduzindo o Custo Total de Propriedade (TCO) em até 80%.
   - Permite executar infraestrutura privada de alta densidade sem a imprevisibilidade de faturamento das nuvens públicas (AWS/Azure).

2. **Reprodutibilidade e Eliminação de Desvios (Drift Elimination):**
   - Todo o ambiente pode ser destruído e reconstruído do zero de forma idêntica em minutos.
   - O comando `terraform plan` detecta instantaneamente qualquer modificação manual indevida realizada no painel web.

3. **Alta Performance com KVM e Containers LXC Nativos:**
   - O Proxmox suporta nativamente containers LXC que compartilham o kernel do host, alcançando densidades computacionais até 3x superiores a VMs tradicionais com quase zero overhead de I/O.
   - Integração com sistemas de arquivos corporativos como **ZFS** (com compressão LZ4 e snapshots instantâneos) e **Ceph** para storage hiperconvergente.

4. **Governança, Conformidade e Segurança (LGPD / CIS Benchmarks):**
   - O código no Git serve como documentação viva e trilha de auditoria: sabe-se com precisão cirúrgica quem alterou cada porta, servidor ou volume de dados.
   - O Ansible aplica baselines rígidos de segurança de forma idempotente em 100% dos nós.

5. **Disaster Recovery Automatizado com Proxmox Backup Server (PBS):**
   - Snapshots incrementais e deduplicação de nível enterprise em discos do storage.
   - Backups criptografados do lado do cliente com tempo de recuperação (RTO) de minutos.

---

## Grandes Empresas e Organizações que Confiam Nessa Arquitetura

A combinação de Proxmox VE com automação declarativa tornou-se o padrão preferido de grandes organizações globais e nacionais:

- **Hetzner & OVHcloud — Provedores Globais de Infraestrutura:**
  Utilizam o Proxmox VE e KVM em centenas de milhares de servidores bare-metal pelo mundo, gerenciando o provisionamento elástico de instâncias para clientes internacionais com Terraform.
- **CERN (European Organization for Nuclear Research):**
  Emprega hypervisors KVM e automação em larga escala com Ansible para gerenciar os clusters de processamento de dados do Grande Colisor de Hádrons (LHC).
- **Universidade de Oxford & Institutos Acadêmicos Internacionais:**
  Adotam Proxmox VE e Terraform para oferecer ambientes de laboratório isolados para pesquisadores de forma automatizada e com custo previsível.
- **Setor Financeiro, Fintechs e Cooperativas de Crédito:**
  Organizações financeiras que migraram do VMware para infraestruturas privadas dedicadas com Proxmox para garantir soberania de dados regulatória e baixa latência.
- **Órgãos Públicos, Tribunais e Prefeituras no Brasil:**
  Instituições governamentais adotam clusters Proxmox VE com automação Ansible para atender às metas de contenção de gastos públicos e adequação à Lei Geral de Proteção de Dados (LGPD).

---

## Como a Plataforma Nexus Apoia Sua Infraestrutura

O Projeto Nexus integra nativamente essas soluções corporativas:
- Arquitetura de clusters Proxmox VE com alta disponibilidade (HA) e storages ZFS/TrueNAS.
- Desenvolvimento de módulos Terraform e playbooks Ansible homologados para a sua realidade.
- Monitoramento proativo 24x7 integrado via Zabbix e central de atendimento ITIL no GLPI.

👉 **[Conheça as especificações do Proxmox VE no Catálogo de Serviços](/servicos/proxmox)**
👉 **[Conheça o serviço de Infraestrutura como Código com Terraform](/servicos/terraform)**
👉 **[Conheça as soluções de Automação Corporativa com Ansible](/servicos/ansible)**$BODY$,
    pinned = true,
    status = 'published',
    published_at = now(),
    updated_at = now()
WHERE slug = 'artigo-proxmox';

COMMIT;
