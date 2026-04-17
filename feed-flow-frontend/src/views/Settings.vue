<template>
  <div class="settings-container">
    <el-card shadow="hover">
      <div class="settings-header">
        <el-button @click="router.back()">
          <el-icon><ArrowLeft /></el-icon>
          返回
        </el-button>
        <h2>设置</h2>
        <div class="header-placeholder"></div>
      </div>

      <div class="settings-list">
        <button type="button" class="settings-item" @click="passwordDialogVisible = true">
          <div>
            <div class="settings-title">修改密码</div>
            <div class="settings-desc">更新登录密码</div>
          </div>
          <el-icon><ArrowRight /></el-icon>
        </button>

        <button type="button" class="settings-item danger" @click="handleLogout">
          <div>
            <div class="settings-title">退出登录</div>
            <div class="settings-desc">清除当前登录状态</div>
          </div>
          <el-icon><SwitchButton /></el-icon>
        </button>
      </div>
    </el-card>

    <el-dialog v-model="passwordDialogVisible" title="修改密码" width="420px">
      <el-form label-width="90px">
        <el-form-item label="原密码">
          <el-input v-model="passwordForm.oldPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closePasswordDialog">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleChangePassword">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { changePassword } from '../api/user'
import { validatePassword } from '../utils/auth'

const router = useRouter()
const saving = ref(false)
const passwordDialogVisible = ref(false)
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const closePasswordDialog = () => {
  passwordDialogVisible.value = false
  passwordForm.oldPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
}

const handleChangePassword = async () => {
  if (!passwordForm.oldPassword || !passwordForm.newPassword || !passwordForm.confirmPassword) {
    return ElMessage.warning('请填写完整密码信息')
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    return ElMessage.warning('两次输入的新密码不一致')
  }
  const passwordErr = validatePassword(passwordForm.newPassword)
  if (passwordErr) return ElMessage.warning(passwordErr)

  saving.value = true
  try {
    const res = await changePassword({
      old_password: passwordForm.oldPassword,
      new_password: passwordForm.newPassword
    })
    if (res.status_code === 0) {
      ElMessage.success('密码修改成功')
      closePasswordDialog()
    } else {
      ElMessage.error(res.status_msg || '修改密码失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.status_msg || '修改密码失败')
  } finally {
    saving.value = false
  }
}

const handleLogout = async () => {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', { type: 'warning' })
    localStorage.clear()
    router.push('/login')
  } catch {}
}
</script>

<style scoped>
.settings-container {
  max-width: 620px;
  margin: 24px auto;
  padding: 0 20px 96px;
}

.settings-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 18px;
}

.settings-header h2 {
  font-size: 20px;
  color: #303133;
}

.header-placeholder {
  width: 88px;
}

.settings-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.settings-item {
  width: 100%;
  border: 1px solid #ebeef5;
  border-radius: 12px;
  background: #fff;
  padding: 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  text-align: left;
}

.settings-item.danger {
  border-color: #fde2e2;
}

.settings-title {
  font-size: 16px;
  color: #303133;
  font-weight: 600;
}

.settings-desc {
  margin-top: 6px;
  font-size: 13px;
  color: #909399;
}

.danger .settings-title,
.danger :deep(.el-icon) {
  color: #f56c6c;
}
</style>
