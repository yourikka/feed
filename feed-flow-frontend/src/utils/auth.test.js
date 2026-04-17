import test from 'node:test'
import assert from 'node:assert/strict'
import { validatePassword, validateUsername } from './auth.js'

test('validateUsername should accept legal username', () => {
  assert.equal(validateUsername('feed_user01'), '')
})

test('validateUsername should reject invalid username', () => {
  assert.equal(validateUsername('ab'), '用户名长度需在 3 到 20 个字符之间')
  assert.equal(validateUsername('bad name'), '用户名只能包含字母、数字和下划线')
})

test('validatePassword should enforce password length', () => {
  assert.equal(validatePassword('1234567'), '密码长度需在 8 到 64 个字符之间')
  assert.equal(validatePassword('12345678'), '')
})
