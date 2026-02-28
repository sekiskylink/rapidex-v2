import '@testing-library/jest-dom/vitest'

if (typeof globalThis.Response === 'undefined') {
  // TanStack Router checks for Response existence when handling redirects.
  ;(globalThis as any).Response = class {}
}
