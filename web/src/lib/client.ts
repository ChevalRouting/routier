import {
  Configuration,
  FetchError,
  ResponseError,
  AnnouncementsApi,
  ApplyApi,
  AuthApi,
  BackupApi,
  ConfigApi,
  DhcpApi,
  FriendsApi,
  HaApi,
  LogsApi,
  MacrosApi,
  NatApi,
  RoutingApi,
  SetupApi,
  SnapshotsApi,
  StatsApi,
  ToolsApi,
  UiApi,
  WireguardApi,
} from '../api'
import { getToken, clearToken } from './utils'

type StagingListener = (pending: boolean) => void
const stagingListeners = new Set<StagingListener>()

export function addStagingListener(fn: StagingListener): () => void {
  stagingListeners.add(fn)
  return () => stagingListeners.delete(fn)
}

const configuration = new Configuration({
  basePath: '',
  apiKey: () => {
    const token = getToken()
    return token ? `Bearer ${token}` : ''
  },
  middleware: [
    {
      async post({ response }) {
        const staging = response.headers.get('X-Staging-Pending')
        if (staging !== null) stagingListeners.forEach((fn) => fn(staging === 'true'))
        if (response.status === 401 && !response.url.endsWith('/api/auth/login')) {
          clearToken()
          window.location.href = '/login'
        }
        return response
      },
    },
  ],
})

export class ApiError extends Error {
  status: number
  body: unknown

  constructor(message: string, status: number, body: unknown) {
    super(message)
    this.status = status
    this.body = body
  }
}

type ErrorBody = {
  error?: string
  message?: string
  status?: string
  fields?: { field?: string; message?: string }[]
}

async function toApiError(err: unknown): Promise<unknown> {
  if (err instanceof ResponseError) {
    let body: unknown
    try {
      body = await err.response.clone().json()
    } catch {
      body = undefined
    }
    const e = (typeof body === 'object' && body !== null ? body : {}) as ErrorBody
    let message = e.error || e.message || e.status || `HTTP ${err.response.status}`
    if (Array.isArray(e.fields) && e.fields.length > 0) {
      const details = e.fields
        .map((f) => (f.field ? `${f.field}: ${f.message || 'invalid value'}` : f.message || 'invalid value'))
        .join('; ')
      message = `${message}: ${details}`
    }
    return new ApiError(message, err.response.status, body)
  }
  if (err instanceof FetchError) {
    return new ApiError('Could not reach the appliance (connection failed or was dropped); the operation may still have completed', 0, undefined)
  }
  return err
}

type Payload<R> = R extends object ? ('result' extends keyof R ? NonNullable<R['result']> : R) : R
type Operations<T> = { [K in keyof T as K extends `api${string}` ? K : never]: T[K] }
type Unwrapped<T> = {
  [K in keyof T]: T[K] extends (...args: infer A) => Promise<infer R> ? (...args: A) => Promise<Payload<R>> : T[K]
}

function operations(instance: object): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const name of Object.getOwnPropertyNames(Object.getPrototypeOf(instance))) {
    const fn = (instance as Record<string, unknown>)[name]
    if (!name.startsWith('api') || typeof fn !== 'function') continue
    out[name] = async (...args: unknown[]) => {
      let res: unknown
      try {
        res = await (fn as (...a: unknown[]) => Promise<unknown>).apply(instance, args)
      } catch (err) {
        throw await toApiError(err)
      }
      if (res && typeof res === 'object' && 'result' in res) return (res as { result?: unknown }).result
      return res
    }
  }
  return out
}

type Api = Unwrapped<
  Operations<AnnouncementsApi> &
    Operations<ApplyApi> &
    Operations<AuthApi> &
    Operations<BackupApi> &
    Operations<ConfigApi> &
    Operations<DhcpApi> &
    Operations<FriendsApi> &
    Operations<HaApi> &
    Operations<LogsApi> &
    Operations<MacrosApi> &
    Operations<NatApi> &
    Operations<RoutingApi> &
    Operations<SetupApi> &
    Operations<SnapshotsApi> &
    Operations<StatsApi> &
    Operations<ToolsApi> &
    Operations<UiApi> &
    Operations<WireguardApi>
>

export const api = [
  new AnnouncementsApi(configuration),
  new ApplyApi(configuration),
  new AuthApi(configuration),
  new BackupApi(configuration),
  new ConfigApi(configuration),
  new DhcpApi(configuration),
  new FriendsApi(configuration),
  new HaApi(configuration),
  new LogsApi(configuration),
  new MacrosApi(configuration),
  new NatApi(configuration),
  new RoutingApi(configuration),
  new SetupApi(configuration),
  new SnapshotsApi(configuration),
  new StatsApi(configuration),
  new ToolsApi(configuration),
  new UiApi(configuration),
  new WireguardApi(configuration),
].reduce<Record<string, unknown>>((acc, inst) => Object.assign(acc, operations(inst)), {}) as Api
