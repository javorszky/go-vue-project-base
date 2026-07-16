import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import App from './App.vue'

vi.mock('./api/client', () => ({
  checkHealth: vi.fn().mockResolvedValue({ status: 'ok' }),
  getStatus: vi
    .fn()
    .mockResolvedValue({ status: 'ok', git_sha: 'abc1234', build_time: '2026-01-01T00:00:00Z' }),
}))

describe('App', () => {
  it('probes the API on mount and renders the results', async () => {
    const wrapper = mount(App)
    await flushPromises()

    expect(wrapper.text()).toContain('healthy')
    expect(wrapper.text()).toContain('abc1234')
  })
})
