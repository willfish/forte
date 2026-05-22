<script lang="ts">
  interface Props {
    size?: number;
    class?: string;
    title?: string;
  }

  let { size = 20, class: className = '', title }: Props = $props();
</script>

<svg
  viewBox="0 0 512 512"
  width={size}
  height={size}
  class={className}
  aria-hidden={title ? undefined : true}
  aria-label={title}
  focusable="false"
>
  {#if title}
    <title>{title}</title>
  {/if}
  <rect width="512" height="512" rx="96" fill="#1b2636" />
  <text
    x="256"
    y="365"
    text-anchor="middle"
    font-family="Georgia, 'Times New Roman', serif"
    font-size="360"
    font-style="italic"
    font-weight="700"
    fill="#e0e0e0"
  >f</text>
  <path d="M166 244h198M148 294h198" stroke="#3366aa" stroke-width="22" stroke-linecap="round" opacity="0.72" />
</svg>
