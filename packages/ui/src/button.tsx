import type { ButtonHTMLAttributes, PropsWithChildren } from "react";

type ButtonProps = PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "secondary" }
>;

export function Button({ children, className = "", variant = "primary", ...props }: ButtonProps) {
  const variantClass =
    variant === "primary"
      ? "bg-forest text-white hover:bg-[#164a36]"
      : "border border-black/10 bg-white text-ink hover:bg-black/[0.03]";

  return (
    <button
      className={`inline-flex items-center justify-center rounded-xl px-4 py-2.5 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-forest/30 ${variantClass} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}
