// src/tests/__init.ts

/**
 * Test suite for Book Catalog API
 *
 * Contains unit tests for models and schemas, plus integration tests for API
 * endpoints.
 *
 * MIGRATION_NOTE: The Python source (`tests/__init__.py`) was a package marker
 * whose only job was to (a) mark the `tests/` directory as an importable Python
 * package and (b) carry a docstring describing the suite. TypeScript/Node.js
 * has no concept of package-init files — Jest discovers `*.test.ts` /
 * `*.spec.ts` files directly and each test file imports its own dependencies
 * explicitly. There is therefore no runtime or structural equivalent to port.
 *
 * This file is intentionally left as a documentation-only module so the suite's
 * top-level description is preserved in the target tree. It exports a small
 * metadata constant so the module is a real ES module (a bare doc comment would
 * otherwise be an empty, importless file). Nothing here needs to be imported by
 * the actual test files.
 */

export const TEST_SUITE_METADATA = {
  name: 'Book Catalog API Test Suite',
  description:
    'Unit tests for models and schemas, plus integration tests for API endpoints.',
} as const;

export type TestSuiteMetadata = typeof TEST_SUITE_METADATA;
