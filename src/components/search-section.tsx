"use client";
import React, { useEffect } from "react";
import axios from "axios";
import { useMutation, useQuery } from "react-query";
import { SPOTIFY_API_URL } from "@/constants";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { Label } from "./ui/label";
import { Input } from "./ui/input";
import { Separator } from "./ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";
import Image from "next/image";
import { getDuration } from "@/lib/time";
import { Pagination, PaginationContent, PaginationItem } from "./ui/pagination";
import { Button } from "./ui/button";
import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { Skeleton } from "./ui/skeleton";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
  AlertDialogAction,
  AlertDialogCancel,
} from "./ui/alert-dialog";
import { useToast } from "./ui/use-toast";

interface SearchSectionProps {
  accessToken: string;
  language: Language;
}

interface Pagination {
  limit: number;
  offset: number;
}

export function SearchSection(props: SearchSectionProps) {
  const [searchQuery, setSearchQuery] = React.useState("");
  const deferredSearchQuery = React.useDeferredValue(searchQuery);
  const [pagination, setPagination] = React.useState<Pagination>({
    limit: 10,
    offset: 0,
  });
  const [songId, setSongId] = React.useState("");
  const [title, setTitle] = React.useState("");
  const { toast } = useToast();

  // Reset pagination when search query changes
  useEffect(() => {
    setPagination({ limit: 10, offset: 0 });
  }, [deferredSearchQuery]);

  const { data: searchResults, isLoading } = useQuery(
    [
      "searchResults",
      deferredSearchQuery,
      props.accessToken,
      props.language,
      pagination,
    ],
    async () => {
      if (searchQuery) {
        const { data } = await axios.get<SpotifySongSearchResponse>(
          "/v1/search",
          {
            baseURL: SPOTIFY_API_URL,
            params: {
              q: searchQuery,
              type: "track",
              market: props.language === "korean" ? "KR" : "US",
              limit: pagination.limit,
              offset: pagination.offset,
            },
            headers: { Authorization: `Bearer ${props.accessToken}` },
          },
        );
        return data;
      }
    },
    {
      enabled: !!deferredSearchQuery,
    },
  );

  const updatePagination = (type: "next" | "previous") => {
    if (type === "next") {
      setPagination({ ...pagination, offset: pagination.offset + 10 });
    }
    if (type === "previous") {
      setPagination({ ...pagination, offset: pagination.offset - 10 });
    }
  };

  const { mutateAsync: addToPlaylist } = useMutation(
    async (songId: string) => {
      await axios.post("/v1/me/player/queue", null, {
        baseURL: SPOTIFY_API_URL,
        headers: { Authorization: `Bearer ${props.accessToken}` },
        params: { uri: songId },
      });
    },
    {
      onSuccess: () => {
        toast({
          title:
            props.language === "korean"
              ? "노래가 추가되었습니다."
              : "Song added to playlist",
        });
        setSearchQuery("");
        setPagination({ limit: 10, offset: 0 });
      },
    },
  );

  return (
    <Card>
      <AlertDialog>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {props.language === "korean"
                ? `노래 제목: ${title}`
                : `Song Title: ${title}`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {props.language === "korean"
                ? "정말로 노래를 플레이리스트에 추가하시겠습니까?"
                : "Are you sure you want to add the song to the playlist?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {props.language === "korean" ? "취소" : "Cancel"}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                addToPlaylist(songId);
              }}
            >
              {props.language === "korean" ? "추가" : "Add"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
        <CardHeader>
          <CardTitle className="mb-2 flex items-center justify-between">
            {props.language === "korean" ? "노래 신청" : "Request Song"}
          </CardTitle>
          <CardDescription>
            <Label htmlFor="searchQuery" className="mb-2 block">
              {props.language === "korean"
                ? "검색 (노래 제목, 가수, 앨범)"
                : "Search (Song Title, Artist, Album)"}
            </Label>
            <div className="flex w-full items-center space-x-2">
              <Input
                type="text"
                id="searchQuery"
                placeholder={
                  props.language === "korean" ? "검색어" : "Search Term"
                }
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <Button
                variant="outline"
                size="icon"
                onClick={() => setSearchQuery("")}
              >
                <X />
              </Button>
            </div>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>#</TableHead>
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
                <TableHead>
                  {props.language === "korean"
                    ? "플레이리스트 추가"
                    : "Add to Playlist"}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading
                ? Array.from({ length: 10 }).map((_, idx) => (
                    <TableRow key={idx}>
                      <TableCell>
                        <Skeleton className="h-[20px] w-[50px]" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-[50px] w-[50px]" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-[20px] w-[100px]" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-[20px] w-[100px]" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-[20px] w-[100px]" />
                      </TableCell>
                    </TableRow>
                  ))
                : searchResults?.tracks.items.map((item, index) => (
                    <TableRow
                      key={item.id}
                      className="cursor-pointer"
                      onClick={() => {
                        setSongId(item.uri);
                        setTitle(item.name);
                      }}
                    >
                      <TableCell>{index + 1 + pagination.offset}</TableCell>
                      <TableCell>
                        <Image
                          src={item.album.images[0].url}
                          alt={item.album.name}
                          width={50}
                          height={50}
                        />
                      </TableCell>
                      <TableCell>{item.name}</TableCell>
                      <TableCell>{item.artists[0].name}</TableCell>
                      <TableCell>{getDuration(item.duration_ms)}</TableCell>
                      <TableCell>
                        <AlertDialogTrigger asChild>
                          <Button variant="outline" className="w-full">
                            추가
                          </Button>
                        </AlertDialogTrigger>
                      </TableCell>
                    </TableRow>
                  )) || (
                    <CardDescription className="my-5 text-center">
                      {props.language === "korean"
                        ? "검색 결과가 없습니다."
                        : "No results found"}
                    </CardDescription>
                  )}
            </TableBody>
          </Table>
          <Separator className="my-2" />
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <Button
                  variant="secondary"
                  className="disabled:cursor-not-allowed"
                  disabled={!searchResults?.tracks.previous}
                  onClick={() => updatePagination("previous")}
                >
                  <ChevronLeft />
                </Button>
              </PaginationItem>
              <PaginationItem>
                <Button
                  variant="secondary"
                  className="disabled:cursor-not-allowed"
                  disabled={!searchResults?.tracks.next}
                  onClick={() => updatePagination("next")}
                >
                  <ChevronRight />
                </Button>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </CardContent>
      </AlertDialog>
    </Card>
  );
}
