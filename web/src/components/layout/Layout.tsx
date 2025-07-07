import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Package, BarChart3, Plus, List, Menu } from 'lucide-react';
import { ThemeToggle } from '../ui/ThemeToggle';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet';

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
    <div className="min-h-screen bg-background">
      {/* Navigation */}
      <nav className="sticky top-0 z-50 bg-background/80 backdrop-blur-sm border-b">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              {/* Logo */}
              <div className="flex-shrink-0 flex items-center">
                <Package className="h-8 w-8 text-primary" />
                <span className="ml-3 text-xl font-bold text-foreground">
                  Package Tracker
                </span>
              </div>
              
              {/* Desktop Navigation links */}
              <div className="hidden sm:ml-6 sm:flex sm:space-x-1">
                {navigation.map((item) => {
                  const isActive = location.pathname === item.href;
                  return (
                    <Button
                      key={item.name}
                      asChild
                      variant={isActive ? "default" : "ghost"}
                      className="inline-flex items-center"
                    >
                      <Link to={item.href}>
                        <item.icon className="mr-2 h-4 w-4" />
                        {item.name}
                      </Link>
                    </Button>
                  );
                })}
              </div>
            </div>
            
            {/* Desktop Theme Toggle */}
            <div className="hidden sm:flex items-center">
              <ThemeToggle />
            </div>

            {/* Mobile Menu Button */}
            <div className="sm:hidden flex items-center">
              <Sheet>
                <SheetTrigger asChild>
                  <Button variant="ghost" size="icon">
                    <Menu className="h-5 w-5" />
                  </Button>
                </SheetTrigger>
                <SheetContent side="right" className="w-[300px]">
                  <div className="flex flex-col space-y-4 mt-4">
                    <div className="flex items-center space-x-2">
                      <Package className="h-6 w-6 text-primary" />
                      <span className="text-lg font-semibold">Package Tracker</span>
                    </div>
                    <div className="flex flex-col space-y-2">
                      {navigation.map((item) => {
                        const isActive = location.pathname === item.href;
                        return (
                          <Button
                            key={item.name}
                            asChild
                            variant={isActive ? "default" : "ghost"}
                            className="justify-start"
                          >
                            <Link to={item.href}>
                              <item.icon className="mr-2 h-4 w-4" />
                              {item.name}
                            </Link>
                          </Button>
                        );
                      })}
                    </div>
                    <div className="pt-4 border-t">
                      <ThemeToggle />
                    </div>
                  </div>
                </SheetContent>
              </Sheet>
            </div>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="flex-1">
        <div className="py-8">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            {children}
          </div>
        </div>
      </main>
    </div>
  );
}