"use client";

import React from "react";
import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
import { useQuery } from "react-query";
import axios from "axios";
import Image from "next/image";
import { SPOTIFY_API_URL } from "@/constants";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";
import { Card, CardHeader, CardTitle } from "./ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "./ui/collapsible";
import { ChevronUpIcon, ChevronDownIcon } from "@radix-ui/react-icons";
import { Button } from "./ui/button";
dayjs.extend(duration);

interface IPlaylistProps {
  accessToken: string;
  language: Language;
}

export function Playlist(props: IPlaylistProps) {
  const [open, setOpen] = React.useState(false);

  const { data: playerQueue } = useQuery(
    ["get-playlist", props.accessToken],
    async () => {
      const { data } = await axios.get<SpotifyQueueResponse>(
        "/v1/me/player/queue",
        {
          baseURL: SPOTIFY_API_URL,
          headers: {
            Authorization: `Bearer ${props.accessToken}`,
          },
        },
      );
      return data;
    },
  );

  const getDuration = (ms: number) => {
    return dayjs(ms).format("mm:ss");
  };

  return (
    <Card>
      <Collapsible open={open} onOpenChange={setOpen}>
        <CardHeader>
          <CardTitle className="flex w-full items-center justify-between">
            {props.language === "korean" ? "재생목록" : "Playlist"}
            <CollapsibleTrigger asChild>
              <Button variant="outline">
                {open ? <ChevronUpIcon /> : <ChevronDownIcon />}
              </Button>
            </CollapsibleTrigger>
          </CardTitle>
        </CardHeader>
        <CollapsibleContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  {props.language === "korean" ? "순서" : "Order"}
                </TableHead>
                <TableHead>
                  {props.language === "korean" ? "앨범" : "Album"}
                </TableHead>
                <TableHead>
                  {props.language === "korean" ? "제목" : "Title"}
                </TableHead>
                <TableHead>
                  {props.language === "korean" ? "아티스트" : "Artist"}
                </TableHead>
                <TableHead>
                  {props.language === "korean" ? "재생시간" : "Duration"}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {playerQueue?.queue.map((item, index) => (
                <TableRow key={item.id}>
                  <TableCell>{index + 1}</TableCell>
                  <TableHead>
                    <Image
                      src={item.album.images[0].url}
                      alt={item.album.name}
                      width={50}
                      height={50}
                    />
                  </TableHead>
                  <TableCell>{item.name}</TableCell>
                  <TableCell>{item.artists[0].name}</TableCell>
                  <TableCell>{getDuration(item.duration_ms)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
}
