import DOMPurify from 'dompurify';

/**
 * Sanitizes user-generated content to prevent XSS attacks
 * @param dirty - The potentially unsafe HTML/text content
 * @returns Sanitized string safe for display
 */
export function sanitizeContent(dirty: string | undefined | null): string {
  if (!dirty) return '';
  
  // Configure DOMPurify to strip dangerous elements but allow basic formatting
  const config = {
    ALLOWED_TAGS: ['b', 'i', 'em', 'strong', 'span'],
    ALLOWED_ATTR: [],
    KEEP_CONTENT: true,
  };
  
  return DOMPurify.sanitize(dirty, config);
}

/**
 * Sanitizes content and returns it as plain text (no HTML)
 * @param dirty - The potentially unsafe content
 * @returns Plain text string safe for display
 */
export function sanitizePlainText(dirty: string | undefined | null): string {
  if (!dirty) return '';
  
  // Strip all HTML tags and return plain text
  return DOMPurify.sanitize(dirty, { ALLOWED_TAGS: [], KEEP_CONTENT: true });
}