export type FighterId =
  | "ryu"
  | "ken"
  | "chun-li"
  | "e-honda"
  | "blanka"
  | "zangief"
  | "guile"
  | "dhalsim"
  | "balrog"
  | "vega"
  | "sagat"
  | "m-bison"
  | "cammy"
  | "t-hawk"
  | "fei-long"
  | "dee-jay"
  | "akuma";

export type GamePhase =
  | "boot"
  | "title"
  | "attract-intro"
  | "attract-fight"
  | "attract-result"
  | "player-intro"
  | "player-fight"
  | "player-result"
  | "paused";

export type Facing = -1 | 1;
export type PlayerIndex = 0 | 1;
export type AttackButton = "light" | "heavy" | "projectile";
export type FighterPose = "idle" | "walk" | "jump" | "attack" | "hit" | "block" | "ko" | "victory";

export type Box = {
  readonly x: number;
  readonly y: number;
  readonly width: number;
  readonly height: number;
};

export type AnimationName = FighterPose;
export type AnimationDefinition = {
  readonly frames: readonly string[];
  readonly ticksPerFrame: number;
  readonly loop: boolean;
};

export type ActiveFrame = {
  readonly hitId: number;
  readonly startTick: number;
  readonly endTick: number;
  readonly hitbox: Box;
};

export type MoveDefinition = {
  readonly id: string;
  readonly input: AttackButton;
  readonly startupTicks: number;
  readonly activeFrames: readonly ActiveFrame[];
  readonly recoveryTicks: number;
  readonly damage: number;
  readonly blockDamage: number;
  readonly hitstunTicks: number;
  readonly blockstunTicks: number;
  readonly hitstopTicks: number;
  readonly multiHit: boolean;
};

export type ProjectileDefinition = {
  readonly moveId: string;
  readonly spawnTick: number;
  readonly offsetX: number;
  readonly offsetY: number;
  readonly velocityX: number;
  readonly hitbox: Box;
  readonly damage: number;
  readonly blockDamage: number;
  readonly hitstunTicks: number;
  readonly blockstunTicks: number;
  readonly hitstopTicks: number;
};

export type VictoryDefinition = {
  readonly animation: AnimationName;
  readonly quote: string;
};

export type AiBias = {
  readonly aggression: number;
  readonly defense: number;
  readonly projectile: number;
  readonly jump: number;
};

export type FighterDefinition = {
  readonly id: FighterId;
  readonly displayName: string;
  readonly maxHealth: number;
  readonly walkSpeed: number;
  readonly jumpVelocity: number;
  readonly gravity: number;
  readonly pushbox: Box;
  readonly hurtboxes: readonly Box[];
  readonly animations: Readonly<Record<AnimationName, AnimationDefinition>>;
  readonly moves: readonly MoveDefinition[];
  readonly projectile: ProjectileDefinition | null;
  readonly victory: VictoryDefinition;
  readonly aiBias: AiBias;
};

export type PlayerInput = {
  readonly left?: boolean;
  readonly right?: boolean;
  readonly up?: boolean;
  readonly down?: boolean;
  readonly light?: boolean;
  readonly heavy?: boolean;
  readonly projectile?: boolean;
  readonly block?: boolean;
};

export type NormalizedPlayerInput = {
  readonly horizontal: -1 | 0 | 1;
  readonly vertical: -1 | 0 | 1;
  readonly light: boolean;
  readonly heavy: boolean;
  readonly projectile: boolean;
  readonly block: boolean;
};

export type GameInput = {
  readonly players?: readonly [PlayerInput, PlayerInput];
  readonly insertCoin?: boolean;
  readonly pause?: boolean;
};

export type NormalizedGameInput = {
  readonly players: readonly [NormalizedPlayerInput, NormalizedPlayerInput];
  readonly insertCoin: boolean;
  readonly pause: boolean;
};

export type FighterSnapshot = {
  readonly id: FighterId;
  readonly x: number;
  readonly y: number;
  readonly velocityX: number;
  readonly velocityY: number;
  readonly facing: Facing;
  readonly health: number;
  readonly pose: FighterPose;
  readonly moveId: string | null;
  readonly moveTick: number;
  readonly hitstunTicks: number;
  readonly blockstunTicks: number;
  readonly hitstopTicks: number;
};

export type ProjectileSnapshot = {
  readonly owner: PlayerIndex;
  readonly x: number;
  readonly y: number;
  readonly velocityX: number;
};

export type RoundResult = "player-1" | "player-2" | "draw" | null;

export type GameSnapshot = {
  readonly tick: number;
  readonly phase: GamePhase;
  readonly phaseTick: number;
  readonly roundTicksRemaining: number;
  readonly rngState: number;
  readonly fighters: readonly [FighterSnapshot, FighterSnapshot];
  readonly projectiles: readonly ProjectileSnapshot[];
  readonly result: RoundResult;
};

export type CaptureState = {
  readonly tick: number;
  readonly phase: GamePhase;
  readonly complete: boolean;
  readonly rngState: number;
};
