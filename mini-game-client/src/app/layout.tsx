import type { Metadata } from "next";
import "./globals.css";
import ToggleTheme from "@/components/ToggleTheme";

export const metadata: Metadata = {
  title: "Mini Game Hub - Play Tic-Tac-Toe, Chess, and More!",
  description:
    "Enjoy Tic-Tac-Toe, Chess, and other exciting multiplayer games. Connect with friends, chat, and compete for fun online!",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased min-h-screen bg-base-100 font-sans">
        <div className="absolute top-4 right-4">
          <ToggleTheme />
        </div>
        {children}
      </body>
    </html>
  );
}
