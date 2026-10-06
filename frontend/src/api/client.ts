import axios from 'axios'

const TOKEN_KEY = 'ulas_token'
const USERNAME_KEY = 'ulas_username'
const PERMISSION_KEY = 'ulas_permission'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USERNAME_KEY)
      localStorage.removeItem(PERMISSION_KEY)
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export function clearStoredAuth(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USERNAME_KEY)
  localStorage.removeItem(PERMISSION_KEY)
}

// downloadScanReport fetches the PDF report as a blob and triggers a browser
// download. onProgress receives the download percentage (0-100); it is called
// with 0 when generation starts and 100 when the download is complete.
export async function downloadScanReport(
  scanId: string,
  onProgress?: (percent: number) => void,
): Promise<void> {
  onProgress?.(0)
  const response = await api.get(`/scans/${scanId}/report`, {
    responseType: 'blob',
    onDownloadProgress: (event) => {
      if (event.total) {
        onProgress?.(Math.min(99, Math.round((event.loaded / event.total) * 100)))
      }
    },
  })
  const blob = new Blob([response.data], { type: 'application/pdf' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', `scan-report-${scanId}.pdf`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
  onProgress?.(100)
}

export default api
