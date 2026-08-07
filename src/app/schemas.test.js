```typescript
// src/app/schemas.test.ts

import { bookCreateSchema, bookUpdateSchema, toBookResponse } from './schemas';
import type { Book } from './models';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const CURRENT_YEAR = new Date().getFullYear();

function getMessages(result: { error?: { issues: Array<{ message: string }> } }): string[] {
  return result.error?.issues.map((i) => i.message) ?? [];
}

// ---------------------------------------------------------------------------
// BookCreate
// ---------------------------------------------------------------------------

describe('bookCreateSchema', () => {
  // ------------------------------------------------------------------
  // Happy-path
  // ------------------------------------------------------------------

  describe('valid input', () => {
    it('accepts a fully populated valid object and strips whitespace', () => {
      const result = bookCreateSchema.safeParse({
        title: '  Clean Code  ',
        author: '  Robert Martin  ',
        published_year: 2008,
        summary: '  A book about writing clean code.  ',
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBe('Clean Code');
        expect(result.data.author).toBe('Robert Martin');
        expect(result.data.published_year).toBe(2008);
        expect(result.data.summary).toBe('A book about writing clean code.');
      }
    });

    it('accepts minimal valid input without summary', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Dune',
        author: 'Frank Herbert',
        published_year: 1965,
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBe('Dune');
        expect(result.data.author).toBe('Frank Herbert');
        expect(result.data.published_year).toBe(1965);
        expect(result.data.summary).toBeNull();
      }
    });

    it('accepts published_year equal to 1000 (lower boundary)', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Ancient Text',
        author: 'Unknown',
        published_year: 1000,
      });

      expect(result.success).toBe(true);
    });

    it('accepts published_year equal to the current year (upper boundary)', () => {
      const result = bookCreateSchema.safeParse({
        title: 'New Book',
        author: 'Author',
        published_year: CURRENT_YEAR,
      });

      expect(result.success).toBe(true);
    });
  });

  // ------------------------------------------------------------------
  // title
  // ------------------------------------------------------------------

  describe('title field', () => {
    it('trims leading and trailing whitespace from title', () => {
      const result = bookCreateSchema.safeParse({
        title: '   Trimmed Title   ',
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBe('Trimmed Title');
      }
    });

    it('rejects an empty title string', () => {
      const result = bookCreateSchema.safeParse({
        title: '',
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Title cannot be empty');
    });

    it('rejects a whitespace-only title', () => {
      const result = bookCreateSchema.safeParse({
        title: '   ',
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Title cannot be empty');
    });

    it('rejects a title longer than 255 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'A'.repeat(256),
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'ensure this value has at most 255 characters',
      );
    });

    it('accepts a title of exactly 255 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'A'.repeat(255),
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(true);
    });

    it('rejects missing title', () => {
      const result = bookCreateSchema.safeParse({
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
    });
  });

  // ------------------------------------------------------------------
  // author
  // ------------------------------------------------------------------

  describe('author field', () => {
    it('trims leading and trailing whitespace from author', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: '  Jane Doe  ',
        published_year: 2000,
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.author).toBe('Jane Doe');
      }
    });

    it('rejects an empty author string', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: '',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Author cannot be empty');
    });

    it('rejects a whitespace-only author', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: '   ',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Author cannot be empty');
    });

    it('rejects an author longer than 255 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'B'.repeat(256),
        published_year: 2000,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'ensure this value has at most 255 characters',
      );
    });

    it('accepts an author of exactly 255 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'B'.repeat(255),
        published_year: 2000,
      });

      expect(result.success).toBe(true);
    });

    it('rejects missing author', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        published_year: 2000,
      });

      expect(result.success).toBe(false);
    });
  });

  // ------------------------------------------------------------------
  // published_year
  // ------------------------------------------------------------------

  describe('published_year field', () => {
    it('rejects published_year less than 1000', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 999,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'Published year must be after year 1000',
      );
    });

    it('rejects published_year greater than the current year', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: CURRENT_YEAR + 1,
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        `Published year cannot be in the future (current year: ${CURRENT_YEAR})`,
      );
    });

    it('rejects a non-integer published_year', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000.5,
      });

      expect(result.success).toBe(false);
    });

    it('rejects missing published_year', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
      });

      expect(result.success).toBe(false);
    });
  });

  // ------------------------------------------------------------------
  // summary
  // ------------------------------------------------------------------

  describe('summary field', () => {
    it('treats omitted summary as null', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.summary).toBeNull();
      }
    });

    it('treats explicit null summary as null', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: null,
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.summary).toBeNull();
      }
    });

    it('normalizes an empty summary string to null', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: '',
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.summary).toBeNull();
      }
    });

    it('normalizes a whitespace-only summary to null', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: '   ',
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.summary).toBeNull();
      }
    });

    it('trims whitespace from a non-empty summary', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: '  Great book!  ',
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.summary).toBe('Great book!');
      }
    });

    it('rejects a summary longer than 2000 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: 'C'.repeat(2001),
      });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'ensure this value has at most 2000 characters',
      );
    });

    it('accepts a summary of exactly 2000 characters', () => {
      const result = bookCreateSchema.safeParse({
        title: 'Title',
        author: 'Author',
        published_year: 2000,
        summary: 'C'.repeat(2000),
      });

      expect(result.success).toBe(true);
    });
  });

  // ------------------------------------------------------------------
  // Multiple missing required fields
  // ------------------------------------------------------------------

  describe('missing required fields', () => {
    it('rejects an empty object with errors for all required fields', () => {
      const result = bookCreateSchema.safeParse({});

      expect(result.success).toBe(false);
    });
  });
});

// ---------------------------------------------------------------------------
// BookUpdate
// ---------------------------------------------------------------------------

describe('bookUpdateSchema', () => {
  // ------------------------------------------------------------------
  // Happy-path / all optional
  // ------------------------------------------------------------------

  describe('no fields provided', () => {
    it('succeeds with an empty object (all fields undefined)', () => {
      const result = bookUpdateSchema.safeParse({});

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBeUndefined();
        expect(result.data.author).toBeUndefined();
        expect(result.data.published_year).toBeUndefined();
        expect(result.data.summary).toBeUndefined();
      }
    });
  });

  // ------------------------------------------------------------------
  // title
  // ------------------------------------------------------------------

  describe('title field', () => {
    it('allows title to be absent (undefined)', () => {
      const result = bookUpdateSchema.safeParse({ author: 'Author' });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBeUndefined();
      }
    });

    it('trims whitespace from a provided title', () => {
      const result = bookUpdateSchema.safeParse({ title: '  Updated Title  ' });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.title).toBe('Updated Title');
      }
    });

    it('rejects an empty title string', () => {
      const result = bookUpdateSchema.safeParse({ title: '' });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Title cannot be empty');
    });

    it('rejects a whitespace-only title', () => {
      const result = bookUpdateSchema.safeParse({ title: '   ' });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Title cannot be empty');
    });

    it('rejects a title longer than 255 characters', () => {
      const result = bookUpdateSchema.safeParse({ title: 'A'.repeat(256) });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'ensure this value has at most 255 characters',
      );
    });

    it('accepts a title of exactly 255 characters', () => {
      const result = bookUpdateSchema.safeParse({ title: 'A'.repeat(255) });

      expect(result.success).toBe(true);
    });
  });

  // ------------------------------------------------------------------
  // author
  // ------------------------------------------------------------------

  describe('author field', () => {
    it('allows author to be absent (undefined)', () => {
      const result = bookUpdateSchema.safeParse({ title: 'Title' });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.author).toBeUndefined();
      }
    });

    it('trims whitespace from a provided author', () => {
      const result = bookUpdateSchema.safeParse({ author: '  Jane Doe  ' });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.author).toBe('Jane Doe');
      }
    });

    it('rejects an empty author string', () => {
      const result = bookUpdateSchema.safeParse({ author: '' });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Author cannot be empty');
    });

    it('rejects a whitespace-only author', () => {
      const result = bookUpdateSchema.safeParse({ author: '   ' });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain('Author cannot be empty');
    });

    it('rejects an author longer than 255 characters', () => {
      const result = bookUpdateSchema.safeParse({ author: 'B'.repeat(256) });

      expect(result.success).toBe(false);
      expect(getMessages(result)).toContain(
        'ensure this value has at most 255 characters',
      );
    });
  });

  // ------------------------------------------------------------------