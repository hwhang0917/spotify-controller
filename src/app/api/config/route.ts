import { getConfiguration, storeConfiguration } from "@/lib/config";
import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET() {
  const config = await getConfiguration();
  return NextResponse.json(config);
}

export async function POST(request: NextRequest) {
  const body = await request.json();
  await storeConfiguration(body);
  return NextResponse.json({ ok: true });
}
