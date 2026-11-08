import type { ComponentProps, ReactNode } from "react";
import './button.css'

interface BaseProperties{
  icon?: ReactNode; // Erlaubt es, Icons von allen Libraries zu nehmen (Lucide, Phosphor, React Icons..)
  className?: string; // CSS-Eigenschaften für den Button
  children?: ReactNode; // Der Inhalt zwischen den Opening und Closing-Tags: <a> Sample Text </a>
  variant?: "filled" | "outline";
}

type LabelProps =
  | { children: ReactNode }
  | { children?: undefined; "aria-label": string };


type AnchorProps = BaseProperties & Omit<ComponentProps<'a'>, keyof BaseProperties> & { href: string };
type ButtonElementProps = BaseProperties & Omit<ComponentProps<'button'>, keyof BaseProperties> & { href?: undefined };

type ButtonProps = (AnchorProps | ButtonElementProps) & LabelProps;

export function Button(props: ButtonProps) {
  const classes = ["btn", `btn--${props.variant ?? "filled"}`, props.className].filter(Boolean).join(" ");

  const content = (
    <>
      {props.icon && (
        <span className="btn__icon" aria-hidden="true">
          {props.icon}
        </span>
      )}  
      {props.children && <span className="btn__label">{props.children}</span>}
    </>
  );

  
  if (props.href !== undefined) {
    const { icon, className, children, variant, ...anchorProps } = props;
    return (
      <a className={classes} {...anchorProps}>
      {content}
    </a>
    );
  }

  const { icon, className, children, variant, ...buttonProps } = props;
  return (
    <button className={classes} {...buttonProps}>
      {content}
    </button>
  );
}