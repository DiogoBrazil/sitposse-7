# Requisitos não funcionais

| ID | Requisito | Meta / verificação |
|---|---|---|
| RNF-01 | **Offline no Android**: nenhuma função exige internet, login em nuvem, backend remoto ou telemetria. | Suite inteira roda em modo avião |
| RNF-02 | **Offline na Central**, exceto mapa e carta-imagem. | Estatísticas abrem com a rede desligada |
| RNF-03 | **Durabilidade**: um dado confirmado como salvo sobrevive a kill de processo, reinício e queda de energia. | SQLite com WAL + `synchronous=FULL`; testes de kill |
| RNF-04 | **Integridade**: fotos verificadas por SHA‑256 de ponta a ponta; nenhum registro aponta para arquivo ausente. | Testes de invariantes |
| RNF-05 | **Idempotência**: reenviar ou reimportar nunca duplica. | Lote repetido 3× produz o mesmo estado final |
| RNF-06 | **Recuperação**: a interrupção em qualquer fase da sincronização é retomável, sem perda. | Suite de injeção de falhas |
| RNF-07 | **Volume de projeto** (aprovado): até cerca de 5 equipes, 300 propriedades e 1.500 fotos por operação; testes com 3× esse volume. | Teste de carga com 900 propriedades e 4.500 fotos |
| RNF-08 | **Desempenho** da Central: atualização do painel com filtros em < 300 ms no volume 3×. | Benchmark Go |
| RNF-09 | **Desempenho** do Android: abrir o formulário em < 500 ms; salvar em < 300 ms (sem foto); a lista rola sem travar com 300 itens. | Macrobenchmark / medição manual |
| RNF-10 | **Compatibilidade**: Android 8.0+ (minSdk 26); Windows 10 22H2+ 64 bits. | Matriz de dispositivos (Fase 0/7) |
| RNF-11 | **Fotos**: lado maior de cerca de 2560 px, JPEG 85, ~0,8–1,5 MB; nunca carregar a imagem inteira em memória. | Teste com aparelho de 2–3 GB de RAM |
| RNF-12 | **UX em campo**: Material 3, bom contraste ao sol, alvos ≥ 48 dp, uso com uma mão, teclado numérico nos números, poucos toques. | Revisão de UX e teste com equipe |
| RNF-13 | **Acessibilidade**: rótulos para leitor de tela, tamanho de fonte do sistema respeitado e contraste AA. | Accessibility Scanner / revisão |
| RNF-14 | **Migrações não destrutivas**, testadas de cada versão publicada até a atual. | `MigrationTestHelper` e testes Go |
| RNF-15 | **Instalação offline da Central**: o instalador inclui o WebView2 e cria a regra de firewall. | Teste em Windows limpo sem internet |
| RNF-16 | **Rastreabilidade**: origem (equipe/instalação), timestamps de origem e de recebimento e auditoria de toda alteração na Central. | Testes de auditoria |
