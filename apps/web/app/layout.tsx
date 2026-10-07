import type { Metadata } from "next";
import type { ReactNode } from "react";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "ForkLedger",
  description: "Control plane for modified upstream software",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <div className="app">
          <header className="top">
            <div className="brand">ForkLedger</div>
            <nav className="nav">
              <Link href="/">Attention</Link>
              <Link href="/repository">Repository</Link>
            </nav>
          </header>
          {children}
        </div>
      </body>
    </html>
  );
}
