import axios from "axios";

export const apiClient = axios.create({
  baseURL: "/api",
  timeout: 10_000,
});

export interface FormSchema {
  [key: string]: unknown;
}

export interface Form {
  id: string;
  name: string;
  description: string;
  schema: FormSchema;
  createdAt: string;
  updatedAt: string;
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
  assigneeId: string;
  priority: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
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
    tickets: { total: number; byStatus: Record<string, number> };
    users: { total: number };
    workflows: { total: number; published: number };
  };
}

export interface CreateFormPayload {
  name: string;
  description?: string;
  schema?: FormSchema;
}

export interface CreateTicketPayload {
  title: string;
  status?: string;
  formId: string;
  assigneeId?: string;
  priority?: string;
  metadata?: Record<string, unknown>;
}

export interface CreateWorkflowPayload {
  name: string;
  description?: string;
  version?: number;
  blueprint: Record<string, unknown>;
}

export async function listForms() {
  const { data } = await apiClient.get<ListResponse<Form>>("/forms");
  return data;
}

export async function createForm(payload: CreateFormPayload) {
  const { data } = await apiClient.post<ItemResponse<Form>>("/forms", payload);
  return data;
}

export async function listUsers() {
  const { data } = await apiClient.get<ListResponse<User>>("/users");
  return data;
}

export async function listTickets(params?: { status?: string; assigneeId?: string }) {
  const { data } = await apiClient.get<ListResponse<Ticket>>("/tickets", { params });
  return data;
}

export async function createTicket(payload: CreateTicketPayload) {
  const { data } = await apiClient.post<ItemResponse<Ticket>>("/tickets", payload);
  return data;
}

export async function resolveTicket(id: string) {
  const { data } = await apiClient.post<ItemResponse<Ticket>>(`/tickets/${id}/resolve`);
  return data;
}

export async function listWorkflows(params?: { published?: boolean }) {
  const { data } = await apiClient.get<ListResponse<WorkflowDefinition>>("/workflows", { params });
  return data;
}

export async function createWorkflow(payload: CreateWorkflowPayload) {
  const { data } = await apiClient.post<ItemResponse<WorkflowDefinition>>("/workflows", payload);
  return data;
}

export async function publishWorkflow(id: string) {
  const { data } = await apiClient.post<ItemResponse<WorkflowDefinition>>(`/workflows/${id}/publish`);
  return data;
}

export async function getOverview() {
  const { data } = await apiClient.get<OverviewResponse>("/overview");
  return data;
}
