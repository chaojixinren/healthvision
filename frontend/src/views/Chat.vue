<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { getMe, createSession, streamChat, type User } from '../services/api'

interface Message {
  role: 'user' | 'model'
  content: string
}

const messages = ref<Message[]>([])
const input = ref('')
const sending = ref(false)
const streaming = ref(false)
const error = ref('')
const userId = ref('')
const sessionId = 'chat'

const messagesEl = ref<HTMLElement>()
const inputEl = ref<HTMLTextAreaElement>()

let abortController: AbortController | null = null

onMounted(async () => {
  try {
    const user: User = await getMe()
    userId.value = String(user.id)
    await createSession(userId.value, sessionId)
  } catch (e: unknown) {
    error.value = (e as Error).message || '创建会话失败，请刷新页面重试'
  }
})

onUnmounted(() => {
  abortController?.abort()
})

async function scrollToBottom() {
  await nextTick()
  if (messagesEl.value) {
    messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  }
}

async function send() {
  const text = input.value.trim()
  if (!text || sending.value || !userId.value) return

  messages.value.push({ role: 'user', content: text })
  input.value = ''
  messages.value.push({ role: 'model', content: '' })

  sending.value = true
  streaming.value = true
  error.value = ''

  await scrollToBottom()

  abortController = streamChat(userId.value, sessionId, text, {
    onToken(token: string) {
      const last = messages.value[messages.value.length - 1]
      if (last && last.role === 'model') {
        last.content += token
      }
      scrollToBottom()
    },
    onComplete() {
      sending.value = false
      streaming.value = false
      abortController = null
      inputEl.value?.focus()
    },
    onError(msg: string) {
      error.value = msg
      sending.value = false
      streaming.value = false
      abortController = null
      const last = messages.value[messages.value.length - 1]
      if (last && last.role === 'model' && last.content === '') {
        messages.value.pop()
      }
    },
  })
}
</script>

<template>
  <div class="chat-view">
    <div ref="messagesEl" class="chat-messages">
      <div v-if="messages.length === 0 && !error" class="chat-empty">
        <div class="chat-empty-icon">+</div>
        <h2>健康助手</h2>
        <p>有什么健康问题需要咨询？</p>
      </div>

      <div
        v-for="(msg, i) in messages"
        :key="i"
        class="chat-msg"
        :class="msg.role"
      >
        <div class="chat-bubble">
          {{ msg.content }}
          <span
            v-if="msg.role === 'model' && streaming && i === messages.length - 1"
            class="chat-cursor"
          ></span>
        </div>
      </div>

      <div v-if="error" class="chat-error">
        <span>{{ error }}</span>
        <button class="chat-error-close" @click="error = ''">&times;</button>
      </div>
    </div>

    <div class="chat-input-area">
      <textarea
        ref="inputEl"
        v-model="input"
        class="chat-input"
        placeholder="输入你的问题..."
        :disabled="sending"
        rows="1"
        @keydown.enter.exact.prevent="send"
      ></textarea>
      <button
        class="btn-primary chat-send-btn"
        :disabled="sending || !input.trim()"
        @click="send"
      >
        {{ sending ? '等待中' : '发送' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-view {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 3.5rem);
  max-width: var(--container-max);
  margin: 0 auto;
  padding: 0 1rem;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: #8a7b70;
  gap: 0.5rem;
}

.chat-empty-icon {
  font-size: 2.5rem;
  color: var(--primary);
  opacity: 0.6;
  margin-bottom: 0.5rem;
}

.chat-empty h2 {
  font-size: 1.5rem;
  color: var(--foreground);
}

.chat-empty p {
  font-size: 0.9375rem;
}

.chat-msg {
  display: flex;
  max-width: 80%;
}

.chat-msg.user {
  align-self: flex-end;
}

.chat-msg.model {
  align-self: flex-start;
}

.chat-bubble {
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  font-size: 0.9375rem;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.chat-msg.user .chat-bubble {
  background: var(--primary);
  color: #ffffff;
  border-bottom-right-radius: 0.25rem;
}

.chat-msg.model .chat-bubble {
  background: var(--muted);
  color: var(--foreground);
  border-bottom-left-radius: 0.25rem;
}

.chat-cursor {
  display: inline-block;
  width: 2px;
  height: 1.1em;
  background: var(--foreground);
  margin-left: 1px;
  vertical-align: text-bottom;
  animation: chat-blink 0.8s step-end infinite;
}

@keyframes chat-blink {
  0%, 100% { opacity: 1; }
  50%      { opacity: 0; }
}

.chat-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fef0f0;
  border: 1px solid #f5c6cb;
  color: #a94442;
  padding: 0.625rem 1rem;
  border-radius: var(--radius-card);
  font-size: 0.8125rem;
}

.chat-error-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  color: #a94442;
  padding: 0 0.25rem;
  line-height: 1;
}

.chat-input-area {
  display: flex;
  gap: 0.75rem;
  padding: 1rem 0.5rem;
  border-top: 1px solid var(--border);
  background: var(--background);
  position: sticky;
  bottom: 0;
}

.chat-input {
  flex: 1;
  resize: none;
  max-height: 120px;
  padding: 0.625rem 0.875rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  background: var(--card);
  font-family: var(--font-sans);
  font-size: 0.9375rem;
  color: var(--foreground);
  outline: none;
  transition: border-color 0.2s;
}

.chat-input:focus {
  border-color: var(--primary);
}

.chat-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.chat-send-btn {
  flex-shrink: 0;
  padding: 0.625rem 1.5rem;
}

.chat-messages::-webkit-scrollbar {
  width: 4px;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}
</style>
