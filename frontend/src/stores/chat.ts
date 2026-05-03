import { ref } from 'vue'
import { defineStore } from 'pinia'
import { getToken, removeToken } from '../services/auth'
import { listConversations, getMessages, deleteConversation } from '../services/api'
import type { Conversation, ChatMessage } from '../services/api'

export type { Conversation, ChatMessage }

const BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const currentConversationID = ref(0)
  const messages = ref<ChatMessage[]>([])
  const sending = ref(false)
  const error = ref('')

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
    messages.value = []
    loadMessages(id).catch(e => {
      error.value = e.message
    })
  }

  function newConversation() {
    currentConversationID.value = 0
    messages.value = []
    error.value = ''
  }

  async function removeConversation(id: number) {
    await deleteConversation(id)
    conversations.value = conversations.value.filter(c => c.id !== id)

    if (currentConversationID.value === id) {
      if (conversations.value.length > 0) {
        switchConversation(conversations.value[0].id)
      } else {
        newConversation()
      }
    }
  }

  async function send(text: string, images?: string[]) {
    if ((!text.trim() && (!images || images.length === 0)) || sending.value) return

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
            try {
              const parsed = JSON.parse(data) as { error?: string }
              throw new Error(parsed.error || '流式响应错误')
            } catch (e) {
              if (e instanceof Error && e.message !== '流式响应错误') throw e
              throw new Error('流式响应错误')
            }
          }
          if (!data) continue

          try {
            const parsed = JSON.parse(data) as {
              token?: string
              conversation_id?: number
              partial?: boolean
              done?: boolean
            }
            if (parsed.token) {
              const msgs = messages.value
              const last = msgs[msgs.length - 1]
              if (last && last.role === 'assistant') {
                last.content += parsed.token
              }
            }
            if (parsed.done && parsed.conversation_id) {
              if (wasNewConversation) {
                currentConversationID.value = parsed.conversation_id
                loadConversations().catch(() => {})
              }
            }
          } catch {
            /* skip unparseable frames */
          }
        }
      }
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

  function stopStreaming() {
    abortController?.abort()
  }

  return {
    conversations,
    currentConversationID,
    messages,
    sending,
    error,
    loadConversations,
    loadMessages,
    switchConversation,
    newConversation,
    deleteConversation: removeConversation,
    send,
    stopStreaming,
  }
})
