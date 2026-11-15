<script setup lang="ts">
defineProps<{ acao: string; icone: 'editar'|'excluir'|'copiar'|'adicionar'|'baixar'; disabled?: boolean }>()
const paths = { editar:'M4 20l4-1L20 7l-3-3L5 16zM14 7l3 3', excluir:'M4 6h16M9 6V3h6v3M6 6l1 15h10l1-15M10 10v7M14 10v7', copiar:'M9 8h11v13H9zM15 8V3H4v13h5', adicionar:'M12 4v16M4 12h16', baixar:'M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4' }
</script>
<template><button type="button" class="acao-icone" :class="{ perigo: icone === 'excluir' }" :title="acao" :aria-label="acao" :disabled="disabled" @click.stop @keydown.stop><svg viewBox="0 0 24 24" aria-hidden="true"><path :d="paths[icone]" /></svg></button></template>
<style scoped>.acao-icone{width:42px;height:42px;padding:9px;display:inline-flex;align-items:center;justify-content:center;background:transparent;flex:none}.acao-icone svg{width:21px;height:21px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.acao-icone.perigo{color:var(--erro)}</style>
