export type NotificationSeverity = 'info' | 'success' | 'warning' | 'error'

export interface Notification {
  message: string
  severity: NotificationSeverity
}

const listeners = new Set<(notification: Notification) => void>()

export function notify(notification: Notification) {
  for (const listener of listeners) {
    listener(notification)
  }
}

export function subscribeNotifications(listener: (notification: Notification) => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}
