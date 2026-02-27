import type { NextAuthOptions } from "next-auth";

export const authOptions: NextAuthOptions = {
  providers: [
    {
      id: "authentik",
      name: "Kubric SSO",
      type: "oauth",
      wellKnown:
        process.env.AUTHENTIK_ISSUER ??
        "https://auth.kubric.security/application/o/kubric/.well-known/openid-configuration",
      clientId: process.env.AUTHENTIK_CLIENT_ID ?? "",
      clientSecret: process.env.AUTHENTIK_CLIENT_SECRET ?? "",
      authorization: { params: { scope: "openid email profile" } },
      idToken: true,
      profile(profile) {
        return {
          id: profile.sub,
          name: profile.name ?? profile.preferred_username,
          email: profile.email,
          image: profile.picture,
        };
      },
    },
  ],
  callbacks: {
    async jwt({ token, account, profile }) {
      if (account && profile) {
        token.accessToken = account.access_token;
        token.tenantId = (profile as Record<string, unknown>).tenant_id as string;
        token.groups = (profile as Record<string, unknown>).groups as string[];
      }
      return token;
    },
    async session({ session, token }) {
      return {
        ...session,
        accessToken: token.accessToken as string,
        tenantId: token.tenantId as string,
        groups: (token.groups as string[]) ?? [],
      };
    },
  },
  pages: {
    signIn: "/login",
  },
  session: {
    strategy: "jwt",
    maxAge: 3600,
  },
};
