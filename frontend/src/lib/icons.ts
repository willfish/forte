import type { LucideIcon } from '@lucide/svelte';
import ChartNoAxesColumn from '@lucide/svelte/icons/chart-no-axes-column';
import ChevronLeft from '@lucide/svelte/icons/chevron-left';
import GripVertical from '@lucide/svelte/icons/grip-vertical';
import Heart from '@lucide/svelte/icons/heart';
import Library from '@lucide/svelte/icons/library';
import ListMusic from '@lucide/svelte/icons/list-music';
import Maximize2 from '@lucide/svelte/icons/maximize-2';
import Minus from '@lucide/svelte/icons/minus';
import Music from '@lucide/svelte/icons/music';
import Pause from '@lucide/svelte/icons/pause';
import Pencil from '@lucide/svelte/icons/pencil';
import Play from '@lucide/svelte/icons/play';
import Radio from '@lucide/svelte/icons/radio';
import Repeat from '@lucide/svelte/icons/repeat';
import Repeat1 from '@lucide/svelte/icons/repeat-1';
import Search from '@lucide/svelte/icons/search';
import Server from '@lucide/svelte/icons/server';
import Settings from '@lucide/svelte/icons/settings';
import Shuffle from '@lucide/svelte/icons/shuffle';
import SkipBack from '@lucide/svelte/icons/skip-back';
import SkipForward from '@lucide/svelte/icons/skip-forward';
import Square from '@lucide/svelte/icons/square';
import Trash2 from '@lucide/svelte/icons/trash-2';
import Volume1 from '@lucide/svelte/icons/volume-1';
import Volume2 from '@lucide/svelte/icons/volume-2';
import VolumeX from '@lucide/svelte/icons/volume-x';
import X from '@lucide/svelte/icons/x';

export const iconComponents = {
  // Transport
  play: Play,
  pause: Pause,
  prev: SkipBack,
  next: SkipForward,
  stop: Square,

  // Mode toggles
  shuffle: Shuffle,
  repeat: Repeat,
  repeatOne: Repeat1,

  // Volume
  volume: Volume2,
  volumeLow: Volume1,
  volumeMute: VolumeX,

  // Navigation
  radio: Radio,
  library: Library,
  playlists: ListMusic,
  stats: ChartNoAxesColumn,
  settings: Settings,

  // Common actions
  search: Search,
  close: X,
  maximize: Maximize2,
  minimize: Minus,
  back: ChevronLeft,
  heart: Heart,
  heartFilled: Heart,
  trash: Trash2,
  edit: Pencil,
  drag: GripVertical,

  // List / queue
  queue: ListMusic,

  // Supporting
  musicNote: Music,
  server: Server,
} satisfies Record<string, LucideIcon>;

export type IconName = keyof typeof iconComponents;
