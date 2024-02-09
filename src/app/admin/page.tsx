export const dynamic = "force-dynamic";

import Link from "next/link";
import Image from "next/image";
import { cookies } from "next/headers";
import { RocketIcon } from "@radix-ui/react-icons";
import { getMemoryCache } from "@/lib/cache";
import { generateRandomHash } from "@/lib/random";
import { getConfiguration } from "@/lib/config";
import { translation } from "@/translation";
import {
  SPOTIFY_ACCESS_TOKEN,
  SPOTIFY_ACCOUNT_URL,
  SPOTIFY_REDIRECT_ENDPOINT,
  SPOTIFY_SCOPE,
  SPOTIFY_STATE,
} from "@/constants";
import Settings from "./settings";
import Unauthorized from "./unauthorized";

export default async function Home() {
  const config = await getConfiguration();
  const t = translation[config.language].adminLogin;

  const memoryCache = await getMemoryCache();
  const accessToken = await memoryCache.get<string>(SPOTIFY_ACCESS_TOKEN);

  if (!accessToken) {
    const state = generateRandomHash();

    // Store state to cache
    await memoryCache.set("state", state);
    const redirectUri = new URL(
      SPOTIFY_REDIRECT_ENDPOINT,
      `http://localhost:${process.env.PORT}`,
    );

    const spotifyLoginUrl = new URL("/authorize", SPOTIFY_ACCOUNT_URL);

    spotifyLoginUrl.searchParams.append("response_type", "code");
    spotifyLoginUrl.searchParams.append(
      "client_id",
      process.env.SPOTIFY_CLIENT_ID || "",
    );
    spotifyLoginUrl.searchParams.append("scope", SPOTIFY_SCOPE);
    spotifyLoginUrl.searchParams.append("redirect_uri", redirectUri.href);
    spotifyLoginUrl.searchParams.append("state", state);

    return (
      <main className="container grid min-h-screen place-items-center">
        <section>
          <article className="prose my-8 dark:prose-invert">
            <div className="flex justify-center">
              <Image
                src="/icon.png"
                alt="Spotify Controller"
                width={100}
                height={100}
              />
            </div>
            <h1>{t.title}</h1>
            <h2>{t.intro}</h2>
            <h3>{t.requirementsTitle}</h3>
            <ul>
              {t.requirements.map((requirement, idx) => (
                <li key={idx}>{requirement}</li>
              ))}
            </ul>
          </article>
          <Link
            href={spotifyLoginUrl.href}
            className="flex items-center gap-2 rounded bg-spotify p-4 text-black"
          >
            <RocketIcon className="h-4 w-4" />
            <span className="uppercase">{t.loginButton}</span>
          </Link>
        </section>
      </main>
    );
  } else {
    const cookieStore = cookies();
    const adminState = cookieStore.get(SPOTIFY_STATE);
    const cacheState = await memoryCache.get<string>("state");

    if (!adminState || adminState.value !== cacheState) {
      return <Unauthorized />;
    } else {
      return <Settings initialConfig={config} />;
    }
  }
}
