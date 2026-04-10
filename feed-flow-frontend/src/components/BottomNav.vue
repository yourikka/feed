<template>
  <nav class="bottom-nav">
    <button
      type="button"
      class="nav-item"
      :class="{ active: activeTab === 'feed' }"
      @click="goTo('/feed')"
    >
      <el-icon><House /></el-icon>
      <span>首页</span>
    </button>
    <button
      type="button"
      class="nav-item"
      :class="{ active: activeTab === 'publish' }"
      @click="goTo('/publish')"
    >
      <el-icon><Plus /></el-icon>
      <span>发布</span>
    </button>
    <button
      type="button"
      class="nav-item"
      :class="{ active: activeTab === 'profile' }"
      @click="goTo('/profile')"
    >
      <el-icon><User /></el-icon>
      <span>我的</span>
    </button>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const activeTab = computed(() => route.meta.tabKey || route.name?.toString().toLowerCase() || 'feed')

const goTo = (path) => {
  if (route.path !== path) {
    router.push(path)
  }
}
</script>

<style scoped>
.bottom-nav {
  position: fixed;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: min(100%, 800px);
  display: flex;
  justify-content: space-around;
  gap: 12px;
  padding: 12px 20px calc(12px + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.96);
  border-top: 1px solid #ebeef5;
  backdrop-filter: blur(10px);
  z-index: 20;
}

.nav-item {
  border: none;
  background: transparent;
  color: #606266;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  cursor: pointer;
}

.nav-item.active {
  color: #409eff;
}
</style>
