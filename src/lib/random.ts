import { randomBytes } from "crypto";

/**
 * @param byte Byte length of the random hash
 */
export const generateRandomHash = (byte = 5) =>
  randomBytes(byte).toString("hex");
