// Auth types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'editor' | 'viewer'
  avatar?: string
  tenantId: string
}

export interface AuthToken {
  accessToken: string
  refreshToken?: string
  expiresIn: number
}

// Data source types
export type DataSourceType = 'database' | 'datalake' | 'api' | 'file' | 'streaming'
export type DataSourceStatus = 'connected' | 'disconnected' | 'scanning' | 'error'

export interface DataSource {
  id: string
  name: string
  type: DataSourceType
  status: DataSourceStatus
  lastScan?: Date
  recordCount: number
  health: number
  createdAt: Date
  updatedAt: Date
}

export interface Dataset {
  id: string
  dataSourceId: string
  name: string
  description?: string
  recordCount: number
  columnCount: number
  discoveredAt: Date
  classficationStatus: 'pending' | 'in_progress' | 'completed'
}

// Classification types
export type SensitivityLevel = 'public' | 'internal' | 'confidential' | 'restricted'

export interface ClassificationResult {
  id: string
  datasetId: string
  columnName: string
  sensitivity: SensitivityLevel
  confidence: number
  classifiedBy: 'model' | 'rule' | 'manual'
  createdAt: Date
}

export interface ClassificationRule {
  id: string
  name: string
  pattern?: string
  keywords?: string[]
  sensitivity: SensitivityLevel
  enabled: boolean
  order: number
}

export interface ClassificationModel {
  id: string
  name: string
  version: string
  accuracy: number
  installed: boolean
  createdAt: Date
}

// Policy types
export type PolicyType = 'access_control' | 'redaction' | 'masking' | 'anonymization'

export interface Policy {
  id: string
  name: string
  description?: string
  type: PolicyType
  regulations: string[]
  conditions: PolicyCondition[]
  actions: PolicyAction[]
  enabled: boolean
  createdAt: Date
  updatedAt: Date
}

export interface PolicyCondition {
  id: string
  field: string
  operator: string
  value: any
}

export interface PolicyAction {
  id: string
  type: string
  config?: Record<string, any>
}

// AI Gate types
export interface AIGateRequest {
  id: string
  query: string
  dataset: string
  user: string
  decision: 'allowed' | 'blocked'
  reason?: string
  processingTime: number
  createdAt: Date
}

export interface AIGateConfig {
  llmEndpoint: string
  vectorDb: string
  governanceMode: 'strict' | 'balanced' | 'permissive'
  rateLimit: number
}

// Quality types
export interface DataQuality {
  datasetId: string
  completeness: number
  accuracy: number
  freshness: number
  consistency: number
  uniqueness: number
  validity: number
  overall: number
  lastChecked: Date
}

// Privacy types
export interface ComplianceScore {
  gdpr: number
  ccpa: number
  hipaa: number
  dpdp: number
  overall: number
}

export interface DSAR {
  id: string
  dataSubjectId: string
  requestType: 'access' | 'rectification' | 'deletion' | 'portability'
  status: 'pending' | 'in_progress' | 'completed' | 'rejected'
  description?: string
  dueDate: Date
  completedAt?: Date
  createdAt: Date
}

// Audit types
export interface AuditLog {
  id: string
  user: string
  action: string
  resource: string
  resourceId: string
  changes?: Record<string, any>
  timestamp: Date
  ip?: string
}

// Settings types
export interface TenantSettings {
  id: string
  name: string
  logo?: string
  governanceMode: 'strict' | 'balanced' | 'permissive'
  timezone: string
  createdAt: Date
  updatedAt: Date
}

export interface ApiKey {
  id: string
  name: string
  key: string
  lastUsed?: Date
  createdAt: Date
}

export interface Webhook {
  id: string
  name: string
  url: string
  events: string[]
  active: boolean
  createdAt: Date
}

// Notification types
export interface Notification {
  id: string
  type: 'alert' | 'warning' | 'info' | 'success'
  title: string
  message: string
  read: boolean
  actionUrl?: string
  createdAt: Date
}

// Alert types
export interface Alert {
  id: string
  severity: 'critical' | 'warning' | 'info'
  message: string
  source: string
  sourceId?: string
  resolved: boolean
  createdAt: Date
  resolvedAt?: Date
}

// Lineage types
export interface LineageNode {
  id: string
  type: 'dataset' | 'process' | 'policy' | 'model'
  name: string
  description?: string
  x: number
  y: number
}

export interface LineageEdge {
  id: string
  source: string
  target: string
  type: 'input' | 'output' | 'governance'
}

// Job types
export interface Job {
  id: string
  name: string
  type: 'scan' | 'classify' | 'quality_check' | 'export' | 'audit'
  schedule?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  nextRun?: Date
  lastRun?: Date
  createdAt: Date
}

// Pagination types
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
  hasMore: boolean
}

// Filter types
export interface FilterOption {
  id: string
  label: string
  value: any
  count?: number
}

export interface FilterState {
  [key: string]: any
}
