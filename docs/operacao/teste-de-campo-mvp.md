# Roteiro do teste em campo do MVP (A24)

Objetivo: cadastrar propriedades **sem internet** numa área rural, voltar e descarregar tudo na Central pelo **hotspot do Windows com internet** (A21), conferindo que nada se perdeu nem duplicou.

## Antes de sair (na base)
1. **Central:** instale `sitposse7-central-amd64-installer.exe` no computador que vai receber os dados (não precisa desinstalar a versão anterior; os dados ficam em `%LOCALAPPDATA%\SITPOSSE-7`).
2. Abra a Central → **Sincronização** → crie a **operação** (ex.: "Teste de campo — novembro/2026").
3. **Celular:** instale `sitposse7-campo-debug.apk` por cima do app anterior (mesma chave de debug desta máquina; o banco migra sozinho para a versão 2). Abra e configure a **equipe** se pedir.
4. **Parear** (precisa estar perto da Central uma vez):
   1. Windows: ligue o **Hotspot móvel** (com internet na origem; "Economia de energia" desativada) e conecte o celular a ele.
   2. Central: **Sincronização → Gerar QR de pareamento**.
   3. Celular: **Sincronizar → Ler QR de pareamento** → aponte para o QR.
   4. Central: confirme o pedido que aparece na tela. O celular mostra "Nada pendente" (ou envia o que tiver).
5. Confira no celular: GPS ligado; permissão de localização e de câmera concedidas na primeira vez que forem pedidas.

## No campo (modo avião ou sem sinal)
- Para cada propriedade: **+ Nova propriedade** → nome (único campo obrigatório) → quantitativos → itens (inclua um "Outros") → área (teste ha e alqueire) → **Obter localização** (em local aberto; anote a precisão) → **Tirar foto** (2 ou 3) → **Salvar**. Deve aparecer o detalhe do cadastro.
- Teste também: sair do formulário no meio (o rascunho deve aparecer na tela inicial para retomar), girar a tela, fechar o app à força e reabrir, editar um cadastro já salvo, excluir um cadastro de teste.
- Anote: aparelho, quantos cadastros e fotos, precisões do GPS, qualquer travamento ou mensagem estranha.

## Na volta (descarga)
1. Windows: ligue o hotspot; conecte o celular.
2. Celular: **Sincronizar → Enviar para a Central**. Acompanhe "Enviando cadastros" e "Enviando fotos". Resultado esperado: "N cadastro(s) confirmado(s) · 0 em análise · 0 erro(s)". Na tela inicial, **Pendentes = 0**.
3. Central: **Painel** e **Propriedades** — confira contagens, itens, área, coordenadas e **fotos** de cada cadastro; **Sincronização → Lotes recebidos** deve mostrar o lote importado sem recusados.
4. **Prova de não duplicação:** toque em **Enviar** de novo no celular → "Nada pendente". Edite um cadastro e envie → só ele vai (1 aplicado).
5. Opcional: desligue o hotspot no meio do envio de fotos e envie de novo; deve continuar de onde parou, sem duplicar.

## Registrar
Resultados, prints e anotações → `docs/planning/continuacao.md` (próxima sessão) e, se houver defeito, uma pendência.
