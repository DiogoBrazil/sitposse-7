<script setup lang="ts">
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { SincronizacaoService as Sinc } from '../../bindings/sitposse7/desktop/app'
import type { Info } from '../../bindings/sitposse7/desktop/app/models'
import type { LoteLinha, Rejeicao } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import TabelaDados from '../componentes/TabelaDados.vue'
import { mensagem, num, quando } from '../formato'
import type { Coluna } from '../tabela'

const emit = defineEmits<{ info: [] }>()
const info = ref<Info | null>(null)
const lotes = ref<LoteLinha[]>([])
const erro = ref('')

const qr = ref('')
const gerando = ref(false)
const expiraEm = ref(0)
const agora = ref(Date.now())
const restante = computed(() => Math.max(0, Math.round((expiraEm.value - agora.value) / 1000)))

const loteAberto = ref<LoteLinha | null>(null)
const rejeicoes = ref<Rejeicao[]>([])

async function carregar() {
  try {
    const [i, l] = await Promise.all([Sinc.Info(), Sinc.Lotes()])
    info.value = i; lotes.value = l ?? []
  } catch (e) { erro.value = mensagem(e) }
}

async function gerarQr() {
  if (gerando.value) return
  erro.value = ''
  gerando.value = true
  try {
    const q = await Sinc.GerarQRPareamento()
    if (!q) return
    expiraEm.value = q.expira_em
    qr.value = await QRCode.toDataURL(JSON.stringify(q.payload), { errorCorrectionLevel: 'M', margin: 2, width: 360 })
    info.value = await Sinc.Info()
  } catch (e) { qr.value = ''; erro.value = mensagem(e) } finally { gerando.value = false }
}

async function abrirLote(l: LoteLinha) {
  if (!l.rejeitados) return
  loteAberto.value = l
  try { rejeicoes.value = (await Sinc.Rejeicoes(l.uuid)) ?? [] } catch (e) { erro.value = mensagem(e) }
}

const colLotes: Coluna<LoteLinha>[] = [
  { chave: 'recebido_em', rotulo: 'Recebido em', formatar: (l) => quando(l.recebido_em) },
  { chave: 'codigo_informado', rotulo: 'Código recebido' },
  { chave: 'equipe', rotulo: 'Destino' },
  { chave: 'aparelho', rotulo: 'Aparelho', secundaria: true },
  { chave: 'status', rotulo: 'Situação' },
  { chave: 'aplicados', rotulo: 'Registros novos ou atualizados', tipo: 'numero' },
  { chave: 'duplicados', rotulo: 'Já existentes', tipo: 'numero' },
  { chave: 'rejeitados', rotulo: 'Recusados', tipo: 'numero' },
]

function situacao(l: LoteLinha): { texto: string; classe: string } {
  switch (l.status) {
    case 'importado': return { texto: 'Importado', classe: 'ok' }
    case 'aguardando_confirmacao': return { texto: 'Aguardando confirmação', classe: 'alerta' }
    case 'cancelado': return { texto: 'Cancelado', classe: 'erro' }
    default: return { texto: 'Recebendo…', classe: 'alerta' }
  }
}

// Recepção ao vivo (3 s) e contagem do QR, só com a tela visível.
let t1: number | undefined
let t2: number | undefined
function iniciar() {
  carregar()
  window.clearInterval(t1); window.clearInterval(t2)
  t1 = window.setInterval(carregar, 3000)
  t2 = window.setInterval(() => { agora.value = Date.now(); if (qr.value && restante.value === 0) qr.value = '' }, 1000)
}
function parar() { window.clearInterval(t1); window.clearInterval(t2) }
onMounted(iniciar); onActivated(iniciar); onDeactivated(parar); onUnmounted(parar)
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <div class="titulo">
        <h1>Sincronização</h1>
        <p class="suave">Receba os cadastros dos celulares pela rede Wi‑Fi deste computador, sem internet.</p>
      </div>
      <button v-if="info" class="estado" :class="info.servidor_ativo ? 'ativo' : 'parado'" @click="emit('info')">
        <span class="ponto" aria-hidden="true"></span>
        {{ info.servidor_ativo ? `Recepção ativa · ${info.ips?.length ?? 0} rede(s)` : 'Recepção parada' }}
      </button>
    </header>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>
    <p v-if="info?.fotos_sem_arquivo?.length" class="aviso-erro">{{ info.fotos_sem_arquivo.length }} foto(s) registrada(s) sem arquivo na pasta de dados. Não apague nada e avise o suporte.</p>

    <section class="bloco parear">
      <div class="passos">
        <h2>Parear um celular</h2>
        <ol>
          <li><span><strong>Ligue o hotspot móvel</strong> deste computador (com internet na origem) e conecte o celular a ele.</span></li>
          <li><span>No celular, toque em <strong>Sincronizar</strong> e depois em <strong>Ler QR de pareamento</strong>.</span></li>
          <li><span>Aponte a câmera para o QR desta tela.</span></li>
          <li><span>Confirme o pedido que vai aparecer aqui. Depois, o celular envia os cadastros e você <strong>confirma o destino</strong> de cada coleta (estudo, jornada e equipe pelo código).</span></li>
        </ol>
        <p class="suave">Um celular já pareado não precisa de QR: basta tocar em “Enviar para a Central”. Os códigos das equipes ficam em <strong>Estudos</strong>.</p>
      </div>
      <div class="qr">
        <div class="moldura" :class="{ vazia: !qr }">
          <img v-if="qr" :src="qr" alt="QR de pareamento" width="260" height="260" />
          <p v-else-if="!info?.ips?.length" class="aviso">Nenhuma rede Wi‑Fi ativa. Ligue o hotspot do Windows e gere o QR.</p>
          <p v-else class="fraco">O QR aparece aqui. Ele vale por 10 minutos e serve para um celular.</p>
        </div>
        <p v-if="qr" class="validade">Válido por <strong>{{ Math.floor(restante / 60) }}:{{ String(restante % 60).padStart(2, '0') }}</strong> · uso único</p>
        <button class="primario" :disabled="!info?.servidor_ativo || gerando" @click="gerarQr">{{ gerando ? 'Gerando…' : qr ? 'Gerar outro QR' : 'Gerar QR de pareamento' }}</button>
      </div>
    </section>

    <section class="pilha secao">
      <div class="linha" style="justify-content: space-between">
        <h2>Lotes recebidos</h2>
        <span class="fraco">Atualiza sozinho enquanto esta tela está aberta.</span>
      </div>
      <TabelaDados :colunas="colLotes" :linhas="lotes" :chave="(l) => l.uuid" rotulo="Lotes recebidos" :ordem="{ chave: 'recebido_em', dir: 'desc' }"
                   texto-vazio="Nenhum lote recebido ainda." clicavel @abrir="abrirLote">
        <template #status="{ linha }">
          <span class="selo" :class="situacao(linha).classe">{{ situacao(linha).texto }}</span>
          <span v-if="linha.importado_em" class="fraco quando">{{ quando(linha.importado_em) }}</span>
          <span v-if="linha.status === 'cancelado'" class="fraco quando">{{ linha.motivo === 'codigo_inexistente' ? 'Código não encontrado' : 'Cancelado pelo operador' }}</span>
        </template>
        <template #codigo_informado="{ linha }"><span class="mono">{{ linha.codigo_informado }}</span></template>
        <template #equipe="{ linha }">
          <template v-if="linha.equipe">
            <strong>{{ linha.equipe }}</strong> <span class="mono fraco">{{ linha.equipe_codigo }}</span>
            <span class="fraco quando">{{ linha.estudo }} · {{ linha.jornada }}</span>
            <span v-if="linha.corrigido" class="selo alerta">Corrigido na Central</span>
          </template>
          <span v-else class="fraco">—</span>
        </template>
        <template #aparelho="{ linha }">{{ linha.aparelho || '—' }}</template>
        <template #rejeitados="{ linha }">
          <span v-if="linha.rejeitados" class="erro"><u>{{ num(linha.rejeitados) }}</u></span>
          <span v-else>0</span>
        </template>
      </TabelaDados>
    </section>

    <div v-if="loteAberto" class="modal-fundo" @click.self="loteAberto = null">
      <div class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="rej-titulo">
        <h2 id="rej-titulo">Registros recusados</h2>
        <p class="suave">Lote de {{ quando(loteAberto.recebido_em) }}, equipe {{ loteAberto.equipe ?? '—' }}. Esses registros continuam pendentes no celular.</p>
        <ul class="rej"><li v-for="r in rejeicoes" :key="r.uuid"><strong>{{ r.entidade }}</strong>: {{ r.motivo }}<br /><span class="mono fraco">{{ r.uuid }}</span></li></ul>
        <div class="linha" style="justify-content: flex-end"><button autofocus @click="loteAberto = null">Fechar</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.estado { display: inline-flex; align-items: center; gap: .5rem; border-radius: 999px; font-weight: 600; }
.estado .ponto { width: 9px; height: 9px; border-radius: 50%; background: var(--erro); }
.estado.ativo .ponto { background: var(--ok); box-shadow: 0 0 0 3px var(--ok-fundo); }
.parear { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 2rem; padding: 1.4rem 1.5rem; align-items: center; }
.passos { display: flex; flex-direction: column; gap: .9rem; }
.passos ol { margin: 0; padding: 0; list-style: none; counter-reset: passo; display: flex; flex-direction: column; gap: .7rem; }
.passos li { counter-increment: passo; display: grid; grid-template-columns: 1.9rem 1fr; gap: .6rem; align-items: baseline; }
.passos li::before { content: counter(passo); font-weight: 750; font-stretch: 80%; font-size: 1.25rem; color: var(--mata); text-align: center; border: 1.5px solid var(--regua-forte); border-radius: 50%; width: 1.9rem; height: 1.9rem; line-height: 1.75rem; }
.qr { display: flex; flex-direction: column; align-items: center; gap: .7rem; text-align: center; }
.moldura { width: 276px; height: 276px; border-radius: var(--raio-grande); background: #fff; display: grid; place-items: center; border: 1px solid var(--regua); }
.moldura.vazia { background: var(--folha-2); border-style: dashed; padding: 1.25rem; }
.moldura img { display: block; }
.validade { font-size: .95rem; }
.secao { gap: .75rem; }
.quando { display: block; font-size: .82rem; margin-top: .15rem; }
.rej { margin: 0; padding-left: 1.1rem; display: flex; flex-direction: column; gap: .6rem; }
@media (max-width: 860px) { .parear { grid-template-columns: 1fr; } }
</style>
