import { describe, expect, it } from 'vitest'

import { buildCcsImportDeeplink, OPENAI_CCS_DEFAULT_MODEL } from '../ccswitchImport'

function getSearchParams(url: string): URLSearchParams {
  return new URL(url).searchParams
}

describe('buildCcsImportDeeplink', () => {
  it('sets the OpenAI CCS default model to gpt-5.4', () => {
    const params = getSearchParams(buildCcsImportDeeplink({
      baseUrl: 'https://api.example.com',
      apiKey: 'sk-test',
      providerName: 'sub2api',
      platform: 'openai',
      clientType: 'claude',
      usageScriptBase64: 'dGVzdA==',
    }))

    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe('https://api.example.com')
    expect(params.get('model')).toBe(OPENAI_CCS_DEFAULT_MODEL)
  })

  it('keeps non-OpenAI imports unchanged', () => {
    const params = getSearchParams(buildCcsImportDeeplink({
      baseUrl: 'https://api.example.com',
      apiKey: 'sk-test',
      providerName: 'sub2api',
      platform: 'anthropic',
      clientType: 'claude',
      usageScriptBase64: 'dGVzdA==',
    }))

    expect(params.get('app')).toBe('claude')
    expect(params.get('model')).toBeNull()
  })
})
