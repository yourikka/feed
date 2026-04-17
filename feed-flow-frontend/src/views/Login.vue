<template>
  <div class="login-container">
    <el-card class="login-card" shadow="hover">
      <div class="logo-area">
        <el-icon :size="60" color="#409EFF"><VideoPlay /></el-icon>
        <h1>短视频</h1>
        <p>请登录</p>
      </div>

      <el-form :model="form" size="large">
        <el-form-item>
          <el-input
            v-model="form.username"
            placeholder="请输入用户名"
            prefix-icon="User"
          />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" style="width: 100%" @click="handleLogin" :loading="loading">
            登录
          </el-button>
        </el-form-item>
        <el-form-item>
          <el-button style="width: 100%" @click="handleRegister" :loading="registerLoading">
            注册新账号
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { login, register } from '../api/user'
import { ElMessage } from 'element-plus'
import { validatePassword, validateUsername } from '../utils/auth'

const router = useRouter()
const loading = ref(false)
const registerLoading = ref(false)
const form = reactive({
  username: '',
  password: ''
})

const handleLogin = async () => {
  if (!form.username || !form.password) return ElMessage.warning('请输入用户名和密码')
  const usernameErr = validateUsername(form.username)
  if (usernameErr) return ElMessage.warning(usernameErr)
  const passwordErr = validatePassword(form.password)
  if (passwordErr) return ElMessage.warning(passwordErr)

  loading.value = true
  try {
    const res = await login(form)
    if (res.status_code === 0) {
      localStorage.setItem('token', res.token)
      localStorage.setItem('userId', res.user_id)
      ElMessage.success('登录成功')
      router.push('/feed')
    } else ElMessage.error(res.status_msg)
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '登录失败，请稍后重试')
  } finally { loading.value = false }
}

const handleRegister = async () => {
  if (!form.username || !form.password) return ElMessage.warning('请输入用户名和密码')
  const usernameErr = validateUsername(form.username)
  if (usernameErr) return ElMessage.warning(usernameErr)
  const passwordErr = validatePassword(form.password)
  if (passwordErr) return ElMessage.warning(passwordErr)

  registerLoading.value = true
  try {
    const res = await register(form)
    if (res.status_code === 0) {
      localStorage.setItem('token', res.token)
      localStorage.setItem('userId', res.user_id)
      ElMessage.success('注册成功')
      router.push('/feed')
    } else ElMessage.error(res.status_msg)
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '注册失败，请稍后重试')
  } finally { registerLoading.value = false }
}
</script>

<style scoped>
.login-container {
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}
.login-card {
  width: 90%;
  max-width: 420px;
  border-radius: 16px;
}
.logo-area {
  text-align: center;
  margin-bottom: 30px;
}
.logo-area h1 {
  margin: 10px 0 5px;
  color: #303133;
}
.logo-area p {
  color: #909399;
  font-size: 14px;
}
</style>
