# Regras de negócio — fonte única

> Este é o **único** lugar onde as regras de negócio são definidas. Os outros documentos fazem referência a elas pelo ID (RN-xx).
> Status: todas as regras abaixo estão **APROVADAS** (prompt-mestre e respostas do usuário em 13/11/2026), salvo marcação em contrário.

## Estudo, jornada, equipe e coleta
*(Reescrita pela A28 em 15/11/2026; ADR-0023 e ADR-0024. Substitui a antiga seção "Equipe e operação".)*
- **RN-01** Na Central, a hierarquia é **Estudo de Situação → Jornada → Equipe → Cadastros**. O **estudo** tem nome obrigatório (não precisa ser único) e código público único `SITPOSSE7-XXXX-XXXX`. A **jornada** pertence a um estudo e tem data (`AAAA-MM-DD`) e número sequencial no estudo; a data não é identificador (duas jornadas podem ter a mesma data). A **equipe** pertence a uma jornada, tem nome livre (sem nomenclatura obrigatória, único dentro da jornada) e código público único em toda a Central `EQ-XXXX-XXXX`. Equipe com o mesmo nome em outra jornada é **outra equipe, com outro código**.
- **RN-02** Os códigos usam o alfabeto Crockford Base32 (sem `I L O U`), com **7 caracteres aleatórios + 1 dígito verificador** (Luhn mod 32). Na digitação, maiúsculas e minúsculas, espaços e hífens são indiferentes, `O` vale `0` e `I`/`L` valem `1`. O banco garante a unicidade; em colisão, gera-se outro. Casos de referência: `shared/test-fixtures/codigos/codigos.v1.json`.
- **RN-03** Cada instalação do app gera um UUID aleatório próprio (`instalacao`). **Não** é coletado nenhum identificador permanente de hardware.
- **RN-04** Não há conta nem login individual no Android.
- **RN-05** No celular, cada código de equipe informado abre uma **coleta** (UUID próprio). Todo cadastro pertence a **uma** coleta, para sempre; o rascunho pertence à coleta em que começou. O celular **não** consulta a Central para aceitar o código: confere só o formato e o dígito verificador.
- **RN-06** Após reinstalação do app, basta informar de novo o código da equipe para continuar coletando (nova coleta). O que não tinha sido sincronizado antes da perda não é recuperável. A restauração dos cadastros pela Central continua prevista (RF-A15).
- **RN-07** Ciclo da coleta: `em andamento` (aceita cadastros novos) ou `concluída` (não aceita cadastros novos, mas os existentes podem ser editados, excluídos e sincronizados; pode ser reaberta com confirmação). Ao abrir o app: nenhuma em andamento → pedir o código; uma → abrir direto; várias → listar. A conclusão é **manual**; o app pergunta se deve concluir a anterior ao iniciar outro código e avisa quando a coleta começou em outro dia. Digitar o código de uma coleta existente retoma essa coleta, sem duplicá-la.
- **RN-08** O código de uma coleta só pode ser **corrigido no celular** enquanto ela não tiver destino confirmado pela Central e nenhum lote dela estiver aberto. A correção não altera nenhum dado dos cadastros e fica registrada (código anterior e novo).
- **RN-09** Antes de importar, a Central **sempre** mostra ao operador o destino resolvido pelo código (estudo, data da jornada, equipe, código) ou avisa que o código não existe; o operador confirma, altera o destino ou cancela. Nenhuma vinculação é silenciosa. Depois da primeira importação, a coleta fica **presa à equipe** escolhida (destino travado). A correção feita na Central preserva o código recebido, é auditável e vale só para aquela coleta.

## Propriedade / família
- **RN-10** O UUID é a identidade. **Nomes iguais não significam o mesmo cadastro.**
- **RN-11** Os **únicos** campos obrigatórios são `nome_propriedade` e `responsavel_familia` (exibido como "Nome do responsável pela propriedade"), ambos não vazios após remover espaços. *(Alterada pela A28 em 15/11/2026; antes, só o nome da propriedade.)* A navegação entre as etapas do formulário nunca é bloqueada por falta deles; só o salvamento.
- **RN-12** GPS, fotos, área e quantitativos **não** são obrigatórios.
- **RN-13** Não há campo obrigatório de identificação pessoal (CPF, RG etc.).
- **RN-14** Excluir é marcar `excluido_em` (tombstone). Exclusão de negócio **≠** limpeza de cópia local.
- **RN-15** **Observações relevantes** (`observacoes_relevantes`): texto livre opcional, multilinha, com acentos e quebras de linha, até **4000 caracteres**. Vazio é gravado como nulo. *(A28, 15/11/2026.)*

## Quantitativos fixos
- **RN-20** Pessoas: **Adultos (18+ anos)**, **Jovens (12–17 anos)**, **Crianças (0–11 anos)**, conforme as faixas do ECA. O formulário mostra o texto de ajuda com as idades.
- **RN-21** Veículos: **Caminhões**; **Automóveis, caminhonetes e utilitários** (categoria única); **Motocicletas e ciclomotores** (categoria única).
- **RN-22** Bovinos: **Bovinos adultos**; **Bezerros e bezerras** (categoria única, sem distinção de sexo).
- **RN-23** Equinos: **Equinos adultos**; **Potros**.
- **RN-24** Quantidades são inteiros ≥ 0, opcionais na UI e **iguais a 0 quando não preenchidas**. NULL não é usado como significado padrão. Os totais derivados são exibidos.

## Itens dinâmicos
- **RN-30** Catálogo inicial:
  - **Eletrodomésticos:** freezer, geladeira, fogão, máquina de lavar, ar-condicionado, ventilador, Outros.
  - **Móveis:** sofá, mesa de jantar, armário, estante, guarda-roupa, Outros.
  - **Aves de criação:** galinha, ganso, pato, Outros.
- **RN-31** Cada item tem UUID, categoria, código do catálogo **ou** nome personalizado ("Outros"), quantidade, timestamps, `rev` e tombstone.
- **RN-32** Um item predefinido aparece **no máximo uma vez** por propriedade (entre itens vigentes). Adicionar de novo o mesmo item abre a edição da quantidade existente.
- **RN-33** "Outros" pode aparecer **várias vezes**, cada um com nome livre obrigatório. Se o nome normalizado repetir outro "Outros" da mesma propriedade, o app **avisa e oferece somar**, mas não bloqueia.
- **RN-34** Quantidade de item: inteiro **≥ 1 na UI**. Para zerar, o usuário remove o item. (O banco aceita 0 por robustez de importação.)
- **RN-35** O texto original do "Outros" **nunca é alterado**. Para estatística usa-se `nome_normalizado` (trim, colapso de espaços, minúsculas, remoção de acentos) e, na Central, aliases opcionais (RF-C11).

## Área
- **RN-40** Área opcional, decimal ≥ 0, com unidade **hectares** ou **alqueire paulista**.
- **RN-41** **1 alqueire paulista = 2,42 ha = 24.200 m²**; 1 ha = 10.000 m².
- **RN-42** Se houver valor, a unidade é obrigatória. Sem valor, ambos ficam vazios.
- **RN-43** O valor e a unidade digitados são **preservados**. A conversão serve só para cálculos e exibição.
- **RN-44** Armazenamento em **milésimos** da unidade (inteiro, até 3 casas decimais). Cálculo canônico em **dm² inteiros**: 1 milésimo de ha = 1.000 dm²; 1 milésimo de alqueire = 2.420 dm². O arredondamento acontece **só na exibição** (ha com 2 casas, HALF_UP). Casos de referência: `shared/test-fixtures/area/conversao.json`.
- **RN-45** Propriedade **sem área não entra na soma como zero**. É contada à parte ("N sem área informada").

## Coordenadas
- **RN-50** WGS84, graus decimais, latitude e longitude opcionais (ambas ou nenhuma).
- **RN-51** Latitude ∈ [-90, 90]; longitude ∈ [-180, 180].
- **RN-52** Origem: `gnss` ou `manual`. GNSS registra precisão horizontal (m), instante da captura (UTC) e satélites, quando disponível. Coordenada manual **não tem precisão inferida**.
- **RN-53** A meta desejável é precisão ≤ 15 m, **não garantida**. Acima disso, o app avisa e permite repetir ou salvar mesmo assim.
- **RN-54** A coordenada é um **ponto coletado**: não é limite fundiário nem centro garantido da propriedade.
- **RN-55** Obter GNSS **nunca** depende de internet ou de mapas.

## Fotografias
- **RN-60** Várias fotos por propriedade. A foto gravada é reduzida (lado maior de cerca de 2560 px, JPEG 85) e esse arquivo é o "original do sistema".
- **RN-61** O SHA‑256 é calculado sobre o arquivo **efetivamente gravado**.
- **RN-62** Remover foto exige confirmação e gera tombstone.
- **RN-63** A foto só é **confirmada** depois de a Central verificar tamanho e hash e persistir o registro.

## Sincronização e conflitos
- **RN-70** A sincronização é **em sentido único: Android → Central**. Exceções: o recibo informa ao celular o destino confirmado da coleta (RN-09), e a futura restauração de cadastros (RF-A15).
- **RN-71** Edições feitas na Central são **correções da Central** e não voltam aos aparelhos.
- **RN-72** Se o Android enviar nova versão (inclusive exclusão) de um cadastro que a Central corrigiu depois da última versão recebida, abre-se um **conflito manual**. Nada é sobrescrito nem apagado em silêncio. Detalhes: [conflitos.md](../synchronization/conflitos.md).
- **RN-73** A unidade de conflito é o **cadastro**: a propriedade com seus itens e os metadados das fotos. *(Recomendado R3, aprovado com o plano.)*
- **RN-74** Enviado ≠ importado. O Android só marca como confirmado o que consta em **recibo persistido** pela Central. Um envio **cancelado** pelo operador não confirma nada, não apaga nada e pode ser refeito.

### Correções e exclusões na Central (A29, ADR-0026)
- **RN-75** A Central pode corrigir os dados de um cadastro, sem mudar seus vínculos de origem. Até a fila de conflitos, novas versões do cadastro ou de seus filhos são recusadas com motivo e permanecem pendentes no aparelho; versões já processadas continuam idempotentes. Toda correção e exclusão é auditada na mesma transação.
- **RN-76** Excluir estudo gera tombstones nas jornadas, equipes e cadastros vinculados, incluindo itens/fotos; excluir propriedade alcança seus itens/fotos. Coletas e histórico de sincronização são preservados; lotes pendentes associados ao estudo são cancelados e destinos excluídos não recebem nova aplicação. Arquivos e cartas emitidas não são apagados. A exclusão isolada de equipe continua restrita à ausência de coleta.

## Estatísticas
- **RN-80** Consideram apenas registros **vigentes** (sem tombstone), na última versão aplicada.
- **RN-81** Os filtros são combináveis e aplicados de forma **idêntica** a cartões, tabelas, gráficos, listas, mapa e exportações.
- **RN-82** Sempre há visão geral e detalhamento, com drill-down até o cadastro individual.
- **RN-83** Estatísticas **nunca dependem** de mapa ou de internet.

## Mapas e cartas-imagem
- **RN-90** Obter imagem de satélite nova **exige internet** nesta versão (os blocos de imagem e as composições da área consultada também são guardados, ADR-0027). É proibido mapa-base offline e pré-download. As imagens que o operador já consultou e as cartas-imagem geradas ficam guardadas na pasta de dados, **sem validade**, e podem ser reabertas sem internet; só a imagem bruta pode ser apagada pelo operador (ADR-0022, que alterou a regra em 14/11/2026).
- **RN-91** Sem conexão efetiva ao serviço, exibir: *"O mapa e a carta-imagem precisam de conexão com a internet. Seus cadastros, coordenadas, estatísticas e filtros continuam disponíveis offline. Conecte-se e tente novamente."*, com os botões **Tentar novamente** e **Voltar**.
- **RN-92** A carta-imagem traz título "SITPOSSE-7 — 7º BPM/PMRO", identificação, coordenadas, legenda, data, fonte/atribuição e a ressalva *"Ponto(s) coletado(s) no levantamento; não representa(m) limites fundiários nem documento cartográfico oficial"*. Escala e norte só aparecem se forem tecnicamente corretos.
- **RN-93** Não é gerada carta-imagem com área **sem nenhuma imagem**. A carta pode ser um **mosaico** de cenas de datas e resoluções diferentes, desde que sinalizado (RN-97). Sem conexão, usa-se a composição já guardada da mesma área, com aviso da data; sem ela, aparece a mensagem da RN-91. *(Reescrita em 15/11/2026, ADR-0027.)*
- **RN-94** Não se inventam coordenadas para propriedades sem GPS.
- **RN-95** A imagem da carta é escolhida **automaticamente**:
  - **Ordem de preferência:** CBERS-4A de 2 m com até 12 meses; Sentinel-2 de 10 m mais recente sem nuvens; mosaico trimestral do INPE; Copernicus (CDSE) só se o INPE estiver inacessível.
  - **Critério:** nuvem e área sem dados **sobre a área das propriedades**, não sobre a cena inteira.
  - **Nuvem estimada:** quando a nuvem é estimada (CBERS), a interface diz "estimativa".
  - **Reprodutibilidade:** a escolha é reproduzível e fica gravada.
  - **Transparência:** o operador sempre vê a fonte, a data (ou o intervalo) e a resolução. *(ADR-0027.)*
- **RN-96** Cada propriedade recebe um **número fixo no estudo**:
  - atribuído quando o cadastro chega à Central, em ordem de recebimento;
  - nunca reutilizado, nem depois de uma exclusão;
  - é o número dos pinos, das legendas e de todas as folhas e arquivos das cartas. *(ADR-0027.)*
- **RN-97** Quando a carta geral fica com detalhe muito menor que o da imagem, ou com pinos sobrepostos, são geradas automaticamente **folhas de setor**, além da geral: o PDF reúne todas as folhas e há um PNG por folha. Um mosaico traz o quadro de proveniência (fonte, data e resolução de cada parte) e nunca é apresentado como imagem de uma data só. *(ADR-0027.)*
- **RN-98** A carta-imagem usa sempre as **coordenadas atuais** das propriedades.
  - **Detecção:** ao voltar à tela de cartas e antes de gerar, a Central relê as propriedades da seleção.
  - **Mudança que exige nova imagem:** coordenada alterada (edição na Central ou sincronização), ou propriedade que entrou ou saiu da seleção. A tela avisa quais números mudaram e oferece **"Atualizar referências"**, que refaz o enquadramento e os setores e prepara a imagem da área.
  - **Bloqueio:** enquanto isso não for feito, a carta não é gerada (a tela desabilita o botão e a Central recusa).
  - **Mudança só de nome, responsável ou número:** basta redesenhar os pinos; a imagem continua valendo.
  - **Sem internet:** vale a imagem preparada antes para as mesmas propriedades, com o enquadramento anterior e aviso, **somente** se todos os pinos continuarem dentro da folha geral e, havendo setores, de algum setor. Caso contrário, aparece a mensagem da RN-91.
  - **Cartas já geradas** não são alteradas: são o registro do que foi gerado naquela data.

  *(15/11/2026, A31, ADR-0027 §Complemento.)*
