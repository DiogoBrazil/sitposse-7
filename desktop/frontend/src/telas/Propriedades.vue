<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { EquipeLinha, PropriedadeDetalhe, PropriedadeLinha } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import { mensagem, num, quando } from '../formato'

const lista = ref<PropriedadeLinha[]>([])
const equipes = ref<EquipeLinha[]>([])
const busca = ref('')
const equipe = ref('')
const erro = ref('')
const detalhe = ref<PropriedadeDetalhe | null>(null)
const fotoAberta = ref<string | null>(null)

async function carregar() {
  try { lista.value = (await Consulta.Propriedades(busca.value, equipe.value)) ?? []; erro.value = '' } catch (e) { erro.value = mensagem(e) }
}
let atraso: number | undefined
watch([busca, equipe], () => { window.clearTimeout(atraso); atraso = window.setTimeout(carregar, 250) })
onMounted(async () => { carregar(); equipes.value = (await Consulta.Equipes()) ?? [] })

async function abrir(uuid: string) {
  try { detalhe.value = await Consulta.Propriedade(uuid) } catch (e) { erro.value = mensagem(e) }
}
const categorias: Record<string, string> = { eletrodomestico: 'Eletrodomésticos', movel: 'Móveis', ave: 'Aves de criação' }
function itensDe(cat: string) { return (detalhe.value?.itens ?? []).filter((i) => i.categoria === cat) }
function somaItens(cat: string) { return itensDe(cat).reduce((s, i) => s + i.quantidade, 0) }
</script>

<template>
  <div class="pilha">
    <h1>Propriedades</h1>
    <div class="linha filtros">
      <input v-model="busca" placeholder="Buscar por propriedade ou responsável (sem diferenciar acentos)" style="flex: 1 1 320px" />
      <select v-model="equipe" style="flex: 0 1 260px">
        <option value="">Todas as equipes</option>
        <option v-for="e in equipes" :key="e.uuid" :value="e.uuid">{{ e.nome }}</option>
      </select>
    </div>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>
    <div class="layout" :class="{ comDetalhe: detalhe }">
      <div class="cartao rolavel" style="padding: 0">
        <table class="tabela">
          <thead><tr><th>Propriedade</th><th>Equipe</th><th class="num">Pessoas</th><th class="num">Bovinos</th><th class="num">Área (ha)</th><th>GPS</th><th class="num">Fotos</th><th>Recebido</th></tr></thead>
          <tbody>
            <tr v-for="p in lista" :key="p.uuid" :class="{ sel: detalhe?.uuid === p.uuid }" @click="abrir(p.uuid)">
              <td><strong>{{ p.nome }}</strong><br /><span class="suave">{{ p.responsavel || '—' }}</span></td>
              <td>{{ p.equipe }}</td>
              <td class="num">{{ num(p.pessoas) }}</td>
              <td class="num">{{ num(p.bovinos) }}</td>
              <td class="num">{{ p.area_ha ?? '—' }}</td>
              <td><span class="selo" :class="p.tem_gps ? 'ok' : ''">{{ p.tem_gps ? 'sim' : 'não' }}</span></td>
              <td class="num">{{ p.fotos }}</td>
              <td>{{ quando(p.recebido_em) }}</td>
            </tr>
          </tbody>
        </table>
        <div v-if="!lista.length" class="vazio">{{ busca || equipe ? 'Nenhuma propriedade com esse filtro.' : 'Nenhuma propriedade recebida ainda. Sincronize os celulares em “Sincronização”.' }}</div>
      </div>

      <aside v-if="detalhe" class="cartao pilha detalhe">
        <div class="linha" style="justify-content: space-between">
          <h2>{{ detalhe.nome }}</h2>
          <button @click="detalhe = null">Fechar</button>
        </div>
        <dl class="dados">
          <dt>Responsável</dt><dd>{{ detalhe.responsavel || '—' }}</dd>
          <dt>Equipe</dt><dd>{{ detalhe.equipe }}{{ detalhe.equipe_area ? ` · ${detalhe.equipe_area}` : '' }}</dd>
          <dt>Atualizado no celular</dt><dd>{{ quando(detalhe.atualizado_em) }}</dd>
          <dt>Recebido</dt><dd>{{ quando(detalhe.recebido_em) }} (versão {{ detalhe.rev }})</dd>
        </dl>
        <h3>Pessoas</h3>
        <dl class="dados"><dt>Adultos (18+)</dt><dd>{{ detalhe.pessoas_adultos }}</dd><dt>Jovens (12–17)</dt><dd>{{ detalhe.pessoas_jovens }}</dd><dt>Crianças (0–11)</dt><dd>{{ detalhe.pessoas_criancas }}</dd></dl>
        <h3>Veículos</h3>
        <dl class="dados"><dt>Caminhões</dt><dd>{{ detalhe.veic_caminhoes }}</dd><dt>Automóveis, caminhonetes e utilitários</dt><dd>{{ detalhe.veic_automoveis_utilitarios }}</dd><dt>Motocicletas e ciclomotores</dt><dd>{{ detalhe.veic_motos_ciclomotores }}</dd></dl>
        <h3>Animais</h3>
        <dl class="dados"><dt>Bovinos adultos</dt><dd>{{ detalhe.bov_adultos }}</dd><dt>Bezerros e bezerras</dt><dd>{{ detalhe.bov_bezerros }}</dd><dt>Equinos adultos</dt><dd>{{ detalhe.equ_adultos }}</dd><dt>Potros</dt><dd>{{ detalhe.equ_potros }}</dd></dl>
        <template v-for="(titulo, cat) in categorias" :key="cat">
          <h3>{{ titulo }} <span class="suave">· {{ somaItens(cat) }}</span></h3>
          <p v-if="!itensDe(cat).length" class="suave" style="margin: 0">Nenhum item</p>
          <dl v-else class="dados"><template v-for="i in itensDe(cat)" :key="i.uuid"><dt>{{ i.nome }}<span v-if="i.outros" class="suave"> (outros)</span></dt><dd>{{ i.quantidade }}</dd></template></dl>
        </template>
        <h3>Área</h3>
        <p style="margin: 0">{{ detalhe.area_ha ? `${detalhe.area_ha} ha` : 'Não informada' }}<span v-if="detalhe.area_unidade === 'alq_paulista'" class="suave"> (informada em alqueire paulista)</span></p>
        <h3>Localização</h3>
        <p v-if="detalhe.lat == null" style="margin: 0">Não informada</p>
        <p v-else style="margin: 0" class="mono">{{ detalhe.lat?.toFixed(6) }}, {{ detalhe.lon?.toFixed(6) }}
          <span class="suave">· {{ detalhe.gps_origem === 'manual' ? 'digitada' : `GPS, ${detalhe.gps_precisao_m?.toFixed(0)} m` }}</span></p>
        <h3>Fotos <span class="suave">· {{ detalhe.fotos?.length ?? 0 }}</span></h3>
        <div class="fotos">
          <img v-for="f in detalhe.fotos ?? []" :key="f.uuid" :src="f.url" loading="lazy" alt="Foto da propriedade" @click="fotoAberta = f.url" />
        </div>
      </aside>
    </div>
    <div v-if="fotoAberta" class="modal-fundo" @click="fotoAberta = null"><img :src="fotoAberta" class="foto-grande" alt="Foto ampliada" /></div>
  </div>
</template>

<style scoped>
.layout { display: grid; gap: 1rem; grid-template-columns: 1fr; align-items: start; }
.layout.comDetalhe { grid-template-columns: minmax(0, 1fr) minmax(340px, 440px); }
tr.sel { background: var(--primaria-clara) !important; }
.detalhe { position: sticky; top: 1rem; max-height: calc(100vh - 2rem); overflow-y: auto; }
.detalhe h3 { font-size: 1rem; margin-top: .25rem; }
.fotos { display: grid; grid-template-columns: repeat(auto-fill, minmax(110px, 1fr)); gap: .5rem; }
.fotos img { width: 100%; aspect-ratio: 1; object-fit: cover; border-radius: 8px; cursor: zoom-in; background: var(--superficie-2); }
.foto-grande { max-width: 95vw; max-height: 92vh; border-radius: 8px; }
@media (max-width: 1200px) { .layout.comDetalhe { grid-template-columns: 1fr; } .detalhe { position: static; max-height: none; } }
</style>
