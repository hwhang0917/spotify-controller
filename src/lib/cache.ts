import { caching, type MemoryCache } from "cache-manager";

// Create a memory block
let memoryCache: MemoryCache | null = null;

/**
 * Get the memory cache instance
 */
export const getMemoryCache = async () => {
  if (!memoryCache) memoryCache = await caching("memory");
  return memoryCache;
};
