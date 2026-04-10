import request from '../utils/request'

export const login = (data) => {
  return request.post('/user/login/', data)
}

export const register = (data) => {
  return request.post('/user/register/', data)
}

export const getUserInfo = () => {
  return request.get('/user/info/')
}

export const updateAvatar = (data) => request.post('/user/avatar/', data, {
  headers: { 'Content-Type': 'multipart/form-data' }
})

export const getUserVideoList = (userId) => {
  return request.get('/user/video/list/', {
    params: { user_id: userId }
  })
}

export const getRelationList = (type, userId) => {
  const url = type === 'followers' ? '/relation/follower/list/' : '/relation/follow/list/'
  return request.get(url, {
    params: userId ? { user_id: userId } : undefined
  })
}

export const changePassword = (data) => {
  return request.post('/user/password/', data)
}
