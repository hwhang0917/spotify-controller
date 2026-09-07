// Hand-written to match app.go; `wails build`/`wails dev` regenerates this file.
// Types live in src/types.ts so the UI does not depend on generated models.
export function GetConfig(): Promise<any>;
export function SaveConfig(arg1: any): Promise<void>;
export function Sources(): Promise<any[]>;
export function SetActiveSource(arg1: string): Promise<void>;
export function LocalRescan(): Promise<number>;
export function SpotifyConnect(): Promise<void>;
export function SpotifyDisconnect(): Promise<void>;
export function SpotifyDevices(): Promise<any[]>;
export function GetState(): Promise<any>;
export function Pause(): Promise<void>;
export function Resume(): Promise<void>;
export function Skip(): Promise<void>;
export function SetVolume(arg1: number): Promise<void>;
export function StartServer(arg1: number): Promise<string>;
export function StopServer(): Promise<void>;
export function Status(): Promise<any>;
