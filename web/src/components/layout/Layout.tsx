import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Package, BarChart3, Plus, List } from 'lucide-react';
import { cn } from '../../lib/utils';

interface LayoutProps {
  children: ReactNode;
}

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: BarChart3 },
  { name: 'Shipments', href: '/shipments', icon: List },
  { name: 'Add Shipment', href: '/shipments/new', icon: Plus },
];

export function Layout({ children }: LayoutProps) {
  const location = useLocation();

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="bg-white shadow-sm border-b">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              {/* Logo */}
              <div className="flex-shrink-0 flex items-center">
                <Package className="h-8 w-8 text-blue-600" />
                <span className="ml-2 text-xl font-bold text-gray-900">
                  Package Tracker
                </span>
              </div>
              
              {/* Navigation links */}
              <div className="hidden sm:ml-6 sm:flex sm:space-x-8">
                {navigation.map((item) => {
                  const isActive = location.pathname === item.href;
                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      className={cn(
                        'inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium',
                        isActive
                          ? 'border-blue-600 text-blue-600'
                          : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                      )}
                    >
                      <item.icon className="mr-2 h-4 w-4" />
                      {item.name}
                    </Link>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile navigation menu */}
      <div className="sm:hidden">
        <div className="pt-2 pb-3 space-y-1 bg-white border-b">
          {navigation.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.name}
                to={item.href}
                className={cn(
                  'block pl-3 pr-4 py-2 border-l-4 text-base font-medium',
                  isActive
                    ? 'border-blue-600 text-blue-600 bg-blue-50'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-50'
                )}
              >
                <div className="flex items-center">
                  <item.icon className="mr-3 h-5 w-5" />
                  {item.name}
                </div>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Main content */}
      <main className="flex-1">
        <div className="py-6">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            {children}
          </div>
        </div>
      </main>
    </div>
  );
}