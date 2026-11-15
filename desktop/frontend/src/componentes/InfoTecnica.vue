<script setup lang="ts">
/** Informações técnicas da Central (servidor, endereços, identidade, pasta). Aberta pelo rodapé da barra lateral. */
import { onMounted, onUnmounted, ref } from 'vue'
import { OperacaoService as Op, SincronizacaoService as Sinc } from '../../bindings/sitposse7/desktop/app'
import type { Info } from '../../bindings/sitposse7/desktop/app/models'
import type { Operacao } from '../../bindings/sitposse7/desktop/internal/pareamento/models'
import { mensagem } from '../formato'

const emit = defineEmits<{ fechar: [] }>()
const info = ref<Info | null>(null)
const op = ref<Operacao | null>(null)
const erro = ref('')
const fechar = () => emit('fechar')
const tecla = (e: KeyboardEvent) => { if (e.key === 'Escape') fechar() }
onMounted(async () => {
  window.addEventListener('keydown', tecla)
  try { [info.value, op.value] = await Promise.all([Sinc.Info(), Op.Ativa()]) } catch (e) { erro.value = mensagem(e) }
})
onUnmounted(() => window.removeEventListener('keydown', tecla))
</script>

<template>
  <div class="modal-fundo" @click.self="fechar">
    <div class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="info-titulo">
      <div class="linha" style="justify-content: space-between">
        <h2 id="info-titulo">Informações técnicas</h2>
        <button class="discreto" autofocus @click="fechar">Fechar</button>
      </div>
      <p v-if="erro" class="aviso-erro">{{ erro }}</p>
      <dl v-if="info" class="dados">
        <dt>Servidor</dt>
        <dd><span class="selo" :class="info.servidor_ativo ? 'ok' : 'erro'">{{ info.servidor_ativo ? `Ativo na porta ${info.porta}` : info.erro_servidor || 'Inativo' }}</span></dd>
        <dt>Endereços</dt><dd>{{ info.ips?.length ? info.ips.join(', ') : 'Nenhuma rede ativa' }}</dd>
        <dt>Operação ativa</dt><dd>{{ op?.nome ?? 'Nenhuma' }}</dd>
        <dt>Impressão digital</dt><dd class="mono">{{ info.pin }}</dd>
        <dt>ID da Central</dt><dd class="mono">{{ info.central_id }}</dd>
        <dt>Pasta de dados</dt><dd class="mono">{{ info.pasta_dados }}</dd>
        <dt>Esquema do banco</dt><dd>{{ info.versao_esquema }}</dd>
        <dt>Versão</dt><dd>{{ info.versao }}</dd>
      </dl>
      <p v-if="info?.fotos_sem_arquivo?.length" class="aviso-erro">{{ info.fotos_sem_arquivo.length }} foto(s) registrada(s) sem arquivo na pasta de dados. Não apague nada e avise o suporte.</p>
      <p class="fraco">A impressão digital confirma a identidade desta Central para os celulares. Ela vai no QR de pareamento.</p>
    </div>
  </div>
</template>
