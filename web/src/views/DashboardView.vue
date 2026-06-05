<template>
  <div>
    <div class="page-header">
      <h2>仪表盘</h2>
      <el-button :icon="Refresh" :loading="loading" @click="load">刷新</el-button>
    </div>

    <el-row :gutter="16" class="card-block">
      <el-col :span="4" v-for="item in statCards" :key="item.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value" :style="{ color: item.color }">{{ item.value }}</div>
          <div class="stat-label">{{ item.label }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>设备状态分布</template>
          <div ref="deviceChartRef" style="height: 280px" />
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>最近告警</template>
          <el-table :data="recentAlerts" size="small" height="280">
            <el-table-column prop="level" label="级别" width="90">
              <template #default="{ row }">
                <el-tag :type="levelType(row.level)" size="small">{{ row.level }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="消息" min-width="180" show-overflow-tooltip />
            <el-table-column prop="triggered_at" label="时间" width="160">
              <template #default="{ row }">{{ formatTime(row.triggered_at) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getDashboardStats, listAlertEvents } from '@/api/alert'
import type { AlertEvent, DashboardStats } from '@/types'

const loading = ref(false)
const stats = ref<DashboardStats | null>(null)
const recentAlerts = ref<AlertEvent[]>([])
const deviceChartRef = ref<HTMLDivElement>()
let chart: echarts.ECharts | null = null

const statCards = computed(() => {
  const s = stats.value
  return [
    { label: '设备总数', value: s?.device_total ?? 0, color: '#409eff' },
    { label: '已连接', value: s?.device_connected ?? 0, color: '#67c23a' },
    { label: '异常设备', value: s?.device_error ?? 0, color: '#f56c6c' },
    { label: '活跃告警', value: s?.active_alerts ?? 0, color: '#e6a23c' },
    { label: '告警规则', value: s?.alert_rules ?? 0, color: '#909399' },
    { label: '24h 历史记录', value: s?.history_records_24h ?? 0, color: '#626aef' },
  ]
})

function levelType(level: string) {
  return ({ INFO: 'info', WARN: 'warning', ERROR: 'danger', CRITICAL: 'danger' } as Record<string, string>)[level] || 'info'
}

function formatTime(ts: string) {
  return new Date(ts).toLocaleString()
}

function renderChart() {
  if (!deviceChartRef.value || !stats.value) return
  if (!chart) chart = echarts.init(deviceChartRef.value)
  const s = stats.value
  const disconnected = Math.max(0, s.device_total - s.device_connected - s.device_error)
  chart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      data: [
        { name: '已连接', value: s.device_connected, itemStyle: { color: '#67c23a' } },
        { name: '未连接', value: disconnected, itemStyle: { color: '#909399' } },
        { name: '异常', value: s.device_error, itemStyle: { color: '#f56c6c' } },
      ],
    }],
  })
}

async function load() {
  loading.value = true
  try {
    stats.value = await getDashboardStats()
    recentAlerts.value = await listAlertEvents({ status: 'active', limit: 10 })
    renderChart()
  } finally {
    loading.value = false
  }
}

onMounted(load)
onUnmounted(() => chart?.dispose())
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.card-block {
  margin-bottom: 16px;
}

.stat-card {
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
}

.stat-label {
  color: #909399;
  margin-top: 4px;
  font-size: 13px;
}
</style>
