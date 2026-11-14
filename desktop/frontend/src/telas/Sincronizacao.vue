<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import QRCode from 'qrcode'
import { OperacaoService as Op, SincronizacaoService as Sinc } from '../../bindings/sitposse7/desktop/app'
import type { Info } from '../../bindings/sitposse7/desktop/app/models'
import type { LoteLinha, Rejeicao } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import type { Operacao } from '../../bindings/sitposse7/desktop/internal/pareamento/models'
import { mensagem, quando } from '../formato'

const info = ref<Info | null>(null)
const op = ref<Operacao | null | undefined>(undefined) // undefined = carregando
const lotes = ref<LoteLinha[]>([])
const erro = ref('')
const nomeOperacao = ref('')
const criando = ref(false)

const qr = ref('')
const payload = ref('')
const expiraEm = ref(0)
const agora = ref(Date.now())
const restante = computed(() => Math.max(0, Math.round((expiraEm.value - agora.value) / 1000)))

const rejeicoes = ref<Record<string, Rejeicao[]>>({})

async function carregar() {
  try {
    const [i, o, l] = await Promise.all([Sinc.Info(), Op.Ativa(), Sinc.Lotes()])
    info.value = i
    op.value = o
    lotes.value = l ?? []
  } catch (e) { erro.value = mensagem(e) }
}

async function criarOperacao() {
  erro.value = ''
  criando.value = true
  try { op.value = await Op.Criar(nomeOperacao.value, ''); nomeOperacao.value = '' } catch (e) { erro.value = mensagem(e) } finally { criando.value = false }
}

async function gerarQr() {
  erro.value = ''
  try {
    const q = await Sinc.GerarQRPareamento()
    if (!q) return
    payload.value = JSON.stringify(q.payload)
    expiraEm.value = q.expira_em
    qr.value = await QRCode.toDataURL(payload.value, { errorCorrectionLevel: 'M', margin: 2, width: 360 })
    info.value = await Sinc.Info()
  } catch (e) { qr.value = ''; erro.value = mensagem(e) }
}

async function verRejeicoes(lote: string) {
  if (rejeicoes.value[lote]) { delete rejeicoes.value[lote]; return }
  rejeicoes.value[lote] = (await Sinc.Rejeicoes(lote)) ?? []
}

let t1: number | undefined
let t2: number | undefined
onMounted(() => {
  carregar()
  t1 = window.setInterval(carregar, 3000) // recepção ao vivo
  t2 = window.setInterval(() => { agora.value = Date.now(); if (qr.value && restante.value === 0) qr.value = '' }, 1000)
})
onUnmounted(() => { window.clearInterval(t1); window.clearInterval(t2) })
</script>

<template>
  <div class="pilha">
    <h1>Sincronização</h1>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>

    <div v-if="op === null" class="cartao pilha">
      <h2>Criar a operação</h2>
      <p class="suave" style="margin: 0">Os celulares são pareados com a operação ativa. Dê um nome que identifique o levantamento (ex.: “Levantamento Linha 25 — novembro/2026”).</p>
      <form class="linha" @submit.prevent="criarOperacao">
        <input v-model="nomeOperacao" maxlength="200" placeholder="Nome da operação" style="flex: 1 1 320px" />
        <button class="primario" :disabled="!nomeOperacao.trim() || criando">Criar operação</button>
      </form>
    </div>

    <div v-else-if="op" class="grade-sinc">
      <div class="cartao pilha">
        <h2>Parear um celular</h2>
        <ol class="passos">
          <li>Ligue o <strong>hotspot móvel</strong> deste computador (com internet na origem) e conecte o celular a ele.</li>
          <li>No celular: <strong>Sincronizar → Ler QR de pareamento</strong> e aponte para o QR abaixo.</li>
          <li>Confirme o pedido que vai aparecer nesta tela.</li>
        </ol>
        <div class="qr-area">
          <template v-if="qr">
            <img :src="qr" alt="QR de pareamento" width="300" height="300" />
            <p class="suave" style="margin: 0">Válido por {{ Math.floor(restante / 60) }}:{{ String(restante % 60).padStart(2, '0') }} · uso único · operação {{ op.nome }}</p>
          </template>
          <p v-else-if="!info?.ips?.length" class="aviso">Nenhuma rede Wi‑Fi/local ativa. Ligue o hotspot do Windows e gere o QR.</p>
          <button class="primario" :disabled="!info?.servidor_ativo" @click="gerarQr">{{ qr ? 'Gerar outro QR' : 'Gerar QR de pareamento' }}</button>
        </div>
      </div>

      <div class="cartao pilha">
        <h2>Esta Central</h2>
        <dl class="dados">
          <dt>Servidor</dt><dd :class="info?.servidor_ativo ? 'ok' : 'erro'">{{ info?.servidor_ativo ? `Ativo na porta ${info?.porta}` : info?.erro_servidor || 'Inativo' }}</dd>
          <dt>Endereços</dt><dd>{{ info?.ips?.length ? info.ips.join(' · ') : 'nenhum' }}</dd>
          <dt>Operação</dt><dd>{{ op.nome }}</dd>
          <dt>Impressão digital</dt><dd class="mono">{{ info?.pin }}</dd>
          <dt>ID da Central</dt><dd class="mono">{{ info?.central_id }}</dd>
          <dt>Pasta de dados</dt><dd class="mono">{{ info?.pasta_dados }}</dd>
          <dt>Esquema do banco</dt><dd>{{ info?.versao_esquema }}</dd>
        </dl>
        <p v-if="info?.fotos_sem_arquivo?.length" class="aviso-erro">{{ info.fotos_sem_arquivo.length }} foto(s) registrada(s) sem arquivo na pasta de dados. Não apague nada e avise o suporte.</p>
      </div>
    </div>

    <div class="cartao pilha">
      <h2>Lotes recebidos</h2>
      <div class="rolavel">
        <table class="tabela">
          <thead><tr><th>Recebido</th><th>Equipe</th><th>Aparelho</th><th>Situação</th><th class="num">Aplicados</th><th class="num">Já existentes</th><th class="num">Recusados</th></tr></thead>
          <tbody>
            <template v-for="l in lotes" :key="l.uuid">
              <tr @click="l.rejeitados ? verRejeicoes(l.uuid) : undefined">
                <td>{{ quando(l.recebido_em) }}</td><td>{{ l.equipe }}</td><td>{{ l.aparelho || '—' }}</td>
                <td><span class="selo" :class="l.status === 'importado' ? 'ok' : 'alerta'">{{ l.status === 'importado' ? `importado ${quando(l.importado_em)}` : 'recebendo…' }}</span></td>
                <td class="num">{{ l.aplicados }}</td><td class="num">{{ l.duplicados }}</td>
                <td class="num" :class="{ erro: l.rejeitados }">{{ l.rejeitados }}</td>
              </tr>
              <tr v-if="rejeicoes[l.uuid]"><td colspan="7"><ul class="rej"><li v-for="r in rejeicoes[l.uuid]" :key="r.uuid">{{ r.entidade }} <span class="mono">{{ r.uuid }}</span>: {{ r.motivo }}</li></ul></td></tr>
            </template>
          </tbody>
        </table>
        <div v-if="!lotes.length" class="vazio">Nenhum lote recebido ainda.</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.grade-sinc { display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(360px, 1fr)); align-items: start; }
.passos { margin: 0; padding-left: 1.2rem; display: flex; flex-direction: column; gap: .35rem; }
.qr-area { display: flex; flex-direction: column; align-items: center; gap: .75rem; text-align: center; }
.qr-area img { background: #fff; border-radius: 10px; padding: 6px; }
.rej { margin: 0; padding-left: 1.1rem; color: var(--erro); }
</style>
