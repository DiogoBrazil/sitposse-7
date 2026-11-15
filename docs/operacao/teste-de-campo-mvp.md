# Roteiro do teste em campo do MVP (A24, atualizado pela A28)

Objetivo: cadastrar propriedades **sem internet** numa área rural, com o **código da equipe** gerado pela Central, voltar e descarregar tudo na Central pelo **hotspot do Windows com internet** (A21), **confirmando o destino** de cada coleta e conferindo que nada se perdeu nem duplicou.

Versões: Central `0.6.0-mvp` (`sitposse7-central-amd64-installer.exe`) e Campo `0.2.0-mvp` (`sitposse7-campo-debug.apk`), em `Downloads\sitposse7-teste`.

## Reinício dos dados de teste (só nesta mudança: ADR-0025)
A Central `0.6.0` e o Campo `0.2.0` usam uma nova estrutura de banco (estudos, jornadas, equipes e coletas). Os bancos do MVP são **recusados sem serem alterados**: a Central mostra "Dados de teste antigos" e o app mostra a mesma mensagem. Para continuar:

1. **Central** (no computador que for a Central; feche a Central antes):
   - Em `%LOCALAPPDATA%\SITPOSSE-7`, crie a pasta `mvp-antigo-AAAAMMDD-HHMMSS` e **mova** para ela (não apague): `sitposse7.db`, `sitposse7.db-wal` e `sitposse7.db-shm` (se existirem), e as pastas `fotos` e `staging`.
   - **Não mova** `tls` (identidade da Central) nem `cartografia` (imagens e cartas guardadas). As credenciais do Copernicus ficam no Gerenciador de Credenciais do Windows e não são afetadas.
   - Instale `sitposse7-central-amd64-installer.exe` por cima e abra: a Central cria o banco novo vazio.
2. **Celular:** desinstale o **SITPOSSE-7 Campo** (isso apaga só os dados dele) e instale `sitposse7-campo-debug.apk`.
3. Nada fora da pasta de dados do SITPOSSE-7 é tocado. Este procedimento vale só para esta mudança; as próximas atualizações voltam a migrar o banco sem apagar nada.

## Antes de sair (na base)
1. **Central → Estudos → Novo estudo**: nome (ex.: "Teste de campo — Fazenda X"), data da 1ª jornada e as equipes (uma por linha, ex.: "Equipe Alfa", "Equipe Bravo"). Anote o **código de cada equipe** (botão **Copiar** ou **Copiar todos os códigos**).
2. **Celular:** na tela **Código da equipe**, digite o código da equipe (ex.: `EQ-7K3M-9QX1`). Teste também um código com um caractere trocado: o app deve recusar ("Confira os caracteres do código").
3. **Parear o celular** (uma vez por aparelho):
   1. Windows: ligue o **Hotspot móvel** (com internet na origem; "Economia de energia" desativada) e conecte o celular a ele.
   2. Central: **Sincronização → Gerar QR de pareamento**.
   3. Celular: no Início, **Sincronizar → Ler QR de pareamento** → aponte para o QR.
   4. Central: confirme o pedido do **aparelho**.
4. Confira: GPS ligado; permissões de localização e câmera concedidas na primeira vez.

## No campo (modo avião ou sem sinal)
- Para cada propriedade: **+ Nova propriedade** → percorra as **7 etapas** (Identificação e família → Veículos → Eletrodomésticos e mobiliário → Animais e área → Localização e fotografias → Observações relevantes → Revisão) → **Salvar cadastro**. Obrigatórios: **nome da propriedade** e **nome do responsável**.
- Teste também:
  - avançar sem preencher nada e tentar salvar (a Revisão mostra o que falta e leva à etapa);
  - tocar no cabeçalho "Etapa x de 7" e pular direto para uma etapa; na Revisão, **Editar** uma categoria;
  - observação longa com quebras de linha;
  - sair no meio (o rascunho aparece no Início), girar a tela, ir para a câmera e voltar, mandar o app para o segundo plano, **fechar o app à força** e reabrir (deve voltar direto à coleta);
  - editar e excluir um cadastro já salvo.
- **Segunda jornada no mesmo aparelho:** no dia seguinte (ou com um segundo código), o app avisa "coleta de outro dia"; informe o novo código e responda **Concluir** à pergunta sobre a coleta anterior.
- Anote: aparelho, quantos cadastros e fotos, precisões do GPS, travamentos ou mensagens estranhas.

## Na volta (descarga)
1. Windows: ligue o hotspot; conecte o celular.
2. Celular: **Sincronizar** → marque as coletas → **Enviar para a Central**. Acompanhe "Enviando cadastros", "Enviando fotos" e **"Aguardando o operador confirmar o destino"**.
3. Central: aparece **Confirmar destino da sincronização** com o estudo, a jornada, a equipe e o código → **Confirmar e importar**. No celular: "N cadastro(s) confirmado(s) em Equipe … · Jornada … · Estudo …"; **Pendentes = 0**.
4. **Código errado:** antes de sair, crie na Central uma equipe extra (ex.: "Equipe Teste"), anote o código e **exclua** a equipe: o código fica válido no formato, mas não existe mais. Faça uma coleta com ele e envie. A Central mostra "Código de equipe não encontrado":
   - **Corrigir no Central** (Estudo → Jornada → Equipe) → importa, e o celular mostra "código corrigido na Central"; ou
   - **Cancelar sincronização** → o celular mostra o cancelamento → **Corrigir o código** no celular → envie de novo.
5. Central: **Painel** e **Propriedades** (estudo em foco) — confira contagens, jornadas, equipes, itens, área, coordenadas, observações e **fotos**; **Sincronização → Lotes recebidos** com o código recebido, o destino e a situação.
6. **Prova de não duplicação:** **Enviar** de novo → "Nada pendente". Edite um cadastro e envie → só ele vai; o destino vem travado (a Central só confirma ou cancela).
7. Opcional: desligue o hotspot no meio do envio de fotos ou enquanto a Central espera a confirmação; envie de novo: deve continuar de onde parou, sem duplicar.

## Registrar
Resultados, prints e anotações → `docs/planning/continuacao.md` (próxima sessão) e, se houver defeito, uma pendência.
