import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getMemoryCache } from "@/lib/cache";
import { SPOTIFY_ACCESS_TOKEN, SPOTIFY_STATE } from "@/constants";
import { getConfiguration } from "@/lib/config";
import Player from "@/components/player";

export default async function Home() {
  const memoryCache = await getMemoryCache();
  const cookieStore = cookies();
  // Get configuration
  const config = await getConfiguration();

  const accessTokenFromCache =
    await memoryCache.get<string>(SPOTIFY_ACCESS_TOKEN);
  const isAppInitialized = !!accessTokenFromCache;
  const adminSpotifyState = cookieStore.get(SPOTIFY_STATE)?.value;

  if (!isAppInitialized) {
    redirect("/admin");
  }

  return <Player />;
}
