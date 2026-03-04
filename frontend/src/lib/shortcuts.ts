import { PlayerService } from "../../bindings/github.com/willfish/forte";

export interface Shortcut {
  key: string;
  ctrl?: boolean;
  description: string;
  action: () => void | Promise<void>;
}

const VOLUME_STEP = 5;
const SEEK_STEP = 5;

export const shortcuts: Shortcut[] = [
  {
    key: "f",
    ctrl: true,
    description: "Search library",
    action: () => {}, // handled by Content.svelte
  },
  {
    key: " ",
    description: "Play / Pause",
    action: async () => {
      const state = await PlayerService.State();
      if (state === "playing") await PlayerService.Pause();
      else if (state === "paused") await PlayerService.Resume();
    },
  },
  {
    key: "ArrowLeft",
    description: "Seek backward 5s",
    action: async () => {
      const pos = await PlayerService.Position();
      await PlayerService.Seek(Math.max(0, pos - SEEK_STEP));
    },
  },
  {
    key: "ArrowRight",
    description: "Seek forward 5s",
    action: async () => {
      const pos = await PlayerService.Position();
      const dur = await PlayerService.Duration();
      await PlayerService.Seek(Math.min(dur, pos + SEEK_STEP));
    },
  },
  {
    key: "ArrowLeft",
    ctrl: true,
    description: "Previous track",
    action: () => PlayerService.Previous(),
  },
  {
    key: "ArrowRight",
    ctrl: true,
    description: "Next track",
    action: () => PlayerService.Next(),
  },
  {
    key: "ArrowUp",
    ctrl: true,
    description: "Volume up",
    action: async () => {
      const vol = await PlayerService.Volume();
      await PlayerService.SetVolume(Math.min(100, vol + VOLUME_STEP));
    },
  },
  {
    key: "ArrowDown",
    ctrl: true,
    description: "Volume down",
    action: async () => {
      const vol = await PlayerService.Volume();
      await PlayerService.SetVolume(Math.max(0, vol - VOLUME_STEP));
    },
  },
];

function isInputFocused(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

export function handleKeydown(e: KeyboardEvent): boolean {
  if (isInputFocused()) return false;

  const match = shortcuts.find(
    (s) => s.key === e.key && !!s.ctrl === (e.ctrlKey || e.metaKey)
  );
  if (!match) return false;

  e.preventDefault();
  match.action();
  return true;
}

export function formatKey(shortcut: Shortcut): string {
  const parts: string[] = [];
  if (shortcut.ctrl) parts.push("Ctrl");

  switch (shortcut.key) {
    case " ":
      parts.push("Space");
      break;
    case "ArrowLeft":
      parts.push("\u2190");
      break;
    case "ArrowRight":
      parts.push("\u2192");
      break;
    case "ArrowUp":
      parts.push("\u2191");
      break;
    case "ArrowDown":
      parts.push("\u2193");
      break;
    case "?":
      parts.push("?");
      break;
    default:
      parts.push(shortcut.key);
  }

  return parts.join(" + ");
}
