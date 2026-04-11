<template>
  <div class="feed-container">
    <div class="header">
      <h2>🔥 热门视频</h2>
      <el-button type="primary" @click="router.push('/publish')">
        <el-icon><Plus /></el-icon> 发布视频
      </el-button>
    </div>

    <el-skeleton :loading="loading" animated :count="3">
      <div class="video-list">
        <el-empty v-if="videoList.length === 0 && !loading" description="暂无视频" />
        <el-card v-for="video in videoList" :key="video.ID" class="video-card" shadow="hover">
          <div class="video-shell">
            <video
              :src="getMediaUrl(video.PlayUrl)"
              controls
              preload="metadata"
              playsinline
              :poster="getMediaUrl(video.CoverUrl)"
              class="video-player"
            ></video>
            <div class="video-actions">
              <button
                v-if="shouldShowFollow(video)"
                type="button"
                class="floating-action follow"
                :class="{ active: video.IsFollowing }"
                @click="handleFollow(video)"
              >
                <el-icon><UserFilled /></el-icon>
              </button>
              <button
                type="button"
                class="floating-action"
                :class="{ active: video.IsLiked }"
                @click="handleLike(video)"
              >
                <el-icon><HeartFilled /></el-icon>
              </button>
              <button
                type="button"
                class="floating-action comment"
                @click="openComment(video)"
              >
                <el-icon><ChatDotRound /></el-icon>
              </button>
              <button
                type="button"
                class="floating-action favorite"
                :class="{ active: video.IsFavorited }"
                @click="handleFavorite(video)"
              >
                <el-icon><StarFilled /></el-icon>
              </button>
            </div>
          </div>
          <div class="video-info">
            <h3>{{ video.Title }}</h3>
            <div class="author">
              <el-avatar :size="32" :src="getAuthorAvatar(video)" />
              <span>{{ getAuthorName(video) }}</span>
            </div>
            <div class="meta-row">
              <span>{{ video.LikeCount || 0 }} 点赞</span>
              <span>{{ video.CommentCount || 0 }} 评论</span>
              <span>{{ video.FavoriteCount || 0 }} 收藏</span>
            </div>
          </div>
        </el-card>
      </div>
      <div v-if="videoList.length > 0" class="load-more-wrap">
        <el-button :loading="loadingMore" :disabled="!hasMore" @click="fetchList(false)">
          {{ hasMore ? '加载更多' : '没有更多了' }}
        </el-button>
      </div>
    </el-skeleton>

    <CommentDialog
      v-model="showComment"
      :videoId="currentVideoId"
      :videoAuthorId="currentVideoAuthorId"
      @close="showComment = false"
    />
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getFeedList } from '../api/feed'
import { favoriteVideo, followUser, likeVideo } from '../api/interaction'
import CommentDialog from '../components/CommentDialog.vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const defaultAvatar = 'https://via.placeholder.com/150'
const currentUserId = computed(() => Number(localStorage.getItem('userId') || 0))
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const cursor = ref(0)
const videoList = ref([])
const showComment = ref(false)
const currentVideoId = ref(0)
const currentVideoAuthorId = ref(0)

const fetchList = async (reset = true) => {
  if (reset) {
    loading.value = true
    cursor.value = 0
    hasMore.value = true
  } else {
    if (loadingMore.value || !hasMore.value) return
    loadingMore.value = true
  }

  try {
    const params = { limit: 10 }
    if (!reset && cursor.value) params.cursor = cursor.value

    const res = await getFeedList(params)
    if (res.status_code === 0) {
      const list = res.video_list || []
      videoList.value = reset ? list : [...videoList.value, ...list]
      cursor.value = Number(res.next_cursor || 0)
      hasMore.value = !!res.has_more
    } else ElMessage.error(res.status_msg || '获取视频列表失败')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '获取视频列表失败')
  } finally {
    if (reset) loading.value = false
    else loadingMore.value = false
  }
}

const handleLike = async (video) => {
  if (!localStorage.getItem('token')) return ElMessage.warning('请先登录')
  try {
    const res = await likeVideo(video.ID)
    if (res.status_code === 0) {
      const liked = !!res.liked
      video.IsLiked = liked
      video.LikeCount = Math.max(0, (video.LikeCount || 0) + (liked ? 1 : -1))
    } else ElMessage.error(res.status_msg || '点赞失败')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '点赞失败')
  }
}

const handleFavorite = async (video) => {
  if (!localStorage.getItem('token')) return ElMessage.warning('请先登录')
  try {
    const res = await favoriteVideo(video.ID)
    if (res.status_code === 0) {
      const favorited = !!res.favorited
      video.IsFavorited = favorited
      video.FavoriteCount = Math.max(0, (video.FavoriteCount || 0) + (favorited ? 1 : -1))
    } else ElMessage.error(res.status_msg || '收藏失败')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '收藏失败')
  }
}

const handleFollow = async (video) => {
  if (!localStorage.getItem('token')) return ElMessage.warning('请先登录')
  try {
    const res = await followUser(video.Author.ID)
    if (res.status_code === 0) {
      video.IsFollowing = !!res.following
    } else ElMessage.error(res.status_msg || '关注失败')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '关注失败')
  }
}

const openComment = (video) => {
  currentVideoId.value = video.ID
  currentVideoAuthorId.value = video.Author?.ID || 0
  showComment.value = true
}

const getMediaUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  return url
}

const shouldShowFollow = (video) => video.Author?.ID && video.Author.ID !== currentUserId.value
const getAuthorAvatar = (video) => video.Author?.Avatar || defaultAvatar
const getAuthorName = (video) => video.Author?.Username || '匿名用户'

onMounted(fetchList)
</script>

<style scoped>
.feed-container {
  padding: 20px;
  padding-bottom: 96px;
  max-width: 800px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.video-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.video-card {
  border-radius: 12px;
}

.video-shell {
  position: relative;
}

.video-player {
  width: 100%;
  max-height: 500px;
  min-height: 260px;
  border-radius: 8px;
  background: #000;
  object-fit: contain;
}

.video-actions {
  position: absolute;
  right: 16px;
  bottom: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.floating-action {
  width: 46px;
  height: 46px;
  border: none;
  border-radius: 999px;
  background: rgba(20, 20, 20, 0.6);
  color: #c9ced8;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  cursor: pointer;
  backdrop-filter: blur(10px);
  transition: transform 0.2s ease, background-color 0.2s ease, color 0.2s ease;
}

.floating-action:hover {
  transform: scale(1.06);
}

.floating-action.active {
  color: #ff5b6e;
  background: rgba(255, 255, 255, 0.92);
}

.floating-action.comment {
  color: #ffffff;
}

.floating-action.follow.active {
  color: #409eff;
  background: rgba(255, 255, 255, 0.92);
}

.floating-action.favorite.active {
  color: #f7b500;
}

.video-info {
  margin-top: 15px;
}

.video-info h3 {
  margin: 0 0 10px;
  color: #303133;
}

.load-more-wrap {
  margin-top: 8px;
  display: flex;
  justify-content: center;
}

.author {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #606266;
}

.meta-row {
  margin-top: 12px;
  display: flex;
  gap: 10px;
  color: #909399;
  font-size: 13px;
}
</style>
