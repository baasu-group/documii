export interface SystemInfo {
  name: string;
  version: string;
  environment: string;
  airGapped: boolean;
}

export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail?: string;
  requestId?: string;
}
