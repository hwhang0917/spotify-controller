import type { Metadata } from "next";
import { ThemeProvider } from "@/components/theme-provider";
import { QueryProvider } from "@/components/query-provider";
import { Noto_Sans_KR, Roboto } from "next/font/google";
import { GithubLink } from "@/components/github-link";
import { ModeToggle } from "@/components/mode-toggle";
import { cn } from "@/lib/utils";
import "./globals.css";

const notoSansKR = Noto_Sans_KR({
  subsets: ["latin"],
  variable: "--font-noto-kr",
});
const roboto = Roboto({
  subsets: ["latin"],
  weight: "400",
  variable: "--font-roboto",
});

export const metadata: Metadata = {
  title: "Spotify Controller",
  description:
    "Share your controller with friends and family to control Spotify playback.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ko">
      <body
        className={cn(
          "min-h-screen font-baseFont antialiased",
          notoSansKR.variable,
          roboto.variable,
        )}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <QueryProvider>
            <nav className="fixed bottom-0 right-0 flex gap-4 p-4">
              <GithubLink href="https://github.com/hwhang0917/spotify-controller" />
              <ModeToggle />
            </nav>
            {children}
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
