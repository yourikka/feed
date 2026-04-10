// 导入axios
import axios from 'axios'
import router from '../router'
import { ElMessage } from 'element-plus'

// 1. 创建axios实例，配置基础参数
const request = axios.create({
  // 基础请求地址：所有接口都会自动拼接这个前缀
  baseURL: '/douyin',
  // 请求超时时间：5秒后没响应就自动中断，避免页面卡死
  timeout: 5000
})

// 2. 请求拦截器：发送请求之前，统一做处理
request.interceptors.request.use(
  (config) => {
    // 核心功能：自动给请求头加Token，后端鉴权用
    // 从localStorage里取出登录时存的Token
    const token = localStorage.getItem('token')
    // 如果有Token，就加到请求头里，和你后端JWT中间件的格式完全对应
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    // 返回处理后的配置，请求才会发出去
    return config
  },
  (error) => {
    // 请求发送失败的错误处理
    return Promise.reject(error)
  }
)

// 3. 响应拦截器：收到后端响应之后，统一做处理
request.interceptors.response.use(
  (response) => {
    // 响应成功：直接返回后端返回的数据，不用每次都写res.data
    return response.data
  },
  (error) => {
    // 响应失败的统一错误处理
    const status = error.response?.status

    // 401错误：Token无效/过期/未登录，直接跳转到登录页
    if (status === 401) {
      // 清除过期的Token
      localStorage.clear()
      // 跳转到登录页
      router.push('/login')
      ElMessage.error('登录已过期，请重新登录')
    }

    // 其他错误，统一打印，方便调试
    console.error('接口请求错误：', error)
    return Promise.reject(error)
  }
)

// 导出封装好的axios实例，所有api文件都会导入这个工具
export default request