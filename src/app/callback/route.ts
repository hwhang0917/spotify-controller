export const dynamic = "force-dynamic";

import ms from "ms";
import qs from "qs";
import { setInterval, clearInterval } from "timers";
import axios from "axios";
import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import {
  SPOTIFY_ACCESS_TOKEN,
  SPOTIFY_ACCOUNT_URL,
  SPOTIFY_REDIRECT_ENDPOINT,
  SPOTIFY_REFRESH_TOKEN,
  SPOTIFY_STATE,
} from "@/constants";
import Logger from "@/lib/logger";
import { getMemoryCache } from "@/lib/cache";

export let refreshInterval: NodeJS.Timeout | null = null;

export async function GET(request: NextRequest) {
  const logger = new Logger("SpotifyAuthCallback");
  try {
    const memoryCache = await getMemoryCache();
    const cookieStore = cookies();

    const searchParams = request.nextUrl.searchParams;
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const redirectUri = new URL(
      SPOTIFY_REDIRECT_ENDPOINT,
      `http://localhost:${process.env.PORT}`,
    );

    if (!code || !state) {
      return NextResponse.json({ message: "invalid_request" }, { status: 400 });
    }

    // Create a base64 encoded string of the client_id and client_secret
    const basicToken = Buffer.from(
      `${process.env.SPOTIFY_CLIENT_ID}:${process.env.SPOTIFY_SECRET}`,
    ).toString("base64");

    // Request for spotify access token
    const { data } = await axios<SpotifyAccessTokenResponse>({
      method: "POST",
      baseURL: SPOTIFY_ACCOUNT_URL,
      url: "/api/token",
      headers: {
        Authorization: `Basic ${basicToken}`,
        "Content-Type": "application/x-www-form-urlencoded",
      },
      data: qs.stringify({
        code,
        redirect_uri: redirectUri.href,
        grant_type: "authorization_code",
      }),
    });

    await memoryCache.set(
      SPOTIFY_ACCESS_TOKEN,
      data.access_token,
      ms("1 hour"),
    );
    await memoryCache.set(SPOTIFY_REFRESH_TOKEN, data.refresh_token);
    logger.log("Spotify Tokens have been set in cache");
    cookieStore.set(SPOTIFY_STATE, state);
    logger.log("Spotify State has been set in cookie");

    // Set a refresh interval to refresh the access token
    refreshInterval = setInterval(async () => {
      const refreshToken = await memoryCache.get(SPOTIFY_REFRESH_TOKEN);
      if (!refreshToken && refreshInterval) {
        logger.error("Refresh token is not available");
        clearInterval(refreshInterval);
      }

      const { data: refreshData } =
        await axios<SpotifyRenewAccessTokenResponse>({
          method: "POST",
          baseURL: SPOTIFY_ACCOUNT_URL,
          url: "/api/token",
          headers: {
            Authorization: `Basic ${basicToken}`,
            "Content-Type": "application/x-www-form-urlencoded",
          },
          data: qs.stringify({
            grant_type: "refresh_token",
            refresh_token: refreshToken,
          }),
        });
      await memoryCache.set(SPOTIFY_ACCESS_TOKEN, refreshData.access_token);
      logger.log("Spotify Access Token has been refreshed in cache");
    }, ms("55 minutes"));

    return NextResponse.redirect(`http://localhost:${process.env.PORT}/`);
  } catch (error: any) {
    logger.error(error?.message);
    return NextResponse.json(
      { message: "internal_server_error" },
      { status: 500 },
    );
  }
}
