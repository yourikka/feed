-- 1) 去重，保留最早一条记录
DELETE l1
FROM likes l1
JOIN likes l2
  ON l1.video_id = l2.video_id
 AND l1.user_id = l2.user_id
 AND l1.id > l2.id;

DELETE f1
FROM favorites f1
JOIN favorites f2
  ON f1.video_id = f2.video_id
 AND f1.user_id = f2.user_id
 AND f1.id > f2.id;

DELETE r1
FROM follows r1
JOIN follows r2
  ON r1.user_id = r2.user_id
 AND r1.target_user_id = r2.target_user_id
 AND r1.id > r2.id;

-- 2) 索引与唯一约束
ALTER TABLE likes
  ADD UNIQUE KEY uk_likes_video_user (video_id, user_id),
  ADD KEY idx_likes_video (video_id),
  ADD KEY idx_likes_user (user_id);

ALTER TABLE favorites
  ADD UNIQUE KEY uk_favorites_video_user (video_id, user_id),
  ADD KEY idx_favorites_video (video_id),
  ADD KEY idx_favorites_user (user_id);

ALTER TABLE follows
  ADD UNIQUE KEY uk_follows_user_target (user_id, target_user_id),
  ADD KEY idx_follows_target (target_user_id);

ALTER TABLE comments
  ADD KEY idx_comments_video_created (video_id, created_at);

ALTER TABLE videos
  ADD KEY idx_videos_author (author_id),
  ADD KEY idx_videos_created_id (created_at, id);

-- 3) 行为事件幂等约束
ALTER TABLE video_behavior_events
  ADD COLUMN event_id varchar(96) NOT NULL DEFAULT '' AFTER deleted_at;

UPDATE video_behavior_events
SET event_id = CASE
  WHEN request_id <> '' THEN CONCAT('req:', request_id)
  ELSE CONCAT(
    'evt:',
    user_id, ':',
    viewer_key, ':',
    video_id, ':',
    event_type, ':',
    progress_ms, ':',
    duration_ms, ':',
    position_ms
  )
END
WHERE event_id = '';

DELETE b1
FROM video_behavior_events b1
JOIN video_behavior_events b2
  ON b1.event_id = b2.event_id
 AND b1.event_id <> ''
 AND b1.id > b2.id;

ALTER TABLE video_behavior_events
  ADD UNIQUE KEY uk_behavior_event_id (event_id),
  ADD KEY idx_behavior_request_id (request_id);
