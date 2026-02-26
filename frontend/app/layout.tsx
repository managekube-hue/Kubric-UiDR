import "./globals.css";
import type { Metadata } from "next";
import { AuthProvider } from "@/lib/auth-provider";

export const metadata: Metadata = {
  title: "Kubric Security Platform",
  description: "Unified security operations, compliance, and threat intelligence",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
