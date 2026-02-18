import { useCallback, useEffect, useState } from "react";

function readStoredObject(key, fallback) {
  if (typeof window === "undefined") {
    return fallback;
  }
  try {
    const parsed = JSON.parse(window.localStorage.getItem(key) || "null");
    if (!parsed || typeof parsed !== "object") {
      return fallback;
    }
    return { ...fallback, ...parsed };
  } catch {
    return fallback;
  }
}

export default function useResizableColumns(storageKey, defaultWidths, options = {}) {
  const minWidth = options.minWidth ?? 48;
  const maxWidth = options.maxWidth ?? 720;
  const [columnWidths, setColumnWidths] = useState(() => readStoredObject(storageKey, defaultWidths));

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    window.localStorage.setItem(storageKey, JSON.stringify(columnWidths));
  }, [storageKey, columnWidths]);

  const handleResizeStart = useCallback((key, event) => {
    event.preventDefault();
    event.stopPropagation();

    const startX = event.clientX;
    const fallbackWidth = Number(defaultWidths[key]) || minWidth;
    const startWidth = Number(columnWidths[key]) || fallbackWidth;

    const handleMouseMove = (moveEvent) => {
      const rawWidth = startWidth + (moveEvent.clientX - startX);
      const boundedWidth = Math.max(minWidth, Math.min(maxWidth, rawWidth));
      setColumnWidths((prev) => ({ ...prev, [key]: boundedWidth }));
    };

    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  }, [columnWidths, defaultWidths, minWidth, maxWidth]);

  return {
    columnWidths,
    setColumnWidths,
    handleResizeStart,
  };
}
