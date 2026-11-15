# Requisitos funcionais

As regras detalhadas (campos, validações, conversões) estão em [regras-de-negocio.md](regras-de-negocio.md). Este documento lista **funções**.

## RF-A — SITPOSSE-7 Campo (Android, 100% offline)
| ID | Requisito |
|---|---|
| RF-A01 | Iniciar a coleta informando só o **código da equipe** (validado offline: formato e dígito verificador). Gerar o UUID da coleta e, na primeira abertura, o UUID da instalação (RN-05). |
| RF-A02 | Gerir as coletas do aparelho: retomar, escolher entre várias em andamento, concluir, reabrir, corrigir o código (RN-07, RN-08) e ver as pendentes de sincronização. |
| RF-A03 | Criar, consultar, editar e excluir (tombstone) propriedades/famílias offline, com formulário **em etapas** e revisão antes de salvar; inclui **Observações relevantes** (RN-15). |
| RF-A04 | Registrar quantitativos fixos: pessoas, veículos, bovinos e equinos. |
| RF-A05 | Registrar itens dinâmicos (eletrodomésticos, móveis, aves), com vários "Outros" de nome livre; editar e remover itens. |
| RF-A06 | Registrar a área (ha ou alqueire paulista) e exibir a conversão. |
| RF-A07 | Obter coordenadas via GNSS sem internet, mostrando progresso, precisão, satélites e opção de repetir; permitir entrada manual. |
| RF-A08 | Capturar várias fotos por propriedade; exibir miniaturas, visualizar e remover com confirmação. |
| RF-A09 | Lista pesquisável com estados de persistência e sincronização. |
| RF-A10 | Preservar rascunho do formulário em rotação, navegação ou interrupção, sem confundi-lo com cadastro salvo. |
| RF-A11 | Parear com a Central via QR Code e enviar alterações incrementais e fotos pela rede Wi‑Fi local. |
| RF-A12 | Exportar pacote de contingência (USB) e importar o recibo da Central. |
| RF-A13 | Exportar backup completo para local escolhido pelo usuário. |
| RF-A14 | Liberar espaço de fotos somente quando confirmadas e cobertas por backup verificado da Central, por ação explícita (regra final pendente: P3/P4). |
| RF-A15 | Receber restauração dos cadastros a partir da Central, após reinstalação. |

## RF-C — SITPOSSE-7 Central (Windows)
| ID | Requisito | Internet? |
|---|---|---|
| RF-C01 | Gerir **estudos de situação**, suas **jornadas** e as **equipes** de cada jornada, com códigos gerados pela Central; copiar equipes de uma jornada anterior (com códigos novos); escolher o estudo em foco (RN-01, RN-02). | Não |
| RF-C02 | Parear aparelhos (QR Code) e receber lotes por Wi‑Fi local, com **confirmação, correção ou cancelamento do destino** pelo operador antes de importar (RN-09). | Não |
| RF-C03 | Importar pacote de contingência e gerar recibo (arquivo e QR). | Não |
| RF-C04 | Consultar, criar e editar (correção da Central) propriedades, itens e fotos, com auditoria. | Não |
| RF-C05 | Fila de conflitos com resolução manual campo a campo. | Não |
| RF-C06 | Painel com estatísticas, filtros combináveis, tabelas analíticas, gráficos e drill-down ([estatisticas.md](estatisticas.md), §22-A). | Não |
| RF-C07 | Relatórios não cartográficos (PDF/CSV) respeitando os filtros. | Não |
| RF-C08 | Backup verificado e restauração. | Não |
| RF-C09 | Histórico de sincronizações, erros e auditoria. | Não |
| RF-C10 | Restaurar cadastros em aparelho reinstalado. | Não |
| RF-C11 | Agrupar nomes de "Outros" por aliases e sugerir possíveis duplicatas sem fundir (P5). | Não |
| RF-C12 | Mapa individual e geral com imagem online, integrado aos filtros. | **Sim** |
| RF-C13 | Carta-imagem individual e geral (PNG/PDF), se o provedor permitir. | **Sim** |
| RF-C14 | Sem internet, mostrar a mensagem amigável apenas nas funções RF-C12 e RF-C13. | — |
