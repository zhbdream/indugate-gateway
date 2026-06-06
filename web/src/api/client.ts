import axios from 'axios'
import type { ApiResponse } from '@/types'
import { ElMessage } from 'element-plus'
import { getAuthHeaderToken } from '@/utils/auth'

const http = axios.create({
  baseURL: '',
  timeout: 30000,
})

http.interceptors.request.use((config) => {
  const token = getAuthHeaderToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse<unknown>
    if (body && typeof body.code === 'number') {
      if (body.code !== 0) {
        if (!response.config.silent) {
          ElMessage.error(body.message || '请求失败')
        }
        return Promise.reject(new Error(body.message))
      }
      // Return unwrapped payload so callers get T, not AxiosResponse<T>.
      return body.data
    }
    return response.data
  },
  (error) => {
    if (!error.config?.silent) {
      const message = error.response?.data?.message || error.message || '网络错误'
      ElMessage.error(message)
    }
    return Promise.reject(error)
  }
)

export default http
