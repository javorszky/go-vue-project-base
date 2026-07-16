import { afterEach, describe, expect, it, vi } from 'vitest'
import { checkHealth, getStatus } from './client'

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })
}

describe('api client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('checkHealth returns the parsed health payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: 'ok' }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(checkHealth()).resolves.toEqual({ status: 'ok' })
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/health', expect.anything())
  })

  it('getStatus returns build metadata', async () => {
    const payload = { status: 'ok', git_sha: 'abc1234', build_time: '2026-01-01T00:00:00Z' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(payload)))

    await expect(getStatus()).resolves.toEqual(payload)
  })

  it('rejects on non-2xx responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('nope', { status: 503 })))

    await expect(checkHealth()).rejects.toThrow('HTTP 503')
  })

  it('rejects when the response is not JSON', async () => {
    const html = new Response('<!doctype html>', {
      status: 200,
      headers: { 'content-type': 'text/html' },
    })
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(html))

    await expect(getStatus()).rejects.toThrow('Expected JSON')
  })
})
