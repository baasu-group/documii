const paths = {
  home: "M3 11.5 12 4l9 7.5M5.5 10v10h13V10M9 20v-6h6v6",
  documents: "M6 3h8l4 4v14H6zM14 3v5h5M9 13h6M9 17h6",
  data: "M4 6c0-2 16-2 16 0s-16 2-16 0Zm0 0v6c0 2 16 2 16 0V6M4 12v6c0 2 16 2 16 0v-6",
  forms: "M6 3h12v18H6zM9 8h6M9 12h6M9 16h3",
  tasks: "M5 5h14v14H5zM8 12l2.5 2.5L16 9",
  automations: "M13 2 5 14h7l-1 8 8-12h-7z",
  ai: "M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Z",
  integrations:
    "M8 12a4 4 0 0 1 4-4h3V5l4 4-4 4v-3h-3a2 2 0 1 0 0 4M16 12a4 4 0 0 1-4 4H9v3l-4-4 4-4v3h3a2 2 0 1 0 0-4",
  settings:
    "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Zm8 4 2-1-2-4-2 .5-1.5-1L16 4h-4l-.5 2.5-1.5 1L8 7 6 11l2 1v2l-2 1 2 4 2-.5 1.5 1L12 22h4l.5-2.5 1.5-1 2 .5 2-4-2-1Z",
  search: "m20 20-4.5-4.5M18 11a7 7 0 1 1-14 0 7 7 0 0 1 14 0Z",
  plus: "M12 5v14M5 12h14",
  arrow: "M5 12h14M14 7l5 5-5 5",
} as const;

export type IconName = keyof typeof paths;

export function Icon({ name, className = "h-5 w-5" }: { name: IconName; className?: string }) {
  return (
    <svg
      aria-hidden="true"
      className={className}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d={paths[name]} />
    </svg>
  );
}
