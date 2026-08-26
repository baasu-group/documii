import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/*/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#19201c",
        paper: "#f4f6f3",
        forest: "#205c45",
        moss: "#dbe8df",
      },
      boxShadow: { soft: "0 18px 50px rgba(25, 32, 28, 0.08)" },
    },
  },
  plugins: [],
} satisfies Config;
