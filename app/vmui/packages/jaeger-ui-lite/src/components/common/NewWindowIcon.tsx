import React from 'react';
import cx from 'classnames';
import { IoOpenOutline } from 'react-icons/io5';

import './NewWindowIcon.css';

type Props = {
  isLarge?: boolean;
};

export default function NewWindowIcon({ isLarge = false, ...rest }: Props) {
  const cls = cx('NewWindowIcon', { 'is-large': isLarge });
  return <IoOpenOutline className={cls} {...rest} data-testid="NewWindowIcon" />;
}
