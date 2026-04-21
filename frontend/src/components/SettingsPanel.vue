<script setup>
import { ref, watch } from 'vue'
import * as SettingsAPI from '../../wailsjs/wailsjs/go/delivery/SettingsHandler'
import TrustedCaListSection from './TrustedCaListSection.vue'

const props = defineProps({
  active: { type: Boolean, default: false }
})

const emit = defineEmits(['console'])

function log(msg) {
  emit('console', msg)
}

const loading = ref(false)
const saving = ref(false)

const mode = ref('none')
const proxyURL = ref('')
const proxyUser = ref('')
const proxyPassword = ref('')
const noProxy = ref('')

const testURL = ref('https://example.com')
const testing = ref(false)
const testResult = ref(null)

const cas = ref([])
const caLabel = ref('')
const caPem = ref('')

async function load() {
  loading.value = true
  try {
    const s = await SettingsAPI.GetProxySettings()
    if (s && typeof s === 'object') {
      mode.value = String(s.mode || 'none')
      proxyURL.value = String(s.url || '')
      proxyUser.value = String(s.username || '')
      // password is never returned — keep field empty unless user types a new one
      proxyPassword.value = ''
      noProxy.value = String(s.no_proxy || '')
    }
    const list = await SettingsAPI.ListTrustedCAs()
    cas.value = Array.isArray(list) ? list : []
    testResult.value = null
  } catch (e) {
    log(`[Settings] Load failed: ${e?.message || String(e)}`)
  } finally {
    loading.value = false
  }
}

async function saveProxy() {
  saving.value = true
  try {
    await SettingsAPI.SetProxySettings({
      mode: mode.value,
      url: proxyURL.value,
      username: proxyUser.value,
      password: proxyPassword.value,
      no_proxy: noProxy.value
    })
    log('[Settings] Proxy settings saved.')
    proxyPassword.value = ''
    await load()
  } catch (e) {
    log(`[Settings] Save failed: ${e?.message || String(e)}`)
  } finally {
    saving.value = false
  }
}

async function runTest() {
  testing.value = true
  testResult.value = null
  try {
    const r = await SettingsAPI.TestProxy(String(testURL.value || '').trim())
    testResult.value = r && typeof r === 'object' ? r : null
    if (r?.ok) {
      log(`[Settings] Proxy test OK — HTTP ${r.status_code} in ${r.duration_ms} ms`)
    } else {
      log(`[Settings] Proxy test failed: ${r?.error_message || 'unknown error'}`)
    }
  } catch (e) {
    log(`[Settings] Proxy test error: ${e?.message || String(e)}`)
  } finally {
    testing.value = false
  }
}

async function pickCAFile() {
  try {
    const p = await SettingsAPI.PickCACertFile()
    if (!p) return
    const text = await SettingsAPI.ReadTextFile(p)
    caPem.value = String(text || '')
    log(`[Settings] Loaded PEM from ${p}`)
  } catch (e) {
    log(`[Settings] Could not read file: ${e?.message || String(e)}`)
  }
}

async function addCA() {
  const label = String(caLabel.value || '').trim()
  const pem = String(caPem.value || '').trim()
  if (!label || !pem) {
    log('[Settings] CA label + PEM content are required.')
    return
  }
  try {
    await SettingsAPI.AddTrustedCA(label, pem)
    log(`[Settings] CA "${label}" added.`)
    caLabel.value = ''
    caPem.value = ''
    await load()
  } catch (e) {
    log(`[Settings] Add CA failed: ${e?.message || String(e)}`)
  }
}

async function toggleCA(id, enabled) {
  try {
    await SettingsAPI.SetTrustedCAEnabled(String(id), !!enabled)
    await load()
  } catch (e) {
    log(`[Settings] Toggle CA failed: ${e?.message || String(e)}`)
  }
}

async function removeCA(id) {
  try {
    await SettingsAPI.DeleteTrustedCA(String(id))
    await load()
  } catch (e) {
    log(`[Settings] Delete CA failed: ${e?.message || String(e)}`)
  }
}

watch(
  () => props.active,
  (on) => {
    if (on) load()
  },
  { immediate: true }
)
</script>

<template>
  <div class="app-scrollbar min-h-0 flex-1 overflow-y-auto p-3 text-sm" style="color: #e5e7eb">
    <div v-if="loading" class="text-xs text-gray-500">Loading settings…</div>

    <div v-else class="space-y-6">
      <section>
        <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Proxy</div>
        <p class="mt-1 text-xs text-gray-500">
          Use <span class="font-mono text-gray-400">system</span> to honour <span class="font-mono">HTTP(S)_PROXY</span> env vars.
          Use <span class="font-mono text-gray-400">manual</span> for an explicit proxy URL + optional credentials.
        </p>

        <div class="mt-3 space-y-2">
          <label class="block text-xs text-gray-500">Mode</label>
          <select
            v-model="mode"
            class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
          >
            <option value="none">None</option>
            <option value="system">System env</option>
            <option value="manual">Manual</option>
          </select>
        </div>

        <div v-if="mode === 'manual'" class="mt-3 space-y-2">
          <label class="block text-xs text-gray-500">Proxy URL</label>
          <input
            v-model="proxyURL"
            class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
            placeholder="http://proxy.company.local:8080"
          />
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <div>
              <label class="block text-xs text-gray-500">Username (optional)</label>
              <input
                v-model="proxyUser"
                autocomplete="off"
                class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
              />
            </div>
            <div>
              <label class="block text-xs text-gray-500">Password (optional)</label>
              <input
                v-model="proxyPassword"
                type="password"
                autocomplete="new-password"
                class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500"
                placeholder="leave empty to keep existing"
              />
            </div>
          </div>
          <div>
            <label class="block text-xs text-gray-500">NO_PROXY</label>
            <textarea
              v-model="noProxy"
              rows="3"
              class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 font-mono text-[11px] text-gray-200 outline-none focus:border-orange-500"
              placeholder="localhost,127.0.0.1,.internal.corp"
            />
            <div class="mt-1 text-[10px] text-gray-600">Comma-separated hosts. Leading <span class="font-mono">.</span> means suffix match.</div>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="rounded bg-orange-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-orange-700 disabled:opacity-50"
            :disabled="saving"
            @click="saveProxy"
          >
            Save proxy
          </button>
        </div>

        <div class="mt-4 rounded border border-gray-800 bg-[#1a1a1a] p-3">
          <div class="text-xs font-semibold text-gray-300">Test proxy</div>
          <div class="mt-2 flex flex-wrap gap-2">
            <input
              v-model="testURL"
              class="min-w-[200px] flex-1 rounded border border-gray-700 bg-[#141414] px-2 py-1.5 font-mono text-xs text-gray-200 outline-none focus:border-orange-500"
            />
            <button
              type="button"
              class="rounded border border-gray-600 bg-gray-800 px-3 py-1.5 text-xs font-semibold text-gray-200 hover:border-orange-500 hover:text-orange-200 disabled:opacity-50"
              :disabled="testing"
              @click="runTest"
            >
              {{ testing ? 'Testing…' : 'Run test' }}
            </button>
          </div>
          <div v-if="testResult" class="mt-2 text-xs text-gray-400">
            <div v-if="testResult.ok">
              OK — HTTP <span class="font-mono">{{ testResult.status_code }}</span> in
              <span class="font-mono">{{ testResult.duration_ms }}</span> ms
            </div>
            <div v-else class="text-red-300">
              Failed: <span class="font-mono">{{ testResult.error_message }}</span>
            </div>
          </div>
        </div>
      </section>

      <TrustedCaListSection
        :items="cas"
        :disabled="loading"
        @toggle="(id, en) => toggleCA(id, en)"
        @remove="(id) => removeCA(id)"
      />

      <section>
        <div class="text-[10px] font-bold uppercase tracking-wider text-gray-500">Add custom CA certificate</div>
        <p class="mt-1 text-xs text-gray-500">
          Import PEM-encoded CA certificates (corporate SSL inspection). They are appended to the OS trust store for outbound HTTPS.
        </p>

        <div class="mt-3 space-y-2">
          <div class="flex flex-wrap gap-2">
            <input v-model="caLabel" class="flex-1 rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 text-xs text-gray-200 outline-none focus:border-orange-500" placeholder="Label" />
            <button type="button" class="rounded border border-gray-600 bg-gray-800 px-3 py-1.5 text-xs text-gray-200 hover:border-orange-500" @click="pickCAFile">
              Pick file…
            </button>
          </div>
          <textarea
            v-model="caPem"
            rows="6"
            class="w-full rounded border border-gray-700 bg-[#1a1a1a] px-2 py-1.5 font-mono text-[11px] text-gray-200 outline-none focus:border-orange-500"
            placeholder="-----BEGIN CERTIFICATE----- ..."
          />
          <button type="button" class="rounded bg-gray-800 px-3 py-1.5 text-xs font-semibold text-gray-200 hover:bg-gray-700" @click="addCA">
            Add CA
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
