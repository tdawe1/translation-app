import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GengoWatcher SaaS",
  description: "Multi-tenant job monitoring with per-user watcher instances",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  );
}
