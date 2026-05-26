import { useCallback, useEffect, useRef, useState, type FocusEvent, type PointerEvent } from "react";

export function useTactilePress<T extends HTMLElement>(disabled = false) {
  const [isPressing, setIsPressing] = useState(false);
  const targetRef = useRef<T | null>(null);
  const pointerIdRef = useRef<number | null>(null);

  const clearPress = useCallback(() => {
    const target = targetRef.current;
    const pointerId = pointerIdRef.current;

    if (target && pointerId !== null && target.hasPointerCapture?.(pointerId)) {
      target.releasePointerCapture(pointerId);
    }

    pointerIdRef.current = null;
    targetRef.current = null;
    setIsPressing(false);
  }, []);

  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        clearPress();
      }
    };

    window.addEventListener("blur", clearPress);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      window.removeEventListener("blur", clearPress);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [clearPress]);

  useEffect(() => {
    if (disabled) {
      clearPress();
    }
  }, [clearPress, disabled]);

  const onPointerDown = useCallback((event: PointerEvent<T>) => {
    if (disabled || event.button !== 0) {
      return;
    }

    targetRef.current = event.currentTarget;
    pointerIdRef.current = event.pointerId;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    setIsPressing(true);
  }, [disabled]);

  const onBlur = useCallback((_event: FocusEvent<T>) => {
    clearPress();
  }, [clearPress]);

  return {
    isPressing,
    clearPress,
    pressProps: {
      onPointerDown,
      onPointerUp: clearPress,
      onPointerCancel: clearPress,
      onPointerLeave: clearPress,
      onLostPointerCapture: clearPress,
      onBlur,
    },
  };
}
