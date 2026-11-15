// Estudo em foco (A28): Painel, Propriedades, relatórios e cartas mostram um estudo por vez.
// A escolha fica guardada na Central (meta.estudo_em_foco) e é compartilhada pelas telas.
import { computed, ref } from 'vue'
import { EstudoService as Est } from '../bindings/sitposse7/desktop/app'
import type { Estudo } from '../bindings/sitposse7/desktop/internal/estudos/models'

export const estudoFoco = ref('')
export const estudos = ref<Estudo[]>([])
export const estudoAtual = computed(() => estudos.value.find((e) => e.uuid === estudoFoco.value) ?? null)

/** Recarrega a lista; na primeira vez, lê o estudo em foco guardado. */
export async function carregarEstudos(): Promise<void> {
  estudos.value = (await Est.Listar()) ?? []
  if (!estudoFoco.value || !estudos.value.some((e) => e.uuid === estudoFoco.value)) estudoFoco.value = await Est.EmFoco()
}

export async function definirFoco(uuid: string): Promise<void> {
  estudoFoco.value = uuid
  await Est.DefinirEmFoco(uuid)
}

/** "10/10/2026" a partir de "2026-10-10". */
export function dataBR(iso: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso)
  return m ? `${m[3]}/${m[2]}/${m[1]}` : iso
}
