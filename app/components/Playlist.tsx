import { IPlaylist } from "@interfaces/playlist";
import { Box, Container } from "@mui/system";
import { FixedSizeList, ListChildComponentProps } from "react-window";
import axios from "axios";
import { useQuery } from "react-query";
import {
  Avatar,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Skeleton,
  Typography,
} from "@mui/material";
import React from "react";

const Wrapper = ({ children }: { children: React.ReactNode }) => {
  return (
    <Box>
      <Typography variant="h5">재생목록</Typography>
      <List sx={{ width: "100%", height: "calc(80px*9)", overflow: "scroll" }}>
        {children}
      </List>
    </Box>
  );
};

export const Playlist = () => {
  const fetchPlaylist = async () => {
    const { data } = await axios.get<IPlaylist>("/webapi/spotify/queue");
    return data;
  };

  const { isLoading, data } = useQuery(["playlist"], fetchPlaylist);

  const SkeletonLoadingList = Array.from({ length: 10 }, () => null).map(() => (
    <ListItem>
      <ListItemAvatar>
        <Skeleton variant="circular" sx={{ width: 50, height: 50 }} />
      </ListItemAvatar>
      <ListItemText
        primary={
          <Skeleton variant="text" sx={{ fontSize: "1.5rem", width: "10%" }} />
        }
        secondary={
          <Skeleton variant="text" sx={{ fontSize: "1rem", width: "15%" }} />
        }
      />
    </ListItem>
  ));

  if (isLoading) {
    return <Wrapper>{SkeletonLoadingList}</Wrapper>;
  } else {
    return (
      <Wrapper>
        {data?.queue.map((track, idx) => (
          <ListItem alignItems="flex-start" key={idx}>
            <ListItemAvatar>
              <Avatar
                alt={track.name + " album cover"}
                src={track.coverImageUrl}
              />
            </ListItemAvatar>
            <ListItemText
              primary={track.name}
              secondary={
                <React.Fragment>
                  <Typography
                    sx={{ display: "inline" }}
                    component="span"
                    variant="body2"
                    color="text.primary"
                  >
                    By
                  </Typography>
                  {" — " + track.artists}
                </React.Fragment>
              }
            />
          </ListItem>
        ))}
      </Wrapper>
    );
  }
};
