import { FighterContractError, validateFighterDefinition } from "../../fighter-registry.ts";
import type { FighterDefinition, FighterId } from "../../types.ts";
import type { FighterContent } from "../../render/glyph-sprite.ts";
import { ryu } from "./ryu.ts";
import { ken } from "./ken.ts";
import { chunLi } from "./chun-li.ts";
import { eHonda } from "./e-honda.ts";
import { blanka } from "./blanka.ts";
import { zangief } from "./zangief.ts";
import { guile } from "./guile.ts";
import { dhalsim } from "./dhalsim.ts";
import { balrog } from "./balrog.ts";
import { vega } from "./vega.ts";
import { sagat } from "./sagat.ts";
import { mBison } from "./m-bison.ts";
import { cammy } from "./cammy.ts";
import { tHawk } from "./t-hawk.ts";
import { feiLong } from "./fei-long.ts";
import { deeJay } from "./dee-jay.ts";
import { akuma } from "./akuma.ts";

export const FIGHTER_ROSTER = [
  ryu,
  ken,
  chunLi,
  eHonda,
  blanka,
  zangief,
  guile,
  dhalsim,
  balrog,
  vega,
  sagat,
  mBison,
  cammy,
  tHawk,
  feiLong,
  deeJay,
  akuma,
] as const satisfies readonly FighterContent[];

export function validateRosterFighterDefinition(expectedId: FighterId, value: unknown): FighterDefinition {
  try {
    const definition = validateFighterDefinition(value);
    if (definition.id !== expectedId) {
      throw new FighterContractError("id", `must equal roster fighter ${expectedId}`);
    }
    return definition;
  } catch (error) {
    if (error instanceof FighterContractError) {
      const detail = error.message.slice(error.field.length + 2);
      throw new FighterContractError(`${expectedId}.${error.field}`, detail);
    }
    throw error;
  }
}
