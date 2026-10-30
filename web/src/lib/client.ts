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
  FlowsApi,
  FriendsApi,
  HaApi,
  InstallApi,
  LogsApi,
  MacrosApi,
  NatApi,
  RoutingApi,
  SetupApi,
  SnapshotsApi,
  StatsApi,
  SwitchesApi,
  SystemApi,
  ToolsApi,
  UiApi,
  WebApi,
  WireguardApi,
} from '../api'
import { getToken, clearToken } from './utils'
import { activeBaseUrl, subscribeInstance } from './instance'

type BodyShape<T> = { result?: T; error?: string }

type FieldsShape = { field?: string; message?: string }

type ResultShape = { result?: unknown }

export interface StagingState {
  pending: boolean
}

type StagingListener = (state: StagingState) => void
const stagingListeners = new Set<StagingListener>()
let currentStagingState: StagingState = { pending: false }

function publishStagingState(state: StagingState): void {
  currentStagingState = state
  stagingListeners.forEach((fn) => fn(state))
}

let stagingSeedOpen = true

function resetStagingSeed(): void {
  stagingSeedOpen = true
}

subscribeInstance(resetStagingSeed)

function applyStagingHeader(response: Response, method?: string): void {
  const staging = response.headers.get('X-Staging-Pending')
  if (staging === null) return

  const mutating = !!method && method.toUpperCase() !== 'GET'
  if (!mutating && !stagingSeedOpen) return
  if (mutating) stagingSeedOpen = false

  publishStagingState({ pending: staging === 'true' })
}

export function addStagingListener(fn: StagingListener): () => void {
  stagingListeners.add(fn)
  fn(currentStagingState)
  return () => stagingListeners.delete(fn)
}

export function getStagingState(): StagingState {
  return currentStagingState
}

function handleAuthenticationResponse(response: Response): Response {
  const authEndpoint =
    response.url.endsWith('/api/auth/login') || response.url.endsWith('/api/auth/password')
  if (response.status === 401 && !authEndpoint) {
    clearToken()
    window.location.href = '/login'
  }
  return response
}

const configuration = new Configuration({
  get basePath() {
    return activeBaseUrl()
  },
  apiKey: () => {
    const token = getToken()
    return token ? `Bearer ${token}` : ''
  },
  middleware: [
    {
      async post({ init, response }) {
        applyStagingHeader(response, init.method)
        return handleAuthenticationResponse(response)
      },
    },
  ],
})

const selfConfiguration = new Configuration({
  basePath: '',
  apiKey: () => {
    const token = getToken()
    return token ? `Bearer ${token}` : ''
  },
  middleware: [{
    async post({ response }) {
      return handleAuthenticationResponse(response)
    },
  }],
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

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const response = await fetch(`${activeBaseUrl()}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  })
  applyStagingHeader(response, init?.method)
  const body = await response.json().catch(() => undefined) as BodyShape<T> | undefined
  if (!response.ok) throw new ApiError(body?.error || `HTTP ${response.status}`, response.status, body)
  return body?.result as T
}

type ErrorBody = {
  error?: string
  message?: string
  status?: string
  fields?: FieldsShape[]
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
      if (res && typeof res === 'object' && 'result' in res) return (res as ResultShape).result
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
    Operations<FlowsApi> &
    Operations<FriendsApi> &
    Operations<HaApi> &
    Operations<InstallApi> &
    Operations<LogsApi> &
    Operations<MacrosApi> &
    Operations<NatApi> &
    Operations<RoutingApi> &
    Operations<SetupApi> &
    Operations<SnapshotsApi> &
    Operations<StatsApi> &
    Operations<SwitchesApi> &
    Operations<SystemApi> &
    Operations<ToolsApi> &
    Operations<UiApi> &
    Operations<WebApi> &
    Operations<WireguardApi>
>

export const api = [
  new AnnouncementsApi(configuration),
  new ApplyApi(configuration),
  new AuthApi(configuration),
  new BackupApi(configuration),
  new ConfigApi(configuration),
  new DhcpApi(configuration),
  new FlowsApi(configuration),
  new FriendsApi(configuration),
  new HaApi(configuration),
  new InstallApi(configuration),
  new LogsApi(configuration),
  new MacrosApi(configuration),
  new NatApi(configuration),
  new RoutingApi(configuration),
  new SetupApi(configuration),
  new SnapshotsApi(configuration),
  new StatsApi(configuration),
  new SwitchesApi(configuration),
  new SystemApi(configuration),
  new ToolsApi(configuration),
  new UiApi(configuration),
  new WebApi(configuration),
  new WireguardApi(configuration),
].reduce<Record<string, unknown>>((acc, inst) => Object.assign(acc, operations(inst)), {}) as Api

type SelfApi = Unwrapped<Operations<AuthApi> & Operations<ConfigApi> & Operations<FriendsApi>>

export const selfApi = [
  new AuthApi(selfConfiguration),
  new ConfigApi(selfConfiguration),
  new FriendsApi(selfConfiguration),
].reduce<Record<string, unknown>>((acc, inst) => Object.assign(acc, operations(inst)), {}) as SelfApi
