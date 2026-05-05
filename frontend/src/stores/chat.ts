import { ref } from 'vue'
import { defineStore } from 'pinia'
import { getToken, removeToken } from '../services/auth'
import { listConversations, getMessages, deleteConversation } from '../services/api'
import type { Conversation, ChatMessage } from '../services/api'

export type { Conversation, ChatMessage }

export type PendingToolConfirmation = {
  confirmationCallId: string
  hint: string
  originalFunctionCall: Record<string, unknown>
}

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

type SSEParsed = {
  token?: string
  conversation_id?: number
  partial?: boolean
  done?: boolean
  pending_confirmation?: boolean
  tool_confirmation?: {
    confirmation_call_id?: string
    hint?: string
    original_function_call?: Record<string, unknown>
  }
}

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const currentConversationID = ref(0)
  const messages = ref<ChatMessage[]>([])
  const sending = ref(false)
  const error = ref('')
  const pendingToolConfirmation = ref<PendingToolConfirmation | null>(null)

  let abortController: AbortController | null = null

  async function loadConversations() {
    const res = await listConversations()
    conversations.value = res.data || []
  }

  async function loadMessages(conversationID: number) {
    const res = await getMessages(conversationID)
    messages.value = res.data || []
  }

  function switchConversation(id: number) {
    currentConversationID.value = id
    error.value = ''
    pendingToolConfirmation.value = null
    messages.value = []
    loadMessages(id).catch(e => {
      error.value = e.message
    })
  }

  function newConversation() {
    currentConversationID.value = 0
    messages.value = []
    error.value = ''
    pendingToolConfirmation.value = null
  }

  async function removeConversation(id: number) {
    await deleteConversation(id)
    conversations.value = conversations.value.filter(c => c.id !== id)

    if (currentConversationID.value === id) {
      pendingToolConfirmation.value = null
      if (conversations.value.length > 0) {
        switchConversation(conversations.value[0].id)
      } else {
        newConversation()
      }
    }
  }

  async function consumeChatStream(
    res: Response,
    opts: { wasNewConversation: boolean },
  ): Promise<void> {
    const reader = res.body?.getReader()
    if (!reader) throw new Error('无法读取响应流')

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const events = buffer.split('\n\n')
      buffer = events.pop() || ''

      for (const raw of events) {
        if (!raw.trim()) continue
        let eventType = ''
        let data = ''

        for (const line of raw.split('\n')) {
          if (line.startsWith('event: ')) eventType = line.slice(7).trim()
          else if (line.startsWith('data: ')) data = line.slice(6)
        }

        if (eventType === 'error') {
          let errMsg = '流式响应错误'
          try {
            const parsed = JSON.parse(data) as { error?: string }
            if (parsed.error) errMsg = parsed.error
          } catch { /* use default message */ }
          throw new Error(errMsg)
        }
        if (!data) continue

        try {
          const parsed = JSON.parse(data) as SSEParsed
          if (parsed.token) {
            const msgs = messages.value
            const last = msgs[msgs.length - 1]
            if (last && last.role === 'assistant') {
              last.content += parsed.token
            }
          }
          if (parsed.tool_confirmation?.confirmation_call_id) {
            pendingToolConfirmation.value = {
              confirmationCallId: parsed.tool_confirmation.confirmation_call_id,
              hint: parsed.tool_confirmation.hint || '',
              originalFunctionCall: parsed.tool_confirmation.original_function_call || {},
            }
          }
          if (parsed.done && parsed.conversation_id) {
            if (opts.wasNewConversation) {
              currentConversationID.value = parsed.conversation_id
              loadConversations().catch(() => {})
            }
          }
        } catch {
          /* skip unparseable frames */
        }
      }
    }
  }

  async function send(text: string, images?: string[]) {
    if ((!text.trim() && (!images || images.length === 0)) || sending.value) return
    if (pendingToolConfirmation.value) return

    pendingToolConfirmation.value = null

    const imagesJSON = images && images.length > 0 ? JSON.stringify(images) : ''

    const userMsg: ChatMessage = {
      id: 0,
      user_id: 0,
      conversation_id: currentConversationID.value,
      role: 'user',
      content: text,
      images: imagesJSON,
      created_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, userMsg]

    const aiMsg: ChatMessage = {
      id: 0,
      user_id: 0,
      conversation_id: currentConversationID.value,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, aiMsg]

    sending.value = true
    error.value = ''

    const wasNewConversation = currentConversationID.value === 0

    abortController = new AbortController()
    const token = getToken()

    try {
      const res = await fetch(`${BASE_URL}/chat/send`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          conversation_id: currentConversationID.value,
          message: text,
          ...(images && images.length > 0 ? { images } : {}),
        }),
        signal: abortController.signal,
      })

      if (res.status === 401) {
        removeToken()
        window.location.href = '/login'
        return
      }
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as Record<string, string>))
        throw new Error(body.error || `请求失败 (${res.status})`)
      }

      await consumeChatStream(res, { wasNewConversation })
    } catch (e: unknown) {
      if ((e as Error).name !== 'AbortError') {
        error.value = (e as Error).message || '请求失败'
        const msgs = messages.value
        const last = msgs[msgs.length - 1]
        if (last && last.role === 'assistant' && last.content === '') {
          messages.value = msgs.slice(0, -1)
        }
      }
    } finally {
      sending.value = false
      abortController = null
    }
  }

  async function resolveToolConfirmation(confirmed: boolean) {
    const pending = pendingToolConfirmation.value
    if (!pending || sending.value || currentConversationID.value === 0) return

    const snapshot = { ...pending }
    pendingToolConfirmation.value = null

    sending.value = true
    error.value = ''

    abortController = new AbortController()
    const token = getToken()

    try {
      const res = await fetch(`${BASE_URL}/chat/send`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          conversation_id: currentConversationID.value,
          tool_confirmation: {
            confirmation_call_id: snapshot.confirmationCallId,
            confirmed,
          },
        }),
        signal: abortController.signal,
      })

      if (res.status === 401) {
        removeToken()
        window.location.href = '/login'
        return
      }
      if (!res.ok) {
        const body = await res.json().catch(() => ({} as Record<string, string>))
        throw new Error(body.error || `请求失败 (${res.status})`)
      }

      await consumeChatStream(res, { wasNewConversation: false })
    } catch (e: unknown) {
      if ((e as Error).name !== 'AbortError') {
        error.value = (e as Error).message || '请求失败'
        pendingToolConfirmation.value = snapshot
      }
    } finally {
      sending.value = false
      abortController = null
    }
  }

  function stopStreaming() {
    abortController?.abort()
  }

  return {
    conversations,
    currentConversationID,
    messages,
    sending,
    error,
    pendingToolConfirmation,
    loadConversations,
    loadMessages,
    switchConversation,
    newConversation,
    deleteConversation: removeConversation,
    send,
    resolveToolConfirmation,
    stopStreaming,
  }
})
