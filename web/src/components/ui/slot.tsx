import * as React from "react";

// Minimal Slot: merges props/className onto a single child element (asChild pattern),
// avoiding a dependency on radix. Base UI favors the `render` prop; for shadcn-style
// `asChild` ergonomics we keep this tiny local helper.
export interface SlotProps extends React.HTMLAttributes<HTMLElement> {
  children?: React.ReactNode;
}

type SlotChildProps = React.HTMLAttributes<HTMLElement> & {
  ref?: React.Ref<HTMLElement>;
};

export const Slot = React.forwardRef<HTMLElement, SlotProps>(function Slot(
  { children, ...props },
  ref,
) {
  if (!React.isValidElement<SlotChildProps>(children)) return null;
  const child = children;
  return React.cloneElement(child, {
    ...props,
    ...child.props,
    ref,
    className: [props.className, child.props.className].filter(Boolean).join(" "),
    style: { ...props.style, ...child.props.style },
  });
});
