import { PRODUCT } from "@dragzone/shared";
import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";

const title = "DragZone — a free Dropzone 4 for macOS";
const description =
  "Turn your menu bar into a drop shelf. Drag files onto action tiles to zip, AirDrop, upload, move, and more — a free, open-source Dropzone 4 clone for macOS.";

export const metadata: Metadata = {
  metadataBase: new URL(PRODUCT.site),
  title,
  description,
  alternates: { canonical: "/" },
  openGraph: {
    type: "website",
    url: PRODUCT.site,
    siteName: PRODUCT.displayName,
    title,
    description,
  },
  twitter: {
    card: "summary_large_image",
    title,
    description,
  },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
