<script setup lang="ts">
/**
 * Estudos de situação (RF-C01, ADR-0023): lista, página do estudo com as jornadas e as equipes
 * de cada jornada (códigos para entregar às equipes), criação de estudo e de jornada (com cópia
 * dos nomes das equipes de uma jornada anterior: cada equipe recebe código novo).
 */
import { computed, onActivated, onMounted, ref } from 'vue'
import { EstudoService as Est } from '../../bindings/sitposse7/desktop/app'
import type { EstudoDetalhe, Jornada } from '../../bindings/sitposse7/desktop/internal/estudos/models'
import AcaoIcone from '../componentes/AcaoIcone.vue'
import ModalFormulario from '../componentes/ModalFormulario.vue'
import ConfirmarExclusao from '../componentes/ConfirmarExclusao.vue'
import TabelaDados from '../componentes/TabelaDados.vue'
import { carregarEstudos, dataBR, definirFoco, estudoFoco, estudos } from '../estudo'
import { mensagem, num, quando } from '../formato'
import type { Coluna } from '../tabela'
import type { Estudo } from '../../bindings/sitposse7/desktop/internal/estudos/models'

const erro = ref('')
const busca = ref('')
const aberto = ref<EstudoDetalhe | null>(null)
const copiado = ref('')
const jornadaId=ref('')
const jornadaAtual=computed(()=>aberto.value?.lista_jornadas?.find(j=>j.uuid===jornadaId.value)??aberto.value?.lista_jornadas?.[0])
const colJornadas:Coluna<Jornada>[]=[{chave:'numero',rotulo:'Jornada',tipo:'numero'},{chave:'data',rotulo:'Data',formatar:j=>dataBR(j.data)},{chave:'equipes',rotulo:'Equipes',valor:j=>j.equipes?.length??0},{chave:'familias',rotulo:'Famílias',tipo:'numero'},{chave:'acao',rotulo:'Ação',ordenavel:false}]
const colEquipes=[{chave:'nome',rotulo:'Equipe'},{chave:'codigo_exibicao',rotulo:'Código'},{chave:'familias',rotulo:'Famílias',tipo:'numero' as const},{chave:'acao',rotulo:'Ação',ordenavel:false}]
const edicao=ref<{id:string;nome:string;tipo:'estudo'|'equipe'}|null>(null)
const exclusao=ref<{id:string;nome:string;resumo:string;tipo:'estudo'|'equipe'}|null>(null)
async function salvarNome(){if(!edicao.value)return;salvando.value=true;try{const e=edicao.value;if(e.tipo==='estudo')await Est.Renomear(e.id,e.nome);else await Est.RenomearEquipe(e.id,e.nome);edicao.value=null;await recarregar()}catch(e){erro.value=mensagem(e)}finally{salvando.value=false}}
async function pedirExcluirEstudo(e:Estudo){try{const d=await Est.Detalhe(e.uuid);exclusao.value={id:e.uuid,nome:e.nome,tipo:'estudo',resumo:`Serão excluídos ${d?.jornadas??0} jornadas, ${d?.equipes??0} equipes e ${d?.familias??0} propriedades, incluindo seus itens e fotos. Envios pendentes vinculados serão cancelados.`}}catch(e){erro.value=mensagem(e)}}
async function confirmarExclusao(){if(!exclusao.value)return;const e=exclusao.value;if(e.tipo==='estudo')await Est.Excluir(e.id);else await Est.ExcluirEquipe(e.id)}
async function excluiu(){if(exclusao.value?.tipo==='estudo'&&aberto.value?.uuid===exclusao.value.id)aberto.value=null;exclusao.value=null;await recarregar();window.dispatchEvent(new Event('cadastros-atualizados'))}


async function recarregar() {
  try {
    await carregarEstudos()
    if (aberto.value) aberto.value = await Est.Detalhe(aberto.value.uuid)
    erro.value = ''
  } catch (e) { erro.value = mensagem(e) }
}
onMounted(recarregar)
onActivated(recarregar)

const semAcento = (s: string) => s.normalize('NFD').replace(/\p{Mn}/gu, '').toLowerCase()
const lista = computed(() => {
  const t = semAcento(busca.value.trim())
  return t ? estudos.value.filter((e) => semAcento(`${e.nome} ${e.codigo_exibicao}`).includes(t)) : estudos.value
})
const colEstudos: Coluna<Estudo>[] = [
  { chave: 'nome', rotulo: 'Estudo de situação', largura: '28%' },
  { chave: 'codigo_exibicao', rotulo: 'Código' },
  { chave: 'periodo', rotulo: 'Jornadas', valor: (l) => l.primeira_data, formatar: (l) => `${l.jornadas} · ${dataBR(l.primeira_data)}${l.ultima_data !== l.primeira_data ? ` a ${dataBR(l.ultima_data)}` : ''}` },
  { chave: 'equipes', rotulo: 'Equipes', tipo: 'numero' },
  { chave: 'familias', rotulo: 'Famílias', tipo: 'numero' },
  { chave: 'ultima_importacao', rotulo: 'Última importação', valor: (l) => l.ultima_importacao ?? 0, formatar: (l) => (l.ultima_importacao ? quando(l.ultima_importacao) : '—'), secundaria: true },
  { chave: 'acao', rotulo: 'Ação', ordenavel: false, largura: '120px' },
]

async function abrir(e: Estudo) {
  try { aberto.value = await Est.Detalhe(e.uuid) } catch (x) { erro.value = mensagem(x) }
}

async function copiar(texto: string, chave: string) {
  try { await navigator.clipboard.writeText(texto); copiado.value = chave; window.setTimeout(() => { if (copiado.value === chave) copiado.value = '' }, 1800) } catch { /* sem área de transferência */ }
}
const textoCodigos = (j: Jornada) => `${aberto.value?.nome} — Jornada ${j.numero} (${dataBR(j.data)})\n` + (j.equipes ?? []).map((q) => `${q.nome}: ${q.codigo_exibicao}`).join('\n')

// ---------- novo estudo ----------
const hoje = () => new Date().toISOString().slice(0, 10)
const novoEstudo = ref<{ nome: string; data: string; equipes: string } | null>(null)
const salvando = ref(false)
const linhas = (s: string) => s.split('\n').map((x) => x.trim()).filter(Boolean)
async function criarEstudo() {
  if (!novoEstudo.value) return
  salvando.value = true
  try {
    const e = await Est.Criar(novoEstudo.value.nome, novoEstudo.value.data, linhas(novoEstudo.value.equipes))
    novoEstudo.value = null
    await carregarEstudos()
    if (e) { estudoFoco.value = e.uuid; aberto.value = e }
  } catch (x) { erro.value = mensagem(x) } finally { salvando.value = false }
}

// ---------- nova jornada ----------
const novaJornada = ref<{ data: string; copiar: Record<string, boolean>; novas: string } | null>(null)
const nomesAnteriores = computed(() => {
  const vistos = new Map<string, string>()
  for (const j of aberto.value?.lista_jornadas ?? []) for (const q of j.equipes ?? []) vistos.set(semAcento(q.nome), q.nome)
  return [...vistos.values()]
})
function abrirNovaJornada() {
  const ultima = aberto.value?.lista_jornadas?.at(-1)
  const copiar: Record<string, boolean> = {}
  for (const n of nomesAnteriores.value) copiar[n] = (ultima?.equipes ?? []).some((q) => q.nome === n)
  novaJornada.value = { data: hoje(), copiar, novas: '' }
}
async function criarJornada() {
  if (!aberto.value || !novaJornada.value) return
  salvando.value = true
  try {
    const nomes = [...Object.entries(novaJornada.value.copiar).filter(([, v]) => v).map(([k]) => k), ...linhas(novaJornada.value.novas)]
    await Est.CriarJornada(aberto.value.uuid, novaJornada.value.data, nomes)
    novaJornada.value = null
    await recarregar()
  } catch (x) { erro.value = mensagem(x) } finally { salvando.value = false }
}

// ---------- equipes e edições ----------
const novaEquipe = ref<{ jornada: string; nome: string } | null>(null)
async function adicionarEquipe() {
  if (!novaEquipe.value) return
  try { await Est.AdicionarEquipes(novaEquipe.value.jornada, [novaEquipe.value.nome]); novaEquipe.value = null; await recarregar() } catch (x) { erro.value = mensagem(x) }
}
function renomearEstudo(){if(aberto.value)edicao.value={id:aberto.value.uuid,nome:aberto.value.nome,tipo:'estudo'}}
function excluirEquipe(uuid:string,nome:string){exclusao.value={id:uuid,nome,tipo:'equipe',resumo:'Somente equipes sem coleta vinculada podem ser excluídas.'}}
async function emFoco() { if (aberto.value) await definirFoco(aberto.value.uuid) }
</script>

<template>
  <div class="pilha">
    <!-- Página de um estudo -->
    <template v-if="aberto">
      <button class="discreto voltar" @click="aberto = null">← Todos os estudos</button>
      <header class="cabeca">
        <div class="titulo">
          <h1>{{ aberto.nome }}</h1>
          <p class="suave">
            Código do estudo <span class="mono">{{ aberto.codigo_exibicao }}</span> ·
            {{ num(aberto.jornadas) }} jornada(s) · {{ num(aberto.equipes) }} equipe(s) · {{ num(aberto.familias) }} família(s)
          </p>
        </div>
        <div class="linha">
          <button class="discreto" @click="renomearEstudo">Renomear</button>
          <button v-if="estudoFoco !== aberto.uuid" @click="emFoco">Pôr em foco no Painel</button>
          <span v-else class="selo ok">Em foco no Painel</span>
          <button class="primario" @click="abrirNovaJornada">Nova jornada</button>
        </div>
      </header>
      <p v-if="erro" class="aviso-erro">{{ erro }}</p>

      <section class="bloco"><div class="bloco-cabeca"><h2>Jornadas</h2></div>
        <TabelaDados :colunas="colJornadas" :linhas="aberto.lista_jornadas??[]" :chave="j=>j.uuid" rotulo="Jornadas do estudo" clicavel :selecionada="jornadaAtual?.uuid" @abrir="jornadaId=$event.uuid">
          <template #acao="{linha}"><div class="acoes-tabela"><AcaoIcone acao="Adicionar equipe" icone="adicionar" @click="novaEquipe={jornada:linha.uuid,nome:''}"/><AcaoIcone acao="Copiar todos os códigos" icone="copiar" :disabled="!linha.equipes?.length" @click="copiar(textoCodigos(linha),linha.uuid)"/></div></template>
        </TabelaDados>
      </section>
      <section v-if="jornadaAtual" class="bloco"><div class="bloco-cabeca"><h2>Equipes · Jornada {{jornadaAtual.numero}} · {{dataBR(jornadaAtual.data)}}</h2><span role="status">{{copiado?'Códigos copiados':''}}</span></div>
        <TabelaDados :colunas="colEquipes" :linhas="jornadaAtual.equipes??[]" :chave="q=>q.uuid" rotulo="Equipes da jornada">
          <template #codigo_exibicao="{linha}"><span class="mono">{{linha.codigo_exibicao}}</span></template>
          <template #acao="{linha}"><div class="acoes-tabela"><AcaoIcone acao="Copiar código" icone="copiar" @click="copiar(linha.codigo_exibicao,linha.uuid)"/><AcaoIcone acao="Editar nome da equipe" icone="editar" @click="edicao={id:linha.uuid,nome:linha.nome,tipo:'equipe'}"/><AcaoIcone :acao="linha.coletas?'Equipe com coleta vinculada: exclusão bloqueada':'Excluir equipe'" icone="excluir" :disabled="!!linha.coletas" @click="excluirEquipe(linha.uuid,linha.nome)"/></div></template>
        </TabelaDados>
      </section>
    </template>

    <!-- Lista de estudos -->
    <template v-else>
      <header class="cabeca">
        <div class="titulo">
          <h1>Estudos de situação</h1>
          <p class="suave">Cada estudo tem jornadas (dias de levantamento) e, em cada jornada, as equipes com o código que o celular usa.</p>
        </div>
        <button class="primario" @click="novoEstudo = { nome: '', data: hoje(), equipes: '' }">Novo estudo</button>
      </header>
      <p v-if="erro" class="aviso-erro">{{ erro }}</p>
      <input v-model="busca" type="search" placeholder="Buscar por nome ou código" class="busca" />
      <TabelaDados :colunas="colEstudos" :linhas="lista" :chave="(l) => l.uuid" rotulo="Estudos de situação" texto-vazio="Nenhum estudo ainda. Crie o primeiro para gerar os códigos das equipes."
                   clicavel @abrir="abrir">
        <template #acao="{linha}"><div class="acoes-tabela"><AcaoIcone acao="Editar nome do estudo" icone="editar" @click="edicao={id:linha.uuid,nome:linha.nome,tipo:'estudo'}"/><AcaoIcone acao="Excluir estudo e dados vinculados" icone="excluir" @click="pedirExcluirEstudo(linha)"/></div></template>
        <template #nome="{ linha }"><strong>{{ linha.nome }}</strong> <span v-if="linha.uuid === estudoFoco" class="selo ok">Em foco</span></template>
        <template #codigo_exibicao="{ linha }"><span class="mono">{{ linha.codigo_exibicao }}</span></template>
      </TabelaDados>
    </template>

    <!-- Modais -->
    <ConfirmarExclusao v-if="exclusao" :nome="exclusao.nome" :resumo="exclusao.resumo" :executar="confirmarExclusao" @fechar="exclusao=null" @concluido="excluiu" />
    <ModalFormulario v-if="edicao" :titulo="edicao.tipo==='estudo'?'Editar nome do estudo':'Editar nome da equipe'" :ocupado="salvando" @fechar="edicao=null">
      <form id="renomear-registro" class="pilha" @submit.prevent="salvarNome"><p v-if="erro" class="aviso-erro">{{erro}}</p><label class="campo">Nome *<input v-model="edicao.nome" required maxlength="200" autofocus></label><p class="suave">O código e os vínculos permanecem os mesmos.</p></form>
      <template #acoes><button :disabled="salvando" @click="edicao=null">Cancelar</button><button class="primario" type="submit" form="renomear-registro" :disabled="salvando">Salvar nome</button></template>
    </ModalFormulario>
    <div v-if="novoEstudo" class="modal-fundo" @click.self="novoEstudo = null">
      <form class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="ne-titulo" @submit.prevent="criarEstudo">
        <h2 id="ne-titulo">Novo estudo de situação</h2>
        <label class="campo">Nome do estudo *<input v-model="novoEstudo.nome" maxlength="200" required autofocus placeholder="Ex.: Fazenda Santa Maria" /></label>
        <label class="campo">Data da primeira jornada *<input v-model="novoEstudo.data" type="date" required /></label>
        <label class="campo">Equipes da primeira jornada (uma por linha; opcional)
          <textarea v-model="novoEstudo.equipes" rows="4" placeholder="Equipe Alfa&#10;Equipe Bravo"></textarea>
        </label>
        <p class="fraco">O código do estudo e o de cada equipe são gerados pela Central.</p>
        <div class="linha" style="justify-content: flex-end">
          <button type="button" @click="novoEstudo = null">Cancelar</button>
          <button class="primario" :disabled="salvando || !novoEstudo.nome.trim() || !novoEstudo.data">Criar estudo</button>
        </div>
      </form>
    </div>

    <div v-if="novaJornada && aberto" class="modal-fundo" @click.self="novaJornada = null">
      <form class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="nj-titulo" @submit.prevent="criarJornada">
        <h2 id="nj-titulo">Nova jornada · {{ aberto.nome }}</h2>
        <label class="campo">Data da jornada *<input v-model="novaJornada.data" type="date" required /></label>
        <fieldset v-if="nomesAnteriores.length" class="copiar">
          <legend>Equipes de jornadas anteriores</legend>
          <label v-for="n in nomesAnteriores" :key="n" class="caixa"><input v-model="novaJornada.copiar[n]" type="checkbox" /> {{ n }}</label>
        </fieldset>
        <label class="campo">Equipes novas (uma por linha)<textarea v-model="novaJornada.novas" rows="3"></textarea></label>
        <p class="fraco">Cada equipe recebe um código novo nesta jornada, mesmo com o nome repetido.</p>
        <div class="linha" style="justify-content: flex-end">
          <button type="button" @click="novaJornada = null">Cancelar</button>
          <button class="primario" :disabled="salvando || !novaJornada.data">Criar jornada</button>
        </div>
      </form>
    </div>

    <div v-if="novaEquipe" class="modal-fundo" @click.self="novaEquipe = null">
      <form class="modal pilha" role="dialog" aria-modal="true" aria-labelledby="eq-titulo" @submit.prevent="adicionarEquipe">
        <h2 id="eq-titulo">Nova equipe</h2>
        <label class="campo">Nome da equipe *<input v-model="novaEquipe.nome" maxlength="200" required autofocus placeholder="Ex.: Equipe Charlie" /></label>
        <div class="linha" style="justify-content: flex-end">
          <button type="button" @click="novaEquipe = null">Cancelar</button>
          <button class="primario" :disabled="!novaEquipe.nome.trim()">Adicionar</button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.cabeca { display: flex; justify-content: space-between; align-items: flex-end; gap: 1rem; flex-wrap: wrap; }
.titulo { display: flex; flex-direction: column; gap: .35rem; }
.voltar { align-self: flex-start; }
.busca { max-width: 420px; }
.equipes { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: .75rem; }
.equipe { border: 1px solid var(--regua); border-radius: var(--raio); padding: .75rem .9rem; display: flex; flex-direction: column; gap: .5rem; background: var(--folha); }
.equipe > div:first-child { display: flex; flex-direction: column; gap: .1rem; }
.codigo-linha { display: flex; align-items: center; gap: .5rem; flex-wrap: wrap; }
.codigo { font-family: ui-monospace, "Cascadia Mono", Consolas, monospace; font-size: 1.35rem; font-weight: 700; letter-spacing: .06em; color: var(--mata); }
.perigo-texto { color: var(--erro); }
.campo { display: flex; flex-direction: column; gap: .3rem; font-weight: 600; }
.campo input, .campo textarea { font-weight: 400; }
.copiar { border: 1px solid var(--regua); border-radius: var(--raio); padding: .6rem .9rem; display: flex; flex-wrap: wrap; gap: .4rem 1.2rem; }
.caixa { display: inline-flex; align-items: center; gap: .4rem; }
</style>
