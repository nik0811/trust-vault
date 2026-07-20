'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Skeleton } from '@/components/base/skeleton'
import { BookOpen, CheckCircle, Circle, AlertTriangle, Clock, Users, Shield, Database } from 'lucide-react'
import { usePlaybook } from '@/hooks/use-advisor'
import { useState } from 'react'

const issueTypes = [
  { id: 'data_breach', name: 'Data Breach Response', description: 'Steps to handle a data breach incident', icon: AlertTriangle, color: 'text-red-500' },
  { id: 'dsar_overdue', name: 'Overdue DSAR', description: 'Handle overdue data subject requests', icon: Clock, color: 'text-orange-500' },
  { id: 'retention_violation', name: 'Retention Violation', description: 'Address data retention policy violations', icon: Database, color: 'text-yellow-500' },
  { id: 'consent_withdrawal', name: 'Consent Withdrawal', description: 'Process consent withdrawal requests', icon: Users, color: 'text-blue-500' },
  { id: 'quality_degradation', name: 'Quality Degradation', description: 'Remediate data quality issues', icon: Shield, color: 'text-purple-500' },
  { id: 'unauthorized_access', name: 'Unauthorized Access', description: 'Respond to unauthorized data access', icon: AlertTriangle, color: 'text-red-600' },
]

interface PlaybookStep {
  title: string
  description?: string
}

export default function PlaybooksPage() {
  const [selectedIssue, setSelectedIssue] = useState<string | null>(null)
  const [completedSteps, setCompletedSteps] = useState<Set<number>>(new Set())
  const { data: playbook, isLoading } = usePlaybook(selectedIssue || '')

  const handleSelectIssue = (issueId: string) => {
    setSelectedIssue(issueId)
    setCompletedSteps(new Set()) // Reset completed steps when changing playbook
  }

  const toggleStep = (index: number) => {
    setCompletedSteps(prev => {
      const newSet = new Set(prev)
      if (newSet.has(index)) {
        newSet.delete(index)
      } else {
        newSet.add(index)
      }
      return newSet
    })
  }

  // Parse steps - handle both array of strings and array of objects
  const parseSteps = (steps: any): PlaybookStep[] => {
    if (!steps) return []
    if (!Array.isArray(steps)) return []
    
    return steps.map((step: any) => {
      if (typeof step === 'string') {
        return { title: step }
      }
      return {
        title: step.title || step.name || String(step),
        description: step.description || step.details || undefined
      }
    })
  }

  const steps = playbook ? parseSteps(playbook.steps) : []
  const completedCount = completedSteps.size
  const totalSteps = steps.length
  const progressPercent = totalSteps > 0 ? (completedCount / totalSteps) * 100 : 0

  const selectedIssueData = issueTypes.find(i => i.id === selectedIssue)

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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {issueTypes.map((issue) => {
            const Icon = issue.icon
            return (
              <button
                key={issue.id}
                onClick={() => handleSelectIssue(issue.id)}
                className={`text-left p-4 rounded-lg border transition-all ${
                  selectedIssue === issue.id
                    ? 'border-primary bg-primary/10 shadow-sm'
                    : 'border-border hover:border-primary/50 hover:bg-muted/50'
                }`}
              >
                <div className="flex items-start gap-3">
                  <div className={`p-2 rounded-lg bg-muted ${issue.color}`}>
                    <Icon className="h-5 w-5" />
                  </div>
                  <div>
                    <h4 className="font-medium text-foreground">{issue.name}</h4>
                    <p className="text-sm text-muted-foreground mt-1">{issue.description}</p>
                  </div>
                </div>
              </button>
            )
          })}
        </div>

        {/* Playbook Content */}
        {selectedIssue && (
          <div className="rounded-lg border border-border bg-card overflow-hidden">
            {/* Playbook Header */}
            <div className="border-b border-border bg-muted/30 p-6">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg bg-primary/10`}>
                    <BookOpen className="h-6 w-6 text-primary" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-foreground">
                      {playbook?.name || selectedIssueData?.name + ' Playbook'}
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      {totalSteps} steps • {completedCount} completed
                    </p>
                  </div>
                </div>
                {totalSteps > 0 && (
                  <div className="text-right">
                    <p className="text-2xl font-bold text-foreground">{Math.round(progressPercent)}%</p>
                    <p className="text-xs text-muted-foreground">Progress</p>
                  </div>
                )}
              </div>
              
              {/* Progress Bar */}
              {totalSteps > 0 && (
                <div className="mt-4 h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-primary transition-all duration-300"
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
              )}
            </div>

            {/* Steps */}
            <div className="p-6">
              {isLoading ? (
                <div className="space-y-4">
                  <Skeleton className="h-20 w-full" />
                  <Skeleton className="h-20 w-full" />
                  <Skeleton className="h-20 w-full" />
                </div>
              ) : steps.length > 0 ? (
                <div className="space-y-3">
                  {steps.map((step, i) => {
                    const isCompleted = completedSteps.has(i)
                    return (
                      <button
                        key={i}
                        onClick={() => toggleStep(i)}
                        className={`w-full flex gap-4 p-4 rounded-lg text-left transition-all ${
                          isCompleted 
                            ? 'bg-green-500/5 border border-green-500/20' 
                            : 'bg-muted/50 hover:bg-muted border border-transparent'
                        }`}
                      >
                        <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 transition-colors ${
                          isCompleted 
                            ? 'bg-green-500 text-white' 
                            : 'bg-primary/10 text-primary'
                        }`}>
                          {isCompleted ? (
                            <CheckCircle className="h-5 w-5" />
                          ) : (
                            <span className="font-bold text-sm">{i + 1}</span>
                          )}
                        </div>
                        <div className="flex-1">
                          <p className={`font-medium ${isCompleted ? 'text-green-700 dark:text-green-400 line-through' : 'text-foreground'}`}>
                            {step.title}
                          </p>
                          {step.description && (
                            <p className={`text-sm mt-1 ${isCompleted ? 'text-green-600/70 dark:text-green-500/70' : 'text-muted-foreground'}`}>
                              {step.description}
                            </p>
                          )}
                        </div>
                        <div className="flex-shrink-0 self-center">
                          {isCompleted ? (
                            <span className="text-xs px-2 py-1 rounded bg-green-500/10 text-green-600">Done</span>
                          ) : (
                            <Circle className="h-5 w-5 text-muted-foreground/50" />
                          )}
                        </div>
                      </button>
                    )
                  })}
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No steps available for this playbook</p>
                </div>
              )}
            </div>

            {/* Footer */}
            {completedCount === totalSteps && totalSteps > 0 && (
              <div className="border-t border-border bg-green-500/5 p-4">
                <div className="flex items-center justify-center gap-2 text-green-600">
                  <CheckCircle className="h-5 w-5" />
                  <span className="font-medium">Playbook completed! All steps have been marked as done.</span>
                </div>
              </div>
            )}
          </div>
        )}

        {/* No Selection State */}
        {!selectedIssue && (
          <div className="rounded-lg border border-dashed border-border bg-muted/20 p-12 text-center">
            <BookOpen className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <p className="text-foreground font-medium">Select an issue type above</p>
            <p className="text-sm text-muted-foreground mt-1">
              Choose a compliance issue to view its step-by-step remediation playbook
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
