<template>
  <div class="publish-container">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <el-button @click="router.back()">
            <el-icon><ArrowLeft /></el-icon> 返回
          </el-button>
          <span>发布视频</span>
          <div></div>
        </div>
      </template>

      <el-form :model="form" label-width="80px">
        <el-form-item label="视频标题">
          <el-input v-model="form.title" placeholder="请输入视频标题" maxlength="50" show-word-limit />
        </el-form-item>

        <el-form-item label="上传视频">
          <el-upload
            class="video-uploader"
            drag
            accept=".mp4,.avi,.mov"
            :limit="1"
            :auto-upload="false"
            :on-change="handleVideoChange"
            :show-file-list="false"
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <div class="el-upload__text">将视频拖到此处，或<em>点击上传</em></div>
            <template #tip>
              <div class="el-upload__tip">支持 mp4/avi/mov 格式，最大 1GB</div>
            </template>
          </el-upload>
          <video v-if="form.video" class="preview-video" :src="videoPreview" controls></video>
        </el-form-item>

        <el-form-item label="视频封面">
          <el-upload
            class="cover-uploader"
            drag
            accept=".jpg,.jpeg,.png"
            :limit="1"
            :auto-upload="false"
            :on-change="handleCoverChange"
            :show-file-list="false"
          >
            <img v-if="form.cover" :src="coverPreview" class="cover-preview" />
            <el-icon v-else class="cover-uploader-icon"><Plus /></el-icon>
            <template #tip>
              <div class="el-upload__tip">支持 jpg/png 格式，建议尺寸 16:9</div>
            </template>
          </el-upload>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" style="width: 100%" @click="handlePublish" :loading="loading">
            发布视频
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { publishVideo } from '../api/feed'
import { ElMessage } from 'element-plus'

const MAX_VIDEO_SIZE = 1024 * 1024 * 1024

const router = useRouter()
const loading = ref(false)
const videoPreview = ref('')
const coverPreview = ref('')
const form = reactive({
  title: '',
  video: null,
  cover: null
})

// 处理视频选择
const handleVideoChange = (file) => {
  if (file.raw.size > MAX_VIDEO_SIZE) {
    ElMessage.error('视频大小不能超过 1GB')
    return
  }
  form.video = file.raw
  videoPreview.value = URL.createObjectURL(file.raw)
}

// 处理封面选择
const handleCoverChange = (file) => {
  form.cover = file.raw
  coverPreview.value = URL.createObjectURL(file.raw)
}

const handlePublish = async () => {
  if (!form.title) return ElMessage.warning('请输入标题')
  if (!form.video) return ElMessage.warning('请上传视频')
  if (!form.cover) return ElMessage.warning('请上传封面')

  loading.value = true
  try {
    // 组装 FormData
    const fd = new FormData()
    fd.append('title', form.title)
    fd.append('video', form.video)
    fd.append('cover', form.cover)

    const res = await publishVideo(fd)
    if (res.status_code === 0) {
      ElMessage.success('发布成功')
      router.push('/feed')
    } else ElMessage.error(res.status_msg || '发布失败')
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '发布失败，请稍后重试')
  } finally { loading.value = false }
}
</script>

<style scoped>
.publish-container {
  max-width: 700px;
  margin: 20px auto;
  padding: 0 20px 96px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}
.preview-video {
  width: 100%;
  max-height: 300px;
  margin-top: 15px;
  border-radius: 8px;
  background: #000;
}
.cover-uploader {
  width: 300px;
  height: 170px;
}
.cover-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 8px;
}
.cover-uploader :deep(.el-upload) {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.cover-uploader-icon {
  font-size: 28px;
  color: #8c939d;
}
</style>
