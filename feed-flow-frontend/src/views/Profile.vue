<template>
  <div class="profile-container">
    <el-card shadow="hover">
      <div class="profile-topbar">
        <div class="profile-title">我的主页</div>
        <el-button text @click="router.push('/settings')">
          <el-icon><Setting /></el-icon>
          设置
        </el-button>
      </div>

      <div class="user-header">
        <el-upload
          class="avatar-uploader"
          :show-file-list="false"
          accept=".jpg,.jpeg,.png"
          :auto-upload="false"
          :on-change="handleAvatarChange"
        >
          <el-avatar :size="100" :src="getMediaUrl(userInfo.Avatar) || defaultAvatar" />
          <div class="avatar-tip">点击更换头像</div>
        </el-upload>
        <div class="user-info">
          <h2>{{ userInfo.Username || '用户' }}</h2>
          <p>ID：{{ userInfo.ID }}</p>
          <el-button plain size="small" @click="router.push('/settings')">
            进入设置
          </el-button>
        </div>
      </div>

      <el-divider />

      <div class="stats">
        <div class="stat-item">
          <div class="num">{{ stats.workCount }}</div>
          <div class="label">作品</div>
        </div>
        <div class="stat-item">
          <div class="num">{{ stats.likeReceivedCount }}</div>
          <div class="label">获赞</div>
        </div>
        <button type="button" class="stat-item clickable" @click="router.push('/profile/following')">
          <div class="num">{{ stats.followCount }}</div>
          <div class="label">关注</div>
        </button>
        <button type="button" class="stat-item clickable" @click="router.push('/profile/fans')">
          <div class="num">{{ stats.followerCount }}</div>
          <div class="label">粉丝</div>
        </button>
      </div>

      <el-divider />

      <div class="works-section">
        <div class="works-header">
          <h3>我的作品</h3>
          <span>{{ works.length }} 个</span>
        </div>
        <el-empty v-if="works.length === 0" description="还没有发布作品" />
        <div v-else class="works-grid">
          <div v-for="video in works" :key="video.ID" class="work-card">
            <video
              :src="getMediaUrl(video.PlayUrl)"
              :poster="getMediaUrl(video.CoverUrl)"
              class="work-preview"
              controls
              preload="metadata"
            ></video>
            <div class="work-title">{{ video.Title }}</div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getUserInfo, getUserVideoList, updateAvatar } from '../api/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const defaultAvatar = 'https://via.placeholder.com/150'
const userInfo = ref({})
const works = ref([])
const stats = reactive({
  workCount: 0,
  likeReceivedCount: 0,
  followCount: 0,
  followerCount: 0
})

const fetchUserInfo = async () => {
  try {
    const [userRes, worksRes] = await Promise.all([
      getUserInfo(),
      getUserVideoList(localStorage.getItem('userId'))
    ])

    if (userRes.status_code === 0) {
      userInfo.value = userRes.user
      stats.workCount = userRes.stats?.work_count || 0
      stats.likeReceivedCount = userRes.stats?.like_received_count || 0
      stats.followCount = userRes.stats?.follow_count || 0
      stats.followerCount = userRes.stats?.follower_count || 0
    }

    if (worksRes.status_code === 0) {
      works.value = worksRes.video_list || []
      stats.workCount = works.value.length
    }
  } catch (error) {
    ElMessage.error('获取个人信息失败')
  }
}

const handleAvatarChange = async (file) => {
  if (!file.raw) return

  const fd = new FormData()
  fd.append('avatar', file.raw)

  loading.value = true
  try {
    const res = await updateAvatar(fd)
    if (res.status_code === 0) {
      userInfo.value.Avatar = res.avatar_url
      ElMessage.success('头像更新成功')
    } else {
      ElMessage.error(res.status_msg || '头像更新失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '头像更新失败')
  } finally {
    loading.value = false
  }
}

const getMediaUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  return url
}

onMounted(fetchUserInfo)
</script>

<style scoped>
.profile-container {
  max-width: 560px;
  margin: 24px auto;
  padding: 0 20px 96px;
}

.profile-topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}

.profile-title {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
}

.user-header {
  display: flex;
  align-items: center;
  gap: 20px;
}

.avatar-uploader {
  cursor: pointer;
  position: relative;
}

.avatar-tip {
  position: absolute;
  bottom: -25px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
  color: #409eff;
}

.user-info h2 {
  margin: 0 0 5px;
  color: #303133;
}

.user-info p {
  margin: 0;
  color: #909399;
}

.user-info :deep(.el-button) {
  margin-top: 10px;
}

.stats {
  display: flex;
  justify-content: space-around;
  gap: 12px;
  padding: 10px 0;
}

.stat-item {
  flex: 1;
  border: none;
  background: transparent;
  text-align: center;
}

.stat-item .num {
  font-size: 24px;
  font-weight: 500;
  color: #303133;
}

.stat-item .label {
  font-size: 14px;
  color: #909399;
  margin-top: 5px;
}

.clickable {
  cursor: pointer;
  border-radius: 10px;
  transition: background-color 0.2s ease;
}

.clickable:hover {
  background: #f5f7fa;
}

.works-section {
  padding: 10px 0;
}

.works-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.works-header h3 {
  margin: 0;
  color: #303133;
}

.works-header span {
  color: #909399;
  font-size: 13px;
}

.works-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.work-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
  padding: 10px;
  background: #fff;
}

.work-preview {
  width: 100%;
  aspect-ratio: 9 / 16;
  border-radius: 8px;
  background: #000;
  object-fit: cover;
}

.work-title {
  margin-top: 10px;
  color: #303133;
  font-size: 14px;
  line-height: 1.4;
}
</style>
