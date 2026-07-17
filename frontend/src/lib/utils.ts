import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

// cn merges Tailwind class lists, resolving conflicts (later wins).
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
