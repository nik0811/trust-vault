'use client'

import { useState, useCallback, useRef } from 'react'
import { Breadcrumbs } from '@/components/base/breadcrumbs'
import {
  FileText, FileSpreadsheet, Upload, Shield, AlertTriangle,
  Loader2, Eye, CheckCircle2, File
} from 'lucide-react'
import { useClassifyDocument, useDocumentClassifications } from '@/hooks/use-classification'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'

interface UploadedDoc {
  id: string
  name: string
  text: string
  uploadedAt: string
  fileType?: string
}

const PII_COLORS: Record<string, string> = {
  EMAIL: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  SSN: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  CREDIT_CARD: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  PHONE: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  DATE_OF_BIRTH: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  ADDRESS: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  MEDICAL_RECORD: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  DEFAULT: 'bg-muted text-muted-foreground',
}

const LABEL_COLORS: Record<string, string> = {
  RESTRICTED: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  CONFIDENTIAL: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  INTERNAL: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
}

const SUPPORTED_TYPES = '.txt,.md,.log,.csv,.tsv,.json,.pdf,.xlsx,.xls,.docx'
const SUPPORTED_MIMES = [
  'text/plain', 'text/markdown', 'text/csv', 'text/tab-separated-values',
  'application/json',
  'application/pdf',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
]

function fileIcon(name: string) {
  const ext = name.split('.').pop()?.toLowerCase() ?? ''
  if (['csv', 'tsv', 'xlsx', 'xls'].includes(ext)) return <FileSpreadsheet className="h-4 w-4 text-green-500 shrink-0" />
  if (ext === 'pdf') return <File className="h-4 w-4 text-red-500 shrink-0" />
  if (['docx', 'doc'].includes(ext)) return <File className="h-4 w-4 text-blue-500 shrink-0" />
  return <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
}

function DocumentDetail({ doc, onClose }: { doc: UploadedDoc; onClose: () => void }) {
  const { data: classifications = [], isLoading } = useDocumentClassifications(doc.id)
  const classResult = classifications[0]

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-card rounded-xl border border-border w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-card border-b border-border p-5 flex items-start justify-between">
          <div>
            <h2 className="text-lg font-bold">{doc.name}</h2>
            <p className="text-sm text-muted-foreground">Document ID: {doc.id}</p>
          </div>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground text-xl leading-none">×</button>
        </div>
        <div className="p-5 space-y-5">
          {isLoading ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin mr-2" /> Loading classifications...
            </div>
          ) : classResult ? (
            <>
              {classResult.governed && (
                <div className={cn(
                  'flex items-center gap-3 rounded-lg p-4 border',
                  classResult.label_applied
                    ? 'bg-red-50 border-red-200 dark:bg-red-950/20 dark:border-red-800'
                    : 'bg-yellow-50 border-yellow-200 dark:bg-yellow-950/20 dark:border-yellow-800'
                )}>
                  <Shield className={cn('h-5 w-5 shrink-0', classResult.label_applied ? 'text-red-500' : 'text-yellow-500')} />
                  <div>
                    <p className="font-medium text-sm">Governed Document</p>
                    {classResult.label_applied && (
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Label applied: <span className="font-semibold">{classResult.label_applied}</span>
                      </p>
                    )}
                  </div>
                  {classResult.label_applied && (
                    <span className={cn('ml-auto text-xs px-2 py-0.5 rounded-full font-medium', LABEL_COLORS[classResult.label_applied] ?? 'bg-muted text-muted-foreground')}>
                      {classResult.label_applied}
                    </span>
                  )}
                </div>
              )}

              {Array.isArray(classResult.entity_types) && classResult.entity_types.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold mb-2">Detected PII Types</h3>
                  <div className="flex flex-wrap gap-2">
                    {classResult.entity_types.map((et: string) => (
                      <span key={et} className={cn('text-xs px-2 py-1 rounded-full font-medium', PII_COLORS[et] ?? PII_COLORS.DEFAULT)}>
                        {et}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {Array.isArray(classResult.findings) && classResult.findings.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold mb-2">Findings ({classResult.findings.length})</h3>
                  <div className="rounded-lg border border-border overflow-hidden">
                    <table className="w-full text-xs">
                      <thead className="bg-muted/50">
                        <tr>
                          {['Type', 'Value (Masked)', 'Confidence'].map(h => (
                            <th key={h} className="text-left px-3 py-2 font-semibold text-muted-foreground uppercase">{h}</th>
                          ))}
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {classResult.findings.slice(0, 20).map((f: any, i: number) => (
                          <tr key={i} className="hover:bg-muted/30">
                            <td className="px-3 py-2">
                              <span className={cn('px-1.5 py-0.5 rounded text-xs', PII_COLORS[f.entity_type] ?? PII_COLORS.DEFAULT)}>
                                {f.entity_type}
                              </span>
                            </td>
                            <td className="px-3 py-2 font-mono">{f.masked_value ?? f.value ?? '***'}</td>
                            <td className="px-3 py-2">{f.confidence != null ? `${(f.confidence * 100).toFixed(0)}%` : '—'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <AlertTriangle className="h-5 w-5 mr-2" />
              No classification results yet.
            </div>
          )}

          <div>
            <h3 className="text-sm font-semibold mb-2">Document Preview</h3>
            <div className="bg-muted/30 rounded-lg p-3 text-xs font-mono text-muted-foreground max-h-40 overflow-y-auto whitespace-pre-wrap">
              {doc.text.slice(0, 500)}{doc.text.length > 500 ? '…' : ''}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function DocumentRow({ doc }: { doc: UploadedDoc }) {
  const { data: classifications = [] } = useDocumentClassifications(doc.id)
  const classResult = classifications[0]
  const [showDetail, setShowDetail] = useState(false)

  const entityTypes: string[] = Array.isArray(classResult?.entity_types)
    ? classResult.entity_types
    : (typeof classResult?.entity_types === 'string' ? JSON.parse(classResult.entity_types || '[]') : [])

  return (
    <>
      <tr className="hover:bg-muted/30 transition-colors">
        <td className="px-4 py-3">
          <div className="flex items-center gap-2">
            {fileIcon(doc.name)}
            <span className="font-medium text-sm">{doc.name}</span>
          </div>
        </td>
        <td className="px-4 py-3">
          <div className="flex flex-wrap gap-1">
            {entityTypes.length === 0 ? (
              <span className="text-xs text-muted-foreground">—</span>
            ) : entityTypes.slice(0, 3).map(et => (
              <span key={et} className={cn('text-xs px-1.5 py-0.5 rounded', PII_COLORS[et] ?? PII_COLORS.DEFAULT)}>
                {et}
              </span>
            ))}
            {entityTypes.length > 3 && (
              <span className="text-xs text-muted-foreground">+{entityTypes.length - 3} more</span>
            )}
          </div>
        </td>
        <td className="px-4 py-3">
          {classResult?.governed ? (
            <div className="flex items-center gap-1.5">
              <AlertTriangle className="h-4 w-4 text-yellow-500 shrink-0" />
              {classResult.label_applied ? (
                <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', LABEL_COLORS[classResult.label_applied] ?? 'bg-muted text-muted-foreground')}>
                  {classResult.label_applied}
                </span>
              ) : (
                <span className="text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 px-2 py-0.5 rounded-full">Governed</span>
              )}
            </div>
          ) : classResult ? (
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
              <span className="text-xs text-muted-foreground">Clean</span>
            </div>
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          )}
        </td>
        <td className="px-4 py-3 text-xs text-muted-foreground">
          {new Date(doc.uploadedAt).toLocaleDateString()}
        </td>
        <td className="px-4 py-3">
          <button
            onClick={() => setShowDetail(true)}
            className="flex items-center gap-1 text-xs text-primary hover:text-primary/80"
          >
            <Eye className="h-3.5 w-3.5" />
            View
          </button>
        </td>
      </tr>
      {showDetail && <DocumentDetail doc={doc} onClose={() => setShowDetail(false)} />}
    </>
  )
}

export default function DocumentsPage() {
  const [docs, setDocs] = useState<UploadedDoc[]>([])
  const [dragging, setDragging] = useState(false)
  const [textInput, setTextInput] = useState('')
  const [docName, setDocName] = useState('')
  const [uploading, setUploading] = useState(false)
  const [uploadStatus, setUploadStatus] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const classifyDoc = useClassifyDocument()

  // Upload text as document classification
  const handleUpload = useCallback(async () => {
    if (!textInput.trim()) return
    const id = `doc-${Date.now()}`
    const name = docName.trim() || `Document ${docs.length + 1}`
    const newDoc: UploadedDoc = { id, name, text: textInput, uploadedAt: new Date().toISOString() }
    setDocs(prev => [newDoc, ...prev])
    try {
      await classifyDoc.mutateAsync({ document_id: id, document_name: name, text: textInput })
    } catch {}
    setTextInput('')
    setDocName('')
  }, [textInput, docName, docs.length, classifyDoc])

  // Upload a binary/structured file via multipart to /documents/extract
  const handleFileUpload = useCallback(async (file: File) => {
    setUploading(true)
    setUploadStatus(`Uploading ${file.name}…`)

    try {
      const formData = new FormData()
      formData.append('file', file, file.name)

      const resp = await api.post('/documents/extract', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })

      const extractionId: string = resp.data.extraction_id ?? `doc-${Date.now()}`

      // Read file as text for preview (best-effort; binary files show partial content)
      const preview = await new Promise<string>(resolve => {
        const reader = new FileReader()
        reader.onload = ev => resolve((ev.target?.result as string) ?? '')
        reader.onerror = () => resolve('[binary file]')
        reader.readAsText(file)
      })

      const newDoc: UploadedDoc = {
        id: extractionId,
        name: file.name,
        text: preview,
        uploadedAt: new Date().toISOString(),
        fileType: file.type,
      }
      setDocs(prev => [newDoc, ...prev])
      setUploadStatus(`${file.name} uploaded — classification in progress`)

      // Poll for classifications becoming available (backend classifies async)
      let attempts = 0
      const poll = setInterval(async () => {
        attempts++
        try {
          const check = await api.get(`/documents/${extractionId}/classifications`)
          if (check.data?.length > 0 || attempts >= 12) {
            clearInterval(poll)
            setUploadStatus(null)
          }
        } catch {
          if (attempts >= 12) {
            clearInterval(poll)
            setUploadStatus(null)
          }
        }
      }, 2500)
    } catch (err: any) {
      setUploadStatus(`Upload failed: ${err?.response?.data?.error ?? err?.message ?? 'unknown error'}`)
    } finally {
      setUploading(false)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (!file) return
    handleFileUpload(file)
  }, [handleFileUpload])

  const handleFileInput = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    handleFileUpload(file)
    // Reset input so same file can be re-selected
    e.target.value = ''
  }, [handleFileUpload])

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Documents', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Document Governance</h1>
        <p className="text-sm text-muted-foreground mt-1">Upload documents for PII classification and automatic governance</p>
      </div>

      <div className="p-8 space-y-8">
        {/* Drop Zone */}
        <div
          className={cn(
            'rounded-lg border-2 border-dashed p-10 text-center cursor-pointer transition-colors',
            dragging ? 'border-primary bg-primary/5' : 'border-border bg-muted/30 hover:bg-muted/50'
          )}
          onDragOver={e => { e.preventDefault(); setDragging(true) }}
          onDragLeave={() => setDragging(false)}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept={SUPPORTED_TYPES}
            className="hidden"
            onChange={handleFileInput}
          />
          {uploading ? (
            <Loader2 className="h-10 w-10 text-primary mx-auto mb-3 animate-spin" />
          ) : (
            <Upload className="h-10 w-10 text-muted-foreground mx-auto mb-3" />
          )}
          <h3 className="text-lg font-semibold mb-1">
            {uploading ? 'Uploading…' : 'Drop a file here or click to browse'}
          </h3>
          <p className="text-sm text-muted-foreground">
            Supported: TXT, CSV, TSV, JSON, PDF, XLSX, XLS, DOCX, MD
          </p>
          {uploadStatus && (
            <p className="mt-3 text-xs text-primary font-medium">{uploadStatus}</p>
          )}
        </div>

        {/* Paste Text */}
        <div className="rounded-lg border border-border bg-card p-6 space-y-4">
          <h3 className="font-semibold">Classify Pasted Text</h3>
          <input
            className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background focus:outline-none focus:ring-1 focus:ring-primary"
            placeholder="Document name (optional)"
            value={docName}
            onChange={e => setDocName(e.target.value)}
          />
          <textarea
            className="w-full border border-border rounded-md px-3 py-2 text-sm bg-background resize-none focus:outline-none focus:ring-1 focus:ring-primary"
            rows={6}
            placeholder="Paste document text here to classify for PII…"
            value={textInput}
            onChange={e => setTextInput(e.target.value)}
          />
          <button
            onClick={handleUpload}
            disabled={!textInput.trim() || classifyDoc.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:opacity-90 disabled:opacity-50"
          >
            {classifyDoc.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
            Classify &amp; Govern
          </button>
        </div>

        {/* Documents Table */}
        <div className="rounded-lg border border-border bg-card">
          <div className="p-5 border-b border-border">
            <h3 className="font-semibold">Processed Documents</h3>
          </div>
          {docs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <FileText className="h-10 w-10 mb-3 opacity-30" />
              <p>No documents yet. Upload a file or paste text above.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    {['Document', 'Classifications', 'Governance', 'Date', ''].map(h => (
                      <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wide">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {docs.map(doc => <DocumentRow key={doc.id} doc={doc} />)}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
