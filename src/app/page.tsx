import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getMemoryCache } from "@/lib/cache";
import { SPOTIFY_ACCESS_TOKEN } from "@/constants";

export default async function Home() {
  const memoryCache = await getMemoryCache();
  const cookieStore = cookies();

  const accessTokenFromCache =
    await memoryCache.get<string>(SPOTIFY_ACCESS_TOKEN);
  const accessTokenFromCookie = cookieStore.get(SPOTIFY_ACCESS_TOKEN)?.name;

  if (!accessTokenFromCache) {
    redirect("/admin");
  }

  return <h1>home</h1>;
}
