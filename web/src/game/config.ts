export const TICKS_PER_SECOND = 60 as const;
export const LOGICAL_COLUMNS = 96 as const;
export const LOGICAL_ROWS = 40 as const;
export const ROUND_TICKS = 5_940 as const;
export const MAX_CATCH_UP_TICKS = 8 as const;
export const MAX_DEVICE_PIXEL_RATIO = 2 as const;
export const PLAYER_INACTIVITY_TICKS = 1_800 as const;
export const RESULT_TICKS = 300 as const;
export const CAPTURE_MAX_TICKS = 7_200 as const;

export const BOOT_TICKS = 1 as const;
export const TITLE_TICKS = 600 as const;
export const ATTRACT_INTRO_TICKS = 180 as const;
export const PLAYER_INTRO_TICKS = 120 as const;

export const WORLD_UNITS_PER_CELL = 100 as const;
export const STAGE_LEFT = 0 as const;
export const STAGE_RIGHT = LOGICAL_COLUMNS * WORLD_UNITS_PER_CELL;
export const GROUND_Y = (LOGICAL_ROWS - 4) * WORLD_UNITS_PER_CELL;
export const DEFAULT_GRAVITY = 80 as const;
export const DEFAULT_JUMP_VELOCITY = -900 as const;
export const DEFAULT_WALK_SPEED = 120 as const;
