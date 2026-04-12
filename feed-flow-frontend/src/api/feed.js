import request from '../utils/request'

/**
 * 获取Feed视频流列表
 * @returns 后端返回的视频列表数据
 */
export const getFeedList = (params = {}) => {
  return request.get('/feed/', { params })
}

/**
 * 发布视频接口
 * @param {FormData} data 发布参数（title标题、视频文件、封面文件）
 * @returns 后端返回的发布结果
 */
export const publishVideo = (data) => {
  return request.post('/publish/action/', data, {
    // 告诉后端，请求体是FormData格式，和后端的c.PostForm接收方式完全对应
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

/**
 * 删除视频
 * @param {number|string} videoId 视频ID
 * @returns 后端返回的删除结果
 */
export const deleteVideo = (videoId) => {
  return request.delete('/publish/action/', {
    params: { video_id: videoId }
  })
}

/**
 * 获取用户发布的视频列表（预留，个人中心用）
 */
export const getUserVideoList = (userId, params = {}) => {
  return request.get('/user/video/list/', {
    params: { user_id: userId, ...params }
  })
}
