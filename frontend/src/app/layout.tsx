import type { Metadata } from "next";
import { Inter } from "next/font/google";
import Header from "@/components/header";
import { SessionProvider } from "@/contexts/SessionContext";
import "./globals.css";


const inter = Inter({
  subsets: ["latin"],
});


export const metadata: Metadata = {
  title: "ASKQL",
  description: "SQL queries with natural language",
};


export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${inter.className} antialiased`}
      >
        <SessionProvider>
          <Header />
          {children}
        </SessionProvider>
      </body>
    </html>
  );
}

