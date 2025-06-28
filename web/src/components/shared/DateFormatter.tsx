interface DateFormatterProps {
  date: string;
  format?: 'date' | 'datetime';
  className?: string;
}

export function DateFormatter({ date, format = 'datetime', className }: DateFormatterProps) {
  const dateObj = new Date(date);
  
  const formatted = format === 'date' 
    ? dateObj.toLocaleDateString()
    : dateObj.toLocaleString();
  
  return (
    <time dateTime={date} className={className}>
      {formatted}
    </time>
  );
}

// Utility functions for use outside of components
export function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString();
}

export function formatDateOnly(dateString: string): string {
  return new Date(dateString).toLocaleDateString();
}