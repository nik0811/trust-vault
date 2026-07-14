'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { ArrowLeft, Database } from 'lucide-react'
import { useCreateDataSource } from '@/hooks/use-datasources'
import Link from 'next/link'

const dataSourceSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  type: z.string().min(1, 'Type is required'),
  host: z.string().optional(),
  port: z.string().optional(),
  database: z.string().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  bucket: z.string().optional(),
  region: z.string().optional(),
  connection_string: z.string().optional(),
  account: z.string().optional(),
  warehouse: z.string().optional(),
  project_id: z.string().optional(),
  dataset: z.string().optional(),
})

type DataSourceForm = z.infer<typeof dataSourceSchema>

const dataSourceCategories = [
  {
    label: 'Relational Databases',
    types: [
      { value: 'postgres', label: 'PostgreSQL', icon: '🐘' },
      { value: 'mysql', label: 'MySQL', icon: '🐬' },
      { value: 'mssql', label: 'SQL Server', icon: '🗄️' },
      { value: 'oracle', label: 'Oracle', icon: '🔴' },
      { value: 'mariadb', label: 'MariaDB', icon: '🦭' },
      { value: 'cockroachdb', label: 'CockroachDB', icon: '🪳' },
    ],
  },
  {
    label: 'Cloud Data Warehouses',
    types: [
      { value: 'snowflake', label: 'Snowflake', icon: '❄️' },
      { value: 'bigquery', label: 'BigQuery', icon: '📊' },
      { value: 'redshift', label: 'Redshift', icon: '🔶' },
      { value: 'databricks', label: 'Databricks', icon: '🧱' },
      { value: 'synapse', label: 'Azure Synapse', icon: '🔷' },
      { value: 'clickhouse', label: 'ClickHouse', icon: '🖱️' },
    ],
  },
  {
    label: 'Data Lakes & Object Storage',
    types: [
      { value: 's3', label: 'Amazon S3', icon: '☁️' },
      { value: 'gcs', label: 'Google Cloud Storage', icon: '🪣' },
      { value: 'azure_blob', label: 'Azure Blob Storage', icon: '📦' },
      { value: 'delta_lake', label: 'Delta Lake', icon: '🔺' },
      { value: 'iceberg', label: 'Apache Iceberg', icon: '🧊' },
      { value: 'hudi', label: 'Apache Hudi', icon: '📂' },
    ],
  },
  {
    label: 'Streaming & Messaging',
    types: [
      { value: 'kafka', label: 'Apache Kafka', icon: '📡' },
      { value: 'pulsar', label: 'Apache Pulsar', icon: '💫' },
      { value: 'kinesis', label: 'AWS Kinesis', icon: '🌊' },
      { value: 'eventhub', label: 'Azure Event Hub', icon: '⚡' },
      { value: 'rabbitmq', label: 'RabbitMQ', icon: '🐰' },
      { value: 'nats', label: 'NATS', icon: '📬' },
    ],
  },
  {
    label: 'NoSQL & Document Stores',
    types: [
      { value: 'mongodb', label: 'MongoDB', icon: '🍃' },
      { value: 'elasticsearch', label: 'Elasticsearch', icon: '🔍' },
      { value: 'dynamodb', label: 'DynamoDB', icon: '⚙️' },
      { value: 'cassandra', label: 'Cassandra', icon: '👁️' },
      { value: 'redis', label: 'Redis', icon: '🔴' },
      { value: 'couchbase', label: 'Couchbase', icon: '🛋️' },
    ],
  },
  {
    label: 'BI & Analytics Tools',
    types: [
      { value: 'tableau', label: 'Tableau', icon: '📈' },
      { value: 'looker', label: 'Looker', icon: '👀' },
      { value: 'powerbi', label: 'Power BI', icon: '📉' },
      { value: 'metabase', label: 'Metabase', icon: '🎯' },
      { value: 'superset', label: 'Apache Superset', icon: '🦸' },
      { value: 'dbt', label: 'dbt', icon: '🔧' },
    ],
  },
  {
    label: 'Orchestration & ETL',
    types: [
      { value: 'airflow', label: 'Apache Airflow', icon: '🌬️' },
      { value: 'spark', label: 'Apache Spark', icon: '✨' },
      { value: 'flink', label: 'Apache Flink', icon: '🐿️' },
      { value: 'glue', label: 'AWS Glue', icon: '🧪' },
      { value: 'fivetran', label: 'Fivetran', icon: '🔌' },
      { value: 'airbyte', label: 'Airbyte', icon: '🔀' },
    ],
  },
  {
    label: 'APIs & SaaS',
    types: [
      { value: 'salesforce', label: 'Salesforce', icon: '☁️' },
      { value: 'hubspot', label: 'HubSpot', icon: '🟠' },
      { value: 'rest_api', label: 'REST API', icon: '🌐' },
      { value: 'graphql', label: 'GraphQL', icon: '◈' },
      { value: 'sharepoint', label: 'SharePoint', icon: '📎' },
      { value: 'google_sheets', label: 'Google Sheets', icon: '📗' },
    ],
  },
  {
    label: 'Files & Documents',
    types: [
      { value: 'file', label: 'File System', icon: '📁' },
      { value: 'csv', label: 'CSV / Excel', icon: '📄' },
      { value: 'pdf', label: 'PDF Documents', icon: '📕' },
      { value: 'parquet', label: 'Parquet', icon: '🪶' },
      { value: 'avro', label: 'Avro', icon: '🅰️' },
      { value: 'json', label: 'JSON / NDJSON', icon: '{ }' },
    ],
  },
]

export default function NewDataSourcePage() {
  const router = useRouter()
  const createDataSource = useCreateDataSource()
  const [selectedType, setSelectedType] = useState<string>('postgres')
  const [searchQuery, setSearchQuery] = useState('')

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    watch,
    setValue,
  } = useForm<DataSourceForm>({
    resolver: zodResolver(dataSourceSchema),
    defaultValues: {
      type: 'postgres',
    },
  })

  const watchType = watch('type')

  const filteredCategories = dataSourceCategories.map(cat => ({
    ...cat,
    types: cat.types.filter(t => 
      t.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.value.toLowerCase().includes(searchQuery.toLowerCase())
    ),
  })).filter(cat => cat.types.length > 0)

  const onSubmit = async (data: DataSourceForm) => {
    const config: Record<string, any> = {}
    
    if (['postgres', 'mysql', 'mssql', 'oracle', 'mariadb', 'cockroachdb'].includes(data.type)) {
      if (data.host) config.host = data.host
      if (data.port) config.port = parseInt(data.port)
      if (data.database) config.database = data.database
      if (data.username) config.username = data.username
      if (data.password) config.password = data.password
    } else if (['s3', 'gcs', 'azure_blob'].includes(data.type)) {
      if (data.bucket) config.bucket = data.bucket
      if (data.region) config.region = data.region
    } else if (data.type === 'snowflake') {
      if (data.account) config.account = data.account
      if (data.warehouse) config.warehouse = data.warehouse
      if (data.database) config.database = data.database
      if (data.username) config.username = data.username
      if (data.password) config.password = data.password
    } else if (data.type === 'bigquery') {
      if (data.project_id) config.project_id = data.project_id
      if (data.dataset) config.dataset = data.dataset
    } else if (data.connection_string) {
      config.connection_string = data.connection_string
    }

    try {
      await createDataSource.mutateAsync({
        name: data.name,
        type: data.type,
        config: Object.keys(config).length > 0 ? config : undefined,
      })
      router.push('/data-sources')
    } catch (error) {
      // Error handled by hook
    }
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Data Sources', href: '/data-sources' },
            { label: 'New Data Source', active: true },
          ]}
        />
        <div className="flex items-center gap-4 mt-4">
          <Link href="/data-sources" className="p-2 rounded-lg hover:bg-muted transition-colors">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-foreground">Add Data Source</h1>
            <p className="text-sm text-muted-foreground mt-1">Connect a new data source to SecureLens</p>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-8 max-w-3xl">
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Name</label>
            <input
              {...register('name')}
              type="text"
              placeholder="My Database"
              className="w-full px-4 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            {errors.name && <p className="text-sm text-destructive mt-1">{errors.name.message}</p>}
          </div>

          {/* Type Selection */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Type</label>
            <input
              type="text"
              placeholder="Search data sources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full px-4 py-2 mb-4 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <div className="max-h-[400px] overflow-y-auto space-y-4 pr-2">
              {filteredCategories.map((category) => (
                <div key={category.label}>
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">{category.label}</p>
                  <div className="grid grid-cols-3 gap-2">
                    {category.types.map((type) => (
                      <label
                        key={type.value}
                        className={`flex items-center gap-2 p-3 rounded-lg border cursor-pointer transition-colors ${
                          watchType === type.value
                            ? 'border-primary bg-primary/10'
                            : 'border-border hover:border-primary/50'
                        }`}
                        onClick={() => setValue('type', type.value)}
                      >
                        <input
                          {...register('type')}
                          type="radio"
                          value={type.value}
                          className="sr-only"
                        />
                        <span className="text-lg">{type.icon}</span>
                        <span className="text-xs font-medium text-foreground truncate">{type.label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Database Config */}
          {['postgres', 'mysql', 'mssql', 'oracle', 'mariadb', 'cockroachdb'].includes(watchType) && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/50">
              <h3 className="text-sm font-medium text-foreground">Connection Details</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Host</label>
                  <input
                    {...register('host')}
                    type="text"
                    placeholder="localhost"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Port</label>
                  <input
                    {...register('port')}
                    type="text"
                    placeholder={watchType === 'postgres' ? '5432' : watchType === 'mssql' ? '1433' : watchType === 'oracle' ? '1521' : '3306'}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Database</label>
                <input
                  {...register('database')}
                  type="text"
                  placeholder="mydb"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Username</label>
                  <input
                    {...register('username')}
                    type="text"
                    placeholder="user"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Password</label>
                  <input
                    {...register('password')}
                    type="password"
                    placeholder="••••••••"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Cloud Storage Config */}
          {['s3', 'gcs', 'azure_blob'].includes(watchType) && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/50">
              <h3 className="text-sm font-medium text-foreground">Storage Configuration</h3>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Bucket / Container Name</label>
                <input
                  {...register('bucket')}
                  type="text"
                  placeholder="my-bucket"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Region</label>
                <input
                  {...register('region')}
                  type="text"
                  placeholder="us-east-1"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
            </div>
          )}

          {/* Snowflake Config */}
          {watchType === 'snowflake' && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/50">
              <h3 className="text-sm font-medium text-foreground">Snowflake Configuration</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Account</label>
                  <input
                    {...register('account')}
                    type="text"
                    placeholder="xy12345.us-east-1"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Warehouse</label>
                  <input
                    {...register('warehouse')}
                    type="text"
                    placeholder="COMPUTE_WH"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Database</label>
                <input
                  {...register('database')}
                  type="text"
                  placeholder="MY_DB"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Username</label>
                  <input
                    {...register('username')}
                    type="text"
                    placeholder="user"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Password</label>
                  <input
                    {...register('password')}
                    type="password"
                    placeholder="••••••••"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
            </div>
          )}

          {/* BigQuery Config */}
          {watchType === 'bigquery' && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/50">
              <h3 className="text-sm font-medium text-foreground">BigQuery Configuration</h3>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Project ID</label>
                <input
                  {...register('project_id')}
                  type="text"
                  placeholder="my-gcp-project"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Dataset</label>
                <input
                  {...register('dataset')}
                  type="text"
                  placeholder="my_dataset"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
            </div>
          )}

          {/* Generic Connection String */}
          {!['postgres', 'mysql', 'mssql', 'oracle', 'mariadb', 'cockroachdb', 's3', 'gcs', 'azure_blob', 'snowflake', 'bigquery'].includes(watchType) && (
            <div className="space-y-4 p-4 rounded-lg border border-border bg-muted/50">
              <h3 className="text-sm font-medium text-foreground">Connection Configuration</h3>
              <div>
                <label className="block text-sm text-muted-foreground mb-1">Connection String / URL</label>
                <input
                  {...register('connection_string')}
                  type="text"
                  placeholder="protocol://host:port/path"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Username (optional)</label>
                  <input
                    {...register('username')}
                    type="text"
                    placeholder="user"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-muted-foreground mb-1">Password (optional)</label>
                  <input
                    {...register('password')}
                    type="password"
                    placeholder="••••••••"
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Submit */}
          <div className="flex items-center gap-4 pt-4">
            <button
              type="submit"
              disabled={isSubmitting || createDataSource.isPending}
              className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Database className="h-4 w-4" />
              {isSubmitting || createDataSource.isPending ? 'Creating...' : 'Create Data Source'}
            </button>
            <Link
              href="/data-sources"
              className="px-6 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors"
            >
              Cancel
            </Link>
          </div>
        </form>
      </div>
    </div>
  )
}
