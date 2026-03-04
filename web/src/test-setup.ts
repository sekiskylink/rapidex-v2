import '@testing-library/jest-dom/vitest'

if (typeof globalThis.Response === 'undefined') {
  // TanStack Router uses Response in redirect handling.
  Object.defineProperty(globalThis, 'Response', {
    value: class {},
    configurable: true,
    writable: true,
  })
}

if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: query === '(prefers-color-scheme: dark)' ? false : false,
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  })
}
