/**
 * Test to verify the test infrastructure is set up correctly
 */

import fc from 'fast-check';

describe('Test Infrastructure', () => {
  it('should run basic tests', () => {
    expect(true).toBe(true);
  });

  it('should have access to environment variables', () => {
    expect(process.env.JWT_ACCESS_SECRET).toBe('test-jwt-secret-key-for-testing-only');
    expect(process.env.ACCESS_TOKEN_TTL_SEC).toBe('900');
  });

  it('should support property-based testing with fast-check', () => {
    fc.assert(
      fc.property(fc.integer(), (n) => {
        expect(n + 0).toBe(n);
      }),
      { numRuns: 10 }
    );
  });
});
