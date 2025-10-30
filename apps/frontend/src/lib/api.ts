import axios from "axios";

export const apiClient = axios.create({
  baseURL: "/api",
  timeout: 10_000,
});

export interface PaginatedResponse<T> {
  data: T[];
  nextCursor?: string;
}

export async function listForms() {
  const { data } = await apiClient.get<PaginatedResponse<any>>("/forms");
  return data;
}

export async function listTickets() {
  const { data } = await apiClient.get<PaginatedResponse<any>>("/tickets");
  return data;
}
