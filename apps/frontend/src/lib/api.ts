import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_GATEWAY_URL ?? "http://localhost:8000/api/";

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10_000,
});

export interface FormField {
  id: string;
  name: string;
  label: string;
  type: string;
  required: boolean;
  order: number;
  metadata: Record<string, unknown>;
}

export interface Form {
  id: string;
  name: string;
  description: string;
  version: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  schema: {
    fields: FormField[];
  };
}

export interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  createdAt: string;
  updatedAt: string;
}

export interface Ticket {
  id: string;
  title: string;
  status: string;
  formId: string;
  assigneeId?: string;
  requesterId?: string;
  priority: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface TicketSubmission {
  id: string;
  clientReference: string;
  status: string;
  ticket?: Ticket;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface TicketQueueMetrics {
  pending: number;
  processing: number;
  completed: number;
  failed: number;
  oldestPendingSeconds: number;
}

export interface WorkflowDefinition {
  id: string;
  name: string;
  version: number;
  description: string;
  published: boolean;
  blueprint: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface ListResponse<T> {
  data: T[];
}

export interface ItemResponse<T> {
  data: T;
}

export interface OverviewResponse {
  data: {
    forms: { total: number };
    tickets: {
      total: number;
      byStatus: Record<string, number>;
      queue: TicketQueueMetrics;
    };
    users: { total: number };
    workflows: { total: number; published: number };
  };
}

export interface CreateFormPayload {
  name: string;
  description?: string;
  schema?: {
    fields?: Array<{
      name: string;
      label: string;
      type: string;
      required?: boolean;
      order?: number;
      metadata?: Record<string, unknown>;
    }>;
  };
}

export interface CreateTicketPayload {
  title: string;
  status?: string;
  formId: string;
  assigneeId?: string;
  priority?: string;
  metadata?: Record<string, unknown>;
  clientRequestId?: string;
}

export interface CreateWorkflowPayload {
  name: string;
  description?: string;
  blueprint: {
    steps?: Array<{
      id?: string;
      name?: string;
      type?: string;
    }>;
  };
}

function mapFormField(apiField: any): FormField {
  const fallbackId = `field-${Math.random().toString(36).slice(2, 10)}`;
  return {
    id: String(apiField.id ?? fallbackId),
    name: apiField.name,
    label: apiField.label,
    type: apiField.field_type,
    required: Boolean(apiField.required),
    order: Number(apiField.order ?? 0),
    metadata: apiField.metadata ?? {},
  };
}

function mapForm(apiForm: any): Form {
  return {
    id: String(apiForm.id),
    name: apiForm.name,
    description: apiForm.description ?? "",
    version: Number(apiForm.version ?? 1),
    isActive: Boolean(apiForm.is_active),
    createdAt: apiForm.created_at,
    updatedAt: apiForm.updated_at,
    schema: {
      fields: Array.isArray(apiForm.fields) ? apiForm.fields.map(mapFormField) : [],
    },
  };
}

function mapUser(apiUser: any): User {
  return {
    id: String(apiUser.id),
    name: apiUser.display_name ?? apiUser.name ?? "",
    email: apiUser.email,
    role: apiUser.role,
    createdAt: apiUser.created_at,
    updatedAt: apiUser.updated_at,
  };
}

function toNumber(value: any): number {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

function mapTicket(apiTicket: any): Ticket {
  return {
    id: String(apiTicket.id),
    title: apiTicket.title,
    status: apiTicket.status,
    formId: String(apiTicket.form_id),
    assigneeId: apiTicket.assignee_id ? String(apiTicket.assignee_id) : undefined,
    requesterId: apiTicket.requester_id ? String(apiTicket.requester_id) : undefined,
    priority: apiTicket.priority,
    metadata: apiTicket.payload ?? {},
    createdAt: apiTicket.created_at,
    updatedAt: apiTicket.updated_at,
  };
}

function mapTicketSubmission(apiSubmission: any): TicketSubmission {
  return {
    id: String(apiSubmission.id),
    clientReference: String(apiSubmission.client_reference),
    status: apiSubmission.status,
    ticket: apiSubmission.ticket ? mapTicket(apiSubmission.ticket) : undefined,
    errorMessage: apiSubmission.error_message ?? undefined,
    createdAt: apiSubmission.created_at,
    updatedAt: apiSubmission.updated_at,
    completedAt: apiSubmission.completed_at ?? undefined,
  };
}

function mapWorkflow(apiWorkflow: any): WorkflowDefinition {
  const steps = Array.isArray(apiWorkflow.steps) ? apiWorkflow.steps : [];
  const blueprint = apiWorkflow.definition ?? {
    steps: steps.map((step: any) => ({
      name: step.name,
      type: step.step_type,
    })),
  };

  return {
    id: String(apiWorkflow.id),
    name: apiWorkflow.name,
    version: Number(apiWorkflow.version ?? 1),
    description: apiWorkflow.description ?? "",
    published: Boolean(apiWorkflow.is_active),
    blueprint,
    createdAt: apiWorkflow.created_at,
    updatedAt: apiWorkflow.updated_at,
  };
}

export async function listForms(): Promise<ListResponse<Form>> {
  const { data } = await apiClient.get("forms/");
  return { data: Array.isArray(data) ? data.map(mapForm) : [] };
}

export async function createForm(payload: CreateFormPayload): Promise<ItemResponse<Form>> {
  const fields = payload.schema?.fields ?? [];
  const response = await apiClient.post("forms/", {
    name: payload.name,
    description: payload.description ?? "",
    fields: fields.map((field, index) => ({
      name: field.name,
      label: field.label,
      field_type: field.type,
      required: field.required ?? false,
      order: field.order ?? index,
      metadata: field.metadata ?? {},
    })),
  });
  return { data: mapForm(response.data) };
}

export async function listUsers(): Promise<ListResponse<User>> {
  const { data } = await apiClient.get("users/");
  return { data: Array.isArray(data) ? data.map(mapUser) : [] };
}

export async function listTickets(): Promise<ListResponse<Ticket>> {
  const { data } = await apiClient.get("tickets/");
  return { data: Array.isArray(data) ? data.map(mapTicket) : [] };
}

export async function submitTicket(
  payload: CreateTicketPayload,
): Promise<ItemResponse<TicketSubmission>> {
  const response = await apiClient.post("tickets/submissions/", {
    title: payload.title,
    description: "",
    form_id: Number(payload.formId),
    assignee_id: payload.assigneeId ? Number(payload.assigneeId) : null,
    priority: payload.priority ?? "medium",
    status: payload.status ?? "open",
    payload: payload.metadata ?? {},
    client_reference: payload.clientRequestId ?? undefined,
  });
  return { data: mapTicketSubmission(response.data) };
}

export async function getTicketSubmission(
  submissionId: string,
): Promise<ItemResponse<TicketSubmission>> {
  const response = await apiClient.get(`tickets/submissions/${submissionId}/`);
  return { data: mapTicketSubmission(response.data) };
}

export async function resolveTicket(id: string): Promise<ItemResponse<Ticket>> {
  const response = await apiClient.post(`tickets/${id}/resolve/`);
  return { data: mapTicket(response.data) };
}

export async function getTicketQueueMetrics(): Promise<TicketQueueMetrics> {
  const { data } = await apiClient.get("tickets/queue-metrics/");
  return {
    pending: toNumber(data.pending),
    processing: toNumber(data.processing),
    completed: toNumber(data.completed),
    failed: toNumber(data.failed),
    oldestPendingSeconds: toNumber(data.oldestPendingSeconds),
  };
}

export async function listWorkflows(): Promise<ListResponse<WorkflowDefinition>> {
  const { data } = await apiClient.get("workflows/");
  return { data: Array.isArray(data) ? data.map(mapWorkflow) : [] };
}

export async function createWorkflow(payload: CreateWorkflowPayload): Promise<ItemResponse<WorkflowDefinition>> {
  const steps = payload.blueprint.steps ?? [];
  const response = await apiClient.post("workflows/", {
    name: payload.name,
    description: payload.description ?? "",
    definition: payload.blueprint,
    steps: steps.map((step, index) => ({
      name: step.name ?? step.type ?? `步骤${index + 1}`,
      step_type: step.type ?? "manual",
      sequence: index + 1,
      metadata: { type: step.type ?? "manual" },
    })),
  });
  return { data: mapWorkflow(response.data) };
}

export async function publishWorkflow(id: string): Promise<ItemResponse<WorkflowDefinition>> {
  const response = await apiClient.post(`workflows/${id}/publish/`);
  return { data: mapWorkflow(response.data) };
}

export async function getOverview(): Promise<OverviewResponse> {
  const { data } = await apiClient.get("overview/");
  return { data };
}
