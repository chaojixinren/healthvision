<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { isOld, getUser } from '../services/auth'
import {
  listMedicines,
  listReminders,
  listBindings,
  createReminder,
  updateReminder,
  deleteReminder,
  type Medicine,
  type Reminder,
  type Binding,
} from '../services/api'

const medicines = ref<Medicine[]>([])
const reminders = ref<Reminder[]>([])
const bindings = ref<Binding[]>([])
const loading = ref(true)
const error = ref('')
const showForm = ref(false)
const editingId = ref<number | null>(null)
const elderly = ref(isOld())

const form = ref({
  medicine_id: 0,
  time: '08:00',
  target_user_id: 0,
})

// only children can create reminders
const canCreate = computed(() => !elderly.value)

// bound elders that the child can set reminders for
const targetUsers = computed(() => {
  const currentUser = getUser()
  if (!currentUser) return []
  if (elderly.value) return []
  return bindings.value
    .filter((b) => b.status === 'accepted')
    .map((b) => ({
      id: b.elder_id,
      name: b.elder?.name || '老人用户',
      email: b.elder?.email || '',
    }))
})

function medicineNotes(id: number): string {
  return medicines.value.find((m) => m.id === id)?.notes ?? ''
}

function targetUserName(id: number): string {
  return targetUsers.value.find((u) => u.id === id)?.name ?? ''
}

function creatorName(createdBy: number): string {
  const currentUser = getUser()
  if (createdBy === currentUser?.id) return currentUser.name
  // try to find in bindings
  const b = bindings.value.find((b) => b.child_id === createdBy)
  return b?.child?.name || '家人'
}

async function fetchData() {
  loading.value = true
  error.value = ''
  try {
    const [mRes, rRes, bRes] = await Promise.all([listMedicines(), listReminders(), listBindings()])
    medicines.value = mRes.data
    reminders.value = rRes.data
    bindings.value = bRes.bindings || []
  } catch (e: any) {
    error.value = e.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  const defaultTarget = targetUsers.value[0]?.id ?? 0
  form.value = {
    medicine_id: medicines.value[0]?.id ?? 0,
    time: '08:00',
    target_user_id: defaultTarget,
  }
  showForm.value = true
}

function openEdit(r: Reminder) {
  editingId.value = r.id
  form.value = { medicine_id: r.medicine_id, time: r.time, target_user_id: r.user_id }
  showForm.value = true
}

function closeForm() {
  showForm.value = false
}

async function submitForm() {
  error.value = ''
  try {
    if (editingId.value !== null) {
      await updateReminder(editingId.value, { time: form.value.time, enabled: true })
    } else {
      await createReminder({
        medicine_id: form.value.medicine_id,
        time: form.value.time,
        target_user_id: form.value.target_user_id || undefined,
      })
    }
    showForm.value = false
    await fetchData()
  } catch (e: any) {
    error.value = e.message || '保存失败'
  }
}

async function toggle(reminder: Reminder) {
  try {
    await updateReminder(reminder.id, { time: reminder.time, enabled: !reminder.enabled })
    reminder.enabled = !reminder.enabled
  } catch (e: any) {
    error.value = e.message || '操作失败'
  }
}

async function remove(id: number) {
  if (!confirm('确定要删除这个提醒吗？')) return
  try {
    await deleteReminder(id)
    reminders.value = reminders.value.filter(r => r.id !== id)
  } catch (e: any) {
    error.value = e.message || '删除失败'
  }
}

onMounted(fetchData)
</script>

<template>
  <div class="container reminders-page">
    <div class="header-row">
      <h1>用药提醒</h1>
      <button v-if="canCreate" class="btn-primary" @click="openCreate">+ 添加提醒</button>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="reminders.length === 0" class="empty">
      <template v-if="canCreate">暂无提醒，点击上方按钮添加</template>
      <template v-else>暂无提醒，等待子女设置</template>
    </div>

    <div v-else class="reminder-list">
      <div v-for="r in reminders" :key="r.id" class="card reminder-card">
        <div class="reminder-left">
          <div class="medicine-label">{{ r.medicine_name }}</div>
          <div v-if="medicineNotes(r.medicine_id)" class="medicine-notes">{{ medicineNotes(r.medicine_id) }}</div>
          <div v-if="canCreate && targetUserName(r.user_id)" class="target-label">
            为：{{ targetUserName(r.user_id) }}
          </div>
          <div v-if="elderly && r.created_by !== r.user_id" class="target-label created-by">
            由 {{ creatorName(r.created_by) }} 设置
          </div>
          <div class="time">{{ r.time }}</div>
        </div>
        <div class="reminder-right">
          <label class="toggle">
            <input type="checkbox" :checked="r.enabled" @change="toggle(r)" />
            <span class="toggle-label">提醒</span>
          </label>
          <button class="btn-outline btn-sm" @click="openEdit(r)">编辑</button>
          <button class="btn-outline btn-sm danger" @click="remove(r.id)">删除</button>
        </div>
      </div>
    </div>

    <!-- Form Modal -->
    <div v-if="showForm" class="modal-overlay" @click.self="closeForm">
      <div class="card-lg modal">
        <h2>{{ editingId !== null ? '编辑提醒' : '添加提醒' }}</h2>
        <form @submit.prevent="submitForm">
          <div v-if="canCreate && targetUsers.length > 0" class="field">
            <label>为谁设置</label>
            <select v-model="form.target_user_id">
              <option :value="0">自己</option>
              <option v-for="u in targetUsers" :key="u.id" :value="u.id">
                {{ u.name }}（{{ u.email }}）
              </option>
            </select>
          </div>
          <div v-if="editingId === null" class="field">
            <label>药品 *</label>
            <select v-model="form.medicine_id" required>
              <option v-for="m in medicines" :key="m.id" :value="m.id">{{ m.name }}</option>
            </select>
          </div>
          <div class="field">
            <label>时间 *</label>
            <input type="time" v-model="form.time" required />
          </div>
          <div class="form-actions">
            <button type="button" class="btn-outline" @click="closeForm">取消</button>
            <button type="submit" class="btn-primary">保存</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.reminders-page {
  padding: 2rem 1.5rem;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.header-row h1 {
  font-size: 1.875rem;
}

.error-banner {
  background: #fce8ec;
  color: #c4536a;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-card);
  margin-bottom: 1rem;
  font-size: 0.875rem;
}

.loading,
.empty {
  text-align: center;
  padding: 4rem 0;
  color: #8a7b70;
}

.reminder-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.reminder-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
}

.reminder-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.medicine-label {
  font-size: 0.875rem;
  color: #6b5b50;
  background: var(--muted);
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
}

.medicine-notes {
  font-size: 0.75rem;
  color: #8b2252;
  max-width: 12rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.target-label {
  font-size: 0.6875rem;
  color: #1565c0;
  background: #e3f2fd;
  padding: 0.125rem 0.5rem;
  border-radius: 999px;
}

.target-label.created-by {
  color: #e65100;
  background: #fff3e0;
}

.time {
  font-family: var(--font-serif);
  font-size: 1.5rem;
  font-weight: 600;
}

.reminder-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.toggle {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  cursor: pointer;
}

.toggle input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--primary);
}

.toggle-label {
  font-size: 0.75rem;
  color: #8a7b70;
  user-select: none;
}

.actions .btn-sm {
  padding: 0.4rem 1rem;
  font-size: 0.75rem;
}

.danger {
  border-color: #c4536a;
  color: #c4536a;
}

.danger:hover {
  background: #fce8ec;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(45, 27, 20, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  padding: 1rem;
}

.modal {
  width: 100%;
  max-width: 24rem;
  max-height: 90vh;
  overflow-y: auto;
}

.modal h2 {
  font-size: 1.25rem;
  margin-bottom: 1.25rem;
}

.field {
  margin-bottom: 1rem;
}

.form-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}
</style>
