import { useState } from "react";
import { Button, Icon } from "@documi/ui";
import { navigation, type NavigationId } from "./navigation";

export type PlatformKind = "web" | "desktop";

const featureCopy: Record<Exclude<NavigationId, "home">, { title: string; description: string }> = {
  documents: {
    title: "Documents",
    description: "Versioned business records, organized and protected.",
  },
  data: {
    title: "Data",
    description: "Structured company information connected to source documents.",
  },
  forms: {
    title: "Forms",
    description: "Collect consistent information from people and processes.",
  },
  tasks: { title: "Tasks", description: "Human work and approvals that keep operations moving." },
  automations: {
    title: "Automations",
    description: "Durable workflows built from triggers, rules, and actions.",
  },
  ai: { title: "AI", description: "Permission-aware assistance using providers you control." },
  integrations: {
    title: "Integrations",
    description: "Connect Documi to approved company systems.",
  },
  settings: {
    title: "Settings",
    description: "Manage workspace policy, access, providers, and deployment.",
  },
};

function Home() {
  const cards = [
    [
      "documents",
      "Bring in a document",
      "Store an immutable source version and prepare it for processing.",
    ],
    [
      "tasks",
      "Review your work",
      "Approvals and assigned tasks appear here when workflows need you.",
    ],
    [
      "automations",
      "Design a process",
      "Connect documents, data, decisions, and people with durable steps.",
    ],
  ] as const;

  return (
    <>
      <div className="mb-8 flex flex-col justify-between gap-5 md:flex-row md:items-end">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.22em] text-forest">
            Wednesday, 26 August
          </p>
          <h1 className="m-0 max-w-2xl text-4xl font-semibold tracking-[-0.04em] text-ink md:text-5xl">
            Good evening, Basanta.
          </h1>
          <p className="mt-3 text-base text-black/55">
            Everything your team needs to move work forward.
          </p>
        </div>
        <Button>
          <Icon name="plus" className="mr-2 h-4 w-4" /> New document
        </Button>
      </div>

      <section className="grid gap-4 lg:grid-cols-3">
        {cards.map(([icon, title, text], index) => (
          <article
            key={title}
            className="group min-h-52 rounded-2xl border border-black/[0.07] bg-white p-6 shadow-soft transition hover:-translate-y-0.5"
          >
            <div
              className={`mb-8 flex h-11 w-11 items-center justify-center rounded-xl ${index === 1 ? "bg-[#ede8d9] text-[#6b5b2c]" : index === 2 ? "bg-[#e3e4ee] text-[#494f75]" : "bg-moss text-forest"}`}
            >
              <Icon name={icon} />
            </div>
            <h2 className="m-0 text-lg font-semibold tracking-[-0.02em]">{title}</h2>
            <p className="mb-5 mt-2 text-sm leading-6 text-black/55">{text}</p>
            <button
              className="flex items-center border-0 bg-transparent p-0 text-sm font-semibold text-forest"
              type="button"
            >
              Get started{" "}
              <Icon name="arrow" className="ml-1.5 h-4 w-4 transition group-hover:translate-x-1" />
            </button>
          </article>
        ))}
      </section>

      <section className="mt-5 rounded-2xl border border-black/[0.07] bg-white p-6 shadow-soft">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="m-0 text-lg font-semibold">Recent activity</h2>
            <p className="mb-0 mt-1 text-sm text-black/50">Workspace events will appear here.</p>
          </div>
          <span className="rounded-full bg-paper px-3 py-1 text-xs font-semibold text-black/45">
            Bootstrap mode
          </span>
        </div>
        <div className="mt-8 flex min-h-28 flex-col items-center justify-center rounded-xl border border-dashed border-black/10 bg-paper/70 text-center">
          <p className="m-0 text-sm font-medium">Your workspace is ready.</p>
          <p className="mb-0 mt-1 text-xs text-black/45">
            Create a document to begin building a business process.
          </p>
        </div>
      </section>
    </>
  );
}

function FeaturePlaceholder({ active }: { active: Exclude<NavigationId, "home"> }) {
  const feature = featureCopy[active];
  const icon = navigation.find((item) => item.id === active)?.icon ?? "home";
  return (
    <section className="rounded-2xl border border-black/[0.07] bg-white p-8 shadow-soft">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-moss text-forest">
        <Icon name={icon} />
      </div>
      <h1 className="mb-2 mt-7 text-4xl font-semibold tracking-[-0.04em]">{feature.title}</h1>
      <p className="m-0 max-w-xl text-base leading-7 text-black/55">{feature.description}</p>
      <div className="mt-10 rounded-xl border border-dashed border-black/10 bg-paper p-8 text-sm text-black/50">
        This product boundary is established and ready for its first vertical slice.
      </div>
    </section>
  );
}

export function AppShell({ platform }: { platform: PlatformKind }) {
  const [active, setActive] = useState<NavigationId>("home");
  return (
    <div className="min-h-screen bg-paper md:grid md:grid-cols-[240px_1fr]">
      <aside className="border-b border-black/[0.06] bg-[#fbfcfa] px-4 py-4 md:sticky md:top-0 md:h-screen md:border-b-0 md:border-r">
        <div className="flex items-center justify-between px-2 py-2 md:mb-7">
          <button
            type="button"
            onClick={() => setActive("home")}
            className="flex items-center border-0 bg-transparent p-0 text-ink"
          >
            <span className="mr-3 flex h-9 w-9 items-center justify-center rounded-xl bg-forest text-lg font-bold text-white">
              D
            </span>
            <span className="text-lg font-semibold tracking-[-0.03em]">Documi</span>
          </button>
          <span className="rounded-md bg-moss px-2 py-1 text-[10px] font-bold uppercase tracking-wider text-forest">
            {platform}
          </span>
        </div>
        <nav aria-label="Product navigation" className="flex gap-1 overflow-x-auto pb-1 md:block">
          {navigation.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => setActive(item.id)}
              aria-current={active === item.id ? "page" : undefined}
              className={`flex min-w-max items-center rounded-xl border-0 px-3 py-2.5 text-sm transition md:mb-1 md:w-full ${active === item.id ? "bg-moss font-semibold text-forest" : "bg-transparent font-medium text-black/55 hover:bg-black/[0.035] hover:text-ink"}`}
            >
              <Icon name={item.icon} className="mr-3 h-[18px] w-[18px]" />
              {item.label}
            </button>
          ))}
        </nav>
        <div className="absolute bottom-5 left-4 right-4 hidden items-center rounded-xl border border-black/[0.06] bg-white p-3 md:flex">
          <span className="mr-3 flex h-8 w-8 items-center justify-center rounded-full bg-[#d6cdb1] text-xs font-bold text-[#4b4126]">
            BS
          </span>
          <span>
            <strong className="block text-xs">Basanta Shrestha</strong>
            <span className="text-[11px] text-black/45">Documi workspace</span>
          </span>
        </div>
      </aside>
      <main className="min-w-0">
        <header className="flex h-16 items-center justify-end border-b border-black/[0.06] bg-paper/90 px-5 backdrop-blur md:px-10">
          <label className="relative hidden sm:block">
            <span className="sr-only">Search</span>
            <Icon name="search" className="absolute left-3 top-2.5 h-4 w-4 text-black/35" />
            <input
              className="w-56 rounded-xl border border-black/[0.08] bg-white py-2 pl-9 pr-3 text-sm outline-none placeholder:text-black/35 focus:border-forest/40 focus:ring-2 focus:ring-forest/10"
              placeholder="Search workspace"
            />
          </label>
        </header>
        <div className="mx-auto max-w-6xl p-5 md:p-10">
          {active === "home" ? <Home /> : <FeaturePlaceholder active={active} />}
        </div>
      </main>
    </div>
  );
}
