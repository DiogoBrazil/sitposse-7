# AGENTS.md — Regras globais do SITPOSSE-7

> **Retomando o trabalho?** Leia primeiro [`docs/planning/continuacao.md`](docs/planning/continuacao.md): estado atual, próximos passos, preparo do ambiente e proteções contra regressão.

Estas regras valem para todo o monorepositório. `mobile/AGENTS.md` e `desktop/AGENTS.md` **acrescentam** detalhes de plataforma e **nunca podem contradizer** este arquivo. Em caso de conflito, prevalece este arquivo e a divergência deve ser corrigida.

## 1. Prioridade do projeto
1. **Não perder nem corromper dados silenciosamente.**
2. Coleta 100% offline no Android; Central offline, exceto mapa e carta-imagem.
3. Migração íntegra e verificável dos dados para a Central.
4. UX simples e moderna.

## 2. Processo
- **Não implemente funcionalidade fora da fase aprovada** em [`docs/planning/plano.md`](docs/planning/plano.md). Fase nova ou mudança de escopo exige aprovação do usuário.
- **Pare e pergunte** diante de dúvida funcional, de conflito, de segurança, de UX ou de arquitetura com consequência relevante. Registre a pergunta em [`docs/planning/pendencias.md`](docs/planning/pendencias.md).
- **Nunca altere uma decisão aprovada em silêncio.** Toda mudança passa por um ADR novo (ou que substitui o anterior) e pela atualização de [`docs/planning/matriz-decisoes.md`](docs/planning/matriz-decisoes.md).
- Mantenha separadas as categorias **Aprovado**, **Recomendado** e **Pendente**.
- Toda mudança de código atualiza, no mesmo commit, os documentos e testes afetados e o [`docs/changelog.md`](docs/changelog.md).
- Não duplique regras: elas vivem em `docs/requirements/regras-de-negocio.md`, e os demais documentos apenas fazem referência.

## 3. Dados e persistência (inegociável)
- **Proibido** usar migração destrutiva (`fallbackToDestructiveMigration` ou equivalente, `DROP` de tabela com dados, recriação silenciosa de banco).
- **Proibido** fazer DELETE físico de entidade de negócio. Exclusão é **tombstone** (`excluido_em`).
- Só informe "salvo" ao usuário **depois do commit** da transação.
- Identidade global por **UUID**. IDs sequenciais locais nunca identificam entidade fora do banco local.
- Detecção de mudanças e conflitos por **`rev`**, nunca por horário.
- Quantitativos são inteiros ≥0 com padrão 0. **Ausência de área ou de coordenada não é zero.**
- A conversão de área segue `shared/test-fixtures/area/conversao.json` (1 alqueire paulista = 2,42 ha).
- Estatísticas consultam **apenas** as visões vigentes (sem tombstones nem versões antigas).
- Fotos: grave o arquivo com fsync e rename **antes** do commit do registro. Registro apontando para arquivo inexistente é defeito grave.

## 4. Esquema, protocolo e contratos
- Toda mudança de esquema SQLite ou de protocolo exige: **ADR**, **migração não destrutiva testada**, atualização de `shared/schemas` e **fixture** em `shared/test-fixtures`.
- Contratos JSON têm `schema_versao` e `protocolo_versao` e devem permanecer compatíveis entre Kotlin e Go; as mesmas fixtures rodam nos dois lados.
- Timestamps: epoch em **ms UTC** no SQLite e **ISO 8601 UTC com ms e `Z`** no JSON; exibição em horário local.

## 5. Sincronização
- Enviar bytes não é importar. Só é **confirmado** o que tem **recibo persistido** pela Central.
- Reenviar ou reimportar **nunca duplica** dados.
- Conflitos **nunca** são sobrescritos em silêncio.
- **Nunca desabilite a verificação TLS.** A identidade da Central é validada pelo pinning do SPKI recebido no QR Code.

## 6. Internet
- Nenhuma função essencial depende de internet. **Mapas e cartas-imagem são a única exceção** e, sem conexão, mostram a mensagem amigável aprovada sem bloquear o restante.
- Proibido: telemetria, login em nuvem, CDN em runtime, cache de tiles para uso offline (nesta versão).

## 7. Dependências
- Prefira a biblioteca padrão e bibliotecas oficiais. Justifique toda dependência nova (no ADR ou no `docs/changelog.md`) e fixe versões exatas.
- Não invente APIs: consulte a documentação oficial atual (Context7 ou web) antes de usar uma biblioteca.

## 8. Privacidade
- Não colete identificador permanente de hardware.
- Não envie dados a serviços externos, exceto requisições de tiles ao provedor cartográfico aprovado, que não levam dados de cadastro.
