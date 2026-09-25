import type { Metadata } from "next";
import "./globals.css";
import { themeInitializationScript } from "./theme";

export const metadata: Metadata = {
  title: "WeKnora — Find the answers, and put knowledge to work",
  description: "WeKnora, Tencent's open-source knowledge framework, brings together RAG Q&A, Agent reasoning and automatic Wiki. v0.8.2 lets agents operate the local browser, serves knowledge bases as an MCP Server, and supports forking and rewinding conversations; private deployment is supported.",
  // Reuse the documentation favicon so the homepage adds nothing at the site root.
  icons: { icon: "/docs/favicon.ico" },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased" suppressHydrationWarning>
      <head><script dangerouslySetInnerHTML={{ __html: themeInitializationScript }} /></head>
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
