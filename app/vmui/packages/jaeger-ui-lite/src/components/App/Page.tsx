import * as React from 'react';
import { Layout } from 'antd';
import cx from 'classnames';
import { connect } from 'react-redux';

import { ReduxState } from '../../types';
import { EmbeddedState } from '../../types/embedded';

import './Page.css';
import withRouteProps from '../../utils/withRouteProps';

type TProps = {
  children: React.ReactNode;
  embedded: EmbeddedState;
  pathname: string;
  search: string;
};

const { Header, Content } = Layout;

// export for tests
export const PageImpl: React.FC<TProps> = ({ children, embedded, pathname, search }) => {
  const contentCls = cx({ 'Page--content': true, 'Page--content--no-embedded': !embedded });

  return (
    <div>
      <Content className={contentCls}>{children}</Content>
    </div>
  );
};

// export for tests
export function mapStateToProps(state: ReduxState) {
  const { embedded } = state;
  const { pathname, search } = state.router.location;
  return { embedded, pathname, search };
}

export default connect(mapStateToProps)(withRouteProps(PageImpl));
