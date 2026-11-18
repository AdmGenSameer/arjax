# stick-it-up

> Premium vector, metallic chrome, and pixel art aesthetic sticker components for React & Next.js.

A collection of vector graphic stickers featuring specular metallic chrome reflections, retro halftone stars, pixel art mascots (Nori, Pip, Kuro, Mochi), cute character doodles, and tactile badges — designed and created by Samiran De. Built with scoped SVG filters, responsive sizing, and realistic 3D vinyl peel hover physics.

---

## 📦 Installation

```bash
# npm (recommended: @latest)
npm install stick-it-up@latest

# pnpm
pnpm add stick-it-up@latest

# bun
bun add stick-it-up@latest

# yarn
yarn add stick-it-up@latest
```

---

## 🚀 Quick Start

### 1. Direct Imports (Named & Lightweight)

Every sticker is tree-shakeable and imports directly without pulling in unnecessary bundle weight:

```tsx
import {
  FlameSticker,
  MetalSticker,
  NoriCoffeeSticker,
  PipRocketSticker,
  BlobBurstSticker,
  StarBurstSticker,
} from "stick-it-up";

export function App() {
  return (
    <div style={{ display: "flex", gap: "24px", alignItems: "center" }}>
      {/* Basic usage with size in pixels */}
      <FlameSticker size={100} />

      {/* Interactive vinyl peel hover effect */}
      <MetalSticker size={120} peel />

      {/* Pixel mascot with peel */}
      <NoriCoffeeSticker size={96} peel />

      {/* Custom width and height overrides */}
      <StarBurstSticker width={140} height={120} className="my-sticker" />
    </div>
  );
}
```

### 2. Dynamic Component (`<Sticker name="..." />`)

```tsx
import { Sticker, type StickerName } from "stick-it-up";

export function StickerGallery() {
  const stickers: StickerName[] = [
    "metal",
    "flame",
    "star-burst",
    "nori-coffee",
    "pip-rocket",
    "kuro-cool",
    "mochi-laptop",
  ];

  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: "16px" }}>
      {stickers.map((name) => (
        <Sticker key={name} name={name} size={90} peel />
      ))}
    </div>
  );
}
```

---

## 🎨 Available Stickers

### 👾 Pixel Mascots (24 stickers)
- **Nori**: `<NoriCoffeeSticker />`, `<NoriCoolSticker />`, `<NoriHeadphonesSticker />`, `<NoriHeartSticker />`, `<NoriLaptopSticker />`, `<NoriRocketSticker />`
- **Pip**: `<PipCoffeeSticker />`, `<PipDetectiveSticker />`, `<PipHeadphonesSticker />`, `<PipRocketSticker />`, `<PipThinkingSticker />`, `<PipWizardSticker />`
- **Kuro**: `<KuroCoffeeSticker />`, `<KuroCoolSticker />`, `<KuroHeadphonesSticker />`, `<KuroHeartSticker />`, `<KuroLaptopSticker />`, `<KuroRocketSticker />`
- **Mochi**: `<MochiCoffeeSticker />`, `<MochiCoolSticker />`, `<MochiHeadphonesSticker />`, `<MochiHeartSticker />`, `<MochiLaptopSticker />`, `<MochiRocketSticker />`

### 🎭 Characters (17 stickers)
- `bad-cat` (`<BadCatSticker />`)
- `black-cat` (`<BlackCatSticker />`)
- `black-face` (`<BlackFaceSticker />`)
- `bunny` (`<BunnySticker />`)
- `cat` (`<CatSticker />`)
- `cat-cute` (`<CatCuteSticker />`)
- `cat-serious` (`<CatSeriousSticker />`)
- `cat-sit` (`<CatSitSticker />`)
- `curious-globe` (`<CuriousGlobeSticker />`)
- `dog` (`<DogSticker />`)
- `face` (`<FaceSticker />`)
- `fish` (`<FishSticker />`)
- `luffy-hat` (`<LuffyHatSticker />`)
- `pink-cat` (`<PinkCatSticker />`)
- `pink-cat-stamp` (`<PinkCatStampSticker />`)
- `red-cat` (`<RedCatSticker />`)
- `yellow-cat` (`<YellowCatSticker />`)

### 🚀 Tech Icons & Founders (9 stickers)
- `elon` / `elon-1` (`<ElonSticker />`)
- `elon-visionary` / `elon-2` (`<ElonVisionarySticker />`)
- `steve-jobs` / `steve-jobs-1` (`<SteveJobsSticker />`)
- `steve-jobs-classic` / `steve-jobs-2` (`<SteveJobsClassicSticker />`)
- `jensen-huang` (`<JensenHuangSticker />`)
- `mark-zuckerberg` (`<MarkZuckerbergSticker />`)
- `mark-socialnetwork` (`<MarkSocialNetworkSticker />`)
- `erlich-bachman` (`<ErlichBachmanSticker />`)
- `socialnetwork-badge` (`<SocialNetworkBadgeSticker />`)

### ⚡ Chrome & Aesthetic Shapes (9 stickers)
- `flame` (`<FlameSticker />`)
- `metal` (`<MetalSticker />`)
- `flame-star` (`<FlameStarSticker />`)
- `blob-burst` (`<BlobBurstSticker />`)
- `star-burst` (`<StarBurstSticker />`)
- `devil-cone` (`<DevilConeSticker />`)
- `folded-paper` (`<FoldedPaperSticker />`)
- `icy-fleur` (`<IcyFleurSticker />`)
- `goo` (`<GooSticker />`)

### 💎 Symbols & Emblems (15 stickers)
- `app-badge` (`<AppBadgeSticker />`)
- `badge-smiley` (`<BadgeSmileySticker />`)
- `berries` (`<BerriesSticker />`)
- `blueberry` (`<BlueberrySticker />`)
- `cherry` (`<CherrySticker />`)
- `coffee` (`<CoffeeSticker />`)
- `cone` (`<ConeSticker />`)
- `cosmic-eye` (`<CosmicEyeSticker />`)
- `cosmic-pill` (`<CosmicPillSticker />`)
- `eye-splat` (`<EyeSplatSticker />`)
- `flower` (`<FlowerSticker />`)
- `heart` (`<HeartSticker />`)
- `japan-badge` (`<JapanBadgeSticker />`)
- `japan-flag` (`<JapanFlagSticker />`)
- `nazar` (`<NazarSticker />`)
- `peace-hand` (`<PeaceHandSticker />`)
- `red-berry` (`<RedBerrySticker />`)
- `sparkles` (`<SparklesSticker />`)
- `sparkle-trio` (`<SparkleTrioSticker />`)
- `spiral` (`<SpiralSticker />`)

---

## ⚙️ Props Reference

All sticker components support the following props:

```ts
interface StickerProps extends React.SVGProps<SVGSVGElement> {
  /** Size in pixels (sets both width and height equally). Default: 100 */
  size?: number | string;
  /** Explicit width overriding `size` */
  width?: number | string;
  /** Explicit height overriding `size` */
  height?: number | string;
  /** Adds a 3D vinyl peel animation and shadow on hover */
  peel?: boolean;
  /** Additional CSS class names */
  className?: string;
  /** Inline style overrides */
  style?: React.CSSProperties;
}
```

---

## 🛠️ Features

- ⚡ **Zero external runtime CSS dependencies** (Pure SVG, inline CSS transforms & transitions)
- 🔒 **Unique scoped SVG filter IDs** via `useId()` preventing gradient/filter collisions across instances
- 🌲 **Tree-shakeable ESM & CommonJS** dual output with comprehensive TypeScript types (`.d.ts`)
- 🪶 **Ultra lightweight** bundle footprint

---

## 🌟 Sponsors

- **[PureCanvas](https://www.purecanvas.app/)** — Free, open-source, local-first vector design web application that runs entirely inside the browser.
- **[Clonvo](https://clonvo.com/)** — OpenAI-compatible API for 100+ AI models from OpenAI, Anthropic, Google etc plus web, company and people tools for agents. You point the OpenAI SDK to Clonvo API, switch models by changing the model id, and use the same key for those tools.
- **[React Bits](https://reactbits.dev/)** — An open-source collection of animated, interactive & fully customizable React components for building standout web products and experiences.

---

## 📄 License & Design

This package is licensed under the **MIT License**.

All stickers, components, animations, and designs are created and crafted by **[Samiran De](https://github.com/Sam721166)** (drawing creative inspiration from visual moodboards on Pinterest, but handcrafted, vectorized, and coded from scratch by me). Free for both personal and commercial projects.

See [LICENSE](LICENSE) for the complete legal text.

Copyright (c) 2026 [Samiran De](https://github.com/Sam721166).
