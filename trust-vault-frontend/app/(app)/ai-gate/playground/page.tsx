'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Zap, CheckCircle, XCircle, Clock, Shield } from 'lucide-react'
import { useGateQuery } from '@/hooks/use-gate'

export default function AIGatePlaygroundPage() {
  const [query, setQuery] = useState('')
  const [maxChunks, setMaxChunks] = useState(5)
  const gateQuery = useGateQuery()

  const handleQuery = async () => {
    if (!query.trim()) return
    await gateQuery.mutateAsync({ query, max_chunks: maxChunks })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'AI Gate', href: '/ai-gate' },
            { label: 'Playground', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">AI Gate Playground</h1>
        <p className="text-sm text-muted-foreground mt-1">Test AI Gate queries with real-time policy evaluation</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Query Input */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Query</h3>
          <textarea
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            rows={4}
            placeholder="Enter your query...

Example: What are the customer details for order #12345?"
            className="w-full px-4 py-3 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
          />
          <div className="flex items-center gap-4 mt-4">
            <div className="flex items-center gap-2">
              <label className="text-sm text-muted-foreground">Max Chunks:</label>
              <input
                type="number"
                value={maxChunks}
                onChange={(e) => setMaxChunks(parseInt(e.target.value) || 5)}
                min={1}
                max={20}
                className="w-20 px-3 py-1 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <button
              onClick={handleQuery}
              disabled={gateQuery.isPending || !query.trim()}
              className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Zap className="h-4 w-4" />
              {gateQuery.isPending ? 'Processing...' : 'Send Query'}
            </button>
          </div>
        </div>

        {/* Results */}
        {gateQuery.data && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Results</h3>

            {/* Decision */}
            <div className="flex items-center gap-4 mb-6 p-4 rounded-lg bg-muted/50">
              {gateQuery.data.decision === 'allow' ? (
                <CheckCircle className="h-8 w-8 text-green-500" />
              ) : (
                <XCircle className="h-8 w-8 text-red-500" />
              )}
              <div>
                <p className={`text-lg font-semibold ${gateQuery.data.decision === 'allow' ? 'text-green-600' : 'text-red-600'}`}>
                  {gateQuery.data.decision === 'allow' ? 'Allowed' : 'Blocked'}
                </p>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Clock className="h-4 w-4" />
                    {gateQuery.data.latency_ms}ms
                  </span>
                  <span className="flex items-center gap-1">
                    <Shield className="h-4 w-4" />
                    {gateQuery.data.policies_applied?.length || 0} policies applied
                  </span>
                </div>
              </div>
            </div>

            {/* Applied Policies */}
            {gateQuery.data.policies_applied && gateQuery.data.policies_applied.length > 0 && (
              <div className="mb-6">
                <h4 className="text-sm font-medium text-foreground mb-2">Applied Policies</h4>
                <div className="flex flex-wrap gap-2">
                  {gateQuery.data.policies_applied.map((policy, i) => (
                    <span key={i} className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm">
                      {policy}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Chunks */}
            {gateQuery.data.redacted_chunks && gateQuery.data.redacted_chunks.length > 0 && (
              <div>
                <h4 className="text-sm font-medium text-foreground mb-2">
                  Retrieved Chunks ({gateQuery.data.redacted_chunks.length})
                </h4>
                <div className="space-y-3">
                  {gateQuery.data.redacted_chunks.map((chunk: any, i: number) => (
                    <div key={i} className="p-4 rounded-lg bg-muted">
                      <pre className="text-sm whitespace-pre-wrap font-mono">
                        {typeof chunk === 'string' ? chunk : JSON.stringify(chunk, null, 2)}
                      </pre>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Original vs Redacted comparison */}
            {gateQuery.data.chunks && gateQuery.data.redacted_chunks && (
              <div className="mt-6 grid grid-cols-2 gap-4">
                <div>
                  <h4 className="text-sm font-medium text-foreground mb-2">Original</h4>
                  <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/20">
                    <pre className="text-sm whitespace-pre-wrap font-mono text-red-600">
                      {gateQuery.data.chunks.length} chunks retrieved
                    </pre>
                  </div>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-foreground mb-2">After Governance</h4>
                  <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20">
                    <pre className="text-sm whitespace-pre-wrap font-mono text-green-600">
                      {gateQuery.data.redacted_chunks.length} chunks (redacted)
                    </pre>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Example Queries */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Example Queries</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              'What customer information do we have for John Smith?',
              'Show me all transactions over $10,000',
              'List employees with access to financial data',
              'What are the medical records for patient ID 12345?',
            ].map((example, i) => (
              <button
                key={i}
                onClick={() => setQuery(example)}
                className="text-left p-4 rounded-lg border border-border hover:border-primary/50 transition-colors"
              >
                <p className="text-sm text-foreground">{example}</p>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
