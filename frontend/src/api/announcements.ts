/**
 * User Announcements API endpoints
 */

import { apiClient } from './client'
import type { Announcement, UserAnnouncement } from '@/types'

export async function list(unreadOnly: boolean = false): Promise<UserAnnouncement[]> {
  const { data } = await apiClient.get<UserAnnouncement[]>('/announcements', {
    params: unreadOnly ? { unread_only: 1 } : {}
  })
  return data
}

export async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/announcements/${id}/read`)
  return data
}

export interface HomepageAnnouncementsResponse {
  pinned?: Announcement
  recent: Announcement[]
}

export async function getHomepage(): Promise<HomepageAnnouncementsResponse> {
  const { data } = await apiClient.get<HomepageAnnouncementsResponse>('/announcements/homepage')
  return data
}

const announcementsAPI = {
  list,
  markRead,
  getHomepage
}

export default announcementsAPI

