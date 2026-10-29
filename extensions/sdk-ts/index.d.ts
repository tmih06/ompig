import "@earendil-works/pi-coding-agent";
import "@earendil-works/pi-tui";

export type * from "@earendil-works/pi-coding-agent";

/**
 * Data rendered by Ompig's fixed native login template.
 *
 * The host validates grid dimensions, palette symbols, colors, and display
 * widths before replacing the current header.
 */
export interface OmpigLoginDefinition {
  brand: string[];
  hero: string[];
  mascot: string[];
  palette: Record<string, string>;
  name: string;
  description: string;
  tagline: string;
}

/**
 * One sprite for Ompig's /sprite catalogue: the 16-by-14 ompig the startup header
 * draws and /sprite preview shows beside the wordmark, colored by the palette. The host validates the grids, palette symbols, colors and text.
 */
export interface OmpigSpriteDefinition {
  id: string;
  name: string;
  tagline: string;
  mascot: string[];
  palette: Record<string, string>;
}

/** A Pi theme token a surface may override (Ompig component kit, D107 §6). */
export type OmpigViewThemeToken =
  | "accent"
  | "muted"
  | "dim"
  | "text"
  | "border"
  | "borderAccent"
  | "borderMuted"
  | "success"
  | "warning"
  | "error"
  | "selectedBg"
  | "customMessageBg";

/**
 * Token overrides for one surface, each a `#rrggbb` color. Ompig renders the
 * surface with them in every mode, escaped in the theme's color mode; a faint
 * token stays faint. An unknown token or a malformed color is applied
 * nowhere, and a frontend rejects the surface's view.
 */
export type OmpigViewTheme = Partial<Record<OmpigViewThemeToken, `#${string}`>>;

/** One row of a {@link OmpigViewList}: its primary text, a secondary text, and further table cells. */
export interface OmpigViewListItem {
  label: string;
  detail?: string;
  columns?: string[];
}

/** Says a component's rows are a list: `items[i]` is row `i`. `selectedIndex` is the highlighted item, or -1. */
export interface OmpigViewList {
  items: OmpigViewListItem[];
  selectedIndex: number;
}

/**
 * What a component's rows show, for a frontend drawing them natively. Ompig
 * sends it only while a frontend draws, and only for a component it cannot
 * read structure from (a custom `render`). The terminal always draws the rows.
 */
export interface OmpigViewLines {
  /** The rows depict this image (base64 data). */
  image?: { data: string; mimeType: string };
  /** The rows show a position in a range. */
  progress?: { value: number; max: number };
  /** The rows are a list, one item per row; a list whose item count differs from the rows is rejected. */
  list?: OmpigViewList;
}

/** The provenance the loader stamped on an extension (Pi's `SourceInfo`). */
export interface OmpigExtensionSourceInfo {
  path: string;
  source: string;
  scope: string;
  origin: string;
  baseDir?: string;
}

declare module "@earendil-works/pi-coding-agent" {
  interface ExtensionAPI {
    /**
     * Run `handler(ctx)` once the runtime registered and has its state, before
     * it serves a request (`session_start` included) and after the emits its
     * factory made were delivered. Pi has no such hook.
     */
    onLoaded(handler: (ctx: ExtensionContext) => void | Promise<void>): void;
  }
  interface ExtensionContext {
    /** The provenance the loader stamped on this extension; undefined when the host holds none. */
    extensionSourceInfo(): OmpigExtensionSourceInfo | undefined;
    /**
     * In a tool's `execute`: the call's position among the tool calls of the
     * assistant message that issued it; undefined for a nested `executeTool` call.
     */
    readonly toolCallIndex?: number;
    /** In a tool's `execute`: `JSON.stringify` of the call's arguments, members in the model's order. */
    readonly toolInput?: string;
  }
  interface ExtensionUIContext {
    /** Subscribe to changes of the host's theme; the handler receives `ui.theme` after it took the new palette. Returns the unsubscribe function. */
    onThemeChange(handler: (theme: Theme) => void): () => void;
    /** Subscribe to the host toggling tool-output expansion; the handler receives the new state. Returns the unsubscribe function. */
    onToolsExpandedChange(handler: (expanded: boolean) => void): () => void;
    /** Ring the terminal bell of the interactive terminal; nothing without a terminal UI. */
    bell(): void;
    /** Set Ompig's native login template, or reject an invalid definition. */
    setLogin(definition: OmpigLoginDefinition): Promise<void>;
    /**
     * Add a sprite to Ompig's /sprite catalogue, or reject an invalid definition.
     * Registering the same ID again replaces this extension's sprite; the sprite
     * leaves /sprite when the extension unloads.
     */
    registerSprite(definition: OmpigSpriteDefinition): Promise<void>;
  }
}

declare module "@earendil-works/pi-tui" {
  interface Component {
    /** On a surface's root component: token overrides for that surface (Ompig only). */
    viewTheme?: OmpigViewTheme;
    /** Frontend-only annotations of this component's rows (Ompig only). */
    viewLines?: OmpigViewLines;
  }
}
