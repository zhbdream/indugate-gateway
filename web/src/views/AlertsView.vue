<template>
  <div>
    <div class="page-header">
      <h2>告警管理</h2>
      <el-button v-if="canWrite()" type="primary" @click="openRuleDialog()">新建规则</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="告警事件" name="events">
        <div style="margin-bottom: 12px; display: flex; gap: 8px">
          <el-select v-model="eventFilter" placeholder="状态" style="width: 120px" @change="loadEvents">
            <el-option label="全部" value="" />
            <el-option label="活跃" value="active" />
            <el-option label="已确认" value="resolved" />
          </el-select>
          <el-button :icon="Refresh" @click="loadEvents">刷新</el-button>
        </div>
        <el-table v-loading="eventsLoading" :data="events" stripe>
          <el-table-column prop="level" label="级别" width="90">
            <template #default="{ row }">
              <el-tag :type="levelType(row.level)" size="small">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="device_id" label="设备" width="80" />
          <el-table-column prop="node_id" label="Node ID" min-width="140" show-overflow-tooltip />
          <el-table-column prop="message" label="消息" min-width="200" show-overflow-tooltip />
          <el-table-column prop="value" label="值" width="100" />
          <el-table-column prop="status" label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'danger' : 'success'" size="small">
                {{ row.status === 'active' ? '活跃' : '已确认' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="triggered_at" label="触发时间" width="170">
            <template #default="{ row }">{{ formatTime(row.triggered_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="canWrite() && row.status === 'active'"
                link
                type="primary"
                @click="ackEvent(row.id)"
              >确认</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="告警规则" name="rules">
        <el-table v-loading="rulesLoading" :data="rules" stripe>
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column prop="device_id" label="设备" width="80" />
          <el-table-column prop="node_id" label="Node ID" min-width="140" show-overflow-tooltip />
          <el-table-column prop="condition" label="条件" width="80" />
          <el-table-column label="阈值" width="120">
            <template #default="{ row }">
              {{ row.threshold }}{{ row.condition === 'range' ? ` ~ ${row.threshold_max}` : '' }}
            </template>
          </el-table-column>
          <el-table-column prop="level" label="级别" width="90" />
          <el-table-column prop="enabled" label="启用" width="70">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '是' : '否' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column v-if="canWrite()" label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openRuleDialog(row)">编辑</el-button>
              <el-button link type="danger" @click="removeRule(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="ruleVisible" :title="editingRule ? '编辑规则' : '新建规则'" width="520px">
      <el-form :model="ruleForm" label-width="90px">
        <el-form-item label="设备" required>
          <el-select v-model="ruleForm.device_id" placeholder="选择设备" style="width: 100%">
            <el-option v-for="d in devices" :key="d.id" :label="d.name" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Node ID" required>
          <el-input v-model="ruleForm.node_id" placeholder="如 holding:0 或 MQTT topic" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        <el-form-item label="条件" required>
          <el-select v-model="ruleForm.condition" style="width: 100%">
            <el-option label="大于 (gt)" value="gt" />
            <el-option label="小于 (lt)" value="lt" />
            <el-option label="等于 (eq)" value="eq" />
            <el-option label="大于等于 (gte)" value="gte" />
            <el-option label="小于等于 (lte)" value="lte" />
            <el-option label="超出范围 (range)" value="range" />
            <el-option label="变化率 (change_rate)" value="change_rate" />
          </el-select>
        </el-form-item>
        <el-form-item label="阈值" required>
          <el-input-number v-model="ruleForm.threshold" :step="0.1" style="width: 100%" />
        </el-form-item>
        <el-form-item v-if="ruleForm.condition === 'range'" label="上限">
          <el-input-number v-model="ruleForm.threshold_max" :step="0.1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="级别">
          <el-select v-model="ruleForm.level" style="width: 100%">
            <el-option label="INFO" value="INFO" />
            <el-option label="WARN" value="WARN" />
            <el-option label="ERROR" value="ERROR" />
            <el-option label="CRITICAL" value="CRITICAL" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="ruleForm.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  acknowledgeAlertEvent,
  createAlertRule,
  deleteAlertRule,
  listAlertEvents,
  listAlertRules,
  updateAlertRule,
} from '@/api/alert'
import { listDevices } from '@/api/device'
import type { AlertEvent, AlertRule, Device } from '@/types'
import { canWrite } from '@/utils/auth'

const activeTab = ref('events')
const events = ref<AlertEvent[]>([])
const rules = ref<AlertRule[]>([])
const devices = ref<Device[]>([])
const eventsLoading = ref(false)
const rulesLoading = ref(false)
const eventFilter = ref('active')
const ruleVisible = ref(false)
const editingRule = ref<AlertRule | null>(null)
const saving = ref(false)

const ruleForm = reactive({
  device_id: 0,
  node_id: '',
  name: '',
  condition: 'gt' as AlertRule['condition'],
  threshold: 0,
  threshold_max: 100,
  level: 'WARN' as AlertRule['level'],
  enabled: true,
  description: '',
})

function levelType(level: string) {
  return ({ INFO: 'info', WARN: 'warning', ERROR: 'danger', CRITICAL: 'danger' } as Record<string, string>)[level] || 'info'
}

function formatTime(ts: string) {
  return new Date(ts).toLocaleString()
}

async function loadEvents() {
  eventsLoading.value = true
  try {
    events.value = await listAlertEvents({
      status: eventFilter.value || undefined,
      limit: 100,
    })
  } finally {
    eventsLoading.value = false
  }
}

async function loadRules() {
  rulesLoading.value = true
  try {
    rules.value = await listAlertRules()
  } finally {
    rulesLoading.value = false
  }
}

async function ackEvent(id: number) {
  await acknowledgeAlertEvent(id)
  ElMessage.success('已确认')
  await loadEvents()
}

function openRuleDialog(rule?: AlertRule) {
  editingRule.value = rule ?? null
  if (rule) {
    Object.assign(ruleForm, {
      device_id: rule.device_id,
      node_id: rule.node_id,
      name: rule.name,
      condition: rule.condition,
      threshold: rule.threshold,
      threshold_max: rule.threshold_max ?? 100,
      level: rule.level,
      enabled: rule.enabled,
      description: rule.description,
    })
  } else {
    Object.assign(ruleForm, {
      device_id: devices.value[0]?.id ?? 0,
      node_id: '',
      name: '',
      condition: 'gt',
      threshold: 0,
      threshold_max: 100,
      level: 'WARN',
      enabled: true,
      description: '',
    })
  }
  ruleVisible.value = true
}

async function saveRule() {
  saving.value = true
  try {
    if (editingRule.value) {
      await updateAlertRule(editingRule.value.id, { ...ruleForm })
    } else {
      await createAlertRule({ ...ruleForm })
    }
    ElMessage.success('保存成功')
    ruleVisible.value = false
    await loadRules()
  } finally {
    saving.value = false
  }
}

async function removeRule(id: number) {
  await ElMessageBox.confirm('确定删除该规则？', '提示', { type: 'warning' })
  await deleteAlertRule(id)
  ElMessage.success('已删除')
  await loadRules()
}

onMounted(async () => {
  devices.value = await listDevices()
  await Promise.all([loadEvents(), loadRules()])
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
</style>
