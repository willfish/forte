<script lang="ts">
  import type { LucideProps } from '@lucide/svelte';
  import type { IconName } from './icons';
  import { iconComponents } from './icons';

  interface Props extends Omit<LucideProps, 'name' | 'size'> {
    name: IconName;
    size?: number | string;
    filled?: boolean;
  }

  let {
    name,
    size = 16,
    filled = false,
    class: className = '',
    title,
    strokeWidth = 2,
    absoluteStrokeWidth = true,
    ...rest
  }: Props = $props();

  const Component = $derived(iconComponents[name]);
  const classes = $derived(['forte-icon', filled && 'filled', className].filter(Boolean).join(' '));
</script>

<Component
  size={size}
  strokeWidth={strokeWidth}
  absoluteStrokeWidth={absoluteStrokeWidth}
  class={classes}
  title={title}
  aria-hidden={title ? undefined : true}
  aria-label={title}
  focusable="false"
  {...rest}
/>
