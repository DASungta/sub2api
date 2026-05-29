export const OPENAI_CCS_DEFAULT_MODEL = 'gpt-5.4'

export interface BuildCcsImportDeeplinkInput {
  baseUrl: string
  apiKey: string
  providerName: string
  platform?: string | null
  clientType: 'claude' | 'gemini'
  usageScriptBase64: string
}

export function buildCcsImportDeeplink({
  baseUrl,
  apiKey,
  providerName,
  platform,
  clientType,
  usageScriptBase64,
}: BuildCcsImportDeeplinkInput): string {
  const normalizedPlatform = platform || 'anthropic'
  let app: string
  let endpoint: string

  if (normalizedPlatform === 'antigravity') {
    app = clientType === 'gemini' ? 'gemini' : 'claude'
    endpoint = `${baseUrl}/antigravity`
  } else {
    switch (normalizedPlatform) {
      case 'openai':
        app = 'codex'
        endpoint = baseUrl
        break
      case 'gemini':
        app = 'gemini'
        endpoint = baseUrl
        break
      default:
        app = 'claude'
        endpoint = baseUrl
        break
    }
  }

  const params = new URLSearchParams({
    resource: 'provider',
    app,
    name: providerName,
    homepage: baseUrl,
    endpoint,
    apiKey,
    configFormat: 'json',
    usageEnabled: 'true',
    usageScript: usageScriptBase64,
    usageAutoInterval: '30',
  })

  if (normalizedPlatform === 'openai') {
    params.set('model', OPENAI_CCS_DEFAULT_MODEL)
  }

  return `ccswitch://v1/import?${params.toString()}`
}
