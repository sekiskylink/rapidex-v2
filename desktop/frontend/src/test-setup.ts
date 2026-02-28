import '@testing-library/jest-dom/vitest'

if (typeof globalThis.Response === 'undefined') {
  // TanStack Router checks for Response existence when handling redirects.
  Object.defineProperty(globalThis, 'Response', {
    value: class {},
    configurable: true,
    writable: true,
  })
}
