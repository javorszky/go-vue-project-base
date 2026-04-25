const base = import.meta.env.VITE_API_URL ?? ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base}${path}`, init)
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${path}`)
  return res.json() as Promise<T>
}

export interface HealthResponse {
  status: string
}

export function checkHealth(): Promise<HealthResponse> {
  return request<HealthResponse>('/api/v1/health')
}
