<template>
  <router-view v-if="isLoginPage" />
  <el-container v-else class="layout">
    <el-aside width="220px" class="aside">
      <div class="logo">
        <img src="/favicon.svg" alt="logo" width="28" height="28" />
        <span>InduGate</span>
      </div>
      <el-menu :default-active="activeMenu" router background-color="#001529" text-color="#ffffffa6" active-text-color="#fff">
        <el-menu-item index="/dashboard">
          <el-icon><DataBoard /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/devices">
          <el-icon><Monitor /></el-icon>
          <span>设备管理</span>
        </el-menu-item>
        <el-menu-item index="/alerts">
          <el-icon><Bell /></el-icon>
          <span>告警管理</span>
        </el-menu-item>
        <el-menu-item v-if="showAdminMenu" index="/users">
          <el-icon><User /></el-icon>
          <span>用户管理</span>
        </el-menu-item>
        <el-menu-item v-if="showAdminMenu" index="/audit">
          <el-icon><Document /></el-icon>
          <span>操作审计</span>
        </el-menu-item>
        <el-menu-item index="/simulators">
          <el-icon><Cpu /></el-icon>
          <span>模拟器</span>
        </el-menu-item>
      </el-menu>
      <div class="aside-footer">
        <el-tag size="small" type="success">v0.6.0</el-tag>
      </div>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-title">
          工业智能体协议网关
          <el-tag v-if="headerUser" size="small" style="margin-left: 8px">{{ headerUser }} · {{ roleLabel(headerRole) }}</el-tag>
        </div>
        <div class="header-actions">
          <el-button link type="primary" @click="tokenVisible = true">
            <el-icon><Key /></el-icon>
            API Token
          </el-button>
          <el-button link type="primary" @click="handleLogout">退出</el-button>
          <el-link href="/health" target="_blank" type="primary">健康检查</el-link>
          <el-link href="/swagger/index.html" target="_blank" type="primary">API 文档</el-link>
        </div>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>

    <el-dialog v-model="tokenVisible" title="API Token 设置" width="480px">
      <p class="token-hint">启用后端 <code>auth.enabled</code> 后，在此填写 Bearer Token 以访问 API。</p>
      <el-input v-model="apiToken" type="password" show-password placeholder="Bearer Token" />
      <template #footer>
        <el-button @click="clearToken">清除</el-button>
        <el-button type="primary" @click="saveToken">保存</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getApiToken, setApiToken, clearAuth, isAdmin, getUsername, getUserRole, roleLabel } from '@/utils/auth'
import { useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const tokenVisible = ref(false)
const apiToken = ref(getApiToken())

const isLoginPage = computed(() => route.path === '/login')
const showAdminMenu = computed(() => isAdmin())
const headerUser = computed(() => getUsername())
const headerRole = computed(() => getUserRole())

const activeMenu = computed(() => {
  if (route.path.startsWith('/devices')) return '/devices'
  if (route.path.startsWith('/alerts')) return '/alerts'
  if (route.path.startsWith('/dashboard')) return '/dashboard'
  if (route.path.startsWith('/users')) return '/users'
  if (route.path.startsWith('/audit')) return '/audit'
  return route.path
})

function saveToken() {
  setApiToken(apiToken.value.trim())
  tokenVisible.value = false
  ElMessage.success('Token 已保存')
}

function clearToken() {
  apiToken.value = ''
  setApiToken('')
  ElMessage.success('Token 已清除')
}

function handleLogout() {
  clearAuth()
  router.push('/login')
}
</script>

<style scoped>
.layout {
  height: 100vh;
}

.aside {
  background: #001529;
  display: flex;
  flex-direction: column;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 16px;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}

.aside-footer {
  margin-top: auto;
  padding: 16px;
}

.header {
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-title {
  font-size: 16px;
  color: #606266;
}

.header-actions {
  display: flex;
  gap: 16px;
}

.main {
  padding: 20px 24px;
  overflow: auto;
}

.token-hint {
  color: #909399;
  font-size: 13px;
  margin: 0 0 12px;
}
</style>
