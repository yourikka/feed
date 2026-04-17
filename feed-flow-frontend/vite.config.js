import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    chunkSizeWarningLimit: 1200,
    rolldownOptions: {
      output: {
        manualChunks: (id) => {
          if (id.includes('node_modules/element-plus') || id.includes('node_modules/@element-plus/icons-vue')) {
            return 'element-plus'
          }
          if (id.includes('node_modules/vue') || id.includes('node_modules/vue-router')) {
            return 'vue-vendor'
          }
          if (id.includes('node_modules/axios')) {
            return 'http'
          }
        }
      }
    }
  },
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
