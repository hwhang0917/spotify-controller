"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "./ui/button";
import { GearIcon } from "@radix-ui/react-icons";

export function SettingsButton() {
  const pathname = usePathname();

  if (pathname === "/admin") {
    return null;
  }

  return (
    <Link href="/admin">
      <Button variant="outline" size="icon">
        <GearIcon />
      </Button>
    </Link>
  );
}
