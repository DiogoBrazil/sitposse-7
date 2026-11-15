<script setup lang="ts">
/** Página de uma propriedade (família): totais, gráficos com tabela, itens, área, localização e fotos. */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ConsultaService as Consulta } from '../../bindings/sitposse7/desktop/app'
import type { ItemLinha, PropriedadeDetalhe } from '../../bindings/sitposse7/desktop/internal/consultas/models'
import FaixaTotais from '../componentes/FaixaTotais.vue'
import Grafico from '../componentes/Grafico.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import ExportarRelatorio from '../componentes/ExportarRelatorio.vue'
import { alturaBarras, barrasHorizontais, type Dominio, type GrupoTotal, type Ponto } from '../graficos'
import { mensagem, num, quando } from '../formato'
import type { Tela } from '../navegacao'
import type { Coluna } from '../tabela'

const props = defineProps<{ uuid: string; de: Tela }>()
const emit = defineEmits<{ voltar: [] }>()
const p = ref<PropriedadeDetalhe | null>(null)
const erro = ref('')
const fotoAberta = ref<number | null>(null)
const exportando = ref(false)

onMounted(async () => {
  try { p.value = await Consulta.Propriedade(props.uuid); if (!p.value) erro.value = 'Propriedade não encontrada (pode ter sido excluída no celular).' } catch (e) { erro.value = mensagem(e) }
  window.addEventListener('keydown', tecla)
})
onUnmounted(() => window.removeEventListener('keydown', tecla))

const voltarPara: Record<Tela, string> = { painel: 'Painel', propriedades: 'Propriedades', equipes: 'Equipes', sincronizacao: 'Sincronização' }
const soma = (ps: Ponto[]) => ps.reduce((s, x) => s + x.valor, 0)

const pessoas = computed<Ponto[]>(() => p.value ? [
  { nome: 'Adultos (18+)', valor: p.value.pessoas_adultos },
  { nome: 'Jovens (12–17)', valor: p.value.pessoas_jovens },
  { nome: 'Crianças (0–11)', valor: p.value.pessoas_criancas },
] : [])
const itens = computed(() => p.value?.itens ?? [])
const daCategoria = (c: string) => itens.value.filter((i) => i.categoria === c)
const aves = computed(() => soma(daCategoria('ave').map((i) => ({ nome: i.nome, valor: i.quantidade }))))
const rebanho = computed<Ponto[]>(() => p.value ? [
  { nome: 'Bovinos adultos', valor: p.value.bov_adultos },
  { nome: 'Bezerros e bezerras', valor: p.value.bov_bezerros },
  { nome: 'Equinos adultos', valor: p.value.equ_adultos },
  { nome: 'Potros', valor: p.value.equ_potros },
] : [])
const veiculos = computed<Ponto[]>(() => p.value ? [
  { nome: 'Caminhões', valor: p.value.veic_caminhoes },
  { nome: 'Automóveis e utilitários', valor: p.value.veic_automoveis_utilitarios },
  { nome: 'Motos e ciclomotores', valor: p.value.veic_motos_ciclomotores },
] : [])

interface Categoria { id: string; titulo: string; dominio: Dominio }
const categorias: Categoria[] = [
  { id: 'eletrodomestico', titulo: 'Eletrodomésticos', dominio: 'eletro' },
  { id: 'movel', titulo: 'Móveis', dominio: 'moveis' },
  { id: 'ave', titulo: 'Aves de criação', dominio: 'aves' },
]
const aba = ref('eletrodomestico')
const catAtual = computed(() => categorias.find((c) => c.id === aba.value)!)
const itensAba = computed(() => [...daCategoria(aba.value)].sort((a, b) => b.quantidade - a.quantidade))
const totalCat = (c: string) => daCategoria(c).reduce((s, i) => s + i.quantidade, 0)

const totais = computed<GrupoTotal[]>(() => p.value ? [
  { titulo: 'Pessoas', cor: '--s2', itens: [{ valor: num(soma(pessoas.value)), rotulo: 'na família' }] },
  { titulo: 'Rebanho', cor: '--s3', itens: [
    { valor: num(p.value.bov_adultos + p.value.bov_bezerros), rotulo: 'bovinos' },
    { valor: num(p.value.equ_adultos + p.value.equ_potros), rotulo: 'equinos' },
  ] },
  { titulo: 'Aves', cor: '--s6', itens: [{ valor: num(aves.value), rotulo: 'de criação' }] },
  { titulo: 'Veículos', cor: '--s4', itens: [{ valor: num(soma(veiculos.value)), rotulo: 'no total' }] },
  { titulo: 'Bens', cor: '--s1', itens: [
    { valor: num(totalCat('eletrodomestico')), rotulo: 'eletrodomésticos' },
    { valor: num(totalCat('movel')), rotulo: 'móveis' },
  ] },
  { titulo: 'Área', itens: [{ valor: p.value.area_ha ?? '—', rotulo: p.value.area_ha ? 'hectares' : 'não informada' }] },
] : [])

const colPonto: Coluna<Ponto>[] = [
  { chave: 'nome', rotulo: 'Categoria', ordenavel: false },
  { chave: 'valor', rotulo: 'Quantidade', tipo: 'numero', ordenavel: false, formatar: (l) => num(l.valor) },
]
const colItens: Coluna<ItemLinha>[] = [
  { chave: 'nome', rotulo: 'Item' },
  { chave: 'outros', rotulo: 'Origem', valor: (l) => (l.outros ? 'Outros (digitado)' : 'Catálogo') },
  { chave: 'quantidade', rotulo: 'Quantidade', tipo: 'numero', formatar: (l) => num(l.quantidade) },
]
const totalTabela = (ps: Ponto[]) => ({ nome: 'Total', valor: num(soma(ps)) })

const fotos = computed(() => p.value?.fotos ?? [])
function tecla(e: KeyboardEvent) {
  const i = fotoAberta.value
  if (i === null) return
  const n = fotos.value.length
  if (e.key === 'Escape') fotoAberta.value = null
  if (e.key === 'ArrowRight') fotoAberta.value = (i + 1) % n
  if (e.key === 'ArrowLeft') fotoAberta.value = (i - 1 + n) % n
}
const origemGps = computed(() => {
  const x = p.value
  if (!x || x.lat == null) return ''
  if (x.gps_origem === 'manual') return 'Digitada no celular (sem precisão medida)'
  const partes = [`GPS do celular`]
  if (x.gps_precisao_m != null) partes.push(`precisão de ${Math.round(x.gps_precisao_m)} m`)
  if (x.gps_satelites != null) partes.push(`${x.gps_satelites} satélites`)
  return partes.join(', ')
})
const unidadeOriginal = computed(() => {
  const u = p.value?.area_unidade
  return u === 'alq_paulista' ? 'alqueire paulista (1 alq. = 2,42 ha)' : u === 'ha' ? 'hectares' : u ?? ''
})
</script>

<template>
  <div class="pilha">
    <button class="voltar discreto" @click="emit('voltar')">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 5l-7 7 7 7" /></svg>
      {{ voltarPara[de] }}
    </button>
    <p v-if="erro" class="aviso-erro">{{ erro }}</p>

    <template v-if="p">
      <header class="cabeca">
        <div class="titulo">
          <h1>{{ p.nome }}</h1>
          <p class="suave">
            {{ p.responsavel ? `Responsável: ${p.responsavel}` : 'Responsável não informado' }}
            <span class="sep" aria-hidden="true"></span>
            Equipe {{ p.equipe }}<template v-if="p.equipe_area"> ({{ p.equipe_area }})</template>
          </p>
        </div>
        <div class="lado">
          <button @click="exportando = true"><svg class="ic" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 4v11m0 0-4.5-4.5M12 15l4.5-4.5M5 19h14" /></svg>Exportar ficha</button>
        <div class="selos">
          <span class="selo" :class="p.lat != null ? 'ok' : ''">{{ p.lat != null ? 'Com localização' : 'Sem localização' }}</span>
          <span class="selo" :class="fotos.length ? 'ok' : ''">{{ fotos.length === 1 ? '1 foto' : `${fotos.length} fotos` }}</span>
          <span class="selo" :class="p.area_ha ? 'ok' : ''">{{ p.area_ha ? `${p.area_ha} ha` : 'Área não informada' }}</span>
        </div>
        </div>
      </header>

      <FaixaTotais :grupos="totais" />

      <div class="grade">
        <section class="bloco">
          <div class="bloco-cabeca"><h2>Pessoas por faixa etária</h2><span class="total">{{ num(soma(pessoas)) }}</span></div>
          <div class="bloco-corpo par">
            <Grafico :montar="(t) => barrasHorizontais(t, 'pessoas', pessoas, 'pessoa(s)')" :altura="alturaBarras(3)" rotulo="Gráfico de pessoas por faixa etária; os valores estão na tabela ao lado" />
            <TabelaDados :colunas="colPonto" :linhas="pessoas" :chave="(l) => l.nome" rotulo="Pessoas por faixa etária" :por-pagina="0" compacta :totais="totalTabela(pessoas)" />
          </div>
        </section>

        <section class="bloco">
          <div class="bloco-cabeca"><h2>Rebanho</h2><span class="total">{{ num(soma(rebanho)) }}</span></div>
          <div class="bloco-corpo par">
            <Grafico :montar="(t) => barrasHorizontais(t, 'rebanho', rebanho, 'cabeça(s)')" :altura="alturaBarras(4)" rotulo="Gráfico do rebanho; os valores estão na tabela ao lado" />
            <TabelaDados :colunas="colPonto" :linhas="rebanho" :chave="(l) => l.nome" rotulo="Rebanho" :por-pagina="0" compacta :totais="totalTabela(rebanho)" />
          </div>
        </section>

        <section class="bloco">
          <div class="bloco-cabeca"><h2>Veículos</h2><span class="total">{{ num(soma(veiculos)) }}</span></div>
          <div class="bloco-corpo par">
            <Grafico :montar="(t) => barrasHorizontais(t, 'veiculos', veiculos, 'veículo(s)')" :altura="alturaBarras(3)" rotulo="Gráfico de veículos por tipo; os valores estão na tabela ao lado" />
            <TabelaDados :colunas="colPonto" :linhas="veiculos" :chave="(l) => l.nome" rotulo="Veículos" :por-pagina="0" compacta :totais="totalTabela(veiculos)" />
          </div>
        </section>

        <section class="bloco">
          <div class="bloco-cabeca"><h2>Área e localização</h2></div>
          <div class="bloco-corpo">
            <dl class="dados">
              <dt>Área</dt>
              <dd>
                <template v-if="p.area_ha"><strong>{{ p.area_ha }} ha</strong><span class="suave"> · informada em {{ unidadeOriginal }}</span></template>
                <span v-else>Não informada</span>
              </dd>
              <dt>Coordenadas</dt>
              <dd v-if="p.lat != null"><span class="mono">{{ p.lat.toFixed(6) }}, {{ p.lon?.toFixed(6) }}</span></dd>
              <dd v-else>Não informadas</dd>
              <template v-if="p.lat != null">
                <dt>Origem</dt><dd>{{ origemGps }}</dd>
                <dt v-if="p.gps_capturado_em">Capturada em</dt><dd v-if="p.gps_capturado_em">{{ quando(p.gps_capturado_em) }}</dd>
              </template>
            </dl>
          </div>
        </section>
      </div>

      <section class="bloco">
        <div class="bloco-cabeca">
          <h2>Bens e criações</h2>
          <div class="abas" role="tablist" aria-label="Categoria de itens">
            <button v-for="c in categorias" :key="c.id" role="tab" :aria-selected="aba === c.id" :class="{ ativa: aba === c.id }" :style="{ '--marca': `var(--s${c.dominio === 'eletro' ? 1 : c.dominio === 'moveis' ? 5 : 6})` }" @click="aba = c.id">
              {{ c.titulo }} <span class="cont">{{ num(totalCat(c.id)) }}</span>
            </button>
          </div>
        </div>
        <div class="bloco-corpo">
          <p v-if="!itensAba.length" class="suave vazio">Nenhum item de {{ catAtual.titulo.toLowerCase() }} registrado nesta propriedade.</p>
          <div v-else class="par largo">
            <Grafico :key="aba" :montar="(t) => barrasHorizontais(t, catAtual.dominio, itensAba.map((i) => ({ nome: i.nome, valor: i.quantidade })))" :altura="alturaBarras(itensAba.length)" :rotulo="`Gráfico de ${catAtual.titulo}; os valores estão na tabela ao lado`" />
            <TabelaDados :colunas="colItens" :linhas="itensAba" :chave="(l) => l.uuid" :rotulo="catAtual.titulo" :por-pagina="0" compacta :ordem="{ chave: 'quantidade', dir: 'desc' }" :totais="{ nome: 'Total', quantidade: num(totalCat(aba)) }" />
          </div>
        </div>
      </section>

      <section class="bloco">
        <div class="bloco-cabeca"><h2>Fotos</h2><span class="total">{{ num(fotos.length) }}</span></div>
        <div class="bloco-corpo">
          <p v-if="!fotos.length" class="suave">Nenhuma foto nesta propriedade.</p>
          <div v-else class="fotos">
            <button v-for="(f, i) in fotos" :key="f.uuid" class="foto" :aria-label="`Ampliar foto ${i + 1} de ${fotos.length}`" @click="fotoAberta = i">
              <img :src="f.url" loading="lazy" alt="" />
              <span>{{ quando(f.capturada_em) }}</span>
            </button>
          </div>
        </div>
      </section>

      <details class="bloco registro">
        <summary>Registro e sincronização</summary>
        <dl class="dados">
          <dt>Cadastrado no celular</dt><dd>{{ quando(p.criado_em) }}</dd>
          <dt>Última alteração no celular</dt><dd>{{ quando(p.atualizado_em) }}</dd>
          <dt>Recebido pela Central</dt><dd>{{ quando(p.recebido_em) }}</dd>
          <dt>Versão do cadastro</dt><dd>{{ p.rev }}</dd>
          <dt>Identificador</dt><dd class="mono">{{ p.uuid }}</dd>
        </dl>
      </details>
    </template>

    <ExportarRelatorio v-if="exportando && p" modo="ficha" :uuid="p.uuid" :nome="p.nome" @fechar="exportando = false" />
    <div v-if="fotoAberta !== null && fotos[fotoAberta]" class="modal-fundo" role="dialog" aria-modal="true" aria-label="Foto ampliada" @click.self="fotoAberta = null">
      <figure class="foto-grande">
        <img :src="fotos[fotoAberta].url" alt="Foto da propriedade" />
        <figcaption>
          <span>Foto {{ fotoAberta + 1 }} de {{ fotos.length }} · {{ quando(fotos[fotoAberta].capturada_em) }}</span>
          <span class="linha">
            <button v-if="fotos.length > 1" @click="fotoAberta = (fotoAberta - 1 + fotos.length) % fotos.length">Anterior</button>
            <button v-if="fotos.length > 1" @click="fotoAberta = (fotoAberta + 1) % fotos.length">Próxima</button>
            <button autofocus @click="fotoAberta = null">Fechar</button>
          </span>
        </figcaption>
      </figure>
    </div>
  </div>
</template>

<style scoped>
.voltar { align-self: flex-start; display: inline-flex; align-items: center; gap: .35rem; padding-left: .4rem; margin-bottom: -.5rem; }
.voltar svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; }
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .4rem; min-width: 0; }
.titulo h1 { font-size: 2.3rem; overflow-wrap: anywhere; }
.sep { display: inline-block; width: 1px; height: .9em; background: var(--regua-forte); margin: 0 .6rem; vertical-align: -.1em; }
.selos { display: flex; gap: .4rem; flex-wrap: wrap; justify-content: flex-end; }
.lado { display: flex; flex-direction: column; align-items: flex-end; gap: .6rem; }
.ic { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; vertical-align: -3px; margin-right: .35rem; }

.grade { display: grid; gap: 1.25rem; grid-template-columns: repeat(auto-fit, minmax(min(100%, 520px), 1fr)); align-items: start; }
.par { display: grid; gap: 1rem; grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr); align-items: center; container-type: inline-size; }
.par.largo { grid-template-columns: minmax(0, 1.3fr) minmax(0, 1fr); align-items: start; }
@container (max-width: 560px) { .par { grid-template-columns: 1fr; } }
@media (max-width: 760px) { .par, .par.largo { grid-template-columns: 1fr; } }
.bloco-cabeca { flex-wrap: wrap; }
.abas { display: flex; gap: .25rem; flex-wrap: wrap; }
.abas button { min-height: 32px; padding: .3rem .75rem; border-color: transparent; background: transparent; color: var(--tinta-2); position: relative; }
.abas button.ativa { color: var(--tinta); background: var(--folha-2); border-color: var(--regua); }
.abas button.ativa::after { content: ""; position: absolute; left: .75rem; right: .75rem; bottom: -1px; height: 3px; border-radius: 2px 2px 0 0; background: var(--marca); }
.cont { font-stretch: 85%; font-weight: 700; margin-left: .25rem; color: var(--tinta); }
.vazio { padding: .5rem 0; }
.fotos { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: .75rem; }
.foto { all: unset; cursor: zoom-in; display: flex; flex-direction: column; gap: .3rem; font-size: .8rem; color: var(--tinta-3); }
.foto img { width: 100%; aspect-ratio: 4 / 3; object-fit: cover; border-radius: var(--raio); background: var(--folha-2); border: 1px solid var(--regua); }
.foto:focus-visible img { outline: 2px solid var(--foco); outline-offset: 2px; }
.registro { padding: .85rem 1.1rem; }
.registro summary { cursor: pointer; font-weight: 650; color: var(--tinta-2); }
.registro[open] summary { margin-bottom: .85rem; }
.foto-grande { margin: 0; display: flex; flex-direction: column; gap: .6rem; max-width: 95vw; }
.foto-grande img { max-width: 95vw; max-height: 82vh; border-radius: var(--raio); object-fit: contain; }
.foto-grande figcaption { display: flex; justify-content: space-between; align-items: center; gap: 1rem; color: #e6ece6; flex-wrap: wrap; }
</style>
