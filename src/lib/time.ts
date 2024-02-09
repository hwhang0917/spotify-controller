import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
dayjs.extend(duration);

export const getDuration = (ms: number, format: string = "mm:ss") => {
  return dayjs.duration(ms).format(format);
};
