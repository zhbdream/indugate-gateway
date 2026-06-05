import { createRouter, createWebHistory } from 'vue-router'
import { getAuthConfig } from '@/api/auth'
import { getJwtToken, getUserRole } from '@/utils/auth'

let authConfigCache: { auth_enabled: boolean; jwt_enabled: boolean; device_acl_enabled: boolean } | null = null

async function loadAuthConfig() {
  if (!authConfigCache) {
    try {
      authConfigCache = await getAuthConfig()
    } catch {
      authConfigCache = { auth_enabled: false, jwt_enabled: false, device_acl_enabled: false }
    }
  }
  return authConfigCache
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
    { path: '/devices', name: 'devices', component: () => import('@/views/DevicesView.vue') },
    { path: '/devices/:id', name: 'device-detail', component: () => import('@/views/DeviceDetailView.vue') },
    { path: '/simulators', name: 'simulators', component: () => import('@/views/SimulatorsView.vue') },
    { path: '/alerts', name: 'alerts', component: () => import('@/views/AlertsView.vue') },
    { path: '/users', name: 'users', component: () => import('@/views/UsersView.vue'), meta: { admin: true } },
    { path: '/audit', name: 'audit', component: () => import('@/views/AuditLogsView.vue'), meta: { admin: true } },
  ],
})

router.beforeEach(async (to, _from, next) => {
  if (to.meta.public) {
    next()
    return
  }
  const cfg = await loadAuthConfig()
  if (cfg.auth_enabled && cfg.jwt_enabled && !getJwtToken()) {
    next({ path: '/login', query: { redirect: to.fullPath } })
    return
  }
  if (to.meta.admin && getUserRole() !== 'admin') {
    next('/dashboard')
    return
  }
  next()
})

export default router
