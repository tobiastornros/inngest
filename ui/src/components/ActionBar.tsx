import { ComponentChildren } from "preact";

interface ActionBarProps {
  tabs?: ComponentChildren;
  actions?: ComponentChildren;
}

export default function ActionBar({ tabs, actions }: ActionBarProps) {
  return (
    <div className="col-span-2 row-start-2 col-start-2 bg-slate-950/50 relative z-20 backdrop-blur-md border-b border-slate-800/60 flex flex-row justify-between">
      <div className="flex h-full">{tabs}</div>
      <div className="flex flex-row space-x-2 items-center mr-2">{actions}</div>
    </div>
  );
}
