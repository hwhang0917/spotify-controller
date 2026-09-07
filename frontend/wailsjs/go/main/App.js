// @ts-check
// Hand-written to match app.go; `wails build`/`wails dev` regenerates this file.
const app = () => window['go']['main']['App'];
export function GetConfig() { return app()['GetConfig'](); }
export function SaveConfig(arg1) { return app()['SaveConfig'](arg1); }
export function Sources() { return app()['Sources'](); }
export function SetActiveSource(arg1) { return app()['SetActiveSource'](arg1); }
export function LocalRescan() { return app()['LocalRescan'](); }
export function SpotifyConnect() { return app()['SpotifyConnect'](); }
export function SpotifyDisconnect() { return app()['SpotifyDisconnect'](); }
export function SpotifyDevices() { return app()['SpotifyDevices'](); }
export function GetState() { return app()['GetState'](); }
export function Pause() { return app()['Pause'](); }
export function Resume() { return app()['Resume'](); }
export function Skip() { return app()['Skip'](); }
export function SetVolume(arg1) { return app()['SetVolume'](arg1); }
export function StartServer(arg1) { return app()['StartServer'](arg1); }
export function StopServer() { return app()['StopServer'](); }
export function Status() { return app()['Status'](); }
export function Guests() { return app()['Guests'](); }
export function KickGuest(arg1) { return app()['KickGuest'](arg1); }
export function RemoveGuest(arg1) { return app()['RemoveGuest'](arg1); }
export function BlockGuest(arg1, arg2) { return app()['BlockGuest'](arg1, arg2); }
