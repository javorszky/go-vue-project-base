import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatusCard from './StatusCard.vue'

describe('StatusCard', () => {
  it('shows build info when both probes succeed', () => {
    const wrapper = mount(StatusCard, {
      props: {
        health: 'ok',
        buildStatus: 'ok',
        buildInfo: { status: 'ok', git_sha: 'abc1234', build_time: '2026-01-01T00:00:00Z' },
      },
    })

    expect(wrapper.text()).toContain('healthy')
    expect(wrapper.text()).toContain('abc1234')
    expect(wrapper.text()).toContain('2026-01-01T00:00:00Z')
  })

  it('reports an unreachable API and unavailable build info', () => {
    const wrapper = mount(StatusCard, {
      props: { health: 'error', buildStatus: 'error', buildInfo: null },
    })

    expect(wrapper.text()).toContain('unreachable')
    expect(wrapper.text()).toContain('unavailable')
  })

  it('collapses build info when the trigger is clicked', async () => {
    const wrapper = mount(StatusCard, {
      props: {
        health: 'ok',
        buildStatus: 'ok',
        buildInfo: { status: 'ok', git_sha: 'abc1234', build_time: '2026-01-01T00:00:00Z' },
      },
    })

    expect(wrapper.text()).toContain('abc1234')
    await wrapper.get('button').trigger('click')
    expect(wrapper.text()).not.toContain('abc1234')
  })
})
