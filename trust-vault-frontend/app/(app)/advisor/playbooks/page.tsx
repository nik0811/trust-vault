'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { BookOpen } from 'lucide-react'
import { usePlaybook } from '@/hooks/use-advisor'
import { useState } from 'react'

const issueTypes = [
  { id: 'data_breach', name: 'Data Breach Response', description: 'Steps to handle a data breach incident' },
  { id: 'dsar_overdue', name: 'Overdue DSAR', description: 'Handle overdue data subject requests' },
  { id: 'retention_violation', name: 'Retention Violation', description: 'Address data retention policy violations' },
  { id: 'consent_withdrawal', name: 'Consent Withdrawal', description: 'Process consent withdrawal requests' },
  { id: 'quality_degradation', name: 'Quality Degradation', description: 'Remediate data quality issues' },
  { id: 'unauthorized_access', name: 'Unauthorized Access', description: 'Respond to unauthorized data access' },
]

export default function PlaybooksPage() {
  const [selectedIssue, setSelectedIssue] = useState<string | null>(null)
  const { data: playbook, isLoading } = usePlaybook(selectedIssue || '')

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Compliance Advisor', href: '/advisor' },
            { label: 'Playbooks', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Compliance Playbooks</h1>
        <p className="text-sm text-muted-foreground mt-1">Step-by-step guides for common compliance issues</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Issue Types */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {issueTypes.map((issue) => (
            <button
              key={issue.id}
              onClick={() => setSelectedIssue(issue.id)}
              className={`text-left p-4 rounded-lg border transition-colors ${
                selectedIssue === issue.id
                  ? 'border-primary bg-primary/10'
                  : 'border-border hover:border-primary/50'
              }`}
            >
              <h4 className="font-medium text-foreground">{issue.name}</h4>
              <p className="text-sm text-muted-foreground mt-1">{issue.description}</p>
            </button>
          ))}
        </div>

        {/* Playbook Content */}
        {selectedIssue && (
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-6">
              <BookOpen className="h-6 w-6 text-primary" />
              <h3 className="text-lg font-semibold text-foreground">
                {issueTypes.find(i => i.id === selectedIssue)?.name} Playbook
              </h3>
            </div>
            {isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : playbook ? (
              <div className="space-y-4">
                {Array.isArray(playbook.steps) ? playbook.steps.map((step: any, i: number) => (
                  <div key={i} className="flex gap-4 p-4 rounded-lg bg-muted/50">
                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                      <span className="text-primary font-bold">{i + 1}</span>
                    </div>
                    <div>
                      <p className="font-medium text-foreground">{step.title || step}</p>
                      {step.description && (
                        <p className="text-sm text-muted-foreground mt-1">{step.description}</p>
                      )}
                    </div>
                  </div>
                )) : (
                  <pre className="p-4 rounded-lg bg-muted text-sm">
                    {JSON.stringify(playbook, null, 2)}
                  </pre>
                )}
              </div>
            ) : (
              <div className="text-center py-8">
                <p className="text-muted-foreground">Select an issue type to view its playbook</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
