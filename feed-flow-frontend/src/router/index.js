import { createRouter, createWebHashHistory } from 'vue-router'

import Login from '../views/Login.vue'
import Feed from '../views/Feed.vue'
import Publish from '../views/Publish.vue'
import Profile from '../views/Profile.vue'
import RelationList from '../views/RelationList.vue'
import Settings from '../views/Settings.vue'

const routes = [
  {
    path: '/',
    redirect: '/feed'
  },
  {
    path: '/login',
    name: 'Login',
    component: Login,
    meta: {
      title: '登录',
      requireAuth: false
    }
  },
  {
    path: '/feed',
    name: 'Feed',
    component: Feed,
    meta: {
      title: '首页',
      requireAuth: false,
      showTabBar: true,
      tabKey: 'feed'
    }
  },
  {
    path: '/publish',
    name: 'Publish',
    component: Publish,
    meta: {
      title: '发布视频',
      requireAuth: true,
      showTabBar: true,
      tabKey: 'publish'
    }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
    meta: {
      title: '个人中心',
      requireAuth: true,
      showTabBar: true,
      tabKey: 'profile'
    }
  },
  {
    path: '/profile/following',
    name: 'FollowingList',
    component: RelationList,
    meta: {
      title: '我的关注',
      requireAuth: true,
      showTabBar: true,
      tabKey: 'profile',
      relationType: 'following'
    }
  },
  {
    path: '/profile/fans',
    name: 'FollowerList',
    component: RelationList,
    meta: {
      title: '我的粉丝',
      requireAuth: true,
      showTabBar: true,
      tabKey: 'profile',
      relationType: 'followers'
    }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: Settings,
    meta: {
      title: '设置',
      requireAuth: true,
      showTabBar: true,
      tabKey: 'profile'
    }
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/feed'
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = to.meta.title || 'Feed流Demo'

  const requireAuth = to.meta.requireAuth
  const isLogin = !!localStorage.getItem('token')

  if (requireAuth && !isLogin) {
    next('/login')
  } else {
    next()
  }
})

export default router
