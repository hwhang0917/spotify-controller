export const dynamic = "force-dynamic";

import { getMemoryCache } from "@/lib/cache";
import { NextRequest, NextResponse } from "next/server";

export async function GET(request: NextRequest) {
  const memoryCache = await getMemoryCache();
  const key = request.nextUrl.searchParams.get("key");
  const value = await memoryCache.get(key || "");
  return NextResponse.json({ value });
}
