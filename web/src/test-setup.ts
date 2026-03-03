import '@testing-library/jest-dom/vitest'

if (typeof globalThis.Response === 'undefined') {
  // TanStack Router uses Response in redirect handling.
  Object.defineProperty(globalThis, 'Response', {
    value: class {},
    configurable: true,
    writable: true,
  })
}
