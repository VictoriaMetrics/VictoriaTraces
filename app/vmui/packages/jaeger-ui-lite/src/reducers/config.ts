import getConfig from '../utils/config/get-config';
import { Config } from '../types/config';

export default function reduceConfig(state?: Config): Config {
  if (state === undefined) {
    return getConfig();
  }
  return state;
}
