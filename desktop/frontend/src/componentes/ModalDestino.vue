<script setup lang="ts">
/**
 * "Confirmar destino da sincronização" (RN-09, ADR-0024): aparece em qualquer tela quando um
 * lote chega. Mostra o destino resolvido pelo código (estudo, data da jornada, equipe, código)
 * ou "Código de equipe não encontrado", e deixa confirmar, alterar/corrigir ou cancelar.
 * Nada é importado sem a decisão do operador; coleta já importada só aceita a mesma equipe.
 */
import { computed, ref, watch } from 'vue'
import { EstudoService as Est, SincronizacaoService as Sinc } from '../../bindings/sitposse7/desktop/app'
import type { LoteAguardando } from '../../bindings/sitposse7/desktop/internal/importacao/models'
import type { EstudoDetalhe } from '../../bindings/sitposse7/desktop/internal/estudos/models'
import { dataBR, estudos } from '../estudo'
import { mensagem, num } from '../formato'

const props = defineProps<{ lotes: LoteAguardando[] }>()
const emit = defineEmits<{ decidido: [] }>()

const lote = computed(() => props.lotes[0])
const modo = ref<'resumo' | 'alterar'>('resumo')
const ocupado = ref(false)
const erro = ref('')
const confirmarCancelar = ref(false)
watch(() => lote.value?.lote_uuid, () => { modo.value = 'resumo'; erro.value = ''; confirmarCancelar.value = false; equipe.value = '' })

// Seleção em cascata Estudo → Jornada → Equipe (alterar destino / corrigir no Central).
const estudo = ref('')
const detalhe = ref<EstudoDetalhe | null>(null)
const jornada = ref('')
const equipe = ref('')
watch(estudo, async (u) => { jornada.value = ''; equipe.value = ''; detalhe.value = u ? await Est.Detalhe(u) : null })
watch(jornada, () => { equipe.value = '' })
const jornadas = computed(() => detalhe.value?.lista_jornadas ?? [])
const equipesDaJornada = computed(() => jornadas.value.find((j) => j.uuid === jornada.value)?.equipes ?? [])
const equipeEscolhida = computed(() => equipesDaJornada.value.find((q) => q.uuid === equipe.value) ?? null)

function alterar() {
  modo.value = 'alterar'
  estudo.value = lote.value?.destino?.estudo_uuid ?? estudos.value[0]?.uuid ?? ''
}

async function decidir(equipeUuid: string) {
  if (!lote.value) return
  ocupado.value = true; erro.value = ''
  try { await Sinc.DecidirDestino(lote.value.lote_uuid, equipeUuid); emit('decidido') } catch (e) { erro.value = mensagem(e) } finally { ocupado.value = false }
}

const destino = computed(() => (lote.value?.travado ? lote.value.destino_travado : lote.value?.destino) ?? null)
const exibirCodigo = (c: string) => (c.length === 10 ? `${c.slice(0, 2)}-${c.slice(2, 6)}-${c.slice(6)}` : c)
</script>

<template>
  <div v-if="lote" class="modal-fundo">
    <div class="modal pilha largo" role="alertdialog" aria-modal="true" aria-labelledby="dest-titulo">
      <div class="linha" style="justify-content: space-between">
        <h2 id="dest-titulo">{{ !destino && modo === 'resumo' ? 'Código de equipe não encontrado' : 'Confirmar destino da sincronização' }}</h2>
        <span v-if="lotes.length > 1" class="selo">1 de {{ lotes.length }}</span>
      </div>

      <dl class="dados">
        <dt>Aparelho</dt><dd>{{ lote.rotulo_aparelho || '—' }}</dd>
        <dt>Código recebido</dt><dd><span class="mono forte">{{ lote.codigo_exibicao }}</span></dd>
        <dt>Cadastros</dt><dd>{{ num(lote.novos) }} novo(s) · {{ num(lote.atualizados) }} atualizado(s) · {{ num(lote.excluidos) }} excluído(s)</dd>
        <dt>Fotografias</dt><dd>{{ num(lote.fotos) }}</dd>
      </dl>

      <template v-if="modo === 'resumo'">
        <div v-if="destino" class="destino" :class="{ travado: lote.travado }">
          <p v-if="lote.travado" class="fraco">Esta coleta já foi importada antes: o destino não pode mudar.</p>
          <dl class="dados">
            <dt>Estudo de situação</dt><dd><strong>{{ destino.estudo_nome }}</strong></dd>
            <dt>Jornada</dt><dd>Jornada {{ destino.jornada_numero }} · {{ dataBR(destino.jornada_data) }}</dd>
            <dt>Equipe</dt><dd><strong>{{ destino.equipe_nome }}</strong></dd>
            <dt>Código da equipe</dt><dd><span class="mono forte">{{ exibirCodigo(destino.equipe_codigo) }}</span></dd>
          </dl>
          <p v-if="destino.equipe_codigo !== lote.codigo_informado" class="aviso">O código recebido é diferente do código oficial desta equipe.</p>
        </div>
        <p v-else>
          O código informado pelo dispositivo não foi localizado entre as equipes cadastradas no SITPOSSE-7 Central.
          Isso pode ocorrer devido a um erro de digitação.
          Você pode corrigir a vinculação diretamente no Central ou cancelar esta sincronização para que o código seja corrigido no smartphone.
        </p>
      </template>

      <template v-else>
        <p class="suave">Escolha a equipe de destino. O código recebido fica registrado; os códigos oficiais das equipes não mudam.</p>
        <div class="cascata">
          <label>Estudo
            <select v-model="estudo"><option v-for="e in estudos" :key="e.uuid" :value="e.uuid">{{ e.nome }} · {{ e.codigo_exibicao }}</option></select>
          </label>
          <label>Jornada
            <select v-model="jornada" :disabled="!jornadas.length">
              <option value="" disabled>Escolha</option>
              <option v-for="j in jornadas" :key="j.uuid" :value="j.uuid">Jornada {{ j.numero }} · {{ dataBR(j.data) }}</option>
            </select>
          </label>
          <label>Equipe
            <select v-model="equipe" :disabled="!equipesDaJornada.length">
              <option value="" disabled>Escolha</option>
              <option v-for="q in equipesDaJornada" :key="q.uuid" :value="q.uuid">{{ q.nome }}</option>
            </select>
          </label>
        </div>
        <p v-if="equipeEscolhida" class="destino">
          Código oficial da equipe escolhida: <span class="mono forte">{{ equipeEscolhida.codigo_exibicao }}</span>
        </p>
      </template>

      <p v-if="erro" class="aviso-erro">{{ erro }}</p>

      <div v-if="confirmarCancelar" class="aviso pilha">
        <p>Cancelar esta sincronização? Nada será importado nem apagado; o celular mantém os cadastros e poderá enviar de novo depois de corrigir o código.</p>
        <div class="linha" style="justify-content: flex-end">
          <button @click="confirmarCancelar = false">Voltar</button>
          <button class="perigo" :disabled="ocupado" @click="decidir('')">Cancelar sincronização</button>
        </div>
      </div>
      <div v-else class="linha acoes">
        <button class="perigo" :disabled="ocupado" @click="confirmarCancelar = true">Cancelar sincronização</button>
        <span class="espaco"></span>
        <template v-if="modo === 'resumo'">
          <button v-if="!lote.travado" :disabled="ocupado" @click="alterar">{{ destino ? 'Alterar destino' : 'Corrigir no Central' }}</button>
          <button v-if="destino" class="primario" :disabled="ocupado" @click="decidir(destino.equipe_uuid)">Confirmar e importar</button>
        </template>
        <template v-else>
          <button :disabled="ocupado" @click="modo = 'resumo'">Voltar</button>
          <button class="primario" :disabled="ocupado || !equipeEscolhida" @click="decidir(equipe)">Confirmar destino e importar</button>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.largo { max-width: 640px; }
.forte { font-weight: 700; font-size: 1.05rem; letter-spacing: .04em; }
.destino { border: 1px solid var(--regua); border-radius: var(--raio); padding: .7rem .9rem; background: var(--folha-2); }
.destino.travado { border-style: dashed; }
.cascata { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: .75rem; }
.cascata label { display: flex; flex-direction: column; gap: .3rem; font-weight: 600; }
.acoes { flex-wrap: wrap; }
.espaco { flex: 1; }
</style>
