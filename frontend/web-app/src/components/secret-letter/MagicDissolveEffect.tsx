import type React from "react";
import type { RefObject } from "react";

type MagicDissolveEffectProps = {
  dissolveRef: RefObject<HTMLDivElement | null>;
  active: boolean;
};

const shardMap = [
  [-122, -34, -160, -92, 0, 7],
  [-94, 18, -126, 72, 0.05, 6],
  [-72, -88, -108, -144, 0.12, 8],
  [-48, -12, -84, -96, 0.16, 5],
  [-26, 54, -68, 106, 0.08, 7],
  [-12, -122, -34, -174, 0.18, 5],
  [18, -72, 38, -148, 0.04, 9],
  [36, 16, 82, 72, 0.11, 6],
  [58, -28, 116, -78, 0.14, 7],
  [76, 66, 142, 128, 0.2, 5],
  [98, -92, 162, -158, 0.09, 6],
  [122, 24, 184, 86, 0.02, 8],
  [-136, 84, -196, 142, 0.22, 5],
  [-110, -118, -176, -178, 0.06, 6],
  [-62, 96, -104, 164, 0.17, 7],
  [4, 104, 18, 188, 0.1, 5],
  [42, -132, 78, -202, 0.24, 6],
  [88, 112, 148, 184, 0.15, 7],
  [134, -54, 210, -104, 0.19, 5],
  [152, 78, 224, 140, 0.28, 6],
] as const;

const moteMap = [
  [12, 28, -32, -120, 0],
  [18, 72, -76, 88, 0.04],
  [24, 44, -118, -42, 0.11],
  [30, 62, -94, 142, 0.16],
  [36, 30, -36, -188, 0.08],
  [42, 78, -18, 174, 0.2],
  [48, 22, 28, -164, 0.03],
  [54, 68, 52, 142, 0.14],
  [60, 36, 92, -118, 0.06],
  [66, 56, 116, 96, 0.22],
  [72, 26, 154, -78, 0.12],
  [78, 74, 184, 116, 0.18],
  [84, 46, 214, -24, 0.25],
  [15, 52, -210, 16, 0.27],
  [27, 18, -154, -162, 0.31],
  [51, 86, 34, 210, 0.34],
  [69, 14, 138, -196, 0.37],
  [88, 61, 234, 72, 0.41],
] as const;

export function MagicDissolveEffect({ dissolveRef, active }: MagicDissolveEffectProps) {
  return (
    <div
      ref={dissolveRef}
      className={`magic-dissolve ${active ? "magic-dissolve-active" : ""}`}
      aria-hidden="true"
    >
      <span className="spell-ring spell-ring-primary" />
      <span className="spell-ring spell-ring-secondary" />
      <span className="spell-sweep" />
      <span className="spell-core" />

      <div className="spark-shards">
        {shardMap.map(([x, y, tx, ty, delay, size], shard) => (
          <span
            key={shard}
            style={
              {
                "--x": `${x}px`,
                "--y": `${y}px`,
                "--tx": `${tx}px`,
                "--ty": `${ty}px`,
                "--delay": `${delay}s`,
                "--size": `${size}px`,
                "--height": `${size * 1.8}px`,
              } as React.CSSProperties
            }
          />
        ))}
      </div>

      <div className="star-motes">
        {moteMap.map(([x, y, tx, ty, delay], mote) => (
          <span
            key={mote}
            style={
              {
                "--x": `${x}%`,
                "--y": `${y}%`,
                "--tx": `${tx}px`,
                "--ty": `${ty}px`,
                "--delay": `${delay}s`,
              } as React.CSSProperties
            }
          />
        ))}
      </div>
    </div>
  );
}
