<script setup lang="ts">
import { onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import { SincronizacaoService as Central } from '../bindings/sitposse7/desktop/app'
import type { Info } from '../bindings/sitposse7/desktop/app/models'

// Fase 0: tela de diagnóstico para o spike de rede (docs/operacao/rede-em-campo.md).
const info = ref<Info | null>(null)
const qr = ref('')
const payload = ref('')
const erro = ref('')

async function atualizar() {
  erro.value = ''
  try {
    info.value = await Central.Info()
    const p = await Central.QRPareamento()
    payload.value = JSON.stringify(p)
    qr.value = await QRCode.toDataURL(payload.value, { errorCorrectionLevel: 'M', margin: 2, width: 320 })
  } catch (e) {
    qr.value = ''
    erro.value = e instanceof Error ? e.message : String(e)
  }
}

async function copiar() {
  await navigator.clipboard.writeText(payload.value)
}

onMounted(atualizar)
</script>

<template>
  <main>
    <header>
      <h1>SITPOSSE-7 Central</h1>
      <span class="sub">7º BPM/PMRO · versão {{ info?.versao }}</span>
    </header>

    <section class="cartao">
      <h2>Sincronização — diagnóstico (Fase 0)</h2>
      <p class="ajuda">
        Conecte o notebook e os celulares à mesma rede Wi-Fi (hotspot do celular ou do Windows).
        Não é preciso internet.
      </p>

      <div class="grade">
        <dl>
          <dt>Servidor</dt>
          <dd :class="info?.servidor_ativo ? 'ok' : 'erro'">
            {{ info?.servidor_ativo ? `Ativo na porta ${info?.porta}` : 'Inativo' }}
          </dd>
          <dt>Endereços na rede local</dt>
          <dd>
            <template v-if="info?.ips?.length">{{ info.ips.join(' · ') }}</template>
            <span v-else class="erro">Nenhuma rede Wi-Fi/local ativa. Conecte-se ao hotspot.</span>
          </dd>
          <dt>Impressão digital (pin)</dt>
          <dd class="mono">{{ info?.pin }}</dd>
          <dt>ID da Central</dt>
          <dd class="mono">{{ info?.central_id }}</dd>
          <dt>Pasta de dados</dt>
          <dd class="mono">{{ info?.pasta_dados }}</dd>
          <dt>Versão do esquema</dt>
          <dd>{{ info?.versao_esquema }}</dd>
        </dl>

        <figure v-if="qr">
          <img :src="qr" alt="QR Code de conexão com a Central" width="320" height="320" />
          <figcaption>Leia este QR no SITPOSSE-7 Campo (tela de diagnóstico).</figcaption>
        </figure>
      </div>

      <p v-if="info?.erro_servidor" class="erro">{{ info.erro_servidor }}</p>
      <p v-if="erro" class="erro">{{ erro }}</p>

      <div class="acoes">
        <button @click="atualizar">Atualizar</button>
        <button class="secundario" :disabled="!payload" @click="copiar">Copiar dados do QR</button>
      </div>
    </section>
  </main>
</template>

<style scoped>
main { max-width: 1100px; margin: 0 auto; padding: 24px; }
header { display: flex; align-items: baseline; gap: 12px; margin-bottom: 16px; }
h1 { font-size: 1.5rem; margin: 0; }
.sub { color: var(--texto-suave); }
.cartao { background: var(--superficie); border: 1px solid var(--borda); border-radius: 12px; padding: 24px; }
h2 { margin: 0 0 4px; font-size: 1.15rem; }
.ajuda { color: var(--texto-suave); margin: 0 0 20px; }
.grade { display: grid; grid-template-columns: 1fr auto; gap: 24px; align-items: start; }
dl { display: grid; grid-template-columns: max-content 1fr; gap: 10px 16px; margin: 0; }
dt { color: var(--texto-suave); }
dd { margin: 0; word-break: break-all; }
.mono { font-family: ui-monospace, Consolas, monospace; font-size: 0.9rem; }
figure { margin: 0; text-align: center; }
figure img { background: #fff; border-radius: 8px; }
figcaption { color: var(--texto-suave); font-size: 0.85rem; max-width: 320px; }
.ok { color: var(--ok); font-weight: 600; }
.erro { color: var(--erro); }
.acoes { display: flex; gap: 12px; margin-top: 20px; }
button { font: inherit; padding: 10px 18px; border-radius: 8px; border: 1px solid var(--primaria); background: var(--primaria); color: #fff; cursor: pointer; }
button.secundario { background: transparent; color: var(--primaria); }
button:disabled { opacity: 0.5; cursor: default; }
@media (max-width: 900px) { .grade { grid-template-columns: 1fr; } }
</style>
