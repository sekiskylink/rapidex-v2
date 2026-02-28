declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          LoadSettings?: () => Promise<unknown>
          SaveSettings?: (patch: unknown) => Promise<unknown>
          ResetSettings?: () => Promise<unknown>
        }
      }
    }
  }
}

export {}
