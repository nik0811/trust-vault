'use client'

import { useState, useEffect } from 'react'
import { useCurrentUser, useUpdateProfile } from '@/hooks/use-auth'
import { Mail, Phone, MapPin, Briefcase, ArrowLeft, Loader2 } from 'lucide-react'
import Link from 'next/link'
import { toast } from 'sonner'

export default function ProfilePage() {
  const { data: user, isLoading, error } = useCurrentUser()
  const updateProfile = useUpdateProfile()
  const [isEditing, setIsEditing] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    phone: '',
    location: '',
    department: '',
    bio: '',
  })

  useEffect(() => {
    if (error) toast.error('Failed to load profile')
  }, [error])

  useEffect(() => {
    if (user) {
      setFormData({
        name: (user as any).name || '',
        phone: (user as any).phone || '',
        location: (user as any).location || '',
        department: (user as any).department || '',
        bio: (user as any).bio || '',
      })
    }
  }, [user])

  const handleSave = async () => {
    try {
      await updateProfile.mutateAsync(formData)
      setIsEditing(false)
    } catch {
      // Error handled by hook
    }
  }

  const profileImage = formData.name ? formData.name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2) : 'U'

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="border-b border-border bg-card">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center gap-4 mb-4">
            <Link href="/dashboard" className="p-2 hover:bg-muted rounded-lg transition-colors">
              <ArrowLeft className="h-5 w-5 text-foreground" />
            </Link>
            <h1 className="text-3xl font-bold text-foreground">Profile</h1>
          </div>
          <p className="text-muted-foreground">Manage your account and profile information</p>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-1">
            <div className="rounded-lg border border-border bg-card p-6 text-center">
              <div className="h-24 w-24 rounded-full bg-primary/20 flex items-center justify-center mx-auto mb-4">
                <span className="text-3xl font-bold text-primary">{profileImage}</span>
              </div>
              <h2 className="text-2xl font-bold text-foreground mb-1">{formData.name || 'User'}</h2>
              <p className="text-sm text-muted-foreground mb-4">{(user as any)?.role || 'Member'}</p>
              <button
                onClick={() => isEditing ? handleSave() : setIsEditing(true)}
                disabled={updateProfile.isPending}
                className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
              >
                {updateProfile.isPending ? 'Saving...' : isEditing ? 'Save Changes' : 'Edit Profile'}
              </button>
              {isEditing && (
                <button
                  onClick={() => setIsEditing(false)}
                  className="w-full mt-2 px-4 py-2 border border-border rounded-lg hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
              )}
            </div>

            <div className="mt-6 rounded-lg border border-border bg-card p-6">
              <h3 className="font-semibold text-foreground mb-4">Account Info</h3>
              <div className="space-y-3">
                <div>
                  <p className="text-xs text-muted-foreground uppercase font-semibold">Member Since</p>
                  <p className="text-sm text-foreground mt-1">{user?.created_at ? new Date(user.created_at).toLocaleDateString() : '—'}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground uppercase font-semibold">Status</p>
                  <p className="text-sm text-foreground mt-1">{user?.status || 'Active'}</p>
                </div>
              </div>
            </div>
          </div>

          <div className="lg:col-span-2 space-y-6">
            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-6">Personal Information</h3>
              <div className="space-y-6">
                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Full Name</label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    disabled={!isEditing}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                  />
                </div>

                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Email Address</label>
                  <div className="flex items-center gap-3">
                    <Mail className="h-5 w-5 text-muted-foreground" />
                    <input
                      type="email"
                      value={user?.email || ''}
                      disabled
                      className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Phone</label>
                  <div className="flex items-center gap-3">
                    <Phone className="h-5 w-5 text-muted-foreground" />
                    <input
                      type="tel"
                      value={formData.phone}
                      onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                      disabled={!isEditing}
                      className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Location</label>
                  <div className="flex items-center gap-3">
                    <MapPin className="h-5 w-5 text-muted-foreground" />
                    <input
                      type="text"
                      value={formData.location}
                      onChange={(e) => setFormData({ ...formData, location: e.target.value })}
                      disabled={!isEditing}
                      className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                    />
                  </div>
                </div>
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-6">Professional Information</h3>
              <div className="space-y-6">
                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Department</label>
                  <div className="flex items-center gap-3">
                    <Briefcase className="h-5 w-5 text-muted-foreground" />
                    <input
                      type="text"
                      value={formData.department}
                      onChange={(e) => setFormData({ ...formData, department: e.target.value })}
                      disabled={!isEditing}
                      className="flex-1 px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs text-muted-foreground uppercase font-semibold mb-2">Bio</label>
                  <textarea
                    value={formData.bio}
                    onChange={(e) => setFormData({ ...formData, bio: e.target.value })}
                    disabled={!isEditing}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground disabled:opacity-50 disabled:cursor-not-allowed resize-none"
                    rows={4}
                  />
                </div>
              </div>
            </div>

            <div className="rounded-lg border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground mb-6">Security</h3>
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-foreground">Password</p>
                    <p className="text-xs text-muted-foreground">Change your account password</p>
                  </div>
                  <button className="px-4 py-2 border border-border rounded-lg hover:bg-muted transition-colors text-sm font-medium">
                    Change Password
                  </button>
                </div>
                <hr className="border-border" />
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-foreground">Two-Factor Authentication</p>
                    <p className="text-xs text-muted-foreground">Add an extra layer of security</p>
                  </div>
                  <button className="px-4 py-2 border border-border rounded-lg hover:bg-muted transition-colors text-sm font-medium">
                    Enable
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
