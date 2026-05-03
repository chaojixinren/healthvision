<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useChatStore } from '../stores/chat'
import type { Conversation } from '../stores/chat'

marked.setOptions({ gfm: true, breaks: true })

function renderMarkdown(text: string): string {
  return DOMPurify.sanitize(marked.parse(text, { async: false }) as string)
}

const store = useChatStore()

const input = ref('')
const images = ref<string[]>([])
const messagesEl = ref<HTMLElement>()
const inputEl = ref<HTMLTextAreaElement>()
const fileInputEl = ref<HTMLInputElement>()
const loading = ref(true)
const sidebarOpen = ref(false)
const isLastMessageAssistant = computed(() => {
  const msgs = store.messages
  return msgs.length > 0 && msgs[msgs.length - 1].role === 'assistant'
})

const resolvedImages = computed(() =>
  store.messages.map((m) => (m.images ? parseImagesFromJSON(m.images) : ([] as string[]))),
)

onMounted(async () => {
  try {
    await store.loadConversations()
    if (store.conversations.length === 0) {
      store.newConversation()
    } else {
      store.switchConversation(store.conversations[0].id)
    }
  } catch (e: unknown) {
    store.error = (e as Error).message || '加载失败'
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  store.stopStreaming()
  document.body.style.overflow = ''
})

watch(
  () => {
    const msgs = store.messages
    return msgs.length > 0 ? msgs[msgs.length - 1].content.length : 0
  },
  () => nextTick(scrollToBottom),
)

function scrollToBottom() {
  if (messagesEl.value) {
    messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  }
}

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function closeSidebar() {
  sidebarOpen.value = false
}

watch(sidebarOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
})

function handleSelect(conv: Conversation) {
  store.switchConversation(conv.id)
  closeSidebar()
  nextTick(() => inputEl.value?.focus())
}

function handleNew() {
  store.newConversation()
  closeSidebar()
  nextTick(() => inputEl.value?.focus())
}

async function handleDelete(conv: Conversation) {
  if (!confirm('确定要删除这个会话吗？')) return
  try {
    await store.deleteConversation(conv.id)
  } catch (e: unknown) {
    store.error = (e as Error).message || '删除失败'
  }
}

async function handleSend() {
  const text = input.value.trim()
  if (!text && images.value.length === 0) return
  input.value = ''
  const imgs = images.value.length > 0 ? [...images.value] : undefined
  images.value = []
  await store.send(text || '请分析这张图片', imgs)
  nextTick(() => inputEl.value?.focus())
}

// --- Image handling ---

function compressImage(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const img = new Image()
      img.onload = () => {
        let { width, height } = img
        const maxDim = 2000
        if (width > maxDim || height > maxDim) {
          if (width > height) {
            height = Math.round(height * maxDim / width)
            width = maxDim
          } else {
            width = Math.round(width * maxDim / height)
            height = maxDim
          }
        }
        const canvas = document.createElement('canvas')
        canvas.width = width
        canvas.height = height
        const ctx = canvas.getContext('2d')!
        ctx.drawImage(img, 0, 0, width, height)
        resolve(canvas.toDataURL('image/jpeg', 0.8))
      }
      img.onerror = () => reject(new Error('Failed to load image'))
      img.src = reader.result as string
    }
    reader.onerror = () => reject(new Error('Failed to read file'))
    reader.readAsDataURL(file)
  })
}

function parseImagesFromJSON(json: string): string[] {
  try {
    return JSON.parse(json) as string[]
  } catch {
    return []
  }
}

async function handleImageInput(files: FileList | null) {
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (!file.type.startsWith('image/')) continue
    try {
      const dataURL = await compressImage(file)
      images.value = [...images.value, dataURL]
    } catch {
      // skip unreadable files
    }
  }
}

function removeImage(index: number) {
  images.value = images.value.filter((_, i) => i !== index)
}

function triggerFileInput() {
  fileInputEl.value?.click()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && e.ctrlKey) {
    e.preventDefault()
    handleSend()
    return
  }
  if (e.key === 'Enter' && !e.shiftKey && input.value.trim()) {
    e.preventDefault()
    handleSend()
  }
}

function autoResize() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}
</script>

<template>
  <div class="chat-page">
    <!-- Background glows -->
    <div class="bg-glow glow-1"></div>
    <div class="bg-glow glow-2"></div>

    <div class="chat-layout">
      <!-- Sidebar -->
      <aside class="chat-sidebar" :class="{ open: sidebarOpen }">
        <div class="sidebar-glass">
          <div class="sidebar-header">
            <div class="sidebar-header-row">
              <h2 class="sidebar-title">AI 助手</h2>
              <button class="sidebar-close-btn" @click="closeSidebar" aria-label="关闭菜单">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
            <button class="new-chat-btn" @click="handleNew">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              <span>新对话</span>
            </button>
          </div>

          <div class="sidebar-divider"></div>

          <div class="sidebar-section-label">历史对话</div>

          <div class="sidebar-list">
            <div
              v-for="c in store.conversations"
              :key="c.id"
              class="history-item"
              :class="{ active: c.id === store.currentConversationID }"
              @click="handleSelect(c)"
            >
              <svg class="history-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
              </svg>
              <span class="history-title">{{ c.title }}</span>
              <button
                class="history-delete"
                @click.stop="handleDelete(c)"
                title="删除会话"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>

            <div v-if="loading" class="sidebar-empty">
              <div class="sidebar-empty-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" opacity="0.25">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                </svg>
              </div>
              <span>加载中...</span>
            </div>
            <div v-else-if="store.conversations.length === 0" class="sidebar-empty">
              <div class="sidebar-empty-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" opacity="0.25">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                </svg>
              </div>
              <span>暂无对话</span>
            </div>
          </div>
        </div>
      </aside>

      <!-- Main chat -->
      <div class="chat-main" @click="closeSidebar">
        <div class="chat-container">
          <!-- Menu toggle (mobile) -->
          <button class="menu-toggle" @click.stop="toggleSidebar" aria-label="打开菜单">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <line x1="3" y1="6" x2="21" y2="6"></line>
              <line x1="3" y1="12" x2="21" y2="12"></line>
              <line x1="3" y1="18" x2="21" y2="18"></line>
            </svg>
          </button>

          <!-- Welcome -->
          <div v-if="!loading && store.messages.length === 0 && !store.error" class="welcome-view">
            <div class="welcome-icon">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                <path d="M8 9h8" stroke-width="1.5"></path>
                <path d="M8 13h5" stroke-width="1.5"></path>
              </svg>
            </div>
            <h2 class="welcome-heading">我是你的AI助手，需要我做些什么？</h2>
            <p class="welcome-sub">可以向我咨询健康问题、用药建议或病症分析</p>
          </div>

          <!-- Messages -->
          <div
            v-else
            ref="messagesEl"
            class="messages-area"
          >
            <div
              v-for="(msg, i) in store.messages"
              :key="i"
              class="message-row"
              :class="msg.role"
            >
              <div class="message-content">
                <div class="bubble" :class="{ 'bubble-md': msg.role === 'assistant' }">
                  <span v-if="msg.role === 'assistant' && !msg.content && store.sending && i === store.messages.length - 1" class="loading-dots">
                    <span class="dot"></span>
                    <span class="dot"></span>
                    <span class="dot"></span>
                  </span>

                  <template v-else-if="msg.role === 'user'">
                    <div v-if="msg.content" class="user-text">{{ msg.content }}</div>
                    <div v-if="msg.images" class="user-images">
                      <img
                        v-for="(img, imgIdx) in resolvedImages[i]"
                        :key="imgIdx"
                        :src="img"
                        class="user-image-thumb"
                        loading="lazy"
                      />
                    </div>
                  </template>
                  <div v-else class="chat-md" v-html="renderMarkdown(msg.content)"></div>
                </div>
              </div>

            </div>

            <!-- Standalone loading row -->
            <div v-if="store.sending && !isLastMessageAssistant" class="message-row assistant">
              <div class="message-content">
                <div class="bubble">
                  <span class="loading-dots">
                    <span class="dot"></span>
                    <span class="dot"></span>
                    <span class="dot"></span>
                  </span>
                </div>
              </div>
            </div>

            <div v-if="store.error" class="chat-error">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              <span>{{ store.error }}</span>
              <button class="chat-error-close" @click="store.error = ''">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
          </div>

          <!-- Input -->
          <div class="input-section">
            <!-- Image previews -->
            <div v-if="images.length > 0" class="image-previews">
              <div v-for="(img, idx) in images" :key="idx" class="image-preview-item">
                <img :src="img" class="image-preview-thumb" />
                <button class="image-preview-remove" @click="removeImage(idx)" title="移除图片">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </svg>
                </button>
              </div>
            </div>
            <div class="input-wrapper">
              <button
                class="image-upload-btn"
                :disabled="store.sending || loading"
                @click="triggerFileInput"
                title="上传图片"
              >
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                  <circle cx="8.5" cy="8.5" r="1.5"></circle>
                  <polyline points="21 15 16 10 5 21"></polyline>
                </svg>
              </button>
              <textarea
                ref="inputEl"
                v-model="input"
                class="chat-input"
                placeholder="发信息..."
                :disabled="store.sending || loading"
                rows="1"
                @keydown="handleKeydown"
                @input="autoResize"
              ></textarea>
              <button
                class="send-btn"
                :class="{ loading: store.sending }"
                :disabled="store.sending || (!input.trim() && images.length === 0) || loading"
                @click="handleSend"
                title="发送"
              >
                <svg v-if="!store.sending" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="22" y1="2" x2="11" y2="13"></line>
                  <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
                </svg>
                <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" class="spinner">
                  <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
                </svg>
              </button>
            </div>
            <input
              ref="fileInputEl"
              type="file"
              accept="image/*"
              multiple
              class="file-input-hidden"
              @change="handleImageInput(($event.target as HTMLInputElement).files)"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── Page & background glows ── */
.chat-page {
  position: relative;
  min-height: calc(100vh - 3.5rem);
  overflow: hidden;
}

.bg-glow {
  position: fixed;
  border-radius: 50%;
  filter: blur(120px);
  opacity: 0.12;
  pointer-events: none;
  z-index: 0;
}

.glow-1 {
  width: 500px;
  height: 500px;
  background: var(--glow-1);
  top: -150px;
  right: -100px;
}

.glow-2 {
  width: 400px;
  height: 400px;
  background: var(--glow-2);
  bottom: -100px;
  left: -80px;
}

/* ── Layout ── */
.chat-layout {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 24px;
  height: calc(100vh - 3.5rem);
  max-width: 1100px;
  margin: 0 auto;
  padding: 16px 24px;
}

/* ── Sidebar ── */
.chat-sidebar {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.sidebar-glass {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--glass-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 20px;
  overflow: hidden;
}

.sidebar-header {
  padding: 20px 20px 16px;
}

.sidebar-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.sidebar-title {
  font-family: var(--font-serif);
  font-size: 1.35rem;
  font-weight: 700;
  background: linear-gradient(135deg, var(--primary) 0%, var(--accent) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.sidebar-close-btn {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: none;
  border: 1px solid var(--border);
  color: var(--foreground);
  cursor: pointer;
  flex-shrink: 0;
}

.new-chat-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 16px;
  background: linear-gradient(135deg, var(--primary) 0%, var(--accent) 100%);
  color: #fff;
  border: none;
  border-radius: 12px;
  font-size: 0.9rem;
  font-weight: 600;
  font-family: var(--font-sans);
  cursor: pointer;
  transition: opacity 0.2s, transform 0.15s;
}

.new-chat-btn:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

.new-chat-btn:active {
  transform: translateY(0);
}

.sidebar-divider {
  height: 1px;
  background: var(--border);
  margin: 0 20px;
}

.sidebar-section-label {
  padding: 14px 20px 8px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.sidebar-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px 12px;
}

/* History items */
.history-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s;
  position: relative;
}

.history-item:hover {
  background: var(--btn-secondary-bg);
}

.history-item.active {
  background: var(--chat-history-active-bg);
  border-right: 3px solid var(--primary);
  border-radius: 10px 0 0 10px;
}

.history-icon {
  flex-shrink: 0;
  color: var(--text-secondary);
  opacity: 0.6;
}

.history-item.active .history-icon {
  color: var(--primary);
  opacity: 1;
}

.history-title {
  flex: 1;
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--foreground);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-delete {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  opacity: 0;
  transition: opacity 0.15s, background 0.15s;
}

.history-item:hover .history-delete {
  opacity: 0.5;
}

.history-delete:hover {
  opacity: 1 !important;
  background: rgba(0, 0, 0, 0.06);
}

/* Sidebar empty */
.sidebar-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  opacity: 0.6;
}

.sidebar-empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-list::-webkit-scrollbar {
  width: 4px;
}
.sidebar-list::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

/* ── Menu toggle (mobile) ── */
.menu-toggle {
  display: none;
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 10;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: var(--glass-bg);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--border);
  color: var(--foreground);
  cursor: pointer;
  transition: background 0.15s;
}

.menu-toggle:active {
  background: var(--btn-secondary-bg);
}

/* ── Main chat area ── */
.chat-main {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.chat-container {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--glass-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 20px;
  overflow: hidden;
}

/* ── Welcome view ── */
.welcome-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 48px 24px;
  gap: 12px;
}

.welcome-icon {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20px;
  background: var(--btn-secondary-bg);
  color: var(--primary);
  margin-bottom: 8px;
}

.welcome-heading {
  font-family: var(--font-sans);
  font-size: 1.35rem;
  font-weight: 600;
  color: var(--foreground);
  line-height: 1.5;
}

.welcome-sub {
  font-size: 0.95rem;
  color: var(--text-secondary);
  max-width: 360px;
  line-height: 1.6;
}

/* ── Messages area ── */
.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 24px 20px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Message row */
.message-row {
  display: flex;
  align-items: flex-start;
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
}

.message-row.user {
  justify-content: flex-end;
}

/* Message content wrapper */
.message-content {
  min-width: 0;
  max-width: 80%;
}

/* Bubbles */
.bubble {
  padding: 14px 18px;
  font-size: 0.9375rem;
  line-height: 1.65;
  word-break: break-word;
}

.message-row.user .bubble {
  background: var(--primary);
  color: #fff;
  border-radius: 16px 4px 16px 16px;
}

.message-row.assistant .bubble {
  background: var(--btn-secondary-bg);
  border: 1px solid var(--border);
  color: var(--foreground);
  border-radius: 4px 16px 16px 16px;
}

.bubble-md {
  white-space: normal;
}

/* Loading dots */
.loading-dots {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 0;
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-secondary);
  animation: bounce 1.4s infinite ease-in-out both;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }
.dot:nth-child(3) { animation-delay: 0s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

/* ── Error ── */
.chat-error {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
  background: #fef0f0;
  border: 1px solid #f5c6cb;
  color: #a94442;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 0.8125rem;
}

.chat-error-close {
  margin-left: auto;
  background: none;
  border: none;
  color: #a94442;
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  opacity: 0.6;
  transition: opacity 0.15s;
}

.chat-error-close:hover {
  opacity: 1;
}

/* ── Input section ── */
.input-section {
  padding: 16px 20px 20px;
  background: var(--glass-bg);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-top: 1px solid var(--border);
}

.input-wrapper {
  display: flex;
  align-items: flex-end;
  gap: 10px;
  max-width: 800px;
  margin: 0 auto;
  background: var(--btn-secondary-bg);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 10px 14px;
  transition: background 0.2s, box-shadow 0.2s;
}

.input-wrapper:focus-within {
  background: var(--card);
  box-shadow: var(--glass-shadow);
}

.chat-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  resize: none;
  max-height: 200px;
  min-height: 24px;
  font-family: var(--font-sans);
  font-size: 0.9375rem;
  line-height: 1.5;
  color: var(--foreground);
  padding: 0;
}

.chat-input::placeholder {
  color: #b8a99a;
}

.chat-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.send-btn {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--primary);
  color: #fff;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 0.2s, transform 0.15s;
}

.send-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: scale(1.04);
}

.send-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.send-btn.loading {
  background: var(--accent);
}

.spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}

/* ── Markdown ── */
.chat-md :deep(p) { margin: 0 0 0.5rem; }
.chat-md :deep(p:last-child) { margin-bottom: 0; }
.chat-md :deep(ul), .chat-md :deep(ol) { margin: 0 0 0.5rem; padding-left: 1.25rem; }
.chat-md :deep(li) { margin-bottom: 0.125rem; }
.chat-md :deep(strong) { font-weight: 600; }
.chat-md :deep(code) {
  background: rgba(45, 27, 20, 0.08);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.875em;
  font-family: var(--font-mono);
}
.chat-md :deep(pre) {
  background: rgba(45, 27, 20, 0.06);
  padding: 0.625rem 0.875rem;
  border-radius: 0.5rem;
  margin: 0.5rem 0;
  overflow-x: auto;
}
.chat-md :deep(pre code) { background: none; padding: 0; }
.chat-md :deep(blockquote) {
  border-left: 3px solid var(--primary);
  margin: 0.5rem 0;
  padding: 0.25rem 0.75rem;
  color: var(--text-secondary);
}
.chat-md :deep(a) { color: var(--primary); text-decoration: underline; }
.chat-md :deep(h1), .chat-md :deep(h2), .chat-md :deep(h3) {
  font-family: var(--font-sans);
  font-size: 1em;
  font-weight: 600;
  margin: 0.5rem 0 0.25rem;
}
.chat-md :deep(h1) { font-size: 1.15em; }
.chat-md :deep(table) { border-collapse: collapse; margin: 0.5rem 0; width: 100%; }
.chat-md :deep(th), .chat-md :deep(td) {
  border: 1px solid var(--border);
  padding: 0.375rem 0.625rem;
  text-align: left;
  font-size: 0.875em;
}
.chat-md :deep(th) { background: rgba(45, 27, 20, 0.04); }
.chat-md :deep(hr) { border: none; border-top: 1px solid var(--border); margin: 0.75rem 0; }

/* Messages scrollbar */
.messages-area::-webkit-scrollbar {
  width: 5px;
}
.messages-area::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}
.messages-area::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* ── Mobile ── */
@media (max-width: 860px) {
  .chat-layout {
    grid-template-columns: 1fr;
    padding: 0;
    gap: 0;
  }

  /* Sidebar becomes a fixed drawer */
  .chat-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 300px;
    max-width: 85vw;
    z-index: 200;
    transform: translateX(-100%);
    transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    padding: 0;
    padding-top: env(safe-area-inset-top);
    padding-bottom: env(safe-area-inset-bottom);
    will-change: transform;
  }

  .chat-sidebar.open {
    transform: translateX(0);
  }

  .sidebar-glass {
    border-radius: 0;
    border: none;
    border-right: 1px solid var(--border);
  }

  .sidebar-header {
    padding-top: 12px;
  }

  .sidebar-close-btn {
    display: flex;
  }

  .menu-toggle {
    display: flex;
  }

  .chat-container {
    border-radius: 0;
    border: none;
  }

  .messages-area {
    padding: 56px 12px 16px;
    gap: 16px;
  }

  .message-row {
    max-width: 100%;
  }

  .input-section {
    padding: 12px;
    padding-bottom: calc(12px + env(safe-area-inset-bottom));
  }
}

/* ── Image previews (input area) ── */
.image-previews {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-width: 800px;
  margin: 0 auto 10px;
}

.image-preview-item {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.image-preview-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-preview-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  transition: background 0.15s;
}

.image-preview-remove:hover {
  background: rgba(0, 0, 0, 0.8);
}

/* ── Upload button ── */
.image-upload-btn {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: none;
  border: 1px solid var(--border);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.image-upload-btn:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary);
}

.image-upload-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.file-input-hidden {
  display: none;
}

/* ── User message images ── */
.user-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.user-image-thumb {
  max-width: 200px;
  max-height: 160px;
  border-radius: 8px;
  object-fit: cover;
  cursor: pointer;
  transition: opacity 0.15s;
}

.user-image-thumb:hover {
  opacity: 0.85;
}

.user-text {
  white-space: pre-wrap;
}

/* ── Mobile tweaks ── */
@media (max-width: 860px) {
  .image-previews {
    padding: 0 4px;
  }
}
</style>
