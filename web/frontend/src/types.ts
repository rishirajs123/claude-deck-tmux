export interface Session {
  id: string
  cwd: string
  project: string
  created_at: number
  last_used_at: number
  duration_secs: number
  prompt_count: number
  message_count: number
  model: string
  git_branch: string
  first_prompt: string
  tokens_in: number
  tokens_out: number
  current_task: string
  status: 'running' | 'ended'
  favorite: boolean
  tags: string
  notes: string
  cpu: number
  mem_mb: number
}

export interface Stats {
  total: number
  running: number
  prompts: number
  projects: number
  tokens_in: number
  tokens_out: number
}

export interface Agg { key: string; sessions: number; prompts: number; tokens: number }
export interface Day { day: string; prompts: number }
export interface Analytics { daily: Day[]; topProjects: Agg[]; models: Agg[] }

export type Action = 'focus' | 'resume' | 'new' | 'kill' | 'reveal' | 'send'
