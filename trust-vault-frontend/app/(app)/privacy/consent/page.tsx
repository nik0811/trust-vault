'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { CheckCircle, XCircle, Plus } from 'lucide-react'
import { useRecordConsent, useWithdrawConsent } from '@/hooks/use-privacy'
import { toast } from 'sonner'

export default function ConsentPage() {
  const recordConsent = useRecordConsent()
  const withdrawConsent = useWithdrawConsent()
  const [formData, setFormData] = useState({ subject_id: '', purpose: '' })
  const [withdrawId, setWithdrawId] = useState('')

  const handleRecord = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await recordConsent.mutateAsync(formData)
      setFormData({ subject_id: '', purpose: '' })
    } catch (error) {
      // Error handled by hook
    }
  }

  const handleWithdraw = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!withdrawId.trim()) return
    try {
      await withdrawConsent.mutateAsync(withdrawId)
      setWithdrawId('')
    } catch (error) {
      // Error handled by hook
    }
  }

  const purposes = [
    'Marketing communications',
    'Analytics and performance',
    'Personalization',
    'Third-party sharing',
    'AI/ML training',
    'Research and development',
  ]

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Privacy', href: '/privacy' },
            { label: 'Consent', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Consent Management</h1>
        <p className="text-sm text-muted-foreground mt-1">Track and manage user consent for data processing</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          {/* Record Consent */}
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-4">
              <CheckCircle className="h-6 w-6 text-green-500" />
              <h3 className="text-lg font-semibold text-foreground">Record Consent</h3>
            </div>
            <form onSubmit={handleRecord} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Subject ID</label>
                <input
                  type="text"
                  value={formData.subject_id}
                  onChange={(e) => setFormData({ ...formData, subject_id: e.target.value })}
                  placeholder="user@example.com or user ID"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Purpose</label>
                <select
                  value={formData.purpose}
                  onChange={(e) => setFormData({ ...formData, purpose: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  required
                >
                  <option value="">Select purpose...</option>
                  {purposes.map((p) => (
                    <option key={p} value={p}>{p}</option>
                  ))}
                </select>
              </div>
              <button
                type="submit"
                disabled={recordConsent.isPending}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-green-600 text-white hover:bg-green-700 transition-colors disabled:opacity-50"
              >
                <Plus className="h-4 w-4" />
                {recordConsent.isPending ? 'Recording...' : 'Record Consent'}
              </button>
            </form>
          </div>

          {/* Withdraw Consent */}
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center gap-3 mb-4">
              <XCircle className="h-6 w-6 text-red-500" />
              <h3 className="text-lg font-semibold text-foreground">Withdraw Consent</h3>
            </div>
            <form onSubmit={handleWithdraw} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">Subject ID</label>
                <input
                  type="text"
                  value={withdrawId}
                  onChange={(e) => setWithdrawId(e.target.value)}
                  placeholder="user@example.com or user ID"
                  className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  required
                />
              </div>
              <p className="text-sm text-muted-foreground">
                This will withdraw all consent records for the specified subject.
              </p>
              <button
                type="submit"
                disabled={withdrawConsent.isPending}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                <XCircle className="h-4 w-4" />
                {withdrawConsent.isPending ? 'Withdrawing...' : 'Withdraw Consent'}
              </button>
            </form>
          </div>
        </div>

        {/* Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Consent Purposes</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {purposes.map((purpose) => (
              <div key={purpose} className="p-4 rounded-lg bg-muted/50">
                <p className="font-medium text-foreground">{purpose}</p>
                <p className="text-sm text-muted-foreground mt-1">
                  Requires explicit user consent
                </p>
              </div>
            ))}
          </div>
        </div>

        {/* Compliance Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Compliance Requirements</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h4 className="font-medium text-foreground mb-2">GDPR (EU)</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Consent must be freely given, specific, informed, and unambiguous</li>
                <li>• Must be as easy to withdraw as to give</li>
                <li>• Records must be maintained</li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium text-foreground mb-2">CCPA (California)</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Right to opt-out of sale of personal information</li>
                <li>• Must provide clear notice</li>
                <li>• Cannot discriminate against users who opt-out</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
