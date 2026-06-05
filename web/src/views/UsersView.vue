<template>
  <div>
    <div class="page-header">
      <h2>用户管理</h2>
      <el-button type="primary" @click="openCreate">新建用户</el-button>
    </div>

    <el-table v-loading="loading" :data="users" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="username" label="用户名" min-width="120" />
      <el-table-column prop="role" label="角色" width="120">
        <template #default="{ row }">
          <el-tag size="small">{{ roleLabel(row.role) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="340" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">改角色</el-button>
          <el-button v-if="deviceACLEnabled && row.role !== 'admin'" link @click="openDevices(row)">设备权限</el-button>
          <el-button link @click="openPassword(row)">改密码</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" title="新建用户" width="420px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="用户名"><el-input v-model="createForm.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="createForm.password" type="password" show-password /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="操作员" value="operator" />
            <el-option label="只读" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="修改角色" width="360px">
      <el-select v-model="editRole" style="width: 100%">
        <el-option label="管理员" value="admin" />
        <el-option label="操作员" value="operator" />
        <el-option label="只读" value="viewer" />
      </el-select>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordVisible" title="修改密码" width="360px">
      <el-input v-model="newPassword" type="password" show-password placeholder="新密码" />
      <template #footer>
        <el-button @click="passwordVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitPassword">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="devicesVisible" title="设备权限" width="480px">
      <p class="hint">仅分配列表中的设备；未分配任何设备时无法访问设备数据。</p>
      <el-select v-model="selectedDeviceIDs" multiple filterable placeholder="选择可访问设备" style="width: 100%">
        <el-option v-for="d in allDevices" :key="d.id" :label="`${d.name} (#${d.id})`" :value="d.id" />
      </el-select>
      <template #footer>
        <el-button @click="devicesVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitDevices">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { changeUserPassword, createUser, deleteUser, getUserDevices, listUsers, setUserDevices, updateUser, type User } from '@/api/users'
import { getAuthConfig } from '@/api/auth'
import { listDevices } from '@/api/device'
import type { Device } from '@/types'

const loading = ref(false)
const saving = ref(false)
const users = ref<User[]>([])
const deviceACLEnabled = ref(false)
const allDevices = ref<Device[]>([])
const createVisible = ref(false)
const editVisible = ref(false)
const passwordVisible = ref(false)
const devicesVisible = ref(false)
const editingUser = ref<User | null>(null)
const editRole = ref<User['role']>('operator')
const newPassword = ref('')
const selectedDeviceIDs = ref<number[]>([])

const createForm = reactive({
  username: '',
  password: '',
  role: 'operator' as User['role'],
})

function roleLabel(role: string) {
  return ({ admin: '管理员', operator: '操作员', viewer: '只读' } as Record<string, string>)[role] || role
}

function formatTime(ts: string) {
  return new Date(ts).toLocaleString()
}

async function load() {
  loading.value = true
  try {
    users.value = await listUsers()
  } finally {
    loading.value = false
  }
}

async function loadConfig() {
  try {
    const cfg = await getAuthConfig()
    deviceACLEnabled.value = cfg.device_acl_enabled
  } catch {
    deviceACLEnabled.value = false
  }
}

async function openDevices(row: User) {
  editingUser.value = row
  allDevices.value = await listDevices()
  const result = await getUserDevices(row.id)
  selectedDeviceIDs.value = result.device_ids
  devicesVisible.value = true
}

async function submitDevices() {
  if (!editingUser.value) return
  saving.value = true
  try {
    await setUserDevices(editingUser.value.id, selectedDeviceIDs.value)
    ElMessage.success('设备权限已更新')
    devicesVisible.value = false
  } finally {
    saving.value = false
  }
}

function openCreate() {
  Object.assign(createForm, { username: '', password: '', role: 'operator' })
  createVisible.value = true
}

async function submitCreate() {
  saving.value = true
  try {
    await createUser({ ...createForm })
    ElMessage.success('用户已创建')
    createVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function openEdit(row: User) {
  editingUser.value = row
  editRole.value = row.role
  editVisible.value = true
}

async function submitEdit() {
  if (!editingUser.value) return
  saving.value = true
  try {
    await updateUser(editingUser.value.id, editRole.value)
    ElMessage.success('角色已更新')
    editVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function openPassword(row: User) {
  editingUser.value = row
  newPassword.value = ''
  passwordVisible.value = true
}

async function submitPassword() {
  if (!editingUser.value || !newPassword.value) return
  saving.value = true
  try {
    await changeUserPassword(editingUser.value.id, newPassword.value)
    ElMessage.success('密码已更新')
    passwordVisible.value = false
  } finally {
    saving.value = false
  }
}

async function remove(row: User) {
  await ElMessageBox.confirm(`确定删除用户 ${row.username}？`, '提示', { type: 'warning' })
  await deleteUser(row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(async () => {
  await loadConfig()
  await load()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.hint {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
}
</style>
