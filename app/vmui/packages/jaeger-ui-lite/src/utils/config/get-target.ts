import { getConfigValue } from './get-config';

export function getTargetEmptyOrBlank() {
  return getConfigValue('forbidNewPage') ? '' : '_blank';
}

export function getTargetBlankOrTop() {
  return getConfigValue('forbidNewPage') ? '_top' : '_blank';
}
