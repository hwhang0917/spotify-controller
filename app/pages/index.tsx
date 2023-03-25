import Head from "next/head";
import { serverSideTranslations } from "next-i18next/serverSideTranslations";
import { useTranslation } from "next-i18next";
import { GetStaticProps } from "next";
import { Box, Container, Stack } from "@mui/system";
import {
  Alert,
  Button,
  CardMedia,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Grid,
  InputAdornment,
  List,
  ListItem,
  ListItemButton,
  Snackbar,
  TextField,
  Typography,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import { useDeferredValue, useState } from "react";
import { CurrentSong } from "@components/CurrentSong";
import { useMutation, useQuery } from "react-query";
import { ISearchSongResponse } from "@interfaces/search-songs";
import axios from "axios";
import { Playlist } from "@components/Playlist";
import { ICommonResponse } from "@interfaces/common-response";
import { ITrack } from "@interfaces/track";
import { IPlaylist } from "@interfaces/playlist";

export const getStaticProps: GetStaticProps = async ({ locale = "en" }) => ({
  props: {
    ...(await serverSideTranslations(locale, ["common"])),
  },
});

export default function Index() {
  const { t } = useTranslation("common");
  const [query, setQuery] = useState<string>("");
  const deferredQuery = useDeferredValue(query);

  // Queue Add Modal
  const [selectedTrack, setSelectedTrack] = useState<ITrack>();
  const [addQueueModalOpen, setAddQueueModalOpen] = useState(false);
  const handleQueueModalOpen = () => {
    setAddQueueModalOpen(true);
  };
  const handleQueueModalClose = () => {
    setAddQueueModalOpen(false);
    setSelectedTrack(undefined);
  };

  // Snackbar
  const [snackbarOpen, setSnackbarOpen] = useState(false);

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

  const addToPlaylist = async (id: string) => {
    const { data } = await axios.post<ICommonResponse>(
      "/webapi/spotify/queue",
      {},
      {
        params: {
          id,
        },
      }
    );
    return data;
  };

  const fetchPlaylist = async () => {
    const { data } = await axios.get<IPlaylist>("/webapi/spotify/queue");
    return data;
  };

  const {
    isLoading: isPlaylistLoading,
    data: playlistData,
    refetch: refetchPlaylist,
  } = useQuery(["playlist"], fetchPlaylist);

  const { data: searchResult, isLoading: isSearchLoading } = useQuery(
    ["search-song", deferredQuery],
    async () => {
      return await fetchSearchTrack(deferredQuery);
    },
    {
      enabled: deferredQuery !== "",
    }
  );

  const { mutate: addMusicToPlaylist } = useMutation(addToPlaylist, {
    onSuccess: () => {
      setSnackbarOpen(true);
      refetchPlaylist();
    },
  });

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
      {/* Queue Modal Section */}
      <Dialog open={addQueueModalOpen} onClose={handleQueueModalClose}>
        <DialogTitle>{t("add-confirm")}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            <Typography variant="subtitle1">{selectedTrack?.name}</Typography>
            <Typography variant="body2">By {selectedTrack?.artists}</Typography>
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleQueueModalClose}>{t("cancel")}</Button>
          <Button
            onClick={() => {
              if (selectedTrack) {
                addMusicToPlaylist(selectedTrack.id);
              }
              handleQueueModalClose();
            }}
          >
            {t("confirm")}
          </Button>
        </DialogActions>
      </Dialog>
      {/* Queue Add Snackbar */}
      <Snackbar
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
        open={snackbarOpen}
        autoHideDuration={3000}
        onClose={() => setSnackbarOpen(false)}
      >
        <Alert severity="success">{t("add-queue-success")}</Alert>
      </Snackbar>
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
              {/* Search Loading */}
              {isSearchLoading && (
                <ListItem
                  key={1}
                  sx={{
                    display: "flex",
                    justifyContent: "center",
                    margin: "1.5em",
                  }}
                >
                  <CircularProgress />
                </ListItem>
              )}
              {/* No Search Result */}
              {searchResult?.data?.length === 0 && (
                <ListItem key={1}>
                  <Typography variant="h5">{t("no-result")} 🤷</Typography>
                </ListItem>
              )}
              {/* Search List Buttons */}
              {searchResult?.data?.map((track, idx) => (
                <ListItemButton
                  key={idx}
                  onClick={() => {
                    setSelectedTrack(track);
                    handleQueueModalOpen();
                  }}
                >
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
          <Playlist isLoading={isPlaylistLoading} data={playlistData} />
        </Grid>
      </Grid>
    </>
  );
}
