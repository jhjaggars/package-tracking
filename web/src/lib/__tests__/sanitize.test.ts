import { describe, it, expect } from 'vitest';
import { sanitizeContent, sanitizePlainText } from '../sanitize';

describe('sanitizeContent', () => {
  it('returns empty string for null/undefined input', () => {
    expect(sanitizeContent(null)).toBe('');
    expect(sanitizeContent(undefined)).toBe('');
    expect(sanitizeContent('')).toBe('');
  });

  it('allows basic formatting tags', () => {
    const input = '<b>Bold</b> and <i>italic</i> text';
    const result = sanitizeContent(input);
    expect(result).toBe('<b>Bold</b> and <i>italic</i> text');
  });

  it('removes dangerous script tags', () => {
    const input = '<script>alert("xss")</script>Safe content';
    const result = sanitizeContent(input);
    expect(result).toBe('Safe content');
  });

  it('removes dangerous event handlers', () => {
    const input = '<span onclick="alert(\\"xss\\")">Click me</span>';
    const result = sanitizeContent(input);
    expect(result).toBe('<span>Click me</span>');
  });

  it('removes iframe and object tags', () => {
    const input = '<iframe src="evil.com"></iframe>Normal text';
    const result = sanitizeContent(input);
    expect(result).toBe('Normal text');
  });

  it('preserves content while removing dangerous tags', () => {
    const input = '<div><script>evil()</script>Good content<style>body{}</style></div>';
    const result = sanitizeContent(input);
    expect(result).toBe('Good content');
  });
});

describe('sanitizePlainText', () => {
  it('returns empty string for null/undefined input', () => {
    expect(sanitizePlainText(null)).toBe('');
    expect(sanitizePlainText(undefined)).toBe('');
    expect(sanitizePlainText('')).toBe('');
  });

  it('strips all HTML tags', () => {
    const input = '<b>Bold</b> and <i>italic</i> text';
    const result = sanitizePlainText(input);
    expect(result).toBe('Bold and italic text');
  });

  it('removes dangerous content and returns plain text', () => {
    const input = '<script>alert("xss")</script>Safe content';
    const result = sanitizePlainText(input);
    expect(result).toBe('Safe content');
  });

  it('handles complex HTML structure', () => {
    const input = '<div class="container"><h1>Title</h1><p>Paragraph with <a href="link">link</a></p></div>';
    const result = sanitizePlainText(input);
    expect(result).toBe('TitleParagraph with link');
  });

  it('handles HTML entities', () => {
    const input = 'Price: $29.99 &amp; shipping';
    const result = sanitizePlainText(input);
    // DOMPurify preserves entities in text-only mode
    expect(result).toBe('Price: $29.99 &amp; shipping');
  });

  it('handles malformed HTML gracefully', () => {
    const input = '<div>Unclosed tag<span>nested content</div>';
    const result = sanitizePlainText(input);
    expect(result).toBe('Unclosed tagnested content');
  });
});