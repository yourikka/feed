import request from '../utils/request'

/**
 * 搜索视频接口（预留）
 */
export const searchVideo = (keyword) => {
  return request.get('/video/search', {
    params: { keyword }
  })
}

/**
 * 获取视频详情接口（预留）
 */
export const getVideoDetail = (videoId) => {
  return request.get('/video/detail', {
    params: { video_id: videoId }
  })
}