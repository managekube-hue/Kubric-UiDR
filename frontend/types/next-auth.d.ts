import "next-auth";
import "next-auth/jwt";

declare module "next-auth" {
  interface Session {
    accessToken: string;
    tenantId: string;
    groups: string[];
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    tenantId?: string;
    groups?: string[];
  }
}
