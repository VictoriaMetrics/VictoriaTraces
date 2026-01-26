import * as React from 'react';
import { List } from 'antd';

const { Item } = List;

export default function DetailList({ details }: { details: string[] }) {
  return (
    <List
      dataSource={details}
      renderItem={(s: string) => (
        <Item>
          <span>{s}</span>
        </Item>
      )}
    />
  );
}
