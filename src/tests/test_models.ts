// src/tests/test_models.ts
import { PrismaClient, Prisma, Book } from '@prisma/client';

/**
 * MIGRATION: The original Python test used an in-memory SQLite database created
 * per-fixture via SQLAlchemy. Here we use Prisma with a dedicated test database
 * (configured via DATABASE_URL pointing to a SQLite/Postgres test instance).
 *
 * The SQLAlchemy `Book` model defined `__repr__` and `__str__`. Prisma models are
 * plain data objects without methods, so the formatting helpers below reproduce
 * those representations exactly. They should live alongside the Book model in the
 * application code (e.g. src/models/book.ts) and be imported here once available.
 *
 * Expected schema.prisma equivalent of the SQLAlchemy Book model:
 *
 *   model Book {
 *     id            Int     @id @default(autoincrement())
 *     title         String
 *     author        String
 *     publishedYear Int     @map("published_year")
 *     summary       String?
 *
 *     @@unique([title, author])
 *     @@map("books")
 *   }
 */

/**
 * formatBookRepr reproduces the SQLAlchemy `__repr__` output exactly:
 * <Book(id=..., title='...', author='...', year=...)>
 */
export function formatBookRepr(book: Book): string {
  return `<Book(id=${book.id}, title='${book.title}', author='${book.author}', year=${book.publishedYear})>`;
}

/**
 * formatBookStr reproduces the SQLAlchemy `__str__` output exactly:
 * 'title by author (year)'
 */
export function formatBookStr(book: Book): string {
  return `${book.title} by ${book.author} (${book.publishedYear})`;
}

let prisma: PrismaClient;

/**
 * Test suite for the Book model, ported from the Python pytest TestBookModel class.
 * Each test runs against a clean database state (cleared in beforeEach), mirroring
 * the per-test SQLAlchemy in-memory session fixture.
 */
describe('Book model', () => {
  beforeAll(() => {
    prisma = new PrismaClient();
  });

  afterAll(async () => {
    await prisma.$disconnect();
  });

  // Equivalent of the per-test db_session fixture: ensure a clean table.
  beforeEach(async () => {
    await prisma.book.deleteMany();
  });

  it('creates a basic book', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Test Book',
        author: 'Test Author',
        publishedYear: 2023,
        summary: 'A test book summary',
      },
    });

    expect(book.id).toBeDefined();
    expect(book.id).not.toBeNull();
    expect(book.title).toBe('Test Book');
    expect(book.author).toBe('Test Author');
    expect(book.publishedYear).toBe(2023);
    expect(book.summary).toBe('A test book summary');
  });

  it('creates a book without summary (optional field)', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Test Book No Summary',
        author: 'Test Author',
        publishedYear: 2023,
      },
    });

    expect(book.id).toBeDefined();
    expect(book.id).not.toBeNull();
    expect(book.title).toBe('Test Book No Summary');
    expect(book.author).toBe('Test Author');
    expect(book.publishedYear).toBe(2023);
    expect(book.summary).toBeNull();
  });

  it('produces the expected repr string', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Repr Test',
        author: 'Repr Author',
        publishedYear: 2023,
        summary: 'Test summary',
      },
    });

    const expectedRepr = `<Book(id=${book.id}, title='Repr Test', author='Repr Author', year=2023)>`;
    expect(formatBookRepr(book)).toBe(expectedRepr);
  });

  it('produces the expected str string', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'String Test',
        author: 'String Author',
        publishedYear: 2023,
      },
    });

    const expectedStr = 'String Test by String Author (2023)';
    expect(formatBookStr(book)).toBe(expectedStr);
  });

  it('rejects duplicate title-author combinations', async () => {
    await prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2023,
      },
    });

    // Attempt to create a second book with the same title and author but a
    // different year. The composite unique constraint on (title, author) must
    // cause a unique violation (Prisma error code P2002).
    await expect(
      prisma.book.create({
        data: {
          title: 'Duplicate Test',
          author: 'Duplicate Author',
          publishedYear: 2024,
        },
      }),
    ).rejects.toMatchObject({
      code: 'P2002',
    } as Partial<Prisma.PrismaClientKnownRequestError>);
  });

  it('allows books with the same title but different authors', async () => {
    const book1 = await prisma.book.create({
      data: {
        title: 'Common Title',
        author: 'Author One',
        publishedYear: 2023,
      },
    });

    const book2 = await prisma.book.create({
      data: {
        title: 'Common Title',
        author: 'Author Two',
        publishedYear: 2023,
      },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });

  it('allows books with the same author but different titles', async () => {
    const book1 = await prisma.book.create({
      data: {
        title: 'First Book',
        author: 'Prolific Author',
        publishedYear: 2023,
      },
    });

    const book2 = await prisma.book.create({
      data: {
        title: 'Second Book',
        author: 'Prolific Author',
        publishedYear: 2024,
      },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });
});
