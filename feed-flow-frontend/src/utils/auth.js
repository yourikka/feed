export const USERNAME_MIN_LENGTH = 3
export const USERNAME_MAX_LENGTH = 20
export const PASSWORD_MIN_LENGTH = 8
export const PASSWORD_MAX_LENGTH = 64

const usernameRegex = /^[a-zA-Z0-9_]+$/

export const validateUsername = (username) => {
  const normalized = String(username || '').trim()
  if (normalized.length < USERNAME_MIN_LENGTH || normalized.length > USERNAME_MAX_LENGTH) {
    return `用户名长度需在 ${USERNAME_MIN_LENGTH} 到 ${USERNAME_MAX_LENGTH} 个字符之间`
  }
  if (!usernameRegex.test(normalized)) {
    return '用户名只能包含字母、数字和下划线'
  }
  return ''
}

export const validatePassword = (password) => {
  const raw = String(password || '')
  if (raw.length < PASSWORD_MIN_LENGTH || raw.length > PASSWORD_MAX_LENGTH) {
    return `密码长度需在 ${PASSWORD_MIN_LENGTH} 到 ${PASSWORD_MAX_LENGTH} 个字符之间`
  }
  return ''
}
