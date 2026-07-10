'use client'

import { Breadcrumbs } from '@/components/base/breadcrumbs'
import { FileText, Upload } from 'lucide-react'

export default function DocumentsPage() {
  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card px-8 py-6">
        <Breadcrumbs items={[{ label: 'Documents', active: true }]} />
        <h1 className="text-3xl font-bold text-foreground mt-4">Documents</h1>
        <p className="text-sm text-muted-foreground mt-1">Upload and process documents for classification</p>
      </div>

      <div className="p-8">
        <div className="rounded-lg border border-dashed border-border bg-muted/50 p-12 text-center cursor-pointer hover:bg-muted transition-colors">
          <Upload className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-foreground mb-2">Drop files here to upload</h3>
          <p className="text-sm text-muted-foreground mb-4">or click to browse</p>
          <p className="text-xs text-muted-foreground">Supported formats: PDF, DOCX, TXT</p>
        </div>

        <div className="mt-8 rounded-lg border border-border bg-card p-6">
          <h3 className="text-lg font-semibold text-foreground mb-4">Recent Documents</h3>
          <div className="flex items-center justify-center py-8 text-muted-foreground">
            <FileText className="h-6 w-6 mr-2" />
            <p>No documents uploaded yet</p>
          </div>
        </div>
      </div>
    </div>
  )
}
