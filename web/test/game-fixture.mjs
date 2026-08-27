export const FIGHTER_IDS = [
  "ryu",
  "ken",
  "chun-li",
  "e-honda",
  "blanka",
  "zangief",
  "guile",
  "dhalsim",
  "balrog",
  "vega",
  "sagat",
  "m-bison",
  "cammy",
  "t-hawk",
  "fei-long",
  "dee-jay",
  "akuma",
];

const animation = { frames: ["frame-1"], ticksPerFrame: 1, loop: true };
const hitbox = { x: 200, y: -800, width: 500, height: 500 };

export function makeFighter(id, options = {}) {
  const damage = options.damage ?? 20;
  const activeFrames = options.activeFrames ?? [{ hitId: 0, startTick: 2, endTick: 4, hitbox }];
  const move = (moveId, input) => ({
    id: moveId,
    input,
    startupTicks: 2,
    activeFrames,
    recoveryTicks: 2,
    damage,
    blockDamage: 2,
    hitstunTicks: 8,
    blockstunTicks: 5,
    hitstopTicks: 3,
    multiHit: options.multiHit ?? false,
  });
  return {
    id,
    displayName: id.toUpperCase(),
    maxHealth: 100,
    walkSpeed: 120,
    jumpVelocity: -900,
    gravity: 80,
    pushbox: { x: 0, y: -1_000, width: 600, height: 1_000 },
    hurtboxes: [{ x: -300, y: -1_000, width: 600, height: 1_000 }],
    animations: {
      idle: animation,
      walk: animation,
      jump: animation,
      attack: animation,
      hit: animation,
      block: animation,
      ko: animation,
      victory: animation,
    },
    moves: [move("jab", "light"), move("roundhouse", "heavy"), move("fireball", "projectile")],
    projectile: {
      moveId: "fireball",
      spawnTick: 1,
      offsetX: 600,
      offsetY: -600,
      velocityX: 200,
      hitbox: { x: 0, y: -200, width: 300, height: 300 },
      damage: 12,
      blockDamage: 1,
      hitstunTicks: 6,
      blockstunTicks: 4,
      hitstopTicks: 2,
    },
    victory: { animation: "victory", quote: `${id} wins` },
    aiBias: options.aiBias ?? { aggression: 0, defense: 0, projectile: 0, jump: 0 },
  };
}

export function makeCoreOptions(overrides = {}) {
  return {
    seed: overrides.seed ?? 123_456,
    fighters: [
      makeFighter("ryu", overrides.first ?? {}),
      makeFighter("ken", overrides.second ?? {}),
    ],
  };
}

export function enterPlayerFight(core) {
  core.advanceTicks(1);
  core.insertCoin();
  core.advanceTicks(120);
}

export function closeFighters(core) {
  core.advanceTicks(14, { players: [{ right: true }, { left: true }] });
}
