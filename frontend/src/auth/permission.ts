const TOKEN_KEY = 'ulas_token'
export const PERMISSION_STORAGE_KEY = 'ulas_permission'

export type Permission = 'read' | 'admin'

function parsePermission(value: unknown): Permission | null {
  return value === 'read' || value === 'admin' ? value : null
}

function permissionFromToken(token: string): Permission | null {
  const parts = token.split('.')
  if (parts.length < 2) return null
  try {
    const base64Url = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const base64 = base64Url + '='.repeat((4 - (base64Url.length % 4)) % 4)
    const payload = JSON.parse(atob(base64))
    return parsePermission(payload?.permission)
  } catch {
    return null
  }
}

export function getPermission(): Permission | null {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    const fromToken = permissionFromToken(token)
    if (fromToken) return fromToken
  }
  return parsePermission(localStorage.getItem(PERMISSION_STORAGE_KEY))
}

export function isAdmin(): boolean {
  return getPermission() === 'admin'
}
