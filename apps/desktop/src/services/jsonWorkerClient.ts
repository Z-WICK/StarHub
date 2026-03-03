import type { JsonWorkerResult } from '../types/models'

type ParseRequest = {
  id: string
  type: 'parse'
  payload: string
}

type StringifyRequest = {
  id: string
  type: 'stringify'
  payload: unknown
}

type WorkerRequest = ParseRequest | StringifyRequest

type WorkerResponse = {
  id: string
  type: WorkerRequest['type']
  result: JsonWorkerResult<unknown>
}

const worker = new Worker(new URL('../workers/json.worker.ts', import.meta.url), {
  type: 'module',
})

const pending = new Map<string, (response: WorkerResponse) => void>()

worker.onmessage = (event: MessageEvent<WorkerResponse>) => {
  const response = event.data
  const resolver = pending.get(response.id)
  if (!resolver) {
    return
  }
  pending.delete(response.id)
  resolver(response)
}

const postMessage = <T>(request: WorkerRequest): Promise<JsonWorkerResult<T>> => {
  return new Promise((resolve) => {
    pending.set(request.id, (response) => {
      resolve(response.result as JsonWorkerResult<T>)
    })
    worker.postMessage(request)
  })
}

const createRequestId = () => {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export const jsonWorkerClient = {
  async parse<T>(payload: string): Promise<JsonWorkerResult<T>> {
    return postMessage<T>({
      id: createRequestId(),
      type: 'parse',
      payload,
    })
  },
  async stringify<T>(payload: T): Promise<JsonWorkerResult<string>> {
    return postMessage<string>({
      id: createRequestId(),
      type: 'stringify',
      payload,
    })
  },
}
