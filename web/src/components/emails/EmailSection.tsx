import { Mail, AlertCircle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useShipmentEmails } from '../../hooks/api';
import { EmailCard } from './EmailCard';

interface EmailSectionProps {
  shipmentId: number;
}

export function EmailSection({ shipmentId }: EmailSectionProps) {
  const { data: emails, isLoading, error } = useShipmentEmails(shipmentId);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Mail className="mr-2 h-5 w-5" />
            Related Emails
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-4">
            <div className="text-muted-foreground">Loading emails...</div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    console.log('Email loading error:', error);
    // If it's a 404, treat it as no emails found rather than an error
    const is404 = error.message?.includes('404') || error.status === 404;
    
    if (is404) {
      return (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Mail className="mr-2 h-5 w-5" />
              Related Emails
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-center py-6">
              <Mail className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-2 text-sm font-medium text-foreground">No related emails</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                This shipment was not created from email processing or no emails have been linked to it.
              </p>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Mail className="mr-2 h-5 w-5" />
            Related Emails
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-4 text-muted-foreground">
            <AlertCircle className="mr-2 h-4 w-4" />
            Failed to load emails
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!emails || emails.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <Mail className="mr-2 h-5 w-5" />
            Related Emails
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-6">
            <Mail className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-2 text-sm font-medium text-foreground">No related emails</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              This shipment was not created from email processing or no emails have been linked to it.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center">
            <Mail className="mr-2 h-5 w-5" />
            Related Emails
          </div>
          <span className="text-sm font-normal text-muted-foreground">
            {emails.length} email{emails.length !== 1 ? 's' : ''}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-0">
        {emails.map((email) => (
          <EmailCard key={email.id} email={email} />
        ))}
      </CardContent>
    </Card>
  );
}