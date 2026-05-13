<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import type { Medicine } from '../services/api'
import { useCareStore } from '../stores/care'

const care = useCareStore()
const medicines = computed(() => care.medicines)
const loading = computed(() => care.loading)
const error = computed({
  get: () => care.error,
  set: (value: string) => { care.error = value },
})
const showForm = ref(false)
const editingId = ref<number | null>(null)

const form = ref({
  name: '',
  image_url: '',
  description: '',
  notes: '',
})

async function fetchMedicines() {
  try {
    await care.loadMedicines()
  } catch (e: any) {
    error.value = e.message || '加载失败'
  }
}

function openCreate() {
  editingId.value = null
  form.value = { name: '', image_url: '', description: '', notes: '' }
  showForm.value = true
}

function openEdit(m: Medicine) {
  editingId.value = m.id
  form.value = {
    name: m.name,
    image_url: m.image_url,
    description: m.description,
    notes: m.notes,
  }
  showForm.value = true
}

function closeForm() {
  showForm.value = false
}

async function submitForm() {
  error.value = ''
  try {
    if (editingId.value !== null) {
      await care.updateMedicine(editingId.value, form.value)
    } else {
      await care.createMedicine(form.value)
    }
    showForm.value = false
  } catch (e: any) {
    error.value = e.message || '保存失败'
  }
}

async function remove(id: number) {
  if (!confirm('确定要删除这个药品吗？')) return
  try {
    await care.deleteMedicine(id)
  } catch (e: any) {
    error.value = e.message || '删除失败'
  }
}

onMounted(fetchMedicines)
</script>

<template>
  <div class="container medicines">
    <div class="header-row">
      <h1>我的药品</h1>
      <button class="btn-primary" @click="openCreate">+ 添加药品</button>
    </div>

    <div v-if="error" class="error-banner">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="medicines.length === 0" class="empty">
      暂无药品，点击上方按钮添加
    </div>

    <div v-else class="medicine-grid">
      <div v-for="m in medicines" :key="m.id" class="card medicine-card">
        <div class="medicine-image" v-if="m.image_url">
          <img :src="m.image_url" :alt="m.name" />
        </div>
        <div class="medicine-body">
          <h3>{{ m.name }}</h3>
          <p v-if="m.description" class="desc">{{ m.description }}</p>
          <p v-if="m.notes" class="notes">
            <span class="label">备注：</span>{{ m.notes }}
          </p>
          <div class="actions">
            <button class="btn-outline btn-sm" @click="openEdit(m)">编辑</button>
            <button class="btn-outline btn-sm danger" @click="remove(m.id)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Form Modal -->
    <div v-if="showForm" class="modal-overlay" @click.self="closeForm">
      <div class="card-lg modal">
        <h2>{{ editingId !== null ? '编辑药品' : '添加药品' }}</h2>
        <form @submit.prevent="submitForm">
          <div class="field">
            <label>药品名称 *</label>
            <input v-model="form.name" required maxlength="100" placeholder="例如：阿莫西林胶囊" />
          </div>
          <div class="field">
            <label>图片地址</label>
            <input v-model="form.image_url" type="url" maxlength="500" placeholder="https://..." />
          </div>
          <div class="field">
            <label>药品说明</label>
            <textarea v-model="form.description" rows="3" maxlength="2000" placeholder="药品用途、成分等说明" />
          </div>
          <div class="field">
            <label>服用备注</label>
            <textarea v-model="form.notes" rows="2" maxlength="1000" placeholder="饭后服用、每日两次等" />
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
.medicines {
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

.medicine-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.medicine-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.medicine-image {
  width: 100%;
  height: 160px;
  background: var(--muted);
  overflow: hidden;
}

.medicine-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.medicine-body {
  padding: 1rem;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.medicine-body h3 {
  font-size: 1.125rem;
  margin-bottom: 0.5rem;
}

.desc {
  font-size: 0.875rem;
  color: #6b5b50;
  margin-bottom: 0.5rem;
  line-height: 1.5;
}

.notes {
  font-size: 0.8125rem;
  color: #8b2252;
  margin-bottom: 1rem;
}

.notes .label {
  font-weight: 500;
}

.actions {
  margin-top: auto;
  display: flex;
  gap: 0.5rem;
}

.actions .btn-sm {
  padding: 0.4rem 1rem;
  font-size: 0.75rem;
}

.actions .danger {
  border-color: #c4536a;
  color: #c4536a;
}

.actions .danger:hover {
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
  max-width: 28rem;
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
