"use client";

import { useSession, signOut } from "next-auth/react";
import { redirect } from "next/navigation";
import Link from "next/link";
import {
  LayoutDashboard,
  Bug,
  ShieldCheck,
  Gauge,
  Cpu,
  CreditCard,
  LogOut,
  Shield,
  ShieldAlert,
} from "lucide-react";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/detection", label: "Detection", icon: ShieldAlert },
  { href: "/vulns", label: "Vulnerabilities", icon: Bug },
  { href: "/compliance", label: "Compliance", icon: ShieldCheck },
  { href: "/kiss", label: "KiSS Score", icon: Gauge },
  { href: "/agents", label: "Agents", icon: Cpu },
  { href: "/billing", label: "Billing", icon: CreditCard },
];

export default function TenantLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { data: session, status } = useSession();

  if (status === "loading") {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="animate-spin h-8 w-8 border-4 border-kubric-400 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (!session) {
    redirect("/login");
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="w-64 bg-kubric-950 text-white flex flex-col">
        <div className="p-6 flex items-center gap-3">
          <Shield className="h-8 w-8 text-kubric-400" />
          <span className="text-lg font-bold">Kubric</span>
        </div>
        <nav className="flex-1 px-3 space-y-1">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="flex items-center gap-3 px-3 py-2 rounded-md text-sm text-gray-300 hover:bg-kubric-800 hover:text-white transition-colors"
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="p-4 border-t border-kubric-800">
          <div className="text-xs text-gray-400 mb-2 truncate">
            {session.user?.email}
          </div>
          <button
            onClick={() => signOut({ callbackUrl: "/login" })}
            className="flex items-center gap-2 text-sm text-gray-400 hover:text-white transition-colors"
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-7xl mx-auto p-6">{children}</div>
      </main>
    </div>
  );
}
