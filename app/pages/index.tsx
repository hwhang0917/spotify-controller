import Head from "next/head";
import { serverSideTranslations } from "next-i18next/serverSideTranslations";
import { useTranslation } from "next-i18next";
import { GetStaticProps } from "next";
import "@fontsource/roboto/300.css";
import "@fontsource/roboto/400.css";
import "@fontsource/roboto/500.css";
import "@fontsource/roboto/700.css";
import { Box, Container } from "@mui/system";
import { Grid, InputAdornment, TextField } from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import { useDeferredValue, useState } from "react";
import { CurrentSong } from "@components/CurrentSong";

export const getStaticProps: GetStaticProps = async ({ locale = "en" }) => ({
  props: {
    ...(await serverSideTranslations(locale, ["common"])),
  },
});

export default function Index() {
  const { t } = useTranslation("common");
  const [query, setQuery] = useState<string>("");
  const deferredQuery = useDeferredValue(query);

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
      {/* 검색 창 */}
      <Grid container spacing={2}>
        <Grid item xs={4}>
          <Container maxWidth="md">
            <Box
              component="form"
              sx={{
                padding: "2em",
              }}
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
          </Container>
        </Grid>
        <Grid item xs={8} sx={{ overflowY: "scroll", height: "100vh" }}>
          {/* 현재 재생 중 */}
          <Container
            sx={{
              position: "sticky",
              top: "0",
              backgroundColor: "rgba(0,0,0,0.3)",
            }}
          >
            <CurrentSong />
          </Container>
        </Grid>
      </Grid>
    </>
  );
}
