<script setup lang="ts">
/**
 * Acesso ao Copernicus (ADR-0022): credenciais do cliente OAuth guardadas no Gerenciador
 * de Credenciais do Windows, teste de conexão e espaço usado pelas imagens e cartas.
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { CartaService as Cartas } from '../../bindings/sitposse7/desktop/app'
import type { EstadoConexao } from '../../bindings/sitposse7/desktop/app/models'
import { megabytes } from '../cartas'
import { mensagem, plural } from '../formato'

const emit = defineEmits<{ fechar: []; mudou: [] }>()
const estado = ref<EstadoConexao | null>(null)
const clientId = ref('')
const segredo = ref('')
const ocupado = ref(false)
const erro = ref('')
const ok = ref('')

async function carregar() { try { estado.value = await Cartas.Estado() } catch (e) { erro.value = mensagem(e) } }
async function agir(f: () => Promise<unknown>, sucesso: string) {
  erro.value = ''; ok.value = ''; ocupado.value = true
  try { await f(); ok.value = sucesso; await carregar(); emit('mudou') } catch (e) { erro.value = mensagem(e) } finally { ocupado.value = false }
}
const salvar = () => agir(async () => { await Cartas.SalvarCredenciais(clientId.value, segredo.value); segredo.value = '' }, 'Conectado. As credenciais foram guardadas no Gerenciador de Credenciais do Windows.')
const testar = () => agir(() => Cartas.TestarConexao(), 'Conexão com o Copernicus funcionando.')
const remover = () => agir(() => Cartas.RemoverCredenciais(), 'Credenciais removidas deste computador.')
async function limpar() {
  let n = 0
  await agir(async () => { n = await Cartas.LimparImagens() }, '')
  if (!erro.value) ok.value = `${plural(n, 'imagem apagada', 'imagens apagadas')}. As cartas geradas continuam guardadas.`
}

const tecla = (e: KeyboardEvent) => { if (e.key === 'Escape' && !ocupado.value) emit('fechar') }
onMounted(() => { carregar(); window.addEventListener('keydown', tecla) })
onUnmounted(() => window.removeEventListener('keydown', tecla))
</script>

<template>
  <div class="modal-fundo" @click.self="!ocupado && emit('fechar')">
    <div class="modal largo pilha" role="dialog" aria-modal="true" aria-labelledby="cfg-titulo">
      <div class="linha" style="justify-content: space-between">
        <h2 id="cfg-titulo">Fonte de reserva: Copernicus (opcional)</h2>
        <button class="discreto" :disabled="ocupado" @click="emit('fechar')">Fechar</button>
      </div>
      <p class="suave">
        As imagens vêm do INPE, sem cadastro. O Copernicus Data Space Ecosystem (gratuito) é usado só como reserva, quando o
        INPE não responde. A Central envia só a área do mapa e o período; nenhum dado dos cadastros sai deste computador.
      </p>

      <section v-if="estado" class="estado" :class="estado.configurado ? 'ok' : ''">
        <strong>{{ estado.configurado ? 'Configurado' : 'Ainda não configurado' }}</strong>
        <span v-if="estado.configurado" class="suave"> · cliente {{ estado.client_id }}</span>
      </section>

      <details class="passos" :open="!estado?.configurado">
        <summary>Como obter o acesso gratuito (uma vez só)</summary>
        <ol>
          <li>Num navegador, crie uma conta em <span class="mono">dataspace.copernicus.eu</span> (de preferência com um e-mail institucional).</li>
          <li>Entre no painel do Sentinel Hub: <span class="mono">shapps.dataspace.copernicus.eu/dashboard</span>.</li>
          <li>Em <strong>User Settings → OAuth clients</strong>, clique em <strong>Create</strong>, dê um nome (ex.: “SITPOSSE-7 Central”) e escolha a validade.</li>
          <li>Copie o <strong>Client ID</strong> e o <strong>Client secret</strong>. O segredo aparece uma única vez.</li>
          <li>Cole os dois abaixo e clique em <strong>Salvar e testar</strong>. Não contrate planos pagos: a cota gratuita (10.000 unidades por mês) é suficiente para cerca de 900 cartas.</li>
        </ol>
      </details>

      <form class="pilha campos" @submit.prevent="salvar">
        <label>Client ID <input v-model="clientId" autocomplete="off" spellcheck="false" :placeholder="estado?.configurado ? 'Para trocar, informe um novo' : 'sh-…'" /></label>
        <label>Client secret <input v-model="segredo" type="password" autocomplete="off" placeholder="Não é exibido depois de salvo" /></label>
        <div class="linha" style="justify-content: flex-end">
          <button v-if="estado?.configurado" type="button" class="perigo" :disabled="ocupado" @click="remover">Remover credenciais</button>
          <button v-if="estado?.configurado" type="button" :disabled="ocupado" @click="testar">Testar conexão</button>
          <button class="primario" :disabled="ocupado || !clientId.trim() || !segredo.trim()">{{ ocupado ? 'Testando…' : 'Salvar e testar' }}</button>
        </div>
      </form>

      <p v-if="ok" class="msg-ok">{{ ok }}</p>
      <p v-if="erro" class="aviso-erro">{{ erro }}</p>

      <section v-if="estado" class="uso">
        <h3>Guardado neste computador</h3>
        <p class="suave">
          {{ plural(estado.uso.imagens, 'imagem de satélite', 'imagens de satélite') }} ({{ megabytes(estado.uso.bytes_imagens) }}) e
          {{ plural(estado.uso.cartas, 'carta gerada', 'cartas geradas') }} ({{ megabytes(estado.uso.bytes_cartas) }}). Nada vence:
          as imagens já consultadas reabrem sem internet e sem gastar cota.
        </p>
        <button :disabled="ocupado || !estado.uso.imagens" @click="limpar">Limpar imagens guardadas</button>
        <p class="fraco dica">Apaga só as imagens brutas. As cartas geradas (PDF e PNG) não são apagadas.</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.largo { max-width: 640px; }
.estado { padding: .6rem .85rem; border-radius: var(--raio); background: var(--alerta-fundo); color: var(--alerta); }
.estado.ok { background: var(--ok-fundo); color: var(--ok); }
.passos summary { cursor: pointer; font-weight: 650; }
.passos ol { margin: .6rem 0 0; padding-left: 1.2rem; display: flex; flex-direction: column; gap: .4rem; }
.campos label { display: flex; flex-direction: column; gap: .25rem; color: var(--tinta-2); font-size: .92rem; }
.msg-ok { background: var(--ok-fundo); color: var(--ok); border-radius: var(--raio); padding: .6rem .9rem; }
.uso { border-top: 1px solid var(--regua); padding-top: .9rem; display: flex; flex-direction: column; gap: .5rem; align-items: flex-start; }
.uso h3 { font-size: 1rem; }
.dica { font-size: .85rem; }
</style>
