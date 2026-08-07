```typescript
// src/app/models.test.ts

import {
  Book,
  CreateBookInput,
  UpdateBookInput,
  createBookSchema,
  updateBookSchema,
  bookToString,
  bookToRepr,
} from './models';

// ---------------------------------------------------------------------------
// Helper factory
// ---------------------------------------------------------------------------

function makeBook(overrides: Partial<Book> = {}): Book {
  return {
    id: 1,
    title: 'Dune',
    author: 'Frank Herbert',
    publishedYear: 1965,
    summary: null,
    ...overrides,
  };
}

// ===========================================================================
// bookToRepr  (Book.__repr__)
// ===========================================================================

describe('bookToRepr (Book.__repr__)', () => {
  describe('format invariants', () => {
    it('always starts with "<Book(" and ends with ")>"', () => {
      const result = bookToRepr(makeBook());
      expect(result.startsWith('<Book(')).toBe(true);
      expect(result.endsWith(')>')).toBe(true);
    });

    it('always contains id, title, author, and year fields in that order', () => {
      const result = bookToRepr(makeBook());
      const idIdx = result.indexOf('id=');
      const titleIdx = result.indexOf('title=');
      const authorIdx = result.indexOf('author=');
      const yearIdx = result.indexOf('year=');

      expect(idIdx).toBeGreaterThan(-1);
      expect(titleIdx).toBeGreaterThan(idIdx);
      expect(authorIdx).toBeGreaterThan(titleIdx);
      expect(yearIdx).toBeGreaterThan(authorIdx);
    });

    it('wraps title and author in single quotes but not id or year', () => {
      const book = makeBook({ id: 42, title: 'Dune', author: 'Frank Herbert', publishedYear: 1965 });
      const result = bookToRepr(book);

      expect(result).toMatch(/title='[^']*'/);
      expect(result).toMatch(/author='[^']*'/);
      // id and year should not be wrapped in quotes
      expect(result).toMatch(/id=\d+[^']/);
      expect(result).toMatch(/year=\d+/);
      expect(result).not.toMatch(/id='\d+'/);
      expect(result).not.toMatch(/year='\d+'/);
    });
  });

  describe('scenario: Book with id=1, title="Dune", author="Frank Herbert", published_year=1965', () => {
    it('returns the exact expected string', () => {
      const book = makeBook({ id: 1, title: 'Dune', author: 'Frank Herbert', publishedYear: 1965 });
      expect(bookToRepr(book)).toBe("<Book(id=1, title='Dune', author='Frank Herbert', year=1965)>");
    });
  });

  describe('scenario: Book whose id has not yet been assigned (None before persistence)', () => {
    it('returns a string containing "id=None" when id is null/undefined', () => {
      // TypeScript does not allow undefined for id on the Book interface, but
      // in a pre-persistence scenario the value could be coerced. We cast to
      // test the runtime path the spec describes.
      const book = makeBook({ id: null as unknown as number });
      const result = bookToRepr(book);
      expect(result).toContain('id=null');
    });
  });

  describe('scenario: Book with title or author containing single quotes', () => {
    it('embeds raw title/author values inside single quotes verbatim', () => {
      const book = makeBook({
        id: 7,
        title: "O'Brien's Book",
        author: "Pat O'Brien",
        publishedYear: 2000,
      });
      const result = bookToRepr(book);
      expect(result).toContain("title='O'Brien's Book'");
      expect(result).toContain("author='Pat O'Brien'");
    });
  });

  describe('various id and year values', () => {
    it('correctly formats id=0 and year=0', () => {
      const book = makeBook({ id: 0, title: 'Genesis', author: 'Anon', publishedYear: 0 });
      expect(bookToRepr(book)).toBe("<Book(id=0, title='Genesis', author='Anon', year=0)>");
    });

    it('correctly formats large id and year values', () => {
      const book = makeBook({ id: 99999, title: 'Future', author: 'Bot', publishedYear: 9999 });
      expect(bookToRepr(book)).toBe("<Book(id=99999, title='Future', author='Bot', year=9999)>");
    });
  });
});

// ===========================================================================
// bookToString  (Book.__str__)
// ===========================================================================

describe('bookToString (Book.__str__)', () => {
  describe('format invariants', () => {
    it('always follows the pattern "<title> by <author> (<published_year>)"', () => {
      const book = makeBook({ title: 'Neuromancer', author: 'William Gibson', publishedYear: 1984 });
      expect(bookToString(book)).toBe('Neuromancer by William Gibson (1984)');
    });

    it('never includes the summary in the output', () => {
      const book = makeBook({ summary: 'A great novel about sand and spice.' });
      const result = bookToString(book);
      expect(result).not.toContain('sand');
      expect(result).not.toContain('great novel');
      expect(result).not.toContain(book.summary!);
    });

    it('never includes the id in the output', () => {
      const book = makeBook({ id: 42 });
      const result = bookToString(book);
      expect(result).not.toContain('42');
    });
  });

  describe('scenario: Book with title="Dune", author="Frank Herbert", published_year=1965', () => {
    it('returns the exact expected string', () => {
      const book = makeBook({ title: 'Dune', author: 'Frank Herbert', publishedYear: 1965 });
      expect(bookToString(book)).toBe('Dune by Frank Herbert (1965)');
    });
  });

  describe('scenario: Book with a summary set', () => {
    it('does not include the summary (only title, author, year)', () => {
      const book = makeBook({
        title: 'Dune',
        author: 'Frank Herbert',
        publishedYear: 1965,
        summary: 'Epic science-fiction novel set in a distant future.',
      });
      const result = bookToString(book);
      expect(result).toBe('Dune by Frank Herbert (1965)');
      expect(result).not.toContain('Epic');
    });
  });

  describe('edge cases', () => {
    it('handles titles/authors with special characters', () => {
      const book = makeBook({ title: 'Léa & Friends', author: 'Ö. Müller', publishedYear: 2022 });
      expect(bookToString(book)).toBe('Léa & Friends by Ö. Müller (2022)');
    });

    it('handles publishedYear=0', () => {
      const book = makeBook({ title: 'Ancient', author: 'Unknown', publishedYear: 0 });
      expect(bookToString(book)).toBe('Ancient by Unknown (0)');
    });
  });
});

// ===========================================================================
// createBookSchema validation
// ===========================================================================

describe('createBookSchema', () => {
  const VALID_INPUT: CreateBookInput = {
    title: 'Dune',
    author: 'Frank Herbert',
    publishedYear: 1965,
    summary: null,
  };

  it('accepts a fully valid input', () => {
    expect(() => createBookSchema.parse(VALID_INPUT)).not.toThrow();
  });

  it('accepts input without summary (summary is optional)', () => {
    const { summary, ...rest } = VALID_INPUT;
    expect(() => createBookSchema.parse(rest)).not.toThrow();
  });

  it('accepts input with summary as null', () => {
    expect(() => createBookSchema.parse({ ...VALID_INPUT, summary: null })).not.toThrow();
  });

  it('accepts input with a non-null summary', () => {
    expect(() => createBookSchema.parse({ ...VALID_INPUT, summary: 'A classic.' })).not.toThrow();
  });

  describe('title validation', () => {
    it('rejects an empty title', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, title: '' })).toThrow();
    });

    it('rejects a title exceeding 255 characters', () => {
      expect(() =>
        createBookSchema.parse({ ...VALID_INPUT, title: 'a'.repeat(256) }),
      ).toThrow();
    });

    it('accepts a title of exactly 255 characters', () => {
      expect(() =>
        createBookSchema.parse({ ...VALID_INPUT, title: 'a'.repeat(255) }),
      ).not.toThrow();
    });

    it('rejects a missing title', () => {
      const { title, ...rest } = VALID_INPUT;
      expect(() => createBookSchema.parse(rest)).toThrow();
    });
  });

  describe('author validation', () => {
    it('rejects an empty author', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, author: '' })).toThrow();
    });

    it('rejects an author exceeding 255 characters', () => {
      expect(() =>
        createBookSchema.parse({ ...VALID_INPUT, author: 'a'.repeat(256) }),
      ).toThrow();
    });

    it('accepts an author of exactly 255 characters', () => {
      expect(() =>
        createBookSchema.parse({ ...VALID_INPUT, author: 'a'.repeat(255) }),
      ).not.toThrow();
    });

    it('rejects a missing author', () => {
      const { author, ...rest } = VALID_INPUT;
      expect(() => createBookSchema.parse(rest)).toThrow();
    });
  });

  describe('publishedYear validation', () => {
    it('rejects a missing publishedYear', () => {
      const { publishedYear, ...rest } = VALID_INPUT;
      expect(() => createBookSchema.parse(rest)).toThrow();
    });

    it('rejects a non-integer publishedYear', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, publishedYear: 1965.5 })).toThrow();
    });

    it('rejects a publishedYear below the minimum (< 0)', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, publishedYear: -1 })).toThrow();
    });

    it('rejects a publishedYear above the maximum (> 9999)', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, publishedYear: 10000 })).toThrow();
    });

    it('accepts publishedYear=0 (boundary)', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, publishedYear: 0 })).not.toThrow();
    });

    it('accepts publishedYear=9999 (boundary)', () => {
      expect(() => createBookSchema.parse({ ...VALID_INPUT, publishedYear: 9999 })).not.toThrow();
    });

    it('rejects a string publishedYear', () => {
      expect(() =>
        createBookSchema.parse({ ...VALID_INPUT, publishedYear: '1965' as unknown as number }),
      ).toThrow();
    });
  });
});

// ===========================================================================
// updateBookSchema validation
// ===========================================================================

describe('updateBookSchema', () => {
  it('accepts an empty object (all fields optional)', () => {
    expect(() => updateBookSchema.parse({})).not.toThrow();
  });

  it('accepts a partial update with only title', () => {
    expect(() => updateBookSchema.parse({ title: 'New Title' })).not.toThrow();
  });

  it('accepts a partial update with only author', () => {
    expect(() => updateBookSchema.parse({ author: 'New Author' })).not.toThrow();
  });

  it('accepts a partial update with only publishedYear', () => {
    expect(() => updateBookSchema.parse({ publishedYear: 2024 })).not.toThrow();
  });

  it('still enforces title max length when title is provided', () => {
    expect(() =>
      updateBookSchema.parse({ title: 'a'.repeat(256) }),
    ).toThrow();
  });

  it('still enforces author max length when author is provided', () => {
    expect(() =>
      updateBookSchema.parse({ author: 'a'.repeat(256) }),
    ).toThrow();
  });

  it('still enforces non-empty title when title is provided', () => {
    expect(() => updateBookSchema.parse({ title: '' })).toThrow();
  });

  it('still enforces integer publishedYear when provided', () => {
    expect(() => updateBookSchema.parse({ publishedYear: 1965.7 })).toThrow();
  });

  it('still enforces year range when publishedYear is provided', () => {
    expect(() => updateBookSchema.parse({ publishedYear: -10 })).toThrow();
  });

  it('accepts a full update object identical to createBookSchema input', () => {
    const fullUpdate: UpdateBookInput = {
      title: 'Dune Messiah',
      author: 'Frank Herbert',
      publishedYear: 1969,
      summary: 'The sequel.',
    };
    expect(() => updateBookSchema.parse(fullUpdate)).not.toThrow();
  });
});

// ===========================================================================
// Book interface shape (structural / type-level sanity checks)
// ===========================================================================

describe('Book interface', () => {
  it('can be constructed with all required fields and a null summary', () => {
    const book: Book = {
      id: 1,
      title: 'Dune',
      author: 'Frank Herbert',
      publishedYear: 1965,
      summary: null,
    };
    expect(book.id).toBe(1);
    expect(book.summary).toBeNull();
  });

  it('can be constructed with a non-null summary', () => {
    const book: Book = {
      id: 2,
      title: 'Dune',
      author: 'Frank Herbert',
      publishedYear: 1965,
      summary: 'A sprawling epic.',
    };
    expect(typeof book.summary).toBe('string');
  });
});

// ===========================================================================
// Global invariants – schema definition documentation tests
// ===========================================================================

describe('Global model invariants (schema documentation)', () => {
  /**
   * These tests assert that the Zod schemas enforce the constraints described
   * in the global_invariants. They do not test the Prisma schema directly
   * (that is a build-time artefact), but they validate the same rules at the
   * application layer.
   */

  it('title is required and non-nullable in createBookSchema', () => {
    expect(() => createBookSchema.parse({ author: 'A', publishedYear: 2000 })).toThrow();
  });

  it('author is required and non-nullable in createBookSchema', () => {
    expect(() => createBookSchema.parse({ title: 'T', publishedYear: 2000 })).toThrow();
  });

  it('publishedYear is required and non-nullable in createBookSchema', () => {
    expect(() => createBookSchema.parse({ title: 'T', author: 'A' })).toThrow();
  });

  it('summary is nullable/optional in createBookSchema', () => {
    expect(() =>
      createBookSchema.parse({ title: 'T', author: 