import axios from "axios";
import {
  Card,
  CardContent,
  CardMedia,
  Chip,
  Skeleton,
  Typography,
} from "@mui/material";
import { Box, Container } from "@mui/system";
import { useQuery } from "react-query";
import type { ICurrentSongResponse } from "@interfaces/current-song";

const Wrapper = ({ children }: { children: React.ReactNode }) => {
  return (
    <Container maxWidth="md">
      <Box
        sx={{
          padding: "2em 0",
          display: "flex",
          justifyContent: "center",
        }}
      >
        <Card
          sx={{
            display: "flex",
            justifyContent: "space-between",
            width: "100%",
            backgroundColor: "rgba(17,17,17,0.3)",
            backdropFilter: "blur(1.5rem)",
          }}
        >
          {children}
        </Card>
      </Box>
    </Container>
  );
};

export const CurrentSong: React.FC = () => {
  const fetchCurrentSong = async () => {
    const { data } = await axios.get<ICurrentSongResponse>(
      "/webapi/spotify/current"
    );
    return data;
  };

  const { isLoading, data } = useQuery(["current-track"], fetchCurrentSong);

  // Loading
  if (isLoading) {
    return (
      <Wrapper>
        <Box sx={{ display: "flex", flexDirection: "column" }}>
          <CardContent sx={{ flex: "1 0 auto" }}>
            <Skeleton
              variant="text"
              sx={{ width: 512 * 0.7, fontSize: "1.5rem" }}
            />
            <Skeleton
              variant="text"
              sx={{ width: 512 * 0.3, fontSize: "1rem" }}
            />
            <Box sx={{ marginTop: "1rem" }}>
              <Skeleton
                variant="rounded"
                sx={{
                  width: 70,
                  height: 30,
                  borderRadius: "1em",
                  marginTop: "1em",
                }}
              />
            </Box>
          </CardContent>
        </Box>
        <Skeleton variant="rectangular" sx={{ width: 151, height: 151 }} />
      </Wrapper>
    );
  } else {
    // No current song
    if (!data) {
      return (
        <Wrapper>
          <Box sx={{ display: "flex", flexDirection: "column" }}>
            <CardContent sx={{ flex: "1 0 auto" }}>
              <Typography component="div" variant="h5">
                No song
              </Typography>
              <Typography
                variant="subtitle1"
                color="text.secondary"
                component="div"
              >
                There is no current playing song
              </Typography>
            </CardContent>
          </Box>
          <CardMedia
            component="img"
            sx={{ width: 151 }}
            image="https://via.placeholder.com/512.png/202020/ffffff?text=No%20Album%20Cover"
            alt="not playing"
          />
        </Wrapper>
      );
    } // Current Song Exists
    else {
      return (
        <Wrapper>
          <Box sx={{ display: "flex", flexDirection: "column" }}>
            <CardContent sx={{ flex: "1 0 auto" }}>
              {/* Song title */}
              <Typography component="div" variant="h5">
                {data.name}
              </Typography>
              {/* Artists names */}
              <Typography
                variant="subtitle1"
                color="text.secondary"
                component="div"
              >
                {data.artists}
              </Typography>
              {/* Playing chip */}
              {data.isPlaying && (
                <Box sx={{ marginTop: "1rem" }}>
                  <Chip label="playing" color="primary" />
                </Box>
              )}
            </CardContent>
          </Box>
          <CardMedia
            component="img"
            sx={{ width: 151 }}
            image={data.coverImageUrl}
            alt={data.name + " album cover"}
          />
        </Wrapper>
      );
    }
  }
};
