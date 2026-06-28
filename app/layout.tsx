import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ABI Wound Billing Triage",
  description: "Medicare Part B wound-care eligibility worklist for billers"
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
