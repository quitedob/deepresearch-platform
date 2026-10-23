import { after, before, beforeEach, test } from 'node:test'
import assert from 'node:assert/strict'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import { createPinia, setActivePinia } from 'pinia'

let server
let mainAPI
let researchStore
let serviceAPI
let streamClient
let chatStoreModule

function memoryStorage() {
  const values = new Map()
  return {
    getItem: key => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: key => values.delete(key)
  }
}

before(async () => {
  server = await createServer({
    configFile: false,
    cacheDir: fileURLToPath(new URL('../../build/temp/vite-api-contract-tests', import.meta.url)),
    optimizeDeps: { noDiscovery: true, include: [] },
    root: fileURLToPath(new URL('..', import.meta.url)),
    resolve: { alias: { '@': fileURLToPath(new URL('../src', import.meta.url)) } },
    server: { middlewareMode: true, watch: null, hmr: false }
  })
  mainAPI = await server.ssrLoadModule('/src/api/index.js')
  researchStore = await server.ssrLoadModule('/src/stores/research.js')
  serviceAPI = await server.ssrLoadModule('/src/services/api.js')
  streamClient = await server.ssrLoadModule('/src/utils/apiClient.js')
  chatStoreModule = await server.ssrLoadModule('/src/store/index.js')
})

after(async () => { await server?.close() })

beforeEach(() => {
  globalThis.localStorage = memoryStorage()
  globalThis.sessionStorage = memoryStorage()
  globalThis.window = { location: { pathname: '/home', href: '/home' } }
  for (const storage of [localStorage, sessionStorage]) {
    storage.setItem('auth_token', 'test-access')
    storage.setItem('refresh_token', 'test-refresh')
    storage.setItem('user', '{"id":"test-user"}')
  }
  setActivePinia(createPinia())
})

test('inactive accounts clear access, refresh and cached identity in both stores', async () => {
  for (const data of [
    { success: false, code: 'ERR_ACCOUNT_INACTIVE', message: '账户已被禁用或不可用' },
    { success: false, error: { code: 'ERR_ACCOUNT_INACTIVE', message: '账户已被禁用或不可用' } },
    { code: 403, error: { code: 'ERR_ACCOUNT_INACTIVE', message: '账户已被禁用或不可用' } }
  ]) {
    for (const storage of [localStorage, sessionStorage]) {
      storage.setItem('auth_token', 'test-access')
      storage.setItem('refresh_token', 'test-refresh')
      storage.setItem('user', '{"id":"test-user"}')
    }
    window.location.href = '/home'
    await assert.rejects(mainAPI.apiClient.get('/user/profile', {
      adapter: async config => { throw { config, response: { status: 403, data } } }
    }))
    for (const storage of [localStorage, sessionStorage]) {
      assert.equal(storage.getItem('auth_token'), null)
      assert.equal(storage.getItem('refresh_token'), null)
      assert.equal(storage.getItem('user'), null)
    }
    assert.match(window.location.href, /^\/login\?reason=banned/)
  }
})

test('ordinary forbidden responses retain the login session', async () => {
  await assert.rejects(mainAPI.apiClient.get('/research/start', {
    adapter: async config => { throw { config, response: { status: 403, data: { code: 'ERR_RESEARCH_QUOTA_EXCEEDED', message: '配额已耗尽' } } } }
  }))
  assert.equal(localStorage.getItem('auth_token'), 'test-access')
  assert.equal(window.location.href, '/home')
})

test('expired credentials clear the session while login errors stay on the form', async () => {
  const rejected = config => Promise.reject({ config, response: { status: 401, data: { message: '凭据无效' } } })
  await assert.rejects(mainAPI.apiClient.get('/user/profile', { adapter: rejected }))
  assert.equal(localStorage.getItem('auth_token'), null)
  assert.equal(window.location.href, '/login')
  window.location.pathname = '/login'
  window.location.href = '/login?redirect=/home'
  await assert.rejects(mainAPI.apiClient.post('/auth/login', {}, { adapter: rejected }))
  assert.equal(window.location.href, '/login?redirect=/home')
})

test('recovered chat store switches model provider and restores session drafts', () => {
  const store = chatStoreModule.useChatStore()
  store.availableModels = [{ name: 'test-model', provider: 'test-provider' }]
  store.setModel('test-model')
  assert.equal(store.currentProvider, 'test-provider')
  assert.equal(localStorage.getItem('chat-provider'), 'test-provider')
  store.saveDraft('session-one', 'Draft to restore')
  assert.equal(store.loadDraft('session-one'), 'Draft to restore')
  assert.equal(store.loadDraft('session-two'), '')
})

test('default research store submission uses the durable backend contract', async () => {
  let submitted
  mainAPI.apiClient.defaults.adapter = async config => {
    submitted = JSON.parse(config.data)
    assert.equal(submitted.llm_config, undefined)
    assert.equal(submitted.tools_config, undefined)
    assert.ok(config.headers['Idempotency-Key']?.length >= 8)
    return { status: 201, data: { success: true, session_id: 'research-test' }, headers: {}, config }
  }
  const store = researchStore.useResearchStore()
  const session = await store.startResearch('Explain durable research')
  assert.equal(session?.id, 'research-test', store.errorMessage)
  assert.equal(submitted.query, 'Explain durable research')
})

test('the research service also submits through the idempotent API', async () => {
  let requestCount = 0
  mainAPI.apiClient.defaults.adapter = async config => {
    requestCount++
    assert.ok(config.headers['Idempotency-Key']?.length >= 8)
    return { status: 201, data: { success: true, session_id: 'research-test' }, headers: {}, config }
  }
  const result = await new serviceAPI.ResearchService().startResearch({ query: 'A research question', research_type: 'deep' })
  assert.equal(result.session_id, 'research-test')
  assert.equal(requestCount, 1)
})

test('unexpected SSE EOF triggers reconnect handling', async () => {
  const originalFetch = globalThis.fetch
  globalThis.fetch = async () => new Response('data: {"type":"connected"}\n\n', { headers: { 'Content-Type': 'text/event-stream' } })
  let stream
  try {
    const error = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('EOF did not notify reconnect handler')), 1000)
      stream = streamClient.createSSEConnection('/research/stream/test')
      stream.onerror = error => { clearTimeout(timeout); resolve(error) }
    })
    assert.match(error.message, /连接.*关闭|stream.*closed/i)
  } finally {
    stream?.close()
    globalThis.fetch = originalFetch
  }
})

test('SSE accepts the backend data prefix without a space across chunks', async () => {
  const originalFetch = globalThis.fetch
  const encoder = new TextEncoder()
  globalThis.fetch = async () => new Response(new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode('event:message\ndata:{"type":"comp'))
      controller.enqueue(encoder.encode('leted","data":{"report_text":"Recovered report"}}\n\n'))
      controller.close()
    }
  }))
  let stream
  try {
    const event = await new Promise((resolve, reject) => {
      stream = streamClient.createSSEConnection('/research/stream/test', { onMessage: resolve, onError: reject })
    })
    assert.equal(event.type, 'completed')
    assert.equal(event.data.report_text, 'Recovered report')
  } finally {
    stream?.close()
    globalThis.fetch = originalFetch
  }
})
