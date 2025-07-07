import { useState } from 'react';
import { ChevronDown, ChevronUp, Mail, ExternalLink } from 'lucide-react';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { DateFormatter } from '../shared';
import { sanitizePlainText } from '../../lib/sanitize';
import type { EmailEntry } from '../../types/api';

interface EmailCardProps {
  email: EmailEntry;
}

export function EmailCard({ email }: EmailCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [showFullContent, setShowFullContent] = useState(false);

  // Parse tracking numbers from JSON string
  const trackingNumbers = email.tracking_numbers ? 
    JSON.parse(email.tracking_numbers).join(', ') : 'None found';

  // Truncate content for preview
  const contentPreview = email.body_text.slice(0, 200);
  const hasLongContent = email.body_text.length > 200;

  return (
    <Card className="mb-4">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-start space-x-3">
            <div className="flex-shrink-0">
              <Mail className="h-5 w-5 text-muted-foreground mt-0.5" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-2">
                <h4 className="text-sm font-medium text-foreground">
                  {sanitizePlainText(email.subject)}
                </h4>
                <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                  email.status === 'processed' 
                    ? 'bg-green-100 text-green-800' 
                    : 'bg-yellow-100 text-yellow-800'
                }`}>
                  {email.status}
                </span>
              </div>
              <p className="text-sm text-muted-foreground">
                From: {sanitizePlainText(email.from)}
              </p>
              <p className="text-xs text-muted-foreground">
                <DateFormatter date={email.date} />
              </p>
              {trackingNumbers !== 'None found' && (
                <p className="text-xs text-muted-foreground mt-1">
                  Tracking Numbers: <span className="font-mono">{trackingNumbers}</span>
                </p>
              )}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? (
              <ChevronUp className="h-4 w-4" />
            ) : (
              <ChevronDown className="h-4 w-4" />
            )}
          </Button>
        </div>
      </CardHeader>
      
      {isExpanded && (
        <CardContent className="pt-0">
          <div className="space-y-3">
            {/* Email Details */}
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="font-medium text-muted-foreground">Scan Method:</span>
                <span className="ml-2 capitalize">{email.scan_method}</span>
              </div>
              <div>
                <span className="font-medium text-muted-foreground">Processed:</span>
                <span className="ml-2">
                  <DateFormatter date={email.processed_at} />
                </span>
              </div>
            </div>

            {/* Email Content */}
            <div className="border-t pt-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-foreground">Email Content</span>
                {hasLongContent && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowFullContent(!showFullContent)}
                  >
                    {showFullContent ? 'Show Less' : 'Show More'}
                  </Button>
                )}
              </div>
              <div className="bg-muted/50 rounded-md p-3 text-sm">
                <pre className="whitespace-pre-wrap font-sans">
                  {sanitizePlainText(
                    showFullContent || !hasLongContent 
                      ? email.body_text 
                      : contentPreview + '...'
                  )}
                </pre>
              </div>
            </div>

            {/* Gmail Link */}
            <div className="flex justify-end">
              <Button
                variant="outline"
                size="sm"
                asChild
              >
                <a
                  href={`https://mail.google.com/mail/u/0/#inbox/${email.gmail_message_id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center"
                >
                  <ExternalLink className="mr-2 h-3 w-3" />
                  View in Gmail
                </a>
              </Button>
            </div>

            {/* Error Message */}
            {email.error_message && (
              <div className="bg-red-50 border border-red-200 rounded-md p-3">
                <p className="text-sm text-red-800">
                  <span className="font-medium">Error:</span> {email.error_message}
                </p>
              </div>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  );
}