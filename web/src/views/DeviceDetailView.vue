<template>
  <div>
    <div class="page-header">
      <div style="display: flex; align-items: center; gap: 12px">
        <el-button :icon="ArrowLeft" @click="$router.push('/devices')">返回</el-button>
        <h2>{{ device?.name || '设备详情' }}</h2>
        <el-tag v-if="device" :type="statusType(device.status)" size="small">{{ statusLabel(device.status) }}</el-tag>
      </div>
      <div>
        <el-switch v-model="autoRefresh" active-text="自动刷新" inactive-text="手动" />
        <el-button :icon="Refresh" :loading="loading" @click="refreshAll">刷新</el-button>
        <el-button
          v-if="canWrite() && device?.status !== 'connected'"
          type="success"
          @click="handleConnect"
        >连接</el-button>
        <el-button v-else-if="canWrite()" type="warning" @click="handleDisconnect">断开</el-button>
      </div>
    </div>

    <el-row :gutter="16" class="card-block">
      <el-col :span="24">
        <el-card shadow="never">
          <template #header>设备信息</template>
          <el-descriptions v-if="device" :column="3" border size="small">
            <el-descriptions-item label="ID">{{ device.id }}</el-descriptions-item>
            <el-descriptions-item label="协议">{{ device.protocol }}</el-descriptions-item>
            <el-descriptions-item label="地址">{{ device.address }}</el-descriptions-item>
            <el-descriptions-item label="配置" :span="3">
              <span class="mono">{{ device.config || '-' }}</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :span="14">
        <el-card shadow="never">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>数据点列表</span>
              <el-button size="small" :loading="nodesLoading" @click="loadNodes">浏览节点</el-button>
            </div>
          </template>
          <el-table
            v-loading="nodesLoading"
            :data="nodes"
            height="420"
            highlight-current-row
            @current-change="onNodeSelect"
          >
            <el-table-column prop="browse_name" label="名称" min-width="120" />
            <el-table-column prop="node_id" label="Node ID" min-width="180" show-overflow-tooltip />
            <el-table-column prop="data_type" label="类型" width="90" />
            <el-table-column label="可写" width="70">
              <template #default="{ row }">
                <el-tag :type="row.writable ? 'success' : 'info'" size="small">{{ row.writable ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="当前值" min-width="120">
              <template #default="{ row }">
                <span class="mono">{{ formatValue(values[row.node_id]) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="10">
        <el-card shadow="never" class="card-block">
          <template #header>实时数据</template>
          <div v-if="selectedNode">
            <p><strong>Node ID:</strong> <span class="mono">{{ selectedNode.node_id }}</span></p>
            <el-statistic title="当前值" :value="displayValue" />
            <p style="color: #909399; font-size: 12px">更新时间: {{ lastUpdate || '-' }}</p>
            <div style="margin-top: 16px; display: flex; gap: 8px">
              <el-button type="primary" @click="readSelected">读取</el-button>
              <el-button v-if="canWrite() && selectedNode.writable" @click="openWrite">写入</el-button>
              <el-button v-if="canWrite()" type="success" @click="subscribeSelected">订阅</el-button>
            </div>
          </div>
          <el-empty v-else description="请选择一个数据点" />
        </el-card>

        <el-card shadow="never">
          <template #header>订阅事件</template>
          <el-table :data="events" height="220" size="small">
            <el-table-column prop="node_id" label="Node" min-width="120" show-overflow-tooltip />
            <el-table-column label="值" min-width="80">
              <template #default="{ row }">{{ formatValue(row.value) }}</template>
            </el-table-column>
            <el-table-column prop="timestamp" label="时间" width="160" />
          </el-table>
        </el-card>

        <el-card shadow="never" class="card-block">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>历史趋势</span>
              <div style="display: flex; gap: 8px">
                <el-button size="small" :disabled="!selectedNode" @click="loadHistory">加载</el-button>
                <el-button size="small" :disabled="!selectedNode" @click="exportCSV">导出 CSV</el-button>
              </div>
            </div>
          </template>
          <div ref="historyChartRef" style="height: 220px" />
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="writeVisible" title="写入数据" width="420px">
      <el-form label-width="80px">
        <el-form-item label="Node ID">
          <span class="mono">{{ selectedNode?.node_id }}</span>
        </el-form-item>
        <el-form-item label="值">
          <el-input v-model="writeValue" placeholder="输入要写入的值" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="writeVisible = false">取消</el-button>
        <el-button type="primary" :loading="writing" @click="submitWrite">写入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import { exportHistoryCSV, queryHistory } from '@/api/alert'
import {
  browseNodes,
  connectDevice,
  disconnectDevice,
  getDevice,
  pollSubscriptionEvents,
  readData,
  subscribeData,
  writeData,
} from '@/api/device'
import type { DataChangeEvent, Device, NodeInfo } from '@/types'
import { canWrite } from '@/utils/auth'

const route = useRoute()
const deviceId = computed(() => Number(route.params.id))

const device = ref<Device | null>(null)
const nodes = ref<NodeInfo[]>([])
const values = ref<Record<string, unknown>>({})
const selectedNode = ref<NodeInfo | null>(null)
const lastUpdate = ref('')
const loading = ref(false)
const nodesLoading = ref(false)
const autoRefresh = ref(true)
const writeVisible = ref(false)
const writeValue = ref('')
const writing = ref(false)
const events = ref<DataChangeEvent[]>([])
const subscriptionId = ref<string | null>(null)
const historyChartRef = ref<HTMLDivElement>()
let historyChart: echarts.ECharts | null = null

let refreshTimer: ReturnType<typeof setInterval> | null = null

const displayValue = computed(() => formatValue(selectedNode.value ? values.value[selectedNode.value.node_id] : undefined))

function formatValue(v: unknown) {
  if (v === undefined || v === null) return '-'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

function statusLabel(s: string) {
  return ({ connected: '已连接', disconnected: '未连接', error: '错误' } as Record<string, string>)[s] || s
}

function statusType(s: string) {
  return ({ connected: 'success', disconnected: 'info', error: 'danger' } as Record<string, string>)[s] || 'info'
}

async function loadDevice() {
  device.value = await getDevice(deviceId.value)
}

async function loadNodes() {
  if (device.value?.status !== 'connected') {
    ElMessage.warning('请先连接设备')
    return
  }
  nodesLoading.value = true
  try {
    nodes.value = await browseNodes(deviceId.value, { depth: 3 })
  } finally {
    nodesLoading.value = false
  }
}

async function readNode(nodeId: string) {
  const result = await readData(deviceId.value, nodeId)
  values.value[nodeId] = result.value
  if (selectedNode.value?.node_id === nodeId) {
    lastUpdate.value = new Date(result.timestamp).toLocaleString()
  }
}

async function readAllNodes() {
  if (!nodes.value.length || device.value?.status !== 'connected') return
  for (const node of nodes.value) {
    try {
      await readNode(node.node_id)
    } catch {
      // skip unreadable nodes during batch refresh
    }
  }
}

function onNodeSelect(row: NodeInfo | null) {
  selectedNode.value = row
  if (row) {
    readNode(row.node_id)
    loadHistory()
  }
}

function parseHistoryValue(raw: string): number | null {
  try {
    const v = JSON.parse(raw)
    if (typeof v === 'number') return v
    const n = Number(v)
    return Number.isNaN(n) ? null : n
  } catch {
    const n = Number(raw)
    return Number.isNaN(n) ? null : n
  }
}

async function loadHistory() {
  if (!selectedNode.value) return
  const rows = await queryHistory(deviceId.value, {
    node_id: selectedNode.value.node_id,
    limit: 100,
  })
  if (!historyChartRef.value) return
  if (!historyChart) historyChart = echarts.init(historyChartRef.value)

  const sorted = [...rows].reverse()
  const times = sorted.map(r => new Date(r.timestamp).toLocaleTimeString())
  const values = sorted.map(r => parseHistoryValue(r.value))

  historyChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 16, top: 24, bottom: 28 },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', scale: true },
    series: [{ type: 'line', smooth: true, data: values, areaStyle: { opacity: 0.15 } }],
  })
}

function exportCSV() {
  if (!selectedNode.value) return
  const url = exportHistoryCSV(deviceId.value, { node_id: selectedNode.value.node_id, limit: 1000 })
  window.open(url, '_blank')
}

async function readSelected() {
  if (!selectedNode.value) return
  await readNode(selectedNode.value.node_id)
  ElMessage.success('读取成功')
}

function openWrite() {
  writeValue.value = String(values.value[selectedNode.value?.node_id ?? ''] ?? '')
  writeVisible.value = true
}

async function submitWrite() {
  if (!selectedNode.value) return
  writing.value = true
  try {
    let value: unknown = writeValue.value
    if (writeValue.value === 'true') value = true
    else if (writeValue.value === 'false') value = false
    else if (!Number.isNaN(Number(writeValue.value)) && writeValue.value.trim() !== '') {
      value = Number(writeValue.value)
    }
    await writeData(deviceId.value, selectedNode.value.node_id, value)
    ElMessage.success('写入成功')
    writeVisible.value = false
    await readNode(selectedNode.value.node_id)
  } finally {
    writing.value = false
  }
}

async function subscribeSelected() {
  if (!selectedNode.value) return
  const sub = await subscribeData(deviceId.value, [selectedNode.value.node_id])
  subscriptionId.value = sub.id
  ElMessage.success(`订阅成功: ${sub.id}`)
}

async function pollEvents() {
  if (!subscriptionId.value) return
  try {
    const newEvents = await pollSubscriptionEvents(deviceId.value, subscriptionId.value, true)
    if (newEvents.length) {
      events.value = [...newEvents.reverse(), ...events.value].slice(0, 100)
      for (const e of newEvents) {
        values.value[e.node_id] = e.value
      }
    }
  } catch {
    // subscription may have ended
  }
}

async function refreshAll() {
  loading.value = true
  try {
    await loadDevice()
    if (device.value?.status === 'connected') {
      if (!nodes.value.length) await loadNodes()
      await readAllNodes()
      await pollEvents()
    }
  } finally {
    loading.value = false
  }
}

async function handleConnect() {
  await connectDevice(deviceId.value)
  ElMessage.success('设备已连接')
  await refreshAll()
}

async function handleDisconnect() {
  await disconnectDevice(deviceId.value)
  ElMessage.success('设备已断开')
  nodes.value = []
  values.value = {}
  subscriptionId.value = null
  await loadDevice()
}

function setupAutoRefresh() {
  if (refreshTimer) clearInterval(refreshTimer)
  if (autoRefresh.value) {
    refreshTimer = setInterval(() => {
      if (device.value?.status === 'connected') {
        readAllNodes()
        pollEvents()
      }
    }, 2000)
  }
}

watch(autoRefresh, setupAutoRefresh)

onMounted(async () => {
  await refreshAll()
  setupAutoRefresh()
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  historyChart?.dispose()
})
</script>

<style scoped>
.card-block {
  margin-bottom: 16px;
}
</style>
