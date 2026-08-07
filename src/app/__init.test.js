```typescript
import {
  PACKAGE_METADATA,
  VERSION,
  AUTHOR,
  EMAIL,
} from './  __init';

describe('src/app/__init module', () => {
  describe('Global invariants', () => {
    it('should be importable without raising an error', () => {
      expect(() => {
        require('./__init');
      }).not.toThrow();
    });

    it('should export PACKAGE_METADATA, VERSION, AUTHOR, and EMAIL', () => {
      expect(PACKAGE_METADATA).toBeDefined();
      expect(VERSION).toBeDefined();
      expect(AUTHOR).toBeDefined();
      expect(EMAIL).toBeDefined();
    });

    it('should produce no observable side effects on import', () => {
      // Verifying the module can be required multiple times without side effects
      const mod1 = require('./__init');
      const mod2 = require('./__init');
      expect(mod1).toBe(mod2);
    });
  });

  describe('VERSION (__version__)', () => {
    it('should be defined as a string', () => {
      expect(typeof VERSION).toBe('string');
    });

    it("should equal '1.0.0'", () => {
      expect(VERSION).toBe('1.0.0');
    });

    it('should match PACKAGE_METADATA.version', () => {
      expect(VERSION).toBe(PACKAGE_METADATA.version);
    });
  });

  describe('AUTHOR (__author__)', () => {
    it('should be defined as a string', () => {
      expect(typeof AUTHOR).toBe('string');
    });

    it("should equal 'Abdullah Arshad'", () => {
      expect(AUTHOR).toBe('Abdullah Arshad');
    });

    it('should match PACKAGE_METADATA.author', () => {
      expect(AUTHOR).toBe(PACKAGE_METADATA.author);
    });
  });

  describe('EMAIL (__email__)', () => {
    it('should be defined as a string', () => {
      expect(typeof EMAIL).toBe('string');
    });

    it("should equal 'abdullah.arshad.314@gmail.com'", () => {
      expect(EMAIL).toBe('abdullah.arshad.314@gmail.com');
    });

    it('should match PACKAGE_METADATA.email', () => {
      expect(EMAIL).toBe(PACKAGE_METADATA.email);
    });
  });

  describe('PACKAGE_METADATA', () => {
    it('should contain a version property equal to 1.0.0', () => {
      expect(PACKAGE_METADATA.version).toBe('1.0.0');
    });

    it('should contain an author property equal to Abdullah Arshad', () => {
      expect(PACKAGE_METADATA.author).toBe('Abdullah Arshad');
    });

    it('should contain an email property equal to abdullah.arshad.314@gmail.com', () => {
      expect(PACKAGE_METADATA.email).toBe('abdullah.arshad.314@gmail.com');
    });

    it('should be a readonly/const object', () => {
      // Verifying the shape and immutability intent
      expect(Object.isFrozen(PACKAGE_METADATA) || typeof PACKAGE_METADATA === 'object').toBe(true);
    });

    it('should have exactly the expected keys', () => {
      const keys = Object.keys(PACKAGE_METADATA).sort();
      expect(keys).toEqual(['author', 'email', 'version']);
    });
  });
});
```