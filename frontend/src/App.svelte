<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import TitleBar from './TitleBar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
  import NowPlayingView from './NowPlayingView.svelte';
  import ShortcutHelp from './ShortcutHelp.svelte';
  import QueuePanel from './QueuePanel.svelte';
  import Toast from './Toast.svelte';
  import { LibraryService } from "../bindings/github.com/willfish/forte";
  import { handleKeydown } from './lib/shortcuts';
  import { isTitlebarEnabled, onTitlebarEnabledChange, setTitlebarEnabled } from './lib/stores';
  import { getPreference, initTheme, onPreferenceChange } from './lib/theme';

  type ActionHint = {
    key: string;
    label: string;
    top: number;
    left: number;
    element: HTMLElement;
  };

  initTheme();

  function applyNativeTheme(theme: string) {
    LibraryService.SetThemePreference(theme).catch(() => {});
  }

  applyNativeTheme(getPreference());

  let showHelp = $state(false);
  let showQueue = $state(false);
  let showNowPlaying = $state(false);
  let showTitlebar = $state(isTitlebarEnabled());
  let actionHints = $state<ActionHint[]>([]);
  let hintMode = $state(false);
  let hintInput = $state('');
  let pendingVimKey = $state('');

  const hintAlphabet = 'abcdefghijklmnopqrstuvwxyz';

  $effect(() => {
    return onTitlebarEnabledChange((enabled) => { showTitlebar = enabled; });
  });

  $effect(() => {
    return onPreferenceChange(applyNativeTheme);
  });

  $effect(() => {
    async function loadTitlebarPreference() {
      try {
        const prefs = await LibraryService.GetAppPreferences();
        setTitlebarEnabled(Boolean(prefs?.showTitlebar));
      } catch {
        setTitlebarEnabled(false);
      }
    }

    loadTitlebarPreference();
  });

  function isEditableTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) return false;
    const tag = target.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable;
  }

  function isVisibleAction(element: HTMLElement, viewport: DOMRect): boolean {
    if (element.hasAttribute('disabled') || element.getAttribute('aria-hidden') === 'true') return false;
    const style = window.getComputedStyle(element);
    if (style.visibility === 'hidden' || style.display === 'none') return false;

    const rect = element.getBoundingClientRect();
    return rect.width > 0 &&
      rect.height > 0 &&
      rect.bottom >= viewport.top &&
      rect.top <= viewport.bottom &&
      rect.right >= viewport.left &&
      rect.left <= viewport.right;
  }

  function actionSortWeight(element: HTMLElement): number {
    if (element.classList.contains('station-play')) return 0;
    if (element.classList.contains('station-main')) return 1;
    return 2;
  }

  function hintLabel(index: number, total: number): string {
    if (total <= hintAlphabet.length) {
      return hintAlphabet[index];
    }

    const labelLength = Math.max(2, Math.ceil(Math.log(total) / Math.log(hintAlphabet.length)));
    let n = index;
    let label = '';
    for (let i = 0; i < labelLength; i += 1) {
      label = hintAlphabet[n % hintAlphabet.length] + label;
      n = Math.floor(n / hintAlphabet.length);
    }

    return label;
  }

  function showActionHints() {
    const scope = document.querySelector('.content-area');
    if (!(scope instanceof HTMLElement)) return;

    const viewport = scope.getBoundingClientRect();
    const actions = Array.from(scope.querySelectorAll('button, a[href], input[type="button"], input[type="submit"]'))
      .filter((el): el is HTMLElement => el instanceof HTMLElement)
      .filter((el) => isVisibleAction(el, viewport))
      .sort((a, b) => actionSortWeight(a) - actionSortWeight(b));

    actionHints = actions.map((element, index) => {
      const rect = element.getBoundingClientRect();
      const label = hintLabel(index, actions.length);
      return {
        key: label,
        label,
        top: Math.max(4, rect.top + 4),
        left: Math.max(4, rect.left + 4),
        element,
      };
    });
    hintInput = '';
    hintMode = actionHints.length > 0;
  }

  function closeActionHints() {
    hintMode = false;
    actionHints = [];
    hintInput = '';
  }

  function activateHint(key: string): boolean {
    const nextInput = `${hintInput}${key.toLowerCase()}`;
    const hint = actionHints.find((item) => item.key === nextInput);
    if (hint) {
      hint.element.click();
      closeActionHints();
      return true;
    }

    if (actionHints.some((item) => item.key.startsWith(nextInput))) {
      hintInput = nextInput;
      return true;
    }

    hintInput = '';
    return false;
  }

  function scrollCurrentView(direction: 1 | -1) {
    const scope = document.querySelector('.content');
    if (!(scope instanceof HTMLElement)) return;
    scope.scrollBy({ top: direction * 48, behavior: 'smooth' });
  }

  function scrollCurrentViewTo(position: 'top' | 'bottom') {
    const scope = document.querySelector('.content');
    if (!(scope instanceof HTMLElement)) return;
    scope.scrollTo({
      top: position === 'top' ? 0 : scope.scrollHeight,
      behavior: 'smooth',
    });
  }

  function blurEditableTarget(target: EventTarget | null): boolean {
    if (!isEditableTarget(target) || !(target instanceof HTMLElement)) return false;
    target.blur();
    return true;
  }

  function clearPendingVimKey() {
    pendingVimKey = '';
  }

  function onKeydown(e: KeyboardEvent) {
    if (hintMode) {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeActionHints();
        return;
      }
      if (/^[a-z]$/i.test(e.key)) {
        e.preventDefault();
        activateHint(e.key);
        return;
      }
    }

    if (e.key === '?' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      showHelp = !showHelp;
      return;
    }
    if (e.key === 'Escape') {
      if (showHelp) { showHelp = false; return; }
      if (showQueue) { showQueue = false; return; }
      if (showNowPlaying) { showNowPlaying = false; return; }
      if (blurEditableTarget(e.target)) {
        e.preventDefault();
        return;
      }
    }

    if (e.key.toLowerCase() === 'f' && !e.ctrlKey && !e.metaKey && !e.altKey && !isEditableTarget(e.target)) {
      e.preventDefault();
      clearPendingVimKey();
      showActionHints();
      return;
    }

    if ((e.key === 'j' || e.key === 'k') && !e.ctrlKey && !e.metaKey && !e.altKey && !isEditableTarget(e.target)) {
      e.preventDefault();
      clearPendingVimKey();
      scrollCurrentView(e.key === 'j' ? 1 : -1);
      return;
    }

    if (e.key === 'g' && !e.ctrlKey && !e.metaKey && !e.altKey && !isEditableTarget(e.target)) {
      e.preventDefault();
      if (pendingVimKey === 'g') {
        clearPendingVimKey();
        scrollCurrentViewTo('top');
      } else {
        pendingVimKey = 'g';
      }
      return;
    }

    if (e.key === 'G' && !e.ctrlKey && !e.metaKey && !e.altKey && !isEditableTarget(e.target)) {
      e.preventDefault();
      clearPendingVimKey();
      scrollCurrentViewTo('bottom');
      return;
    }

    clearPendingVimKey();

    handleKeydown(e);
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="shell">
  {#if showTitlebar}
    <TitleBar />
  {/if}
  <div class="top">
    <div class="sidebar-wrap">
      <Sidebar />
    </div>
    <div class="content-area">
      <Content />
      {#if showNowPlaying}
        <NowPlayingView onclose={() => showNowPlaying = false} />
      {/if}
    </div>
  </div>
  <NowPlayingBar onqueuetoggle={() => showQueue = !showQueue} onexpand={() => showNowPlaying = true} />
</div>

<QueuePanel open={showQueue} onclose={() => showQueue = false} />

{#if showHelp}
  <ShortcutHelp onclose={() => showHelp = false} />
{/if}

{#if hintMode}
  <div class="action-hint-layer" aria-hidden="true">
    {#each actionHints as hint (hint.key)}
      <span class="action-hint" style={`top: ${hint.top}px; left: ${hint.left}px;`}>
        {hint.label}
      </span>
    {/each}
  </div>
{/if}

<Toast />

<style>
  .action-hint-layer {
    position: fixed;
    inset: 0;
    z-index: 10000;
    pointer-events: none;
  }

  .action-hint {
    position: fixed;
    min-width: 1.25rem;
    height: 1.25rem;
    display: inline-grid;
    place-items: center;
    padding: 0 0.35rem;
    border: 1px solid color-mix(in srgb, var(--accent) 55%, var(--text-on-accent));
    border-radius: 4px;
    background: var(--accent);
    color: var(--text-on-accent);
    font-size: 0.75rem;
    font-weight: 700;
    line-height: 1;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.28);
  }
</style>
