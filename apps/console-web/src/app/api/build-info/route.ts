import { NextResponse } from "next/server";

export const dynamic = "force-static";

export function GET() {
  return NextResponse.json({
    revision: process.env.NEXT_PUBLIC_ENX_BUILD_REVISION ?? "dev",
  });
}
