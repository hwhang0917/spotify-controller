import Head from "next/head";
import { serverSideTranslations } from "next-i18next/serverSideTranslations";
import { useTranslation } from "next-i18next";
import { GetStaticProps } from "next";
import "@fontsource/roboto/300.css";
import "@fontsource/roboto/400.css";
import "@fontsource/roboto/500.css";
import "@fontsource/roboto/700.css";
import { Box, Container, Stack } from "@mui/system";
import {
  CardMedia,
  Fab,
  Grid,
  InputAdornment,
  List,
  ListItem,
  ListItemButton,
  TextField,
  Typography,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import { useContext, useDeferredValue, useState } from "react";
import { CurrentSong } from "@components/CurrentSong";
import { useQuery } from "react-query";
import { ISearchSongResponse } from "@interfaces/search-songs";
import axios from "axios";
import { Playlist } from "@components/Playlist";

export const getStaticProps: GetStaticProps = async ({ locale = "en" }) => ({
  props: {
    ...(await serverSideTranslations(locale, ["common"])),
  },
});

export default function Index() {
  const { t } = useTranslation("common");
  const [query, setQuery] = useState<string>("");
  const deferredQuery = useDeferredValue(query);

  const fetchSearchTrack = async (q: string) => {
    const { data } = await axios.get<ISearchSongResponse>(
      "/webapi/spotify/search",
      {
        params: {
          q,
          page: 1,
        },
      }
    );
    return data;
  };

  const { data: searchResult, isLoading } = useQuery(
    ["search-song", deferredQuery],
    async () => {
      return await fetchSearchTrack(deferredQuery);
    },
    {
      enabled: deferredQuery !== "",
    }
  );

  return (
    <>
      <Head>
        <title>{t("title")}</title>
        <meta
          name="description"
          content="This website controls Spotify player."
        />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      {/* Search Section */}
      <Grid container spacing={2}>
        <Grid item xs={4} sx={{ overflowY: "scroll", height: "100vh" }}>
          <Container maxWidth="md">
            <Box
              component="form"
              sx={{
                position: "sticky",
                marginTop: "2em",
                top: "2em",
                zIndex: 100,
                backgroundColor: "rgba(0,0,0,0.1)",
                backdropFilter: "blur(1.5rem)",
              }}
              onSubmit={(e) => e.preventDefault()}
            >
              <TextField
                label={t("search")}
                variant="outlined"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                }}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                }}
                fullWidth
              />
            </Box>
            <List>
              {searchResult?.data?.map((track, idx) => (
                <ListItemButton key={idx} id={track.id}>
                  <CardMedia
                    component="img"
                    sx={{ width: 75, marginRight: "1em" }}
                    image={track.coverImageUrl}
                    alt={track.name + " album cover"}
                  />
                  <Stack>
                    <Typography variant="h5">{track.name}</Typography>
                    <Typography variant="body2">By {track.artists}</Typography>
                  </Stack>
                </ListItemButton>
              ))}
            </List>
          </Container>
        </Grid>
        {/* Playlist section */}
        <Grid item xs={8}>
          <Container>
            <CurrentSong />
          </Container>
          <Playlist />
        </Grid>
      </Grid>
    </>
  );
}
