"use client";

import { motion } from "framer-motion";
import type { ReactNode } from "react";

export function PageFade({ children }: { children: ReactNode }) {
  return (
    <motion.div
      initial={{ y: 6 }}
      animate={{ y: 0 }}
      transition={{ duration: 0.16, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  );
}

export function Appear({ children, delay = 0 }: { children: ReactNode; delay?: number }) {
  return (
    <motion.div
      initial={{ y: 4 }}
      animate={{ y: 0 }}
      transition={{ duration: 0.14, delay, ease: "easeOut" }}
    >
      {children}
    </motion.div>
  );
}
