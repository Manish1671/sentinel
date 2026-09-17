"use client";

import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

type IconButtonProps = ComponentProps<typeof Button> &
  VariantProps<typeof buttonVariants> & {
    label: string;
  };

export function IconButton({ label, size = "icon", className, children, ...props }: IconButtonProps) {
  return (
    <Button size={size} aria-label={label} title={label} className={cn(className)} {...props}>
      {children}
    </Button>
  );
}
