import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ABI Wound Billing Dashboard",
  description: "Eligibility dashboard for the wound care billing pipeline"
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
