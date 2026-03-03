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

self.onmessage = (event: MessageEvent<WorkerRequest>) => {
  const request = event.data
  const response: WorkerResponse = {
    id: request.id,
    type: request.type,
    result: { success: false, data: null, error: '未知错误' },
  }

  try {
    if (request.type === 'parse') {
      response.result = {
        success: true,
        data: JSON.parse(request.payload),
        error: null,
      }
    } else {
      response.result = {
        success: true,
        data: JSON.stringify(request.payload, null, 2),
        error: null,
      }
    }
  } catch {
    response.result = {
      success: false,
      data: null,
      error: request.type === 'parse' ? 'JSON 格式无效' : '导出 JSON 失败',
    }
  }

  self.postMessage(response)
}
