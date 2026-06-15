import type {
  ApiErrorBody,
  Assignee,
  AssigneeInput,
  Dashboard,
  Ticket,
  TicketInput,
  TicketSort,
  TicketStatus,
} from '../types/ticket'

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly fields: Record<string, string> = {},
  ) {
    super(message)
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const body = (await response.json().catch(() => ({
      message: 'Não foi possível concluir a operação.',
    }))) as ApiErrorBody
    throw new ApiError(body.message, response.status, body.fields)
  }

  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

export interface TicketFilters {
  search?: string
  status?: string
  priority?: string
  assigneeId?: number
  sort?: TicketSort
  limit?: number
}

export const api = {
  dashboard: () => request<Dashboard>('/api/dashboard'),
  assignees: () => request<Assignee[]>('/api/assignees'),
  createAssignee: (input: AssigneeInput) =>
    request<Assignee>('/api/assignees', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  updateAssignee: (id: number, input: AssigneeInput) =>
    request<Assignee>(`/api/assignees/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),
  deleteAssignee: (id: number) =>
    request<void>(`/api/assignees/${id}`, {
      method: 'DELETE',
    }),
  tickets(filters: TicketFilters = {}) {
    const params = new URLSearchParams()
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== '' && value !== 0) {
        params.set(key, String(value))
      }
    })
    const query = params.size ? `?${params.toString()}` : ''
    return request<Ticket[]>(`/api/tickets${query}`)
  },
  ticket: (id: number) => request<Ticket>(`/api/tickets/${id}`),
  createTicket: (input: TicketInput) =>
    request<Ticket>('/api/tickets', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  updateTicket: (id: number, input: TicketInput) =>
    request<Ticket>(`/api/tickets/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    }),
  changeTicketStatus: (id: number, status: TicketStatus) =>
    request<Ticket>(`/api/tickets/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    }),
}
