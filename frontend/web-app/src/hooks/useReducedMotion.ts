import { useEffect, useState } from "react";

export function useReducedMotion() {
  const getReducedMotionPreference = () =>
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  const [reducedMotion, setReducedMotion] = useState(getReducedMotionPreference);

  useEffect(() => {
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    const updatePreference = () => setReducedMotion(media.matches);

    updatePreference();
    media.addEventListener("change", updatePreference);

    return () => {
      media.removeEventListener("change", updatePreference);
    };
  }, []);

  return reducedMotion;
}
