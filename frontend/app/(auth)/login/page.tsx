"use client";

import { signIn } from "next-auth/react";
import { Shield } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-kubric-950">
      <div className="w-full max-w-md space-y-8 rounded-xl bg-white p-10 shadow-2xl">
        <div className="flex flex-col items-center gap-4">
          <Shield className="h-16 w-16 text-kubric-600" />
          <h1 className="text-2xl font-bold text-kubric-950">
            Kubric Security Platform
          </h1>
          <p className="text-sm text-gray-500">
            Sign in with your organization credentials
          </p>
        </div>
        <Button
          onClick={() => signIn("authentik", { callbackUrl: "/dashboard" })}
          className="w-full"
          size="lg"
        >
          Sign in with SSO
        </Button>
        <p className="text-center text-xs text-gray-400">
          Protected by Authentik OIDC. Contact your admin for access.
        </p>
      </div>
    </div>
  );
}
