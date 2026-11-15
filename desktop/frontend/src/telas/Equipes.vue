<script setup lang="ts">
import { onActivated, onMounted, ref } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { EquipeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import TabelaDados from '../componentes/TabelaDados.vue'
import { mensagem, num, plural, quando } from '../formato'
import type { Coluna } from '../tabela'

const equipes = ref<EquipeLinha[]>([])
const erro = ref('')
const carregando = ref(true)
async function carregar() {
  try { equipes.value = (await Consulta.Equipes()) ?? []; erro.value = '' } catch (e) { erro.value = mensagem(e) } finally { carregando.value = false }
}
onMounted(carregar)
onActivated(carregar)

const colunas: Coluna<EquipeLinha>[] = [
  { chave: 'nome', rotulo: 'Equipe' },
  { chave: 'responsavel', rotulo: 'Responsável' },
  { chave: 'identificacao_area', rotulo: 'Área de atuação' },
  { chave: 'aparelhos', rotulo: 'Aparelhos', tipo: 'numero' },
  { chave: 'propriedades', rotulo: 'Famílias', tipo: 'numero', formatar: (l) => num(l.propriedades) },
  { chave: 'ultima_importacao', rotulo: 'Última importação', formatar: (l) => quando(l.ultima_importacao), secundaria: true },
]
</script>

<template>
  <div class="pilha">
    <header class="cabeca">
      <h1>Equipes</h1>
      <p class="suave">{{ plural(equipes.length, 'equipe', 'equipes') }} na operação. Uma equipe entra na operação quando o primeiro celular dela é pareado.</p>
    </header>
    <TabelaDados :colunas="colunas" :linhas="equipes" :chave="(l) => l.uuid" rotulo="Equipes" :carregando="carregando" :erro="erro"
                 :ordem="{ chave: 'nome', dir: 'asc' }" texto-vazio="Nenhuma equipe ainda. Pareie o primeiro celular em Sincronização.">
      <template #nome="{ linha }"><strong>{{ linha.nome }}</strong></template>
    </TabelaDados>
  </div>
</template>

<style scoped>
.cabeca { display: flex; flex-direction: column; gap: .35rem; }
</style>
