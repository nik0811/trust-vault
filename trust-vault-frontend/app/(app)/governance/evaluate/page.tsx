'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Zap, CheckCircle, XCircle, AlertTriangle } from 'lucide-react'
import { useEvaluatePolicy } from '@/hooks/use-policies'

export default function PolicyEvaluatePage() {
  const [testData, setTestData] = useState('')
  const evaluatePolicy = useEvaluatePolicy()

  const handleEvaluate = async () => {
    if (!testData.trim()) return
    await evaluatePolicy.mutateAsync({ data: testData })
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Governance', href: '/governance' },
            { label: 'Policy Evaluation', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Policy Evaluation</h1>
        <p className="text-sm text-muted-foreground mt-1">Test your policies against sample data</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Input Section */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Test Data</h3>
          <textarea
            value={testData}
            onChange={(e) => setTestData(e.target.value)}
            rows={6}
            placeholder="Enter sample data to evaluate against your policies...

Example:
John Doe's email is john.doe@company.com and his SSN is 123-45-6789.
His credit card number is 4111-1111-1111-1111."
            className="w-full px-4 py-3 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none font-mono text-sm"
          />
          <button
            onClick={handleEvaluate}
            disabled={evaluatePolicy.isPending || !testData.trim()}
            className="mt-4 flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Zap className="h-4 w-4" />
            {evaluatePolicy.isPending ? 'Evaluating...' : 'Evaluate Policies'}
          </button>
        </div>

        {/* Results Section */}
        {evaluatePolicy.data && (
          <div className="rounded-lg border border-border bg-card p-6">
            <h3 className="text-lg font-semibold text-foreground mb-4">Evaluation Results</h3>
            
            {/* Decision */}
            <div className="flex items-center gap-3 mb-6">
              {evaluatePolicy.data.decision === 'allow' ? (
                <>
                  <CheckCircle className="h-8 w-8 text-green-500" />
                  <div>
                    <p className="text-lg font-semibold text-green-600">Allowed</p>
                    <p className="text-sm text-muted-foreground">Data passes all policy checks</p>
                  </div>
                </>
              ) : evaluatePolicy.data.decision === 'deny' ? (
                <>
                  <XCircle className="h-8 w-8 text-red-500" />
                  <div>
                    <p className="text-lg font-semibold text-red-600">Denied</p>
                    <p className="text-sm text-muted-foreground">Data blocked by policy</p>
                  </div>
                </>
              ) : (
                <>
                  <AlertTriangle className="h-8 w-8 text-yellow-500" />
                  <div>
                    <p className="text-lg font-semibold text-yellow-600">Redacted</p>
                    <p className="text-sm text-muted-foreground">Sensitive data has been redacted</p>
                  </div>
                </>
              )}
            </div>

            {/* Applied Policies */}
            {evaluatePolicy.data.applied_policies && evaluatePolicy.data.applied_policies.length > 0 && (
              <div className="mb-6">
                <h4 className="text-sm font-medium text-foreground mb-2">Applied Policies</h4>
                <div className="flex flex-wrap gap-2">
                  {evaluatePolicy.data.applied_policies.map((policy, i) => (
                    <span key={i} className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm">
                      {policy}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Redacted Output */}
            {evaluatePolicy.data.redacted_data && (
              <div>
                <h4 className="text-sm font-medium text-foreground mb-2">Redacted Output</h4>
                <pre className="p-4 rounded-lg bg-muted text-sm font-mono whitespace-pre-wrap">
                  {evaluatePolicy.data.redacted_data}
                </pre>
              </div>
            )}
          </div>
        )}

        {/* Help Section */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">How Policy Evaluation Works</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">1</span>
              </div>
              <h4 className="font-medium text-foreground">Classification</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Data is first classified to identify sensitive entities (PII, PHI, etc.)
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">2</span>
              </div>
              <h4 className="font-medium text-foreground">Policy Matching</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Active policies are evaluated in priority order against the classified data
              </p>
            </div>
            <div>
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-3">
                <span className="text-primary font-bold">3</span>
              </div>
              <h4 className="font-medium text-foreground">Action</h4>
              <p className="text-sm text-muted-foreground mt-1">
                Based on policy rules: allow, deny, or redact the data
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
