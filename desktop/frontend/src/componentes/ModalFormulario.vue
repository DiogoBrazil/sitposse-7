<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
defineProps<{ titulo:string; ocupado?:boolean }>()
const emit=defineEmits<{ fechar:[] }>()
const dialogo=ref<HTMLDialogElement>()
let anterior:HTMLElement|null=null
onMounted(()=>{anterior=document.activeElement as HTMLElement;dialogo.value?.showModal()})
onBeforeUnmount(()=>{dialogo.value?.close();anterior?.focus()})
</script>
<template><Teleport to="body"><dialog ref="dialogo" class="modal-editor" aria-modal="true" :aria-label="titulo" @cancel.prevent="!ocupado && emit('fechar')"><header><h2>{{ titulo }}</h2><button type="button" class="discreto" :disabled="ocupado" aria-label="Fechar" @click="emit('fechar')">✕</button></header><div class="conteudo"><slot /></div><footer><slot name="acoes" /></footer></dialog></Teleport></template>
<style scoped>.modal-editor{border:1px solid var(--regua);border-radius:16px;padding:0;background:var(--folha);color:var(--tinta);width:min(900px,calc(100vw - 32px));max-height:calc(100dvh - 40px);box-shadow:0 24px 80px #0004}.modal-editor[open]{display:flex;flex-direction:column}.modal-editor::backdrop{background:#101e20aa;backdrop-filter:blur(3px)}header,footer{display:flex;align-items:center;gap:1rem;padding:1rem 1.4rem;flex-shrink:0}header{justify-content:space-between;border-bottom:1px solid var(--regua)}footer{justify-content:flex-end;border-top:1px solid var(--regua);flex-wrap:wrap}.conteudo{overflow:auto;padding:1.4rem}h2{margin:0;font-size:1.35rem}@media(max-width:600px){header,footer,.conteudo{padding:1rem}.modal-editor{width:calc(100vw - 16px);max-height:calc(100dvh - 16px)}}</style>
