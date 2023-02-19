import axios from "axios";
import qs from "qs";
import dotenv from "dotenv";
dotenv.config();

const CLIENT_ID = process.env.SPOTIFY_CLIENT_ID;
const CLIENT_SECRET = process.env.SPOTIFY_CLIENT_SECRET;
const auth_token = new Buffer.from(`${CLIENT_ID}:${CLIENT_SECRET}`).toString(
  "base64"
);

/**
 * Get Spotify Auth token
 */
const getAuth = async () => {
  try {
    const token_url = "https://accounts.spotify.com/api/token";
    const data = qs.stringify({ grant_type: "client_credentials" });

    const response = await axios.post(token_url, data, {
      headers: {
        Authorization: `Basic ${auth_token}`,
        "Content-Type": "application/x-www-form-urlencoded",
      },
    });
    return response.data.access_token;
  } catch (error) {
    console.log(error);
  }
};

const searchSong = async (searchInput) => {
  try {
    const token = await getAuth();
    const q = searchInput;
    const response = await axios.get("https://api.spotify.com/v1/search/", {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      params: {
        q,
        type: "track",
        market: "KR",
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

try {
  const result = await searchSong("스텔라 장");
  const parsedSongs = result.tracks.items.map((v) => {
    const artists = v.artists.map(({ name }) => name);
    const artist = new Intl.ListFormat("ko-KR").format(artists);
    const coverImage = v.album.images?.at(0)?.url;

    return {
      id: v.id,
      name: v.name,
      artist,
      preview_url: v.preview_url,
      coverImage,
    };
  });
  console.log(parsedSongs);
  //console.log(parsedSongs);
  //const token = await getAuth();
  //const queue = await axios.get("https://api.spotify.com/v1/me/player/queue", {
  //headers: {
  //Authorization: `Bearer ${token}`,
  //},
  //});
  //console.log(queue);
} catch (err) {
  throw err;
}
