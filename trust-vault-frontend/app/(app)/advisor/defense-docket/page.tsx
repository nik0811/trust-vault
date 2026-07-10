'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { FileText, Download } from 'lucide-react'
import { useGenerateDefenseDocket } from '@/hooks/use-advisor'

const regulations = ['GDPR', 'CCPA', 'HIPAA', 'SOC2', 'PCI-DSS', 'DPDP', 'EU AI Act']

export default function DefenseDocketPage() {
  const generateDocket = useGenerateDefenseDocket()
  const [selectedRegs, setSelectedRegs] = useState<string[]>(['GDPR'])
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')

  const handleGenerate = async () => {
    if (selectedRegs.length === 0) return
    await generateDocket.mutateAsync({
      regulations: selectedRegs,
      date_from: dateFrom || new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(),
      date_to: dateTo || new Date().toISOString(),
    })
  }

  const toggleReg = (reg: string) => {
    setSelectedRegs(prev =>
      prev.includes(reg) ? prev.filter(r => r !== reg) : [...prev, reg]
    )
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs
          items={[
            { label: 'Compliance Advisor', href: '/advisor' },
            { label: 'Defense Docket', active: true },
          ]}
        />
        <h1 className="text-3xl font-bold text-foreground mt-4">Defense Docket</h1>
        <p className="text-sm text-muted-foreground mt-1">Generate audit-ready compliance documentation</p>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8">
        {/* Configuration */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Generate Defense Docket</h3>
          
          {/* Regulations */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-foreground mb-2">Select Regulations</label>
            <div className="flex flex-wrap gap-2">
              {regulations.map((reg) => (
                <button
                  key={reg}
                  onClick={() => toggleReg(reg)}
                  className={`px-4 py-2 rounded-lg border transition-colors ${
                    selectedRegs.includes(reg)
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border text-foreground hover:border-primary/50'
                  }`}
                >
                  {reg}
                </button>
              ))}
            </div>
          </div>

          {/* Date Range */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">From Date</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">To Date</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>
          </div>

          <button
            onClick={handleGenerate}
            disabled={generateDocket.isPending || selectedRegs.length === 0}
            className="flex items-center gap-2 px-6 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <FileText className="h-4 w-4" />
            {generateDocket.isPending ? 'Generating...' : 'Generate Docket'}
          </button>
        </div>

        {/* Result */}
        {generateDocket.data && (
          <div className="rounded-lg border border-border bg-card p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-foreground">Generated Docket</h3>
              <button className="flex items-center gap-2 px-4 py-2 rounded-lg border border-border text-foreground hover:bg-muted transition-colors">
                <Download className="h-4 w-4" />
                Download PDF
              </button>
            </div>
            <pre className="p-4 rounded-lg bg-muted text-sm font-mono whitespace-pre-wrap overflow-auto max-h-96">
              {JSON.stringify(generateDocket.data, null, 2)}
            </pre>
          </div>
        )}

        {/* Info */}
        <div className="rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">What&apos;s Included</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h4 className="font-medium text-foreground mb-2">Compliance Evidence</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Policy configurations and enforcement logs</li>
                <li>• Data classification summaries</li>
                <li>• Access control audit trails</li>
                <li>• DSAR processing records</li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium text-foreground mb-2">Risk Assessment</h4>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Data quality scores and trends</li>
                <li>• Identified gaps and remediation status</li>
                <li>• AI governance compliance</li>
                <li>• Retention policy adherence</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
