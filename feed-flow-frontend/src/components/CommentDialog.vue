<template>
  <el-dialog
    v-model="visible"
    title="评论列表"
    width="90%"
    :close-on-click-modal="false"
    @closed="handleClose"
  >
    <div class="comment-list">
      <el-empty v-if="comments.length === 0" description="暂无评论" />
      <div v-for="item in comments" :key="item.ID" class="comment-item">
        <el-avatar :size="32" :src="getCommentUserAvatar(item)" />
        <div class="comment-content">
          <div class="username">{{ getCommentUserName(item) }}</div>
          <div class="text">{{ item.Content }}</div>
          <button
            v-if="canDeleteComment(item)"
            type="button"
            class="delete-comment"
            @click="removeComment(item)"
          >
            <el-icon><Delete /></el-icon>
          </button>
        </div>
      </div>
    </div>
    <template #footer>
      <div class="footer-input">
        <el-input
          v-model="content"
          placeholder="说点什么..."
          @keyup.enter="submit"
        />
        <el-button type="primary" @click="submit">发送</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { getComments, addComment, deleteComment } from '../api/interaction'
import { ElMessage, ElMessageBox } from 'element-plus'

const defaultAvatar = 'https://via.placeholder.com/150'
const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  videoId: {
    type: Number,
    default: 0
  },
  videoAuthorId: {
    type: Number,
    default: 0
  }
})
const emit = defineEmits(['update:modelValue', 'close'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
const comments = ref([])
const content = ref('')
const currentUserId = computed(() => Number(localStorage.getItem('userId') || 0))

watch(
  () => [visible.value, props.videoId],
  async ([isVisible, videoId]) => {
    if (!isVisible || !videoId) return

    try {
      const res = await getComments(videoId)
      if (res.status_code === 0) comments.value = res.comments || []
      else ElMessage.error(res.status_msg || '获取评论失败')
    } catch (error) {
      ElMessage.error(error.response?.data?.status_msg || '获取评论失败')
    }
  }
)

const submit = async () => {
  if (!content.value.trim()) return ElMessage.warning('请输入评论内容')
  if (!localStorage.getItem('token')) return ElMessage.warning('请先登录')

  try {
    const res = await addComment({
      videoId: props.videoId,
      content: content.value.trim()
    })

    if (res.status_code !== 0) {
      ElMessage.error(res.status_msg || '评论失败')
      return
    }

    content.value = ''
    const latestComments = await getComments(props.videoId)
    if (latestComments.status_code === 0) comments.value = latestComments.comments || []
    ElMessage.success('评论成功')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '评论失败')
  }
}

const handleClose = () => {
  content.value = ''
  comments.value = []
  emit('close')
}

const getCommentUserAvatar = (item) => item.User?.Avatar || defaultAvatar

const getCommentUserName = (item) => item.User?.Username || '匿名用户'

const canDeleteComment = (item) => {
  if (!currentUserId.value) return false
  return item.UserID === currentUserId.value || props.videoAuthorId === currentUserId.value
}

const removeComment = async (item) => {
  try {
    await ElMessageBox.confirm('确定删除这条评论吗？', '提示', { type: 'warning' })
    const res = await deleteComment(item.ID)
    if (res.status_code !== 0) {
      ElMessage.error(res.status_msg || '删除评论失败')
      return
    }
    comments.value = comments.value.filter((comment) => comment.ID !== item.ID)
    ElMessage.success('评论已删除')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error.response?.data?.status_msg || '删除评论失败')
  }
}
</script>

<style scoped>
.comment-list {
  max-height: 400px;
  overflow-y: auto;
}
.comment-item {
  display: flex;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f5f7fa;
}
.comment-content {
  flex: 1;
  position: relative;
  padding-right: 32px;
}
.username {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
  margin-bottom: 4px;
}
.text {
  color: #606266;
  font-size: 14px;
}
.delete-comment {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #c0c4cc;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.delete-comment:hover {
  color: #f56c6c;
  background: #fef0f0;
}
.footer-input {
  display: flex;
  gap: 10px;
}
</style>
