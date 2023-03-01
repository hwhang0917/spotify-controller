import { IPaginationMeta } from "./pagination";
import { ITrack } from "./track";

export interface ISearchSongResponse {
  data: Array<ITrack>;
  meta: IPaginationMeta;
}
