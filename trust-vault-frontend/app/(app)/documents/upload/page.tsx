'use client'

import { useState } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Upload, FileText, CheckCircle2, Loader2 } from 'lucide-react'

export default function DocumentUploadPage() {
  const [dragging, setDragging] = useState(false)
  const [files, setFiles] = useState<{ name: string; size: string; status: 'scanning' | 'done'; findings: number }[]>([])

  const addFiles = (list: FileList | null) => {
    if (!list) return
    const newFiles = Array.from(list).map((f) => ({
      name: f.name,
      size: `${(f.size / 1024).toFixed(1)} KB`,
      status: 'scanning' as const,
      findings: 0,
    }))
    setFiles((prev) => [...prev, ...newFiles])
    newFiles.forEach((nf, i) => {
      setTimeout(() => {
        setFiles((prev) =>
          prev.map((f) =>
            f.name === nf.name ? { ...f, status: 'done', findings: Math.floor(Math.random() * 6) } : f
          )
        )
      }, 1200 + i * 600)
    })
  }

  return (
    <div className="space-y-6">
      <Breadcrumbs items={[{ label: 'Documents', href: '/documents' }, { label: 'Upload & Analyze' }]} />

      <div>
        <h1 className="text-2xl font-bold text-foreground">Upload & Analyze</h1>
        <p className="text-muted-foreground mt-1">Scan documents for sensitive entities before storage or sharing</p>
      </div>

      <label
        onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => { e.preventDefault(); setDragging(false); addFiles(e.dataTransfer.files) }}
        className={`flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed p-12 cursor-pointer transition-colors ${
          dragging ? 'border-primary bg-primary/5' : 'border-border bg-card hover:border-primary/40'
        }`}
      >
        <div className="rounded-full bg-primary/10 p-3">
          <Upload className="h-6 w-6 text-primary" />
        </div>
        <p className="text-sm font-medium text-foreground">Drag & drop documents here or click to browse</p>
        <p className="text-xs text-muted-foreground">PDF, DOCX, TXT, CSV up to 50MB</p>
        <input type="file" multiple className="sr-only" onChange={(e) => addFiles(e.target.files)} />
      </label>

      {files.length > 0 && (
        <div className="space-y-3">
          {files.map((f, i) => (
            <div key={i} className="flex items-center justify-between rounded-lg border border-border bg-card p-4">
              <div className="flex items-center gap-3">
                <FileText className="h-5 w-5 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium text-foreground">{f.name}</p>
                  <p className="text-xs text-muted-foreground">{f.size}</p>
                </div>
              </div>
              {f.status === 'scanning' ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Scanning...
                </div>
              ) : (
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                  <span className={f.findings > 0 ? 'text-amber-500 font-medium' : 'text-emerald-500'}>
                    {f.findings > 0 ? `${f.findings} sensitive entities found` : 'No sensitive data'}
                  </span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
