<template>
  <div class="feed-container">
    <div class="header">
      <div class="header-copy">
        <h2>视频 Feed</h2>
        <p>加入曝光去重、自动播放、预加载和播放行为埋点。</p>
      </div>
      <div class="header-actions">
        <el-segmented v-model="sortType" :options="sortOptions" @change="handleSortChange" />
        <el-button type="primary" @click="router.push('/publish')">
          <el-icon><Plus /></el-icon> 发布视频
        </el-button>
      </div>
    </div>

    <el-skeleton :loading="loading" animated :count="3">
      <div class="video-list">
        <el-empty v-if="videoList.length === 0 && !loading" description="暂无视频" />
        <el-card
          v-for="(video, index) in videoList"
          :key="video.ID"
          :ref="(el) => setCardRef(el, index)"
          class="video-card"
          shadow="hover"
          :class="{ active: index === activeIndex }"
        >
          <div class="video-shell">
            <video
              :ref="(el) => setVideoRef(el, index)"
              :src="getMediaUrl(video.PlayUrl)"
              :poster="getMediaUrl(video.CoverUrl)"
              :preload="getPreloadMode(index)"
              :muted="index === activeIndex"
              controls
              playsinline
              webkit-playsinline
              class="video-player"
              @play="handleVideoPlay(video, index)"
              @pause="handleVideoPause(video, index)"
              @ended="handleVideoEnded(video, index)"
              @timeupdate="handleTimeUpdate(video, index)"
            ></video>
            <div v-if="index === activeIndex" class="video-badge">当前播放</div>
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
                <el-icon><Pointer /></el-icon>
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
            <div class="video-topline">
              <h3>{{ video.Title }}</h3>
              <span class="sort-tag">{{ sortLabel }}</span>
            </div>
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getFeedList, trackFeedEvent } from '../api/feed'
import { favoriteVideo, followUser, likeVideo } from '../api/interaction'
import CommentDialog from '../components/CommentDialog.vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const defaultAvatar = 'https://via.placeholder.com/150'
const sortOptions = [
  { label: '最新', value: 'latest' },
  { label: '热门', value: 'hot' }
]

const currentUserId = computed(() => Number(localStorage.getItem('userId') || 0))
const sortType = ref('latest')
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const cursor = ref(0)
const cursorToken = ref('')
const videoList = ref([])
const showComment = ref(false)
const currentVideoId = ref(0)
const currentVideoAuthorId = ref(0)
const activeIndex = ref(-1)

const cardRefs = []
const videoRefs = []
const exposureTimers = new Map()
const exposedVideoIds = new Set()
const playStartedVideoIds = new Set()
const progressMarks = new Map()
let observer = null
let sessionId = ''
let clientId = ''

const sortLabel = computed(() => (sortType.value === 'hot' ? '热度混排' : '时间流'))

const ensureClientId = () => {
  const storageKey = 'feed_client_id'
  let value = localStorage.getItem(storageKey)
  if (!value) {
    value = `client_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
    localStorage.setItem(storageKey, value)
  }
  clientId = value
  return value
}

const createSessionId = () => `feed_session_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
const createRequestId = (eventType, videoId) =>
  `${eventType}_${videoId}_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`

const getFeedParams = (reset) => {
  const params = {
    limit: 10,
    sort: sortType.value,
    filter_seen: 1,
    client_id: ensureClientId()
  }
  if (!reset && cursor.value) params.cursor = cursor.value
  if (!reset && cursorToken.value) params.cursor_token = cursorToken.value
  return params
}

const fetchList = async (reset = true) => {
  if (reset) {
    loading.value = true
    cursor.value = 0
    cursorToken.value = ''
    hasMore.value = true
    activeIndex.value = -1
    clearExposureTimers()
    exposedVideoIds.clear()
    playStartedVideoIds.clear()
    progressMarks.clear()
    cardRefs.length = 0
    videoRefs.length = 0
    pauseAllVideos()
  } else {
    if (loadingMore.value || !hasMore.value) return
    loadingMore.value = true
  }

  try {
    const res = await getFeedList(getFeedParams(reset))
    if (res.status_code !== 0) {
      ElMessage.error(res.status_msg || '获取视频列表失败')
      return
    }

    const list = Array.isArray(res.video_list) ? res.video_list : []
    videoList.value = reset ? list : [...videoList.value, ...list]
    cursor.value = Number(res.next_cursor || 0)
    cursorToken.value = res.next_token || ''
    hasMore.value = !!res.has_more

    await nextTick()
    initObserver()
    if (videoList.value.length > 0 && activeIndex.value < 0) {
      setActiveVideo(0)
    } else {
      preloadNeighbors(activeIndex.value)
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '获取视频列表失败')
  } finally {
    if (reset) loading.value = false
    else loadingMore.value = false
  }
}

const reportEvent = async (payload) => {
  try {
    await trackFeedEvent({
      client_id: ensureClientId(),
      session_id: sessionId,
      request_id: createRequestId(payload.event_type, payload.video_id),
      ...payload
    })
  } catch (error) {
    console.error('事件上报失败', error)
  }
}

const scheduleExposure = (video, index) => {
  if (!video || exposedVideoIds.has(video.ID) || exposureTimers.has(index)) return
  const timer = window.setTimeout(() => {
    exposedVideoIds.add(video.ID)
    exposureTimers.delete(index)
    reportEvent({
      video_id: video.ID,
      event_type: 'exposure'
    })
    if (index >= videoList.value.length - 3 && hasMore.value && !loadingMore.value) {
      fetchList(false)
    }
  }, 900)
  exposureTimers.set(index, timer)
}

const clearExposureTimer = (index) => {
  const timer = exposureTimers.get(index)
  if (timer) {
    window.clearTimeout(timer)
    exposureTimers.delete(index)
  }
}

const clearExposureTimers = () => {
  exposureTimers.forEach((timer) => window.clearTimeout(timer))
  exposureTimers.clear()
}

const pauseAllVideos = () => {
  videoRefs.forEach((videoEl) => {
    if (videoEl && !videoEl.paused) videoEl.pause()
  })
}

const attemptPlay = async (index) => {
  const videoEl = videoRefs[index]
  if (!videoEl) return
  videoEl.muted = true
  try {
    await videoEl.play()
  } catch (error) {
    console.error('自动播放失败', error)
  }
}

const preloadNeighbors = (index) => {
  ;[index + 1, index + 2].forEach((targetIndex) => {
    const videoEl = videoRefs[targetIndex]
    if (!videoEl) return
    try {
      videoEl.load()
    } catch (error) {
      console.error('预加载失败', error)
    }
  })
}

const setActiveVideo = async (index) => {
  if (index < 0 || index >= videoList.value.length) return
  if (activeIndex.value === index) return

  const prevIndex = activeIndex.value
  const prevVideoData = prevIndex >= 0 ? videoList.value[prevIndex] : null
  const prevVideoEl = prevIndex >= 0 ? videoRefs[prevIndex] : null
  activeIndex.value = index

  if (prevIndex >= 0) {
    if (prevVideoEl && !prevVideoEl.paused) {
      const duration = Number(prevVideoEl.duration || 0)
      const currentTime = Number(prevVideoEl.currentTime || 0)
      if (prevVideoData && duration > 0 && currentTime / duration < 0.85) {
        reportEvent({
          video_id: prevVideoData.ID,
          event_type: 'skip',
          position_ms: Math.round(currentTime * 1000),
          duration_ms: Math.round(duration * 1000)
        })
      }
      prevVideoEl.pause()
    }
  }

  preloadNeighbors(index)
  await nextTick()
  attemptPlay(index)
}

const initObserver = () => {
  if (observer) observer.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        const index = Number(entry.target.dataset.index)
        if (Number.isNaN(index)) return
        if (entry.isIntersecting && entry.intersectionRatio >= 0.6) {
          scheduleExposure(videoList.value[index], index)
          setActiveVideo(index)
        } else {
          clearExposureTimer(index)
        }
      })
    },
    {
      threshold: [0.25, 0.6, 0.9]
    }
  )

  cardRefs.forEach((card, index) => {
    if (!card) return
    card.dataset.index = String(index)
    observer.observe(card)
  })
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

const handleVideoPlay = (video) => {
  if (playStartedVideoIds.has(video.ID)) return
  playStartedVideoIds.add(video.ID)
  reportEvent({
    video_id: video.ID,
    event_type: 'play_start'
  })
}

const handleVideoPause = (video, index) => {
  if (index !== activeIndex.value) return
  const videoEl = videoRefs[index]
  reportEvent({
    video_id: video.ID,
    event_type: 'pause',
    position_ms: Math.round((videoEl?.currentTime || 0) * 1000),
    duration_ms: Math.round((videoEl?.duration || 0) * 1000)
  })
}

const handleVideoEnded = (video, index) => {
  const videoEl = videoRefs[index]
  reportEvent({
    video_id: video.ID,
    event_type: 'play_finish',
    progress_ms: Math.round((videoEl?.duration || 0) * 1000),
    duration_ms: Math.round((videoEl?.duration || 0) * 1000),
    position_ms: Math.round((videoEl?.currentTime || 0) * 1000)
  })
}

const handleTimeUpdate = (video, index) => {
  if (index !== activeIndex.value) return
  const videoEl = videoRefs[index]
  if (!videoEl || !videoEl.duration) return

  const ratio = videoEl.currentTime / videoEl.duration
  const checkpoints = progressMarks.get(video.ID) || { mid: false, high: false }

  if (!checkpoints.mid && ratio >= 0.5) {
    checkpoints.mid = true
    reportEvent({
      video_id: video.ID,
      event_type: 'play_progress',
      progress_ms: Math.round(videoEl.currentTime * 1000),
      duration_ms: Math.round(videoEl.duration * 1000),
      position_ms: Math.round(videoEl.currentTime * 1000)
    })
  }

  if (!checkpoints.high && ratio >= 0.85) {
    checkpoints.high = true
    reportEvent({
      video_id: video.ID,
      event_type: 'play_progress',
      progress_ms: Math.round(videoEl.currentTime * 1000),
      duration_ms: Math.round(videoEl.duration * 1000),
      position_ms: Math.round(videoEl.currentTime * 1000)
    })
  }

  progressMarks.set(video.ID, checkpoints)
}

const handleSortChange = async () => {
  sessionId = createSessionId()
  await fetchList(true)
}

const openComment = (video) => {
  currentVideoId.value = video.ID
  currentVideoAuthorId.value = video.Author?.ID || 0
  showComment.value = true
}

const setCardRef = (el, index) => {
  cardRefs[index] = el?.$el || el || null
}

const setVideoRef = (el, index) => {
  videoRefs[index] = el || null
}

const getMediaUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  return url
}

const getPreloadMode = (index) => {
  if (index === activeIndex.value) return 'auto'
  if (index === activeIndex.value + 1 || index === activeIndex.value + 2) return 'metadata'
  return 'none'
}

const shouldShowFollow = (video) => video.Author?.ID && video.Author.ID !== currentUserId.value
const getAuthorAvatar = (video) => video.Author?.Avatar || defaultAvatar
const getAuthorName = (video) => video.Author?.Username || '匿名用户'

onMounted(async () => {
  ensureClientId()
  sessionId = createSessionId()
  await fetchList(true)
})

onBeforeUnmount(() => {
  clearExposureTimers()
  if (observer) observer.disconnect()
  pauseAllVideos()
})
</script>

<style scoped>
.feed-container {
  padding: 24px;
  padding-bottom: 96px;
  max-width: 860px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
  margin-bottom: 24px;
}

.header-copy h2 {
  margin: 0;
  font-size: 28px;
  color: #1f2937;
}

.header-copy p {
  margin: 8px 0 0;
  color: #6b7280;
  font-size: 14px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.video-list {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.video-card {
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.video-card.active {
  transform: translateY(-2px);
  border-color: rgba(37, 99, 235, 0.25);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.12);
}

.video-shell {
  position: relative;
}

.video-player {
  width: 100%;
  max-height: 620px;
  min-height: 320px;
  border-radius: 12px;
  background: #000;
  object-fit: contain;
}

.video-badge {
  position: absolute;
  left: 14px;
  top: 14px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.92);
  color: #fff;
  font-size: 12px;
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

.video-topline {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.video-info h3 {
  margin: 0;
  color: #303133;
}

.sort-tag {
  flex-shrink: 0;
  padding: 4px 10px;
  border-radius: 999px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
}

.load-more-wrap {
  margin-top: 8px;
  display: flex;
  justify-content: center;
}

.author {
  margin-top: 10px;
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

@media (max-width: 768px) {
  .feed-container {
    padding: 16px;
    padding-bottom: 88px;
  }

  .header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-actions {
    justify-content: space-between;
  }

  .video-player {
    min-height: 240px;
  }
}
</style>
