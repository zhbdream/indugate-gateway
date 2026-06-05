<template>
  <div>
    <div class="page-header">
      <h2>模拟器控制</h2>
      <el-button :icon="Refresh" @click="loadSimulators">刷新</el-button>
    </div>

    <el-row :gutter="16">
      <el-col v-for="sim in simulators" :key="sim.type" :span="8">
        <el-card shadow="hover" class="sim-card">
          <template #header>
            <div class="sim-header">
              <span>{{ simTitle(sim.type) }}</span>
              <el-tag :type="sim.status === 'running' ? 'success' : 'info'" size="small">
                {{ sim.status === 'running' ? '运行中' : '已停止' }}
              </el-tag>
            </div>
          </template>

          <p class="sim-desc">{{ sim.description }}</p>
          <p v-if="sim.endpoint" class="sim-endpoint">
            <el-icon><Link /></el-icon>
            <span class="mono">{{ sim.endpoint }}</span>
          </p>

          <div v-if="sim.nodes?.length" class="sim-meta">
            <div class="meta-label">数据点</div>
            <el-tag v-for="n in sim.nodes.slice(0, 5)" :key="n" size="small" style="margin: 2px">{{ n }}</el-tag>
          </div>
          <div v-if="sim.topics?.length" class="sim-meta">
            <div class="meta-label">Topics</div>
            <el-tag v-for="t in sim.topics" :key="t" size="small" type="warning" style="margin: 2px">{{ t }}</el-tag>
          </div>

          <div v-if="canWrite()" class="sim-actions">
            <el-button
              v-if="sim.status !== 'running'"
              type="primary"
              :loading="actionType === sim.type"
              @click="handleStart(sim.type)"
            >启动</el-button>
            <el-button
              v-else
              type="danger"
              :loading="actionType === sim.type"
              @click="handleStop(sim.type)"
            >停止</el-button>
            <el-button @click="openConfig(sim)">配置</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>快速体验指南</template>
      <el-steps :active="3" align-center>
        <el-step title="启动模拟器" description="启动 OPC UA / Modbus / MQTT 模拟器" />
        <el-step title="添加设备" description="在设备管理中添加对应协议的设备" />
        <el-step title="连接设备" description="点击连接按钮建立通信" />
        <el-step title="查看数据" description="进入设备详情页浏览实时数据" />
      </el-steps>
    </el-card>

    <el-dialog v-model="configVisible" title="模拟器配置" width="520px">
      <el-form label-width="80px">
        <el-form-item label="类型">
          <el-tag>{{ configType }}</el-tag>
        </el-form-item>
        <el-form-item label="JSON">
          <el-input v-model="configJSON" type="textarea" :rows="10" placeholder='{"host":"0.0.0.0","port":502}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configVisible = false">取消</el-button>
        <el-button type="primary" :loading="configSaving" @click="saveConfig">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Link, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listSimulators, startSimulator, stopSimulator, updateSimulatorConfig } from '@/api/simulator'
import type { Simulator } from '@/types'
import { canWrite } from '@/utils/auth'

const simulators = ref<Simulator[]>([])
const actionType = ref<string | null>(null)
const configVisible = ref(false)
const configType = ref('')
const configJSON = ref('')
const configSaving = ref(false)

const defaultConfig: Record<string, object> = {
  opcua: { host: '0.0.0.0', port: 4840 },
  modbus: { host: '0.0.0.0', port: 502, holding_registers: { '0': 2500, '1': 200 } },
  mqtt: { host: '0.0.0.0', port: 1883, topics: ['factory/device1/telemetry', 'factory/device2/telemetry'] },
}

function openConfig(sim: Simulator) {
  configType.value = sim.type
  configJSON.value = JSON.stringify(defaultConfig[sim.type] || { host: '0.0.0.0' }, null, 2)
  configVisible.value = true
}

async function saveConfig() {
  configSaving.value = true
  try {
    JSON.parse(configJSON.value)
    await updateSimulatorConfig(configType.value, configJSON.value)
    ElMessage.success('配置已更新')
    configVisible.value = false
  } catch (e) {
    ElMessage.error('JSON 格式无效')
  } finally {
    configSaving.value = false
  }
}

function simTitle(type: string) {
  return ({ opcua: 'OPC UA 模拟器', modbus: 'Modbus TCP 模拟器', mqtt: 'MQTT 模拟器' } as Record<string, string>)[type] || type
}

async function loadSimulators() {
  simulators.value = await listSimulators()
}

async function handleStart(type: string) {
  actionType.value = type
  try {
    await startSimulator(type)
    ElMessage.success(`${simTitle(type)} 已启动`)
    await loadSimulators()
  } finally {
    actionType.value = null
  }
}

async function handleStop(type: string) {
  actionType.value = type
  try {
    await stopSimulator(type)
    ElMessage.success(`${simTitle(type)} 已停止`)
    await loadSimulators()
  } finally {
    actionType.value = null
  }
}

onMounted(loadSimulators)
</script>

<style scoped>
.sim-card {
  margin-bottom: 16px;
  min-height: 280px;
}

.sim-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sim-desc {
  color: #606266;
  font-size: 14px;
  min-height: 40px;
}

.sim-endpoint {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #409eff;
  font-size: 13px;
}

.sim-meta {
  margin: 12px 0;
}

.meta-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}

.sim-actions {
  margin-top: 16px;
}
</style>
