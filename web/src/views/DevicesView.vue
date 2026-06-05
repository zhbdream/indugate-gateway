<template>
  <div>
    <div class="page-header">
      <h2>设备管理</h2>
      <div>
        <el-button :icon="Refresh" @click="loadDevices">刷新</el-button>
        <el-button v-if="canWrite()" type="primary" :icon="Plus" @click="openCreate">添加设备</el-button>
      </div>
    </div>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="devices" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="protocol" label="协议" width="110">
          <template #default="{ row }">
            <el-tag size="small">{{ protocolLabel(row.protocol) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="address" label="地址" min-width="180" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="goDetail(row.id)">数据</el-button>
            <template v-if="canWrite()">
              <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
              <el-button
                v-if="row.status !== 'connected'"
                link type="success"
                :loading="actionId === row.id && actionType === 'connect'"
                @click="handleConnect(row)"
              >连接</el-button>
              <el-button
                v-else
                link type="warning"
                :loading="actionId === row.id && actionType === 'disconnect'"
                @click="handleDisconnect(row)"
              >断开</el-button>
              <el-popconfirm title="确定删除该设备？" @confirm="handleDelete(row.id)">
                <template #reference>
                  <el-button link type="danger">删除</el-button>
                </template>
              </el-popconfirm>
            </template>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <DeviceFormDialog ref="formDialogRef" @saved="loadDevices" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { connectDevice, deleteDevice, disconnectDevice, listDevices } from '@/api/device'
import DeviceFormDialog from '@/components/DeviceFormDialog.vue'
import type { Device } from '@/types'
import { canWrite } from '@/utils/auth'

const router = useRouter()
const loading = ref(false)
const devices = ref<Device[]>([])
const formDialogRef = ref<InstanceType<typeof DeviceFormDialog>>()
const actionId = ref<number | null>(null)
const actionType = ref<'connect' | 'disconnect' | null>(null)

function protocolLabel(p: string) {
  return ({ opcua: 'OPC UA', modbus: 'Modbus', mqtt: 'MQTT', s7: 'S7' } as Record<string, string>)[p] || p
}

function statusLabel(s: string) {
  return ({ connected: '已连接', disconnected: '未连接', error: '错误' } as Record<string, string>)[s] || s
}

function statusType(s: string) {
  return ({ connected: 'success', disconnected: 'info', error: 'danger' } as Record<string, string>)[s] || 'info'
}

async function loadDevices() {
  loading.value = true
  try {
    devices.value = await listDevices()
  } finally {
    loading.value = false
  }
}

function openCreate() {
  formDialogRef.value?.openCreate()
}

function openEdit(device: Device) {
  formDialogRef.value?.openEdit(device)
}

function goDetail(id: number) {
  router.push(`/devices/${id}`)
}

async function handleConnect(device: Device) {
  actionId.value = device.id
  actionType.value = 'connect'
  try {
    await connectDevice(device.id)
    ElMessage.success('设备已连接')
    await loadDevices()
  } finally {
    actionId.value = null
    actionType.value = null
  }
}

async function handleDisconnect(device: Device) {
  actionId.value = device.id
  actionType.value = 'disconnect'
  try {
    await disconnectDevice(device.id)
    ElMessage.success('设备已断开')
    await loadDevices()
  } finally {
    actionId.value = null
    actionType.value = null
  }
}

async function handleDelete(id: number) {
  await deleteDevice(id)
  ElMessage.success('设备已删除')
  await loadDevices()
}

onMounted(loadDevices)
</script>
