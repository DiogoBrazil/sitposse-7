# Estratégia de testes

Os testes validam **dados persistidos e invariantes** ([integridade.md](../synchronization/integridade.md)), não só mensagens de sucesso.

## 1. Fixtures compartilhadas (`shared/test-fixtures`)
| Pasta | Conteúdo | Executado por |
|---|---|---|
| `area/` | conversões e somas (RN-44) | Kotlin (JVM) e Go |
| `normalizacao/` | nomes de "Outros" → `nome_normalizado` (RN-35) | Kotlin e Go |
| `contratos/` (Fase 0+) | payloads válidos e inválidos por versão | validação JSON Schema nos dois lados |
| `sync/` (Fase 2+) | cenários: estado inicial, lotes e estado final esperado | Go (servidor) e cliente Kotlin JVM |
| `estatisticas/` (Fase 4) | base de exemplo e números esperados para cada cruzamento | Go |

## 2. Android
- **JVM:** domínio, validações, área, normalização, máquina de estados do lote.
- **Instrumentados:** Room (CHECKs, transações, índices únicos parciais), migrações com `MigrationTestHelper` de **todas** as versões publicadas, gravação de foto com falha injetada em cada etapa, reconciliador, GNSS com provedor *mock*, Compose UI (cadastro completo, vários "Outros", rascunho em rotação e morte de processo).
- **Aparelho real:** modo avião em tudo; `adb shell am kill` durante o salvamento; reinício do aparelho; armazenamento quase cheio; permissões negadas (câmera, localização, rede local); Android 8, um intermediário e Android 17.

## 3. Central (Go)
- Testes de tabela e integração com SQLite temporário.
- **Injeção de falhas** com checkpoints nomeados (`apos_receber_registros`, `apos_publicar_fotos`, `apos_commit_antes_resposta`): o teste sobe o servidor em subprocesso, mata no checkpoint, reinicia e verifica as invariantes.
- Proxy de teste que derruba a conexão em cada fase.
- Disco cheio simulado (FS wrapper).
- Backup e restauração: gerar backup, corromper o original, restaurar e comparar.
- Benchmark do painel (RNF-08).

## 4. Sincronização ponta a ponta
Cliente Kotlin (JVM) contra o servidor Go real:
1. Primeira carga.
2. Três equipes em paralelo.
3. 35 confirmadas + 3 editadas + 2 novas → **exatamente 5 aplicadas**.
4. Mesmo lote 3× → estado idêntico.
5. ACK perdido.
6. Foto interrompida e retomada.
7. Hash divergente.
8. Tombstone.
9. Conflito (edição e exclusão).
10. Versão incompatível.
11. Cadastros diferentes com o mesmo nome.
12. Volume 3×: 900 propriedades e 4.500 fotos.
13. Pacote USB + recibo.
14. Restauração de equipe.

## 5. Frontend da Central
- Vitest: serialização do `FiltroEstudo`, formatação de área e datas.
- Teste de que Estatísticas abre e calcula **com a rede desligada**.

## 6. Cartografia
Provedor falso: sucesso, timeout, falha de DNS, 401/403, 429 e 5xx. Verificar que **somente** o painel de mapa e a carta-imagem ficam indisponíveis, que os filtros se mantêm e que nenhuma carta é gerada incompleta. Implementado para o Copernicus CDSE (A27): ver `desktop/docs/cartografia.md` §Testes.

## 7. Aceite final (prompt §21, inegociável)
- [ ] Registros persistem offline e não desaparecem ao reiniciar.
- [ ] Fotos preservadas, com hash conferido.
- [ ] GNSS funciona offline.
- [ ] Múltiplas equipes.
- [ ] Transferência local sem internet.
- [ ] Sincronização repetida sem duplicação.
- [ ] Importação verificável (recibo).
- [ ] Recuperação após interrupção em cada fase.
- [ ] Conflitos nunca sobrescritos em silêncio.
- [ ] Exclusões propagadas, sem ressurgimento.
- [ ] Backups restauráveis (testado).
- [ ] Estatísticas e filtros corretos offline.
- [ ] Alqueire paulista = 2,42 ha.
- [ ] Mapa e carta-imagem online, com mensagem amigável quando offline.
- [ ] UX moderna e simples (validada com equipe).
- [ ] Documentação atualizada.
- [ ] Migrações sem destruição de dados.
