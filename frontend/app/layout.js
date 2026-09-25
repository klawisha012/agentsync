import { Geist, JetBrains_Mono } from "next/font/google";
import Shell from "./shell";
import "./globals.css";

const geist = Geist({ subsets: ["latin", "cyrillic"], variable: "--font-geist" });
const mono = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  variable: "--font-mono",
});

export const metadata = {
  title: "AgentSync",
};

export default function RootLayout({ children }) {
  return (
    <html lang="ru" className={`${geist.variable} ${mono.variable}`}>
      <body>
        <Shell>{children}</Shell>
      </body>
    </html>
  );
}
