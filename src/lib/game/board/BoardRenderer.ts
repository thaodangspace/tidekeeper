import {
  Application,
  Assets,
  Container,
  Graphics,
  Sprite,
  Text,
  type FederatedPointerEvent
} from 'pixi.js';
import type {
  BoardKeeper,
  BoardRendererCallbacks,
  BoardSynergy,
  BoardTideState
} from '$lib/game/state/board-types';

interface BoardState {
  keepers: BoardKeeper[];
  maxSlots: number;
  selectedKeeperId: string | null;
  locked: boolean;
  activeSynergies: BoardSynergy[];
  tideState?: BoardTideState;
  reducedMotion: boolean;
}

interface SlotLayout {
  x: number;
  y: number;
  width: number;
  height: number;
  centerX: number;
  centerY: number;
}

interface DragState {
  keeperId: string;
  sourceSlot: number;
  startX: number;
  startY: number;
  moved: boolean;
  object: Container;
}

const sectorColors: Record<string, number> = {
  CREST: 0x8fc8c8,
  HARBOR: 0x78a9aa,
  ECHO: 0xa6bfe9,
  CURRENT: 0x40d9d1,
  EMBER: 0xf25543,
  BLOOM: 0x90c96b,
  UNKNOWN: 0xb7894e
};

const sectorSigils: Record<string, string> = {
  CREST: '♜',
  HARBOR: '⚓',
  ECHO: '◉',
  CURRENT: '≋',
  EMBER: '♨',
  BLOOM: '✦'
};

/**
 * Pixi is deliberately kept behind this small projection renderer. It knows
 * how to draw and hit-test the board, but never owns a Fleet or calls an API.
 */
export class BoardRenderer {
  private app: Application | null = null;
  private readonly callbacks: BoardRendererCallbacks;
  private state: BoardState = {
    keepers: [],
    maxSlots: 5,
    selectedKeeperId: null,
    locked: false,
    activeSynergies: [],
    reducedMotion: false
  };
  private readonly root = new Container();
  private readonly synergyLayer = new Graphics();
  private readonly ambientLayer = new Graphics();
  private readonly slotLayer = new Container();
  private readonly keeperLayer = new Container();
  private readonly layouts = new Map<number, SlotLayout>();
  private readonly keeperObjects = new Map<string, Container>();
  private drag: DragState | null = null;
  private phase = 0;
  private renderToken = 0;
  private destroyed = false;

  constructor(callbacks: BoardRendererCallbacks) {
    this.callbacks = callbacks;
    this.root.addChild(this.ambientLayer, this.synergyLayer, this.slotLayer, this.keeperLayer);
  }

  async init(container: HTMLElement, reducedMotion: boolean): Promise<HTMLCanvasElement> {
    if (this.app) return this.app.canvas;

    const app = new Application();
    await app.init({
      resizeTo: container,
      backgroundAlpha: 0,
      antialias: true,
      autoDensity: true,
      preference: 'webgl'
    });

    if (this.destroyed) {
      app.destroy(true);
      throw new Error('Fleet board was destroyed during initialization.');
    }

    this.app = app;
    this.state.reducedMotion = reducedMotion;
    app.stage.addChild(this.root);
    app.stage.eventMode = 'static';
    app.stage.on('pointermove', this.handlePointerMove, this);
    app.stage.on('pointerup', this.handlePointerUp, this);
    app.stage.on('pointerupoutside', this.handlePointerUp, this);
    app.ticker.add(this.animate, this);
    if (reducedMotion) app.ticker.stop();
    this.resize(container.clientWidth, container.clientHeight);
    return app.canvas;
  }

  setState(state: Omit<BoardState, 'reducedMotion'> & { reducedMotion: boolean }): void {
    this.state = { ...state };
    if (this.app) this.renderScene();
  }

  setReducedMotion(reducedMotion: boolean): void {
    this.state.reducedMotion = reducedMotion;
    if (this.app) {
      if (reducedMotion) this.app.ticker.stop();
      else this.app.ticker.start();
    }
  }

  setVisibility(hidden: boolean): void {
    if (!this.app) return;
    // Keep the board alive while the route is mounted, but stop its ticker
    // while the tab is hidden. This avoids a background animation loop.
    if (hidden) this.app.ticker.stop();
    else if (!this.state.reducedMotion) this.app.ticker.start();
  }

  resize(width: number, height: number): void {
    if (!this.app || width <= 0 || height <= 0) return;
    this.app.renderer.resize(width, height);
    this.app.stage.hitArea = this.app.screen;
    this.root.position.set(0, 0);
    this.renderScene();
  }

  destroy(): void {
    this.destroyed = true;
    this.renderToken += 1;
    this.drag = null;
    this.layouts.clear();
    this.keeperObjects.clear();
    if (!this.app) return;

    this.app.stage.off('pointermove', this.handlePointerMove, this);
    this.app.stage.off('pointerup', this.handlePointerUp, this);
    this.app.stage.off('pointerupoutside', this.handlePointerUp, this);
    this.app.ticker.remove(this.animate, this);
    this.app.destroy(true);
    this.app = null;
  }

  private renderScene(): void {
    if (!this.app) return;
    const width = this.app.screen.width;
    const height = this.app.screen.height;
    const token = ++this.renderToken;
    this.layouts.clear();
    this.keeperObjects.clear();
    this.slotLayer.removeChildren().forEach((child) => child.destroy({ children: true }));
    this.keeperLayer.removeChildren().forEach((child) => child.destroy({ children: true }));

    this.drawBackground(width, height);
    const layout = this.calculateLayout(width, height);
    for (let slotIndex = 0; slotIndex < this.state.maxSlots; slotIndex += 1) {
      const slot = layout[slotIndex];
      if (!slot) continue;
      this.layouts.set(slotIndex, slot);
      this.drawSlot(slotIndex, slot);
    }

    this.drawSynergies();
    for (const keeper of this.state.keepers) {
      const slot = this.layouts.get(keeper.slotIndex);
      if (slot) void this.drawKeeper(keeper, slot, token);
    }
  }

  private calculateLayout(width: number, height: number): SlotLayout[] {
    const count = Math.max(1, this.state.maxSlots);
    const horizontalPadding = Math.min(28, width * 0.06);
    const gap = Math.min(16, Math.max(5, width * 0.018));
    const slotWidth = Math.max(54, (width - horizontalPadding * 2 - gap * (count - 1)) / count);
    const slotHeight = Math.min(230, Math.max(138, height * 0.63));
    const totalWidth = slotWidth * count + gap * (count - 1);
    const startX = (width - totalWidth) / 2;
    const startY = Math.max(48, (height - slotHeight) / 2 + 8);

    return Array.from({ length: count }, (_, index) => {
      const x = startX + index * (slotWidth + gap);
      return {
        x,
        y: startY,
        width: slotWidth,
        height: slotHeight,
        centerX: x + slotWidth / 2,
        centerY: startY + slotHeight / 2
      };
    });
  }

  private drawBackground(width: number, height: number): void {
    this.ambientLayer.clear();
    const tideColor = this.state.tideState === 'SHOCK' ? 0x321d27 : this.state.tideState === 'GROWTH' ? 0x092a2a : 0x041923;
    this.ambientLayer.rect(0, 0, width, height).fill({ color: tideColor, alpha: 0.97 });
    this.ambientLayer.rect(10, 10, Math.max(0, width - 20), Math.max(0, height - 20)).stroke({
      width: 1,
      color: 0x70532f,
      alpha: 0.6
    });
    this.ambientLayer.moveTo(0, height * 0.72).lineTo(width, height * 0.58).stroke({
      width: 1,
      color: 0x1d7e83,
      alpha: 0.18
    });
    this.ambientLayer.moveTo(0, height * 0.82).lineTo(width, height * 0.68).stroke({
      width: 1,
      color: 0x1d7e83,
      alpha: 0.13
    });
  }

  private drawSlot(slotIndex: number, slot: SlotLayout): void {
    const keeper = this.state.keepers.find((candidate) => candidate.slotIndex === slotIndex);
    const occupied = Boolean(keeper);
    const selected = keeper?.id === this.state.selectedKeeperId;
    const graphics = new Graphics();
    const color = occupied ? sectorColors[keeper?.sector.toUpperCase() ?? 'UNKNOWN'] : 0xb7894e;
    const fillAlpha = this.state.locked ? 0.28 : occupied ? 0.52 : 0.2;

    graphics.roundRect(slot.x, slot.y, slot.width, slot.height, 8).fill({ color: 0x06131c, alpha: fillAlpha });
    graphics.roundRect(slot.x, slot.y, slot.width, slot.height, 8).stroke({
      width: selected ? 3 : 1,
      color: selected ? 0x40d9d1 : color,
      alpha: this.state.locked ? 0.62 : occupied ? 0.82 : 0.7
    });

    if (!occupied) {
      graphics.circle(slot.centerX, slot.centerY - 8, Math.min(25, slot.width * 0.18)).stroke({
        width: 1,
        color: 0xb7894e,
        alpha: 0.75
      });
      graphics.moveTo(slot.centerX - 10, slot.centerY - 8).lineTo(slot.centerX + 10, slot.centerY - 8).stroke({
        width: 1,
        color: 0xb7894e,
        alpha: 0.75
      });
      graphics.moveTo(slot.centerX, slot.centerY - 18).lineTo(slot.centerX, slot.centerY + 2).stroke({
        width: 1,
        color: 0xb7894e,
        alpha: 0.75
      });
      const text = new Text({
        text: this.state.locked ? 'KHÓA' : 'TRỐNG',
        style: { fill: 0xaa9e8c, fontSize: Math.max(11, Math.min(15, slot.width / 8)), fontFamily: 'Cormorant Garamond' }
      });
      text.anchor.set(0.5);
      text.position.set(slot.centerX, slot.centerY + 27);
      graphics.addChild(text);
      graphics.eventMode = this.state.locked ? 'none' : 'static';
      graphics.cursor = 'pointer';
      graphics.on('pointertap', () => this.callbacks.onEmptySlotSelect(slotIndex));
    } else if (this.state.locked) {
      const lock = new Text({ text: '◆', style: { fill: 0xb7894e, fontSize: 12 } });
      lock.anchor.set(0.5);
      lock.position.set(slot.x + slot.width - 14, slot.y + 14);
      graphics.addChild(lock);
    }

    this.slotLayer.addChild(graphics);
  }

  private drawSynergies(): void {
    this.synergyLayer.clear();
    for (const synergy of this.state.activeSynergies) {
      const points = synergy.keeperIds
        .map((keeperId) => this.state.keepers.find((keeper) => keeper.id === keeperId))
        .map((keeper) => (keeper ? this.layouts.get(keeper.slotIndex) : undefined))
        .filter((slot): slot is SlotLayout => Boolean(slot));
      if (points.length < 2) continue;

      for (let index = 1; index < points.length; index += 1) {
        const from = points[index - 1];
        const to = points[index];
        this.synergyLayer.moveTo(from.centerX, from.centerY).lineTo(to.centerX, to.centerY).stroke({
          width: 2,
          color: 0x40d9d1,
          alpha: 0.46
        });
      }
    }
  }

  private async drawKeeper(keeper: BoardKeeper, slot: SlotLayout, token: number): Promise<void> {
    const object = new Container();
    object.position.set(slot.x, slot.y);
    object.eventMode = 'static';
    object.cursor = this.state.locked ? 'pointer' : 'grab';
    object.on('pointerdown', (event: FederatedPointerEvent) => this.handlePointerDown(event, keeper, object));
    object.on('pointerup', this.handlePointerUp, this);
    object.on('pointerupoutside', this.handlePointerUp, this);

    const color = sectorColors[keeper.sector.toUpperCase()] ?? sectorColors.UNKNOWN;
    const selected = keeper.id === this.state.selectedKeeperId;
    const card = new Graphics();
    card.roundRect(0, 0, slot.width, slot.height, 8).fill({ color: 0x071721, alpha: 0.94 });
    card.roundRect(0, 0, slot.width, slot.height, 8).stroke({
      width: selected ? 3 : 1,
      color: selected ? 0x40d9d1 : color,
      alpha: 0.94
    });
    card.roundRect(5, 5, Math.max(0, slot.width - 10), Math.max(0, slot.height * 0.58), 5).fill({
      color,
      alpha: 0.12
    });
    object.addChild(card);

    const sigil = new Text({
      text: sectorSigils[keeper.sector.toUpperCase()] ?? '✦',
      style: { fill: color, fontSize: Math.max(16, Math.min(28, slot.width * 0.22)), fontFamily: 'Cinzel' }
    });
    sigil.anchor.set(0.5);
    sigil.position.set(slot.width / 2, slot.height * 0.28);
    object.addChild(sigil);

    const name = new Text({
      text: keeper.name,
      style: {
        fill: 0xece4d1,
        fontSize: Math.max(12, Math.min(18, slot.width / 7)),
        fontFamily: 'Cinzel',
        align: 'center',
        wordWrap: true,
        wordWrapWidth: Math.max(40, slot.width - 12)
      }
    });
    name.anchor.set(0.5, 0);
    name.position.set(slot.width / 2, slot.height * 0.66);
    object.addChild(name);

    const role = new Text({
      text: keeper.role,
      style: { fill: color, fontSize: Math.max(11, Math.min(15, slot.width / 9)), fontFamily: 'Cormorant Garamond', align: 'center' }
    });
    role.anchor.set(0.5, 0);
    role.position.set(slot.width / 2, slot.height * 0.86);
    object.addChild(role);

    this.keeperObjects.set(keeper.id, object);
    this.keeperLayer.addChild(object);
    if (keeper.image) {
      try {
        const texture = await Assets.load(keeper.image);
        if (this.destroyed || token !== this.renderToken || !object.parent) return;
        const sprite = new Sprite(texture);
        sprite.width = Math.max(0, slot.width - 10);
        sprite.height = Math.max(0, slot.height * 0.56);
        sprite.position.set(5, 5);
        sprite.alpha = 0.48;
        sprite.tint = color;
        object.addChildAt(sprite, 1);
      } catch {
        // The sigil/card remains a useful board representation when artwork is
        // unavailable (for example while an asset is being deployed).
      }
    }
  }

  private handlePointerDown(event: FederatedPointerEvent, keeper: BoardKeeper, object: Container): void {
    this.drag = {
      keeperId: keeper.id,
      sourceSlot: keeper.slotIndex,
      startX: event.global.x,
      startY: event.global.y,
      moved: false,
      object
    };
  }

  private handlePointerMove(event: FederatedPointerEvent): void {
    if (!this.drag || this.state.locked) return;
    const distance = Math.hypot(event.global.x - this.drag.startX, event.global.y - this.drag.startY);
    if (distance > 6) {
      this.drag.moved = true;
      this.drag.object.alpha = 0.72;
      this.drag.object.position.set(event.global.x - this.drag.object.width / 2, event.global.y - this.drag.object.height / 2);
      this.highlightTarget(this.findSlot(event.global.x, event.global.y));
    }
  }

  private handlePointerUp(event: FederatedPointerEvent): void {
    const drag = this.drag;
    if (!drag) return;
    this.drag = null;
    drag.object.alpha = 1;
    if (!drag.moved) {
      this.callbacks.onKeeperSelect(drag.keeperId);
      return;
    }

    const target = this.findSlot(event.global.x, event.global.y);
    this.clearTargetHighlight();
    if (target !== undefined && target !== drag.sourceSlot && !this.state.locked) {
      this.callbacks.onKeeperMove(drag.keeperId, target);
    } else {
      this.renderScene();
    }
  }

  private findSlot(x: number, y: number): number | undefined {
    for (const [slotIndex, slot] of this.layouts) {
      if (x >= slot.x && x <= slot.x + slot.width && y >= slot.y && y <= slot.y + slot.height) return slotIndex;
    }
    return undefined;
  }

  private highlightTarget(target: number | undefined): void {
    this.clearTargetHighlight();
    if (target === undefined || this.state.locked || target === this.drag?.sourceSlot) return;
    const slot = this.layouts.get(target);
    if (!slot) return;
    this.synergyLayer.roundRect(slot.x, slot.y, slot.width, slot.height, 8).stroke({ width: 4, color: 0x90c96b, alpha: 0.88 });
  }

  private clearTargetHighlight(): void {
    this.drawSynergies();
  }

  private animate(ticker: { deltaTime: number }): void {
    if (this.state.reducedMotion || !this.app) return;
    this.phase += ticker.deltaTime * 0.018;
    const width = this.app.screen.width;
    const height = this.app.screen.height;
    // Redraw the small ambient layer rather than appending a new path every
    // tick, keeping the animation bounded on long preparation sessions.
    this.drawBackground(width, height);
    this.ambientLayer
      .moveTo(width * 0.1, height * (0.71 + Math.sin(this.phase) * 0.012))
      .quadraticCurveTo(width * 0.48, height * (0.64 + Math.sin(this.phase + 1) * 0.018), width * 0.9, height * (0.69 + Math.sin(this.phase + 2) * 0.012))
      .stroke({ width: 2, color: 0x40d9d1, alpha: 0.17 });
  }
}
