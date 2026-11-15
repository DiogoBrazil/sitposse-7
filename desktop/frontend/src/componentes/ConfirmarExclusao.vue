<script setup lang="ts">
import { ref } from 'vue'
import ModalFormulario from './ModalFormulario.vue'
import { mensagem } from '../formato'
const props=defineProps<{ nome:string; resumo:string; executar:()=>Promise<unknown> }>()
const emit=defineEmits<{ fechar:[]; concluido:[] }>()
const ocupado=ref(false),erro=ref('')
async function confirmar(){ocupado.value=true;try{await props.executar();emit('concluido')}catch(e){erro.value=mensagem(e)}finally{ocupado.value=false}}
</script>
<template><ModalFormulario titulo="Excluir e retirar dos resultados" :ocupado="ocupado" @fechar="emit('fechar')"><div class="pilha"><strong>{{ nome }}</strong><p>Esta ação excluirá este registro e todos os dados vinculados a ele das consultas, estatísticas e novas exportações.</p><p>{{ resumo }}</p><p class="suave">O histórico e os arquivos são preservados. Esta ação não altera os dados no celular. Cartas-imagem já geradas continuam guardadas.</p><p v-if="erro" class="aviso-erro" role="alert">{{ erro }}</p></div><template #acoes><button :disabled="ocupado" @click="emit('fechar')">Cancelar</button><button class="perigo" :disabled="ocupado" @click="confirmar">{{ ocupado?'Excluindo…':'Excluir e confirmar' }}</button></template></ModalFormulario></template>
