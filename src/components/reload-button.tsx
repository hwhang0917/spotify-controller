"use client";

import React from "react";
import { Button } from "./ui/button";
import { ReloadIcon } from "@radix-ui/react-icons";

export function ReloadButton() {
  const reload = React.useCallback(() => {
    if (typeof window !== "undefined") {
      window.location.reload();
    }
  }, []);

  return (
    <Button variant="outline" onClick={reload}>
      <ReloadIcon />
    </Button>
  );
}
