import type { ButtonHTMLAttributes, MouseEvent } from "react";
import { useTactilePress } from "../hooks/useTactilePress";

type TactileButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export function TactileButton({
  className,
  disabled,
  onContextMenu,
  ...props
}: TactileButtonProps) {
  const tactile = useTactilePress<HTMLButtonElement>(Boolean(disabled));
  const composedClassName = [
    className,
    tactile.isPressing ? "is-pressing" : "",
  ].filter(Boolean).join(" ");

  const handleContextMenu = (event: MouseEvent<HTMLButtonElement>) => {
    onContextMenu?.(event);
    if (!event.defaultPrevented) {
      event.preventDefault();
    }
  };

  return (
    <button
      {...props}
      {...tactile.pressProps}
      onContextMenu={handleContextMenu}
      className={composedClassName}
      data-pressed={tactile.isPressing ? "true" : undefined}
      disabled={disabled}
    />
  );
}
