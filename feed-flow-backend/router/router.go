package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/controller"
	"github.com/yourikka/feed-flow/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	// 全局使用跨域中间件
	r.Use(middleware.CORS())

	//公开接口（无需登录）
	public := r.Group("/douyin")
	{
		public.POST("/user/register/", controller.Register)
		public.POST("/user/login/", controller.Login)
		public.GET("/feed/", controller.Feed)
		public.GET("/feed/ids/", controller.FeedIDs)
		public.GET("/feed/details/", controller.FeedDetails)
		public.POST("/feed/event/", controller.TrackVideoEvent)
		public.GET("/comment/list/", controller.GetComment)
	}

	// 私有接口（需要登录）
	private := r.Group("/douyin")
	private.Use(middleware.JWAuth())
	{
		private.POST("/publish/action/", controller.PublishVideo)
		private.DELETE("/publish/action/", controller.DeleteVideo)
		private.POST("/comment/action/", controller.CommentVideo)
		private.DELETE("/comment/action/", controller.DeleteComment)
		private.POST("/like/action/", controller.LikeVideo)
		private.POST("/favorite/action/", controller.FavoriteVideo)
		private.POST("/relation/action/", controller.FollowUser)

		private.GET("/user/info/", controller.GetUserInfo)
		private.GET("/user/video/list/", controller.GetUserVideoList)
		private.GET("/relation/follow/list/", controller.GetFollowList)
		private.GET("/relation/follower/list/", controller.GetFollowerList)
		private.POST("/user/avatar/", controller.UpdateAvatar)
		private.POST("/user/password/", controller.UpdatePassword)

	}
	return r
}
