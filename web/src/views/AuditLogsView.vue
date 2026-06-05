<template>
  <div>
    <div class="page-header">
      <h2>操作审计</h2>
      <el-button :icon="Refresh" @click="loadLogs">刷新</el-button>
    </div>

    <el-card shadow="never">
      <div class="filters">
        <el-input v-model="filters.username" placeholder="用户名" clearable style="width: 160px" @clear="loadLogs" />
        <el-input v-model="filters.action" placeholder="操作类型" clearable style="width: 180px" @clear="loadLogs" />
        <el-button type="primary" @click="loadLogs">查询</el-button>
      </div>

      <el-table v-loading="loading" :data="logs" stripe style="margin-top: 12px">
        <el-table-column prop="created_at" label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="username" label="用户" width="110" />
        <el-table-column prop="role" label="角色" width="90">
          <template #default="{ row }">{{ roleLabel(row.role) }}</template>
        </el-table-column>
        <el-table-column prop="action" label="操作" min-width="160" show-overflow-tooltip />
        <el-table-column prop="method" label="方法" width="80" />
        <el-table-column prop="path" label="路径" min-width="220" show-overflow-tooltip />
        <el-table-column prop="client_ip" label="IP" width="120" />
        <el-table-column prop="status_code" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.status_code }}</el-tag>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadLogs"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { listAuditLogs, type AuditLog } from '@/api/audit'
import { roleLabel } from '@/utils/auth'

const loading = ref(false)
const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const filters = reactive({ username: '', action: '' })

function formatTime(ts: string) {
  return new Date(ts).toLocaleString()
}

async function loadLogs() {
  loading.value = true
  try {
    const result = await listAuditLogs({
      username: filters.username || undefined,
      action: filters.action || undefined,
      limit: pageSize.value,
      offset: (page.value - 1) * pageSize.value,
    })
    logs.value = result.items
    total.value = result.total
  } finally {
    loading.value = false
  }
}

onMounted(loadLogs)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filters {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
