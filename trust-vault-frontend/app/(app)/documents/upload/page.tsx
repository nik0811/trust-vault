'use client'

import { useState, useRef } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { Upload, FileText, CheckCircle2, Loader2, AlertCircle } from 'lucide-react'
import { api } from '@/lib/api'

interface FileResult {
  name: string
  size: string
  status: 'uploading' | 'done' | 'error'
  findings: number
  error?: string
}

export default function DocumentUploadPage() {
  const [dragging, setDragging] = useState(false)
  const [files, setFiles] = useState<FileResult[]>([])
  const uploadingRef = useRef(new Set<string>())

  const uploadFile = async (file: File) => {
    if (uploadingRef.current.has(file.name)) return
    uploadingRef.current.add(file.name)

    const entry: FileResult = {
      name: file.name,
      size: `${(file.size / 1024).toFixed(1)} KB`,
      status: 'uploading',
      findings: 0,
    }
    setFiles(prev => [...prev, entry])

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await api.post('/documents/extract', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })

      const data = response.data as any
      const entityCount = data.entity_count ?? data.entities?.length ?? 0

      setFiles(prev =>
        prev.map(f =>
          f.name === file.name ? { ...f, status: 'done', findings: entityCount } : f
        )
      )
    } catch (err: any) {
      const errMsg = err?.response?.data?.error || 'Upload failed'
      setFiles(prev =>
        prev.map(f =>
          f.name === file.name ? { ...f, status: 'error', error: errMsg } : f
        )
      )
    } finally {
      uploadingRef.current.delete(file.name)
    }
  }

  const addFiles = (list: FileList | null) => {
    if (!list) return
    Array.from(list).forEach(uploadFile)
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
              {f.status === 'uploading' ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Scanning...
                </div>
              ) : f.status === 'error' ? (
                <div className="flex items-center gap-2 text-sm text-red-500">
                  <AlertCircle className="h-4 w-4" />
                  {f.error || 'Upload failed'}
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
