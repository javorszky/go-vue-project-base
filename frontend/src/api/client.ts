const base = import.meta.env.VITE_API_URL ?? ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const timeout = AbortSignal.timeout(10_000)
  const signal = init?.signal ? AbortSignal.any([timeout, init.signal]) : timeout
  const res = await fetch(`${base}${path}`, { ...init, signal })
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${path}`)
  const ct = res.headers.get('content-type')
  if (!ct?.includes('application/json')) {
    throw new Error(`Expected JSON from ${path}, got: ${ct ?? 'no content-type'}`)
  }
  return res.json() as Promise<T>
}

export interface HealthResponse {
  status: string
}

export function checkHealth(): Promise<HealthResponse> {
  return request<HealthResponse>('/api/v1/health')
}

export interface StatusResponse {
  status: string
  git_sha: string
  build_time: string
}

export function getStatus(): Promise<StatusResponse> {
  return request<StatusResponse>('/api/v1/status')
}
