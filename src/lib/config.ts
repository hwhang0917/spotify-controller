import path from "path";
import fs from "fs/promises";
import { existsSync } from "fs";
import { DEFAULT_CONFIG } from "@/constants";

/**
 * Get configuration from the config.json file, create a default one if it doesn't exist
 */
export async function getConfiguration(): Promise<SpotifyControllerConfig> {
  const configPath = path.join(process.cwd(), "config.json");
  if (!existsSync(configPath)) {
    // Create a default config file
    await fs.writeFile(configPath, JSON.stringify(DEFAULT_CONFIG), "utf-8");
    return DEFAULT_CONFIG;
  }
  const config = await fs.readFile(configPath, "utf-8");
  return JSON.parse(config) as SpotifyControllerConfig;
}

/**
 * Store the configuration in the config.json file
 */
export async function storeConfiguration(config: SpotifyControllerConfig) {
  const configPath = path.join(process.cwd(), "config.json");
  await fs.writeFile(configPath, JSON.stringify(config), "utf-8");
}
