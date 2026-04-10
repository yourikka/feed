import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  //开发服务配置
  server: {
    host: '0.0.0.0',
    // 端口号
    port: 3000,
    // 跨域代理
    proxy: {
      '/douyin': {
        target: 'http://localhost:8080', // 后端接口地址
        changeOrigin: true, // 是否改变请求头中的origin字段
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
