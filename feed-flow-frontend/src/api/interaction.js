import request from '../utils/request'

/**
 * 点赞/取消点赞接口
 * @param {Number} videoId 视频ID
 * @returns 后端返回的操作结果
 */
export const likeVideo = (videoId) => {
  return request.post('/like/action/', null, {
    params: { video_id: videoId }
  })
}

/**
 * 收藏/取消收藏接口
 * @param {Number} videoId 视频ID
 * @returns 后端返回的操作结果
 */
export const favoriteVideo = (videoId) => {
  return request.post('/favorite/action/', null, {
    params: { video_id: videoId }
  })
}

/**
 * 关注/取消关注接口
 * @param {Number} userId 用户ID
 * @returns 后端返回的操作结果
 */
export const followUser = (userId) => {
  return request.post('/relation/action/', null, {
    params: { to_user_id: userId }
  })
}

/**
 * 发布评论接口
 * @param {Object} data 评论参数 {videoId: 视频ID, content: 评论内容}
 * @returns 后端返回的发布结果
 */
export const addComment = (data) => {
  return request.post('/comment/action/', null, {
    params: {
      video_id: data.videoId,
      content: data.content
    }
  })
}

/**
 * 获取视频的评论列表
 * @param {Number} videoId 视频ID
 * @returns 后端返回的评论列表
 */
export const getComments = (videoId) => {
  return request.get('/comment/list/', {
    params: { video_id: videoId }
  })
}

/**
 * 删除评论接口
 * @param {Number} commentId 评论ID
 * @returns 后端返回的删除结果
 */
export const deleteComment = (commentId) => {
  return request.delete('/comment/action/', {
    params: { comment_id: commentId }
  })
}
