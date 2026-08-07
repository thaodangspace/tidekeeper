/** Core domain types shared across repositories, services, and handlers. */

export type VoyageStatus = "ACTIVE" | "COMPLETED" | "FAILED" | "ABANDONED";

export interface Player {
  id: string;
  publicId: string;
  displayName: string;
  username: string;
  onboardingCompleted: boolean;
  locale: string;
  timezone: string;
  currentVoyageId: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Session {
  id: string;
  playerId: string;
  /** Hex-encoded SHA-256 digest of the raw token. */
  tokenDigest: string;
  expiresAt: string;
  lastSeenAt: string;
  rotatedAt: string | null;
  replacedBySessionId: string | null;
  revokedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface VoyageDefinition {
  id: string;
  definitionKey: string;
  version: number;
  status: "DRAFT" | "PUBLISHED";
  durationDays: number;
  startingHull: number;
  startingSupplies: number;
  fleetSlotCount: number;
  initialPhase: string;
  launchPresentation: unknown;
  checksum: string;
}

export interface VoyageDefinitionStarterKeeper {
  keeperDefinitionVersionId: string;
  keeperKey: string;
  name: string;
  currentName: string;
  sectorKey: string;
  roleKey: string;
  rarityKey: string;
  passiveRuleKey: string;
  passiveRuleConfig: string;
  rootUpgradeNodeKey: string;
}

export interface VoyageDefinitionInitialOffer {
  keeperDefinitionVersionId: string;
  keeperKey: string;
  name: string;
  currentName: string;
  sectorKey: string;
  roleKey: string;
  rarityKey: string;
  passiveRuleKey: string;
  passiveRuleConfig: string;
  cost: number;
}

export interface Voyage {
  id: string;
  publicId: string;
  playerId: string;
  status: VoyageStatus;
  definitionKey: string;
  definitionVersion: number;
  currentDayNumber: number;
  fundHealth: number;
  maxFundHealth: number;
  capital: number;
  score: string;
  rowVersion: number;
  startedAt: string;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface VoyageLedgerEntry {
  id: string;
  voyageId: string;
  entryType: string;
  amount: number;
  balanceAfter: number;
  sourceType: string;
  sourceId: string;
  reasonKey: string;
  occurredAt: string;
}

export interface VoyageLifecycleEvent {
  id: string;
  publicId: string;
  voyageId: string;
  eventType: string;
  resultingStatus: VoyageStatus;
  occurredAt: string;
}

export interface KeeperInstance {
  id: string;
  publicId: string;
  voyageId: string;
  definitionKey: string;
  definitionVersion: number;
  name: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  upgradeNodeKey: string;
  nodeDepth: number;
  acquiredDay: number;
  acquiredSource: string;
  acquiredAt: string;
}

export interface PlayerKeeperUnlock {
  playerId: string;
  keeperKey: string;
  definitionVersion: number;
  name: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  unlockSource: string;
  unlockedAt: string;
}

export interface DailyTide {
  id: string;
  dayKey: string;
  sequenceNumber: number;
  phase: string;
  lockAt: string;
  settleAfter: string;
  contentVersion: number;
  /** Exact schema-2 content references; absent on legacy KV records. */
  modifierDefinitionVersionId?: string | null;
  objectiveDefinitionVersionId?: string | null;
  gameRuleSetVersionId?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface PlayerDailyState {
  id: string;
  playerId: string;
  voyageId: string;
  dailyTideId: string;
  dayNumber: number;
  phase: string;
  version: number;
  selectedStrategyId: string | null;
  pendingRewardCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface PlayerDailyContextView {
  voyageId: string;
  dayNumber: number;
  schemaVersion: number;
  projection: unknown;
  validatedAt: string;
}

export interface SettlementRecord {
  id: string;
  voyageId: string;
  dayNumber: number;
  ruleVersion: number;
  scoreBps: number;
  score: string;
  hullDelta: number;
  suppliesDelta: number;
  reward: {
    type: "SUPPLIES";
    amount: number;
    reason: string;
  } | null;
  breakdown: unknown;
  settledAt: string;
  claimedAt: string | null;
}

export interface IdempotencyKeyRecord {
  playerId: string;
  scope: string;
  key: string;
  state: "IN_PROGRESS" | "COMPLETED";
  requestHash: string;
  resultVoyageId: string | null;
  responseStatus: number | null;
  responseJson: string | null;
  createdAt: string;
  completedAt: string | null;
}

/** Authenticated principal attached to a request context. */
export interface Principal {
  playerId: string;
}
