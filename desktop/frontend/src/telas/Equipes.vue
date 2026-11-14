<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { EquipeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import { mensagem, num, quando } from '../formato'

const equipes = ref<EquipeLinha[]>([])
const erro = ref('')
onMounted(async () => { try { equipes.value = (await Consulta.Equipes()) ?? [] } catch (e) { erro.value = mensagem(e) } })
</script>

<template>
  <div class="pilha">
    <h1>Equipes</h1>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>
    <div class="cartao rolavel" style="padding: 0">
      <table class="tabela">
        <thead><tr><th>Equipe</th><th>Responsável</th><th>Área</th><th class="num">Aparelhos</th><th class="num">Propriedades</th><th>Última importação</th></tr></thead>
        <tbody>
          <tr v-for="e in equipes" :key="e.uuid">
            <td><strong>{{ e.nome }}</strong></td><td>{{ e.responsavel }}</td><td>{{ e.identificacao_area || '—' }}</td>
            <td class="num">{{ e.aparelhos }}</td><td class="num">{{ num(e.propriedades) }}</td><td>{{ quando(e.ultima_importacao) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="!equipes.length" class="vazio">Nenhuma equipe ainda. As equipes entram na operação quando o celular é pareado.</div>
    </div>
  </div>
</template>
