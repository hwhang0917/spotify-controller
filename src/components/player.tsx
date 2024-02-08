"use client";

import {
  ALBUM_COVER_PLACEHOLDER,
  SPOTIFY_ACCESS_TOKEN,
  SPOTIFY_API_URL,
} from "@/constants";
import axios from "axios";
import { useMutation, useQuery } from "react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import Image from "next/image";
import { Skeleton } from "./ui/skeleton";
import { getListFormatter } from "@/lib/string";
import { Pause, Play, SkipForward } from "lucide-react";
import { Button } from "./ui/button";
import { cn } from "@/lib/utils";

export default function Player() {
  // Get configuration and access token
  const { data: configCache } = useQuery(
    ["config", "accessToken"],
    async () => {
      const configResponse =
        await axios.get<SpotifyControllerConfig>("/api/config");
      const accessTokenResponse = await axios.get<{ value?: unknown }>(
        "/api/cache",
        { params: { key: SPOTIFY_ACCESS_TOKEN } },
      );
      const accessToken = accessTokenResponse?.data?.value as string;

      return { config: configResponse.data, accessToken };
    },
  );

  // Access Token
  const accessToken = configCache?.accessToken;
  // Language settings
  const language = configCache?.config?.language;

  // Get currently playing track
  const { data: currentlyPlayingTrack, refetch } = useQuery(
    ["currentlyPlayingTrack", accessToken, language],
    async () => {
      const { data } = await axios.get<SpotifyCurrentlyPlayingTrack>(
        "/v1/me/player/currently-playing",
        {
          baseURL: SPOTIFY_API_URL,
          params: { market: language === "korean" ? "KR" : "US" },
          headers: { Authorization: `Bearer ${accessToken}` },
        },
      );
      return data;
    },
    {
      enabled: !!configCache,
      retry: 3,
    },
  );

  const { mutate: playTrack } = useMutation({
    mutationFn: async () => {
      await axios.put("/v1/me/player/play", null, {
        baseURL: SPOTIFY_API_URL,
        headers: { Authorization: `Bearer ${accessToken}` },
      });
      refetch();
    },
  });
  const { mutate: pauseTrack } = useMutation({
    mutationFn: async () => {
      await axios.put("/v1/me/player/pause", null, {
        baseURL: SPOTIFY_API_URL,
        headers: { Authorization: `Bearer ${accessToken}` },
      });
      refetch();
    },
  });
  const { mutate: skipTrack } = useMutation({
    mutationFn: async () => {
      await axios.post("/v1/me/player/next", null, {
        baseURL: SPOTIFY_API_URL,
        headers: { Authorization: `Bearer ${accessToken}` },
      });
      refetch();
    },
  });

  const albumCover = currentlyPlayingTrack?.item?.album?.images?.[0]?.url;
  const title = currentlyPlayingTrack?.item?.name;

  return (
    <main className="container grid min-h-screen place-items-center">
      <Card className="min-w-[300px]">
        <CardHeader>
          <CardTitle>
            {language === "korean" ? "현재 재생 중" : "Currently Playing Track"}
          </CardTitle>
          <CardDescription></CardDescription>
        </CardHeader>
        <CardContent>
          <div className="w-full">
            {albumCover ? (
              <Image
                src={albumCover}
                alt={currentlyPlayingTrack?.item?.name || ""}
                width={300}
                height={300}
                className="mx-auto rounded-md"
              />
            ) : (
              <Skeleton className="h-[300px] w-[300px] rounded-md" />
            )}
          </div>
          <div className="mt-4">
            <h2 className="text-lg font-bold">{title || "No track playing"}</h2>
            <p className="text-sm">
              {currentlyPlayingTrack?.item?.artists
                ?.map((artist) => artist.name)
                ?.join(", ") || "No artist"}
            </p>
          </div>
          <div className="mt-8 space-y-2">
            {currentlyPlayingTrack?.is_playing ? (
              <Button
                variant="outline"
                className={cn(
                  "w-full",
                  configCache?.config.allowPausing
                    ? "cursor-pointer"
                    : "cursor-not-allowed",
                )}
                onClick={() => pauseTrack()}
                disabled={!configCache?.config.allowPausing}
              >
                <Pause />
              </Button>
            ) : (
              <Button
                variant="outline"
                className={cn(
                  "w-full",
                  configCache?.config.allowPausing
                    ? "cursor-pointer"
                    : "cursor-not-allowed",
                )}
                onClick={() => playTrack()}
                disabled={!configCache?.config.allowPausing}
              >
                <Play />
              </Button>
            )}
            <Button
              variant="outline"
              className={cn(
                "w-full",
                configCache?.config.allowSkipping
                  ? "cursor-pointer"
                  : "cursor-not-allowed",
              )}
              onClick={() => skipTrack()}
              disabled={!configCache?.config.allowSkipping}
            >
              <SkipForward />
            </Button>
          </div>
        </CardContent>
      </Card>
    </main>
  );
}
