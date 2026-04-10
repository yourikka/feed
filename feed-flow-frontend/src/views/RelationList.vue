<template>
  <div class="relation-container">
    <el-card shadow="hover">
      <div class="relation-header">
        <el-button @click="router.back()">
          <el-icon><ArrowLeft /></el-icon>
          返回
        </el-button>
        <h2>{{ pageTitle }}</h2>
        <div class="header-placeholder"></div>
      </div>

      <el-empty v-if="!loading && userList.length === 0" :description="emptyText" />

      <div v-else class="relation-list">
        <div v-for="user in userList" :key="user.ID" class="relation-item">
          <div class="relation-user">
            <el-avatar :size="52" :src="user.Avatar || defaultAvatar" />
            <div class="relation-meta">
              <div class="username">{{ user.Username || '未命名用户' }}</div>
              <div class="user-id">ID：{{ user.ID }}</div>
            </div>
          </div>
          <el-button
            v-if="user.ID !== currentUserId"
            :type="user.IsFollowing ? 'info' : 'primary'"
            plain
            @click="handleFollow(user)"
          >
            {{ getFollowButtonText(user) }}
          </el-button>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getRelationList } from '../api/user'
import { followUser } from '../api/interaction'

const route = useRoute()
const router = useRouter()
const defaultAvatar = 'https://via.placeholder.com/150'
const loading = ref(false)
const userList = ref([])
const currentUserId = Number(localStorage.getItem('userId') || 0)

const relationType = computed(() => route.meta.relationType || 'following')
const pageTitle = computed(() => (relationType.value === 'followers' ? '我的粉丝' : '我的关注'))
const emptyText = computed(() => (relationType.value === 'followers' ? '还没有粉丝' : '还没有关注任何人'))

const fetchList = async () => {
  loading.value = true
  try {
    const res = await getRelationList(relationType.value, currentUserId)
    if (res.status_code === 0) {
      userList.value = res.user_list || []
    } else {
      ElMessage.error(res.status_msg || '获取列表失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '获取列表失败')
  } finally {
    loading.value = false
  }
}

const handleFollow = async (user) => {
  try {
    const res = await followUser(user.ID)
    if (res.status_code !== 0) {
      ElMessage.error(res.status_msg || '操作失败')
      return
    }

    user.IsFollowing = !!res.following
    if (relationType.value === 'following' && !user.IsFollowing) {
      userList.value = userList.value.filter((item) => item.ID !== user.ID)
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '操作失败')
  }
}

const getFollowButtonText = (user) => {
  if (user.IsFollowing) {
    return relationType.value === 'following' ? '取消关注' : '已关注'
  }
  return relationType.value === 'followers' ? '回关' : '关注'
}

onMounted(fetchList)
watch(relationType, fetchList)
</script>

<style scoped>
.relation-container {
  max-width: 620px;
  margin: 24px auto;
  padding: 0 20px 96px;
}

.relation-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}

.relation-header h2 {
  font-size: 20px;
  color: #303133;
}

.header-placeholder {
  width: 88px;
}

.relation-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.relation-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 0;
  border-bottom: 1px solid #f0f2f5;
}

.relation-user {
  display: flex;
  align-items: center;
  gap: 14px;
}

.username {
  font-size: 16px;
  color: #303133;
  font-weight: 600;
}

.user-id {
  margin-top: 6px;
  font-size: 13px;
  color: #909399;
}
</style>
