# Feed Flow Frontend

## Scripts

- `npm run dev`: 启动开发服务
- `npm run build`: 生产构建
- `npm run preview`: 本地预览构建产物
- `npm run test`: 运行前端单元测试

## Notes

- 默认通过 Vite 代理请求 `/douyin` 和 `/uploads` 到 `http://localhost:8080`
- 路由采用懒加载分包，减少首屏 JS 体积
