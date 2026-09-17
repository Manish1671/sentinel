"use client";

import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

type SearchInputProps = {
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  id?: string;
};

export function SearchInput({
  value,
  onValueChange,
  placeholder = "Search",
  className,
  id,
}: SearchInputProps) {
  return (
    <div className={cn("relative", className)}>
      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-text-muted" aria-hidden />
      <Input
        id={id}
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
        placeholder={placeholder}
        className="pl-8"
        aria-label={placeholder}
      />
    </div>
  );
}
